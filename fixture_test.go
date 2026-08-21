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

func TestFixturesJoin_InitializesRegistryForEmptyReceiver(t *testing.T) {
	var base axiom.Fixtures
	other := axiom.NewFixtures(
		axiom.WithFixture("user", func(*axiom.Config) (any, func(), error) {
			return "user", nil, nil
		}),
	)

	result := base.Join(other)

	assert.NotNil(t, result.Registry)
	assert.Equal(t, "user", getFixtureValue(t, result, "user"))
	assert.NotNil(t, result.Cache)
	assert.Empty(t, result.Cleanups)
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

func TestConfig_Test_RuntimeWrapEnclosesHooksAndFixtureCleanup(t *testing.T) {
	var order []string

	cfg := &axiom.Config{
		SubT: t,
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "before-test") },
			},
			AfterTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "after-test") },
			},
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(*axiom.Config) (any, func(), error) {
					order = append(order, "fixture-setup")
					return struct{}{}, func() {
						order = append(order, "fixture-cleanup")
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					next(current)
					order = append(order, "runtime-after")
				}
			}),
		),
	}

	cfg.Test(func(current *axiom.Config) {
		order = append(order, "action")
		axiom.GetFixture[struct{}](current, "report")
	})

	assert.Equal(t, []string{
		"runtime-before",
		"before-test",
		"action",
		"fixture-setup",
		"after-test",
		"fixture-cleanup",
		"runtime-after",
	}, order)
}

func TestConfig_Test_RuntimeWrapKeepsSetupAndTeardownMiddlewareActiveForFixtureLifecycle(t *testing.T) {
	var order []string

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "fixture lifecycle middleware"},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					order = append(order, "before-test")
					_ = axiom.GetFixture[string](cfg, "report")
				},
			},
			AfterTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "after-test") },
			},
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(cfg *axiom.Config) (any, func(), error) {
					cfg.Setup("create report", func() {
						order = append(order, "fixture-setup")
					})
					return "report", func() {
						cfg.Teardown("finalize report", func() {
							order = append(order, "fixture-cleanup")
						})
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					defer func() { order = append(order, "runtime-after") }()
					next(current)
				}
			}),
			axiom.WithRuntimeSetupWrap(func(_ string, next axiom.SetupAction) axiom.SetupAction {
				return func() {
					order = append(order, "setup-wrap-before")
					defer func() { order = append(order, "setup-wrap-after") }()
					next()
				}
			}),
			axiom.WithRuntimeTeardownWrap(func(_ string, next axiom.TeardownAction) axiom.TeardownAction {
				return func() {
					order = append(order, "teardown-wrap-before")
					defer func() { order = append(order, "teardown-wrap-after") }()
					next()
				}
			}),
		),
		SubT: t,
	}

	cfg.Test(func(*axiom.Config) {
		order = append(order, "body")
	})

	assert.Equal(t, []string{
		"runtime-before",
		"before-test",
		"setup-wrap-before",
		"fixture-setup",
		"setup-wrap-after",
		"body",
		"after-test",
		"teardown-wrap-before",
		"fixture-cleanup",
		"teardown-wrap-after",
		"runtime-after",
	}, order)
}

func TestConfig_Test_BodyPanicRunsDeferredLifecycleInsideRuntime(t *testing.T) {
	var order []string
	var events []axiom.Event
	fakeT := &testing.T{}

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "body panic"},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					order = append(order, "before-test")
					_ = axiom.GetFixture[string](cfg, "report")
				},
			},
			AfterTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "after-test") },
			},
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(*axiom.Config) (any, func(), error) {
					return "report", func() {
						order = append(order, "fixture-cleanup")
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					defer func() { order = append(order, "runtime-after") }()
					next(current)
				}
			}),
			axiom.WithRuntimeEventSink(func(event axiom.Event) {
				events = append(events, event)
			}),
		),
		SubT: fakeT,
	}

	assert.NotPanics(t, func() {
		cfg.Test(func(*axiom.Config) {
			order = append(order, "body")
			panic("body boom")
		})
	})

	assert.True(t, fakeT.Failed())
	assert.Equal(t, []string{
		"runtime-before",
		"before-test",
		"body",
		"after-test",
		"fixture-cleanup",
		"runtime-after",
	}, order)
	requireEventTypes(t, events,
		axiom.EventTypeCaseStart,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupFinish,
		axiom.EventTypeCasePanic,
		axiom.EventTypeCaseFinish,
	)
}

func TestConfig_Test_BeforeTestPanicStillRunsAfterTestAndFixtureCleanup(t *testing.T) {
	var order []string
	var events []axiom.Event
	fakeT := &testing.T{}
	actionCalled := false

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "before-test panic"},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					order = append(order, "before-test-setup")
					_ = axiom.GetFixture[string](cfg, "report")
				},
				func(*axiom.Config) {
					order = append(order, "before-test-panic")
					panic("before boom")
				},
			},
			AfterTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "after-test") },
			},
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(*axiom.Config) (any, func(), error) {
					return "report", func() {
						order = append(order, "fixture-cleanup")
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					defer func() { order = append(order, "runtime-after") }()
					next(current)
				}
			}),
			axiom.WithRuntimeEventSink(func(event axiom.Event) {
				events = append(events, event)
			}),
		),
		SubT: fakeT,
	}

	assert.NotPanics(t, func() {
		cfg.Test(func(*axiom.Config) {
			actionCalled = true
		})
	})

	assert.False(t, actionCalled)
	assert.True(t, fakeT.Failed())
	assert.Equal(t, []string{
		"runtime-before",
		"before-test-setup",
		"before-test-panic",
		"after-test",
		"fixture-cleanup",
		"runtime-after",
	}, order)
	requireEventTypes(t, events,
		axiom.EventTypeCaseStart,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupFinish,
		axiom.EventTypeCasePanic,
		axiom.EventTypeCaseFinish,
	)
}

func TestConfig_Test_FixtureCleanupPanicPropagatesAfterRuntimeUnwinds(t *testing.T) {
	var order []string
	var events []axiom.Event

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "cleanup panic"},
		Hooks: axiom.Hooks{
			AfterTest: []axiom.TestHook{
				func(*axiom.Config) { order = append(order, "after-test") },
			},
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(*axiom.Config) (any, func(), error) {
					return "report", func() {
						order = append(order, "fixture-cleanup")
						panic("cleanup boom")
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					defer func() { order = append(order, "runtime-after") }()
					next(current)
				}
			}),
			axiom.WithRuntimeEventSink(func(event axiom.Event) {
				events = append(events, event)
			}),
		),
		SubT: t,
	}

	assert.PanicsWithValue(t, "cleanup boom", func() {
		cfg.Test(func(current *axiom.Config) {
			order = append(order, "body")
			_ = axiom.GetFixture[string](current, "report")
		})
	})

	assert.Equal(t, []string{
		"runtime-before",
		"body",
		"after-test",
		"fixture-cleanup",
		"runtime-after",
	}, order)
	requireEventTypes(t, events,
		axiom.EventTypeCaseStart,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupPanic,
	)
}

func TestConfig_Test_DeferredLifecyclePreservesOriginalPanicValue(t *testing.T) {
	type panicValue struct {
		phase string
	}
	original := &panicValue{phase: "cleanup"}

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "preserve lifecycle panic"},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(*axiom.Config) (any, func(), error) {
					return "report", func() { panic(original) }, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
		SubT: t,
	}

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		cfg.Test(func(current *axiom.Config) {
			_ = axiom.GetFixture[string](current, "report")
		})
	}()

	assert.Same(t, original, recovered)
}

func TestConfig_Test_DeferredLifecyclePanicPrecedence(t *testing.T) {
	tests := []struct {
		name           string
		bodyPanic      bool
		afterPanic     bool
		cleanupPanic   bool
		wantRecovered  any
		wantFailed     bool
		wantCasePanic  bool
		wantCaseFinish bool
	}{
		{
			name:           "normal lifecycle",
			wantCaseFinish: true,
		},
		{
			name:           "body panic",
			bodyPanic:      true,
			wantFailed:     true,
			wantCasePanic:  true,
			wantCaseFinish: true,
		},
		{
			name:          "after-test panic",
			afterPanic:    true,
			wantRecovered: "after-test boom",
		},
		{
			name:          "cleanup panic",
			cleanupPanic:  true,
			wantRecovered: "cleanup boom",
		},
		{
			name:          "after-test panic replaces body panic",
			bodyPanic:     true,
			afterPanic:    true,
			wantRecovered: "after-test boom",
		},
		{
			name:          "cleanup panic replaces body panic",
			bodyPanic:     true,
			cleanupPanic:  true,
			wantRecovered: "cleanup boom",
		},
		{
			name:          "cleanup panic replaces after-test panic",
			afterPanic:    true,
			cleanupPanic:  true,
			wantRecovered: "cleanup boom",
		},
		{
			name:          "cleanup panic replaces body and after-test panics",
			bodyPanic:     true,
			afterPanic:    true,
			cleanupPanic:  true,
			wantRecovered: "cleanup boom",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			var events []axiom.Event
			fakeT := &testing.T{}

			cfg := &axiom.Config{
				Case: &axiom.Case{Name: test.name},
				Hooks: axiom.Hooks{
					AfterTest: []axiom.TestHook{
						func(*axiom.Config) {
							order = append(order, "after-test")
							if test.afterPanic {
								panic("after-test boom")
							}
						},
					},
				},
				Fixtures: axiom.Fixtures{
					Registry: map[string]axiom.Fixture{
						"report": func(*axiom.Config) (any, func(), error) {
							return "report", func() {
								order = append(order, "fixture-cleanup")
								if test.cleanupPanic {
									panic("cleanup boom")
								}
							}, nil
						},
					},
					Cache: map[string]axiom.FixtureResult{},
				},
				Runtime: axiom.NewRuntime(
					axiom.WithRuntimeEventSink(func(event axiom.Event) {
						events = append(events, event)
					}),
				),
				SubT: fakeT,
			}

			var recovered any
			func() {
				defer func() { recovered = recover() }()
				cfg.Test(func(current *axiom.Config) {
					_ = axiom.GetFixture[string](current, "report")
					order = append(order, "body")
					if test.bodyPanic {
						panic("body boom")
					}
				})
			}()

			assert.Equal(t, test.wantRecovered, recovered)
			assert.Equal(t, test.wantFailed, fakeT.Failed())
			assert.Equal(t, []string{"body", "after-test", "fixture-cleanup"}, order)

			containsEvent := func(eventType axiom.EventType) bool {
				for _, event := range events {
					if event.Type == eventType {
						return true
					}
				}
				return false
			}
			assert.Equal(t, test.wantCasePanic, containsEvent(axiom.EventTypeCasePanic))
			assert.Equal(t, test.wantCaseFinish, containsEvent(axiom.EventTypeCaseFinish))
		})
	}
}

func TestConfig_Test_SkipNowRunsDeferredLifecycleInsideRuntime(t *testing.T) {
	var order []string
	var events []axiom.Event

	ok := t.Run("skipped attempt", func(st *testing.T) {
		cfg := &axiom.Config{
			Case: &axiom.Case{Name: "runtime skip"},
			Hooks: axiom.Hooks{
				AfterTest: []axiom.TestHook{
					func(*axiom.Config) { order = append(order, "after-test") },
				},
			},
			Fixtures: axiom.Fixtures{
				Registry: map[string]axiom.Fixture{
					"report": func(*axiom.Config) (any, func(), error) {
						return "report", func() {
							order = append(order, "fixture-cleanup")
						}, nil
					},
				},
				Cache: map[string]axiom.FixtureResult{},
			},
			Runtime: axiom.NewRuntime(
				axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
					return func(current *axiom.Config) {
						order = append(order, "runtime-before")
						defer func() { order = append(order, "runtime-after") }()
						next(current)
					}
				}),
				axiom.WithRuntimeEventSink(func(event axiom.Event) {
					events = append(events, event)
				}),
			),
			SubT: st,
		}

		cfg.Test(func(current *axiom.Config) {
			_ = axiom.GetFixture[string](current, "report")
			order = append(order, "body")
			current.T().SkipNow()
			order = append(order, "after-skip")
		})
	})

	assert.True(t, ok, "a skipped subtest must not fail its parent")
	assert.Equal(t, []string{
		"runtime-before",
		"body",
		"after-test",
		"fixture-cleanup",
		"runtime-after",
	}, order)
	requireEventTypes(t, events,
		axiom.EventTypeCaseStart,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupFinish,
		axiom.EventTypeCaseFinish,
	)
}

func TestConfig_Test_DrainsFixtureCleanups_WhenAfterTestPanics(t *testing.T) {
	var order []string
	var events []axiom.Event

	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "panic-after-test"},
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
				func(cfg *axiom.Config) {
					order = append(order, "after")
					panic("after boom")
				},
			},
		},
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(current *axiom.Config) {
					order = append(order, "runtime-before")
					defer func() { order = append(order, "runtime-after") }()
					next(current)
				}
			}),
			axiom.WithRuntimeEventSink(func(event axiom.Event) {
				events = append(events, event)
			}),
		),
		SubT: t,
	}

	assert.PanicsWithValue(t, "after boom", func() {
		cfg.Test(func(c *axiom.Config) {
			_ = axiom.GetFixture[string](c, "db")
			order = append(order, "body")
		})
	})

	assert.Equal(t, []string{
		"runtime-before",
		"body",
		"after",
		"fixture-cleanup",
		"runtime-after",
	}, order,
		"fixture cleanup must still run after a panicking AfterTest hook")
	assert.Empty(t, cfg.Fixtures.Cleanups, "cleanups must be drained even when AfterTest panics")
	requireEventTypes(t, events,
		axiom.EventTypeCaseStart,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
		axiom.EventTypeFixtureCleanupStart,
		axiom.EventTypeFixtureCleanupFinish,
	)
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

func TestFixturesCopy_DeepCopiesCleanups(t *testing.T) {
	var calls []string
	f := axiom.Fixtures{
		Cleanups: []axiom.FixtureCleanup{
			func(*axiom.Config) { calls = append(calls, "original") },
		},
	}

	cp := f.Copy()
	cp.Cleanups[0] = func(*axiom.Config) { calls = append(calls, "copy") }

	f.Cleanups[0](nil)
	cp.Cleanups[0](nil)

	assert.Equal(t, []string{"original", "copy"}, calls)
}
