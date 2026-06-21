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

	assert.Len(t, cfg.Hooks.AfterTest, 1)
	requireEventTypes(t, events,
		axiom.EventTypeFixtureSetupStart,
		axiom.EventTypeFixtureSetupFinish,
	)

	cfg.Hooks.AfterTest[0](cfg)
	assert.True(t, cleanupCalled, "cleanup must be executed")
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
	assert.Empty(t, cfg.Hooks.AfterTest, "nil cleanup must not register a cleanup hook")
}

func TestUseFixtures_AddsCleanupToAfterTest(t *testing.T) {
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

	assert.Len(t, cfg.Hooks.AfterTest, 1, "cleanup hook must be registered")

	cfg.Hooks.AfterTest[0](cfg)
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
		cfg.Hooks.AfterTest[0](cfg)
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
