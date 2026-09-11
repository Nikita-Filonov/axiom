package axiom_test

import (
	"errors"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestResourceKey_RegisterAndGet(t *testing.T) {
	key := axiom.NewResourceKey[int]("answer")
	runner := axiom.NewRunner(
		axiom.WithRunnerResourceKey(key, func(r *axiom.Runner) (int, func(), error) {
			return 42, nil, nil
		}),
	)

	assert.Equal(t, 42, key.Get(runner))
}

func TestResourceKey_ConstructsOnce(t *testing.T) {
	calls := 0
	key := axiom.NewResourceKey[int]("answer")
	runner := axiom.NewRunner(
		axiom.WithRunnerResourceKey(key, func(r *axiom.Runner) (int, func(), error) {
			calls++
			return 42, nil, nil
		}),
	)

	_ = key.Get(runner)
	_ = key.Get(runner)

	assert.Equal(t, 1, calls, "resource must be constructed exactly once")
}

func TestResourceKey_InteropWithStringRegistry(t *testing.T) {
	key := axiom.NewResourceKey[string]("token")
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("token", func(r *axiom.Runner) (any, func(), error) {
			return "secret", nil, nil
		}),
	)

	assert.Equal(t, "secret", key.Get(runner))
}

func TestResourceKey_RunsCleanupThroughTypedRegistration(t *testing.T) {
	cleaned := false
	key := axiom.NewResourceKey[string]("pool")
	runner := axiom.NewRunner(
		axiom.WithRunnerResourceKey(key, func(r *axiom.Runner) (string, func(), error) {
			return "pool", func() { cleaned = true }, nil
		}),
	)

	_ = key.Get(runner)
	runner.Resources.Teardown(runner)

	assert.True(t, cleaned, "resource cleanup must run on teardown")
}

func TestResourceKey_TryGetMissing(t *testing.T) {
	key := axiom.NewResourceKey[int]("missing")
	runner := axiom.NewRunner()

	_, err := key.TryGet(runner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResourceKey_GetPanicsOnMissing(t *testing.T) {
	key := axiom.NewResourceKey[string]("missing")
	runner := axiom.NewRunner()

	assert.PanicsWithError(t, `resource "missing" not found`, func() {
		_ = key.Get(runner)
	})
}

func TestResourceKey_PanicsWhenNameIsEmpty(t *testing.T) {
	assert.PanicsWithValue(t, "resource: key name must not be empty", func() {
		_ = axiom.NewResourceKey[int]("")
	})
}

func TestResourceKey_ZeroValueGetPanics(t *testing.T) {
	var key axiom.ResourceKey[int]

	assert.PanicsWithValue(t, "resource: key must be created with NewResourceKey", func() {
		_ = key.Get(axiom.NewRunner())
	})
}

func TestWithRunnerResourceKey_PanicsOnNilConstructor(t *testing.T) {
	key := axiom.NewResourceKey[int]("x")

	assert.PanicsWithValue(t, "resource: nil constructor", func() {
		_ = axiom.WithRunnerResourceKey(key, nil)
	})
}

func TestResourceDef_SelfRegistersAndGet(t *testing.T) {
	def := axiom.DefineResource("answer", func(r *axiom.Runner) (int, func(), error) {
		return 42, nil, nil
	})
	runner := axiom.NewRunner(axiom.WithRunnerResources(def))

	assert.Equal(t, 42, def.Get(runner))
	assert.Equal(t, "answer", def.Name())
	assert.Equal(t, "answer", def.Key().Name())
}

func TestResourceDef_TryGetPropagatesConstructorError(t *testing.T) {
	def := axiom.DefineResource("boom", func(r *axiom.Runner) (int, func(), error) {
		return 0, nil, errors.New("boom")
	})
	runner := axiom.NewRunner(axiom.WithRunnerResources(def))

	_, err := def.TryGet(runner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestDefineResource_PanicsOnNilConstructor(t *testing.T) {
	assert.PanicsWithValue(t, "resource: nil constructor", func() {
		_ = axiom.DefineResource[int]("x", nil)
	})
}
