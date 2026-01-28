package axiom_test

import (
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

func TestGetFixture_HappyPath(t *testing.T) {
	callCount := 0
	cleanupCalled := false

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
		SubT:     t,
	}

	v := axiom.GetFixture[int](cfg, "num")
	assert.Equal(t, 42, v)
	assert.Equal(t, 1, callCount, "fixture must be executed exactly once")

	v2 := axiom.GetFixture[int](cfg, "num")
	assert.Equal(t, 42, v2)
	assert.Equal(t, 1, callCount, "fixture must NOT run twice")

	assert.Len(t, cfg.Hooks.AfterTest, 1)

	cfg.Hooks.AfterTest[0](cfg)
	assert.True(t, cleanupCalled, "cleanup must be executed")
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
