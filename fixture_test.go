package axiom_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

// helper to extract a fixture value without full config
func getFixtureValue(t *testing.T, f axiom.Fixtures, name string) any {
	cfg := &axiom.Config{Fixtures: f}
	fx, ok := f.Registry[name]
	assert.True(t, ok)
	val, _, _ := fx(cfg)
	return val
}

func TestNewFixtures_Defaults(t *testing.T) {
	f := axiom.NewFixtures()

	assert.Nil(t, f.Registry)
	assert.Nil(t, f.Cache)
}

func TestWithFixture(t *testing.T) {
	f := axiom.NewFixtures(
		axiom.WithFixture("user", func(cfg *axiom.Config) (any, func(), error) {
			return 123, nil, nil
		}),
	)

	assert.Contains(t, f.Registry, "user")
}

func TestWithFixturesMap(t *testing.T) {
	m := map[string]axiom.Fixture{
		"a": func(cfg *axiom.Config) (any, func(), error) { return "A", nil, nil },
		"b": func(cfg *axiom.Config) (any, func(), error) { return "B", nil, nil },
	}

	f := axiom.NewFixtures(
		axiom.WithFixturesMap(m),
	)

	assert.Equal(t, 2, len(f.Registry))
}

func TestFixturesJoin(t *testing.T) {
	f1 := axiom.NewFixtures(
		axiom.WithFixture("a", func(cfg *axiom.Config) (any, func(), error) { return "A1", nil, nil }),
		axiom.WithFixture("b", func(cfg *axiom.Config) (any, func(), error) { return "B1", nil, nil }),
	)

	f2 := axiom.NewFixtures(
		axiom.WithFixture("b", func(cfg *axiom.Config) (any, func(), error) { return "B2", nil, nil }), // overrides
		axiom.WithFixture("c", func(cfg *axiom.Config) (any, func(), error) { return "C", nil, nil }),
	)

	result := f1.Join(f2)

	// Registry merge
	assert.Equal(t, "A1", getFixtureValue(t, result, "a"))
	assert.Equal(t, "B2", getFixtureValue(t, result, "b")) // overridden
	assert.Equal(t, "C", getFixtureValue(t, result, "c"))

	// Cache must always be empty in Join result
	assert.Empty(t, result.Cache)
}

func TestFixturesJoin_ResetsCacheFromBothSides(t *testing.T) {
	f1 := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"a": func(cfg *axiom.Config) (any, func(), error) { return "A", nil, nil },
		},
		Cache: map[string]axiom.FixtureResult{
			"a": {Value: "cached-A"},
		},
	}
	f2 := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"b": func(cfg *axiom.Config) (any, func(), error) { return "B", nil, nil },
		},
		Cache: map[string]axiom.FixtureResult{
			"b": {Value: "cached-B"},
		},
	}

	result := f1.Join(f2)

	assert.Contains(t, result.Registry, "a")
	assert.Contains(t, result.Registry, "b")
	assert.Empty(t, result.Cache)
}

func TestFixturesJoin_DoesNotMutateOriginalFixtures(t *testing.T) {
	f1 := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"a": func(cfg *axiom.Config) (any, func(), error) { return "A", nil, nil },
		},
		Cache: map[string]axiom.FixtureResult{
			"a": {Value: "cached-A"},
		},
	}
	f2 := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"b": func(cfg *axiom.Config) (any, func(), error) { return "B", nil, nil },
		},
		Cache: map[string]axiom.FixtureResult{
			"b": {Value: "cached-B"},
		},
	}

	result := f1.Join(f2)
	result.Cache["x"] = axiom.FixtureResult{Value: "X"}
	result.Registry["c"] = func(cfg *axiom.Config) (any, func(), error) { return "C", nil, nil }

	assert.NotContains(t, f1.Registry, "c")
	assert.NotContains(t, f2.Registry, "c")
	assert.NotContains(t, f1.Cache, "x")
	assert.NotContains(t, f2.Cache, "x")
}

func TestGetFixture_HappyPath(t *testing.T) {
	callCount := 0
	cleanupCalled := false
	var events []axiom.Event

	fixtures := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"num": func(cfg *axiom.Config) (any, func(), error) {
				callCount++
				return 42, func() { cleanupCalled = true }, nil
			},
		},
		Cache: map[string]axiom.FixtureResult{},
	}

	cfg := &axiom.Config{
		Fixtures: fixtures,
		Hooks:    axiom.Hooks{},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: t,
	}

	v := axiom.GetFixture[int](cfg, "num")
	assert.Equal(t, 42, v)
	assert.Equal(t, 1, callCount, "fixture must be executed exactly once")

	v2 := axiom.GetFixture[int](cfg, "num")
	assert.Equal(t, 42, v2)
	assert.Equal(t, 1, callCount, "fixture must NOT run twice")

	assert.Empty(t, cfg.Hooks.AfterTest, "cleanups must not pollute user AfterTest hooks")
	assert.Len(t, cfg.Fixtures.Cleanups, 1)
	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
	)

	cfg.Fixtures.Teardown(cfg)
	assert.True(t, cleanupCalled, "cleanup must be executed")
	assert.Empty(t, cfg.Fixtures.Cleanups, "cleanups must be drained")
	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupFinish,
	)
}

func TestUseFixtures_ExecutesAllAndCaches(t *testing.T) {
	calls := map[string]int{}

	fixtures := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"a": func(cfg *axiom.Config) (any, func(), error) {
				calls["a"]++
				return "A", nil, nil
			},
			"b": func(cfg *axiom.Config) (any, func(), error) {
				calls["b"]++
				return "B", nil, nil
			},
		},
		Cache: map[string]axiom.FixtureResult{},
	}

	cfg := &axiom.Config{
		Fixtures: fixtures,
		Hooks:    axiom.Hooks{},
		SubT:     t,
	}

	hook := axiom.UseFixtures("a", "b")
	hook(cfg)

	assert.Equal(t, 1, calls["a"])
	assert.Equal(t, 1, calls["b"])
	assert.Contains(t, cfg.Fixtures.Cache, "a")
	assert.Contains(t, cfg.Fixtures.Cache, "b")
}

func TestUseFixtures_DoesNotExecuteTwice(t *testing.T) {
	callCount := 0

	fixtures := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"num": func(cfg *axiom.Config) (any, func(), error) {
				callCount++
				return 42, nil, nil
			},
		},
		Cache: map[string]axiom.FixtureResult{},
	}

	cfg := &axiom.Config{
		Fixtures: fixtures,
		Hooks:    axiom.Hooks{},
		SubT:     t,
	}

	hook := axiom.UseFixtures("num")

	hook(cfg)
	hook(cfg)

	assert.Equal(t, 1, callCount, "fixture must be executed only once due to cache")
	assert.Empty(t, cfg.Fixtures.Cleanups, "nil cleanup must not register a cleanup")
}

func TestUseFixtures_RegistersCleanupOnFixturesStack(t *testing.T) {
	cleanupCalled := false

	fixtures := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"x": func(cfg *axiom.Config) (any, func(), error) {
				return "X", func() { cleanupCalled = true }, nil
			},
		},
		Cache: map[string]axiom.FixtureResult{},
	}

	cfg := &axiom.Config{
		Fixtures: fixtures,
		Hooks:    axiom.Hooks{},
		SubT:     t,
	}

	axiom.UseFixtures("x")(cfg)

	assert.Empty(t, cfg.Hooks.AfterTest, "fixture cleanup must not touch user AfterTest hooks")
	assert.Len(t, cfg.Fixtures.Cleanups, 1, "cleanup must be registered on the fixture stack")

	cfg.Fixtures.Teardown(cfg)
	assert.True(t, cleanupCalled, "cleanup must be executed")
}

func TestGetFixture_Panic_NilConfig(t *testing.T) {
	assert.PanicsWithValue(t, "fixture: nil config", func() {
		_ = axiom.GetFixture[string](nil, "x")
	})
}

func TestGetFixture_Missing_EmitsFailedFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{},
			Cache:    map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "missing") })

	requireEventTypes(t, events, axiom.EventTypeFixtureSetupFailed)
	assert.Equal(t, "missing", events[0].Name)
	assert.Equal(t, "not found", events[0].Message)
}

func TestGetFixture_NilFixture_EmitsFailedFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{"x": nil},
			Cache:    map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	requireEventTypes(t, events, axiom.EventTypeFixtureSetupFailed)
	assert.Equal(t, "nil fixture", events[0].Message)
}

func TestGetFixture_FactoryError_EmitsFailedFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"x": func(cfg *axiom.Config) (any, func(), error) {
					return nil, nil, fmt.Errorf("boom")
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFailed,
	)
	assert.Equal(t, "boom", events[1].Message)
}

func TestGetFixture_WrongType_EmitsFailedFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"x": func(cfg *axiom.Config) (any, func(), error) {
					return "string", nil, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFailed,
	)
	assert.Equal(t, "unexpected type", events[1].Message)
}

func TestGetFixture_CleanupPanic_EmitsPanicFact(t *testing.T) {
	var events []axiom.Event
	fixtures := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"x": func(cfg *axiom.Config) (any, func(), error) {
				return "X", func() { panic("boom") }, nil
			},
		},
		Cache: map[string]axiom.FixtureResult{},
	}

	cfg := &axiom.Config{
		Fixtures: fixtures,
		Hooks:    axiom.Hooks{},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		SubT: t,
	}

	assert.Equal(t, "X", axiom.GetFixture[string](cfg, "x"))

	assert.PanicsWithValue(t, "boom", func() {
		cfg.Fixtures.Teardown(cfg)
	})

	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupPanic,
	)
	assert.Equal(t, "boom", events[3].Message)
}

func runFixtureFatal(fn func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	<-done
}

func TestFixturesTeardown_LIFOOrder(t *testing.T) {
	var order []string

	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"db": func(cfg *axiom.Config) (any, func(), error) {
					return "db", func() { order = append(order, "db") }, nil
				},
				"user": func(cfg *axiom.Config) (any, func(), error) {
					_ = axiom.GetFixture[string](cfg, "db")
					return "user", func() { order = append(order, "user") }, nil
				},
				"session": func(cfg *axiom.Config) (any, func(), error) {
					_ = axiom.GetFixture[string](cfg, "user")
					return "session", func() { order = append(order, "session") }, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Hooks: axiom.Hooks{},
		SubT:  t,
	}

	_ = axiom.GetFixture[string](cfg, "session")

	cfg.Fixtures.Teardown(cfg)
	assert.Equal(t, []string{"session", "user", "db"}, order, "cleanups must run in reverse setup order")
}

func TestFixturesTeardown_RunsAfterUserAfterTestHooks(t *testing.T) {
	var order []string

	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"db": func(cfg *axiom.Config) (any, func(), error) {
					return "db", func() { order = append(order, "fixture-cleanup") }, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Hooks: axiom.Hooks{
			AfterTest: []axiom.TestHook{
				func(cfg *axiom.Config) { order = append(order, "user-after-test") },
			},
		},
		SubT: t,
	}

	_ = axiom.GetFixture[string](cfg, "db")

	cfg.Hooks.ApplyAfterTest(cfg)
	cfg.Fixtures.Teardown(cfg)

	assert.Equal(t, []string{"user-after-test", "fixture-cleanup"}, order,
		"user AfterTest hooks must observe live fixtures before cleanups run")
}

func TestGetFixture_CachedValueWrongType_EmitsFailedFact(t *testing.T) {
	var events []axiom.Event
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{},
			Cache: map[string]axiom.FixtureResult{
				"x": {Value: "string"},
			},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) { events = append(events, e) }),
		),
		SubT: &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	requireEventTypes(t, events, axiom.EventTypeFixtureSetupFailed)
	assert.Equal(t, "unexpected type", events[0].Message)
}

func TestGetFixture_FactoryError_DoesNotRegisterCleanup(t *testing.T) {
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"x": func(cfg *axiom.Config) (any, func(), error) {
					return nil, func() { t.Fatal("cleanup must not be registered on factory error") }, fmt.Errorf("boom")
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(),
		SubT:    &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	assert.Empty(t, cfg.Fixtures.Cleanups,
		"cleanup must not be registered when factory returned an error")
	assert.NotContains(t, cfg.Fixtures.Cache, "x",
		"value must not be cached when factory returned an error")
}

func TestGetFixture_NilCleanup_DoesNotRegisterAnything(t *testing.T) {
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"x": func(cfg *axiom.Config) (any, func(), error) {
					return "X", nil, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		SubT: t,
	}

	v := axiom.GetFixture[string](cfg, "x")
	assert.Equal(t, "X", v)

	assert.Empty(t, cfg.Fixtures.Cleanups,
		"nil cleanup must not be appended to the cleanup stack")
	assert.Contains(t, cfg.Fixtures.Cache, "x",
		"value must still be cached even with nil cleanup")
}

func TestGetFixture_WrongTypeWithNonNilCleanup_StillRegistersCleanup(t *testing.T) {
	// If the factory created a resource (e.g. opened a connection) and returned
	// a non-nil cleanup, the framework must still register that cleanup even if
	// the caller asked for the wrong type — otherwise we leak the resource.
	cleanupCalled := false
	cfg := &axiom.Config{
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"x": func(cfg *axiom.Config) (any, func(), error) {
					return "string", func() { cleanupCalled = true }, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(),
		SubT:    &testing.T{},
	}

	runFixtureFatal(func() { _ = axiom.GetFixture[int](cfg, "x") })

	assert.Len(t, cfg.Fixtures.Cleanups, 1,
		"cleanup must be registered even when caller-side type assertion fails")
	assert.NotContains(t, cfg.Fixtures.Cache, "x",
		"value with mismatched type must not be cached")

	cfg.Fixtures.Teardown(cfg)
	assert.True(t, cleanupCalled,
		"registered cleanup must run to free the resource the factory created")
}

func TestConfig_Test_DrainsFixtureCleanups_AfterAfterTestHooks(t *testing.T) {
	// Lifecycle contract: Config.Test() must run AfterTest hooks first, then
	// drain Fixtures.Cleanups. Hooks must observe live fixtures; cleanups must
	// observe a cleared state afterwards.
	var order []string

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "lifecycle"},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"db": func(cfg *axiom.Config) (any, func(), error) {
					return "db", func() { order = append(order, "fixture-cleanup-db") }, nil
				},
				"client": func(cfg *axiom.Config) (any, func(), error) {
					_ = axiom.GetFixture[string](cfg, "db")
					return "client", func() { order = append(order, "fixture-cleanup-client") }, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					_ = axiom.GetFixture[string](cfg, "client")
					order = append(order, "before")
				},
			},
			AfterTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					assert.Len(t, cfg.Fixtures.Cleanups, 2,
						"AfterTest must observe fixtures still alive")
					order = append(order, "after")
				},
			},
		},
		SubT: t,
	}

	cfg.Test(func(c *axiom.Config) {
		order = append(order, "body")
	})

	assert.Equal(t, []string{
		"before",
		"body",
		"after",
		"fixture-cleanup-client",
		"fixture-cleanup-db",
	}, order, "Config.Test must drain cleanups LIFO after AfterTest hooks")

	assert.Empty(t, cfg.Fixtures.Cleanups, "cleanups must be drained after Config.Test")
}

func TestFixturesCopy_DeepCopyMaps(t *testing.T) {
	f := axiom.Fixtures{
		Registry: map[string]axiom.Fixture{
			"x": func(cfg *axiom.Config) (any, func(), error) { return 1, nil, nil },
		},
		Cache: map[string]axiom.FixtureResult{
			"x": {Value: 1},
		},
	}

	cp := f.Copy()
	cp.Registry["y"] = func(cfg *axiom.Config) (any, func(), error) { return 2, nil, nil }
	cp.Cache["y"] = axiom.FixtureResult{Value: 2}

	assert.NotContains(t, f.Registry, "y")
	assert.NotContains(t, f.Cache, "y")
}
