package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextKey_ValueAndGet(t *testing.T) {
	key := axiom.NewContextKey[int]("user-id")
	ctx := axiom.NewContext(key.Value(42))

	assert.Equal(t, 42, key.Get(&ctx))
	assert.Equal(t, "user-id", key.Name())
}

func TestContextKey_TryGet(t *testing.T) {
	key := axiom.NewContextKey[string]("token")
	ctx := axiom.NewContext(key.Value("abc"))

	got, ok := key.TryGet(&ctx)

	require.True(t, ok)
	assert.Equal(t, "abc", got)
}

func TestContextKey_InteropWithStringData(t *testing.T) {
	// A typed key must resolve a value stored through the plain string API, and a
	// value stored through the key must be readable through the string API,
	// proving the layer is purely additive.
	key := axiom.NewContextKey[string]("env")

	fromString := axiom.NewContext(axiom.WithContextData("env", "staging"))
	assert.Equal(t, "staging", key.Get(&fromString))

	fromKey := axiom.NewContext(key.Value("prod"))
	raw, ok := axiom.GetContextValue[string](&fromKey, "env")
	require.True(t, ok)
	assert.Equal(t, "prod", raw)
}

func TestContextKey_TryGetMissingReturnsFalse(t *testing.T) {
	key := axiom.NewContextKey[int]("absent")
	ctx := axiom.NewContext()

	got, ok := key.TryGet(&ctx)

	assert.False(t, ok)
	assert.Zero(t, got)
}

func TestContextKey_TryGetWrongTypeReturnsFalse(t *testing.T) {
	key := axiom.NewContextKey[int]("value")
	ctx := axiom.NewContext(axiom.WithContextData("value", "not-an-int"))

	got, ok := key.TryGet(&ctx)

	assert.False(t, ok)
	assert.Zero(t, got)
}

func TestContextKey_GetMissingPanics(t *testing.T) {
	key := axiom.NewContextKey[int]("missing")
	ctx := axiom.NewContext()

	assert.PanicsWithValue(t, `context: expected value for key "missing" of type int`, func() {
		_ = key.Get(&ctx)
	})
}

func TestContextKey_PanicsWhenNameIsEmpty(t *testing.T) {
	assert.PanicsWithValue(t, "context: key name must not be empty", func() {
		_ = axiom.NewContextKey[int]("")
	})
}

func TestContextKey_ZeroValuePanics(t *testing.T) {
	const message = "context: key must be created with NewContextKey"

	t.Run("value", func(t *testing.T) {
		var key axiom.ContextKey[int]
		assert.PanicsWithValue(t, message, func() { _ = key.Value(1) })
	})

	t.Run("get", func(t *testing.T) {
		var key axiom.ContextKey[int]
		ctx := axiom.NewContext()
		assert.PanicsWithValue(t, message, func() { _ = key.Get(&ctx) })
	})

	t.Run("try-get", func(t *testing.T) {
		var key axiom.ContextKey[int]
		ctx := axiom.NewContext()
		assert.PanicsWithValue(t, message, func() { _, _ = key.TryGet(&ctx) })
	})
}
