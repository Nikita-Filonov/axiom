package axiom_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheKey_GetSetDelete(t *testing.T) {
	for name, cache := range map[string]*axiom.Cache{
		"constructor": axiom.NewCache(),
		"zero value":  {},
	} {
		t.Run(name, func(t *testing.T) {
			key := axiom.NewCacheKey[string]("value")
			assert.Equal(t, "value", key.Name())
			value, ok := key.Get(cache)
			assert.False(t, ok)
			assert.Empty(t, value)
			key.Delete(cache)

			key.Set(cache, "first")
			value, ok = key.Get(cache)
			require.True(t, ok)
			assert.Equal(t, "first", value)

			key.Set(cache, "second")
			value, ok = key.Get(cache)
			require.True(t, ok)
			assert.Equal(t, "second", value)

			key.Delete(cache)
			value, ok = key.Get(cache)
			assert.False(t, ok)
			assert.Empty(t, value)
		})
	}
}

func TestCacheKey_IdentityIncludesNameAndType(t *testing.T) {
	cache := axiom.NewCache()
	first := axiom.NewCacheKey[string]("same")
	same := axiom.NewCacheKey[string]("same")
	otherType := axiom.NewCacheKey[int]("same")
	otherName := axiom.NewCacheKey[string]("other")
	first.Set(cache, "first")
	same.Set(cache, "updated")
	otherType.Set(cache, 42)
	otherName.Set(cache, "other")

	value, ok := first.Get(cache)
	require.True(t, ok)
	assert.Equal(t, "updated", value)
	number, ok := otherType.Get(cache)
	require.True(t, ok)
	assert.Equal(t, 42, number)

	same.Delete(cache)
	_, ok = first.Get(cache)
	assert.False(t, ok)
	_, ok = otherType.Get(cache)
	assert.True(t, ok)
	_, ok = otherName.Get(cache)
	assert.True(t, ok)
	_, ok = otherType.Get(axiom.NewCache())
	assert.False(t, ok, "separate caches must be independent")
}

func TestCacheKey_PreservesValueTypesAndNil(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) { checkCacheValue[any](t, nil) })
	t.Run("nil pointer", func(t *testing.T) { checkCacheValue[*int](t, nil) })
	t.Run("nil slice", func(t *testing.T) { checkCacheValue[[]int](t, nil) })
	t.Run("nil map", func(t *testing.T) { checkCacheValue[map[string]int](t, nil) })
	t.Run("slice", func(t *testing.T) { checkCacheValue(t, []int{1, 2}) })
	t.Run("map", func(t *testing.T) { checkCacheValue(t, map[string]int{"id": 1}) })
	t.Run("zero", func(t *testing.T) { checkCacheValue(t, 0) })
	t.Run("interface with slice", func(t *testing.T) { checkCacheValue[any](t, []int{1, 2}) })
	t.Run("interface with typed nil", func(t *testing.T) {
		var pointer *int
		checkCacheValue[any](t, pointer)
	})
}

func checkCacheValue[T any](t *testing.T, value T) {
	t.Helper()
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[T]("value")
	key.Set(cache, value)
	got, ok := key.Get(cache)
	require.True(t, ok)
	assert.Equal(t, value, got)
	key.Delete(cache)

	calls := 0
	create := func() (T, error) {
		calls++
		return value, nil
	}
	for range 2 {
		got, err := key.GetOrCreate(t.Context(), cache, create)
		require.NoError(t, err)
		assert.Equal(t, value, got)
	}
	assert.Equal(t, 1, calls)
	got, ok = key.Get(cache)
	require.True(t, ok)
	assert.Equal(t, value, got)
}

func TestCacheKey_ValuesAreNotCopiedOrClosed(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[*cacheCloseable]("value")
	value := &cacheCloseable{}
	key.Set(cache, value)
	got, err := key.GetOrCreate(t.Context(), cache, func() (*cacheCloseable, error) {
		t.Fatal("constructor called on a hit")
		return nil, nil
	})
	require.NoError(t, err)
	assert.Same(t, value, got)
	replacement := &cacheCloseable{}
	key.Set(cache, replacement)
	key.Delete(cache)
	assert.False(t, value.closed)
	assert.False(t, replacement.closed)
}

type cacheCloseable struct{ closed bool }

func (c *cacheCloseable) Close() error {
	c.closed = true
	return nil
}

func TestCacheKey_GetOrCreate_DoesNotCacheErrorsOrPartialValues(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[string]("value")
	failure := errors.New("creation failed")
	value, err := key.GetOrCreate(t.Context(), cache, func() (string, error) {
		return "partial", failure
	})
	assert.Same(t, failure, err)
	assert.Empty(t, value)
	_, ok := key.Get(cache)
	assert.False(t, ok)

	value, err = key.GetOrCreate(t.Context(), cache, func() (string, error) {
		return "complete", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "complete", value)
}

func TestCacheKey_ValidatesArguments(t *testing.T) {
	assert.PanicsWithValue(t, "cache: key name must not be empty", func() {
		axiom.NewCacheKey[int]("")
	})
	operations := map[string]func(axiom.CacheKey[int], *axiom.Cache){
		"Get":    func(k axiom.CacheKey[int], c *axiom.Cache) { k.Get(c) },
		"Set":    func(k axiom.CacheKey[int], c *axiom.Cache) { k.Set(c, 1) },
		"Delete": func(k axiom.CacheKey[int], c *axiom.Cache) { k.Delete(c) },
		"GetOrCreate": func(k axiom.CacheKey[int], c *axiom.Cache) {
			_, _ = k.GetOrCreate(t.Context(), c, func() (int, error) { return 1, nil })
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			assert.PanicsWithValue(t, "cache: key must be created with NewCacheKey", func() {
				operation(axiom.CacheKey[int]{}, axiom.NewCache())
			})
			assert.PanicsWithValue(t, "cache: nil *Cache", func() {
				operation(axiom.NewCacheKey[int]("value"), nil)
			})
		})
	}

	cache := axiom.NewCache()
	key := axiom.NewCacheKey[int]("value")
	for _, state := range []string{"miss", "hit"} {
		t.Run(state, func(t *testing.T) {
			if state == "hit" {
				key.Set(cache, 1)
			}
			assert.PanicsWithValue(t, "cache: nil context", func() {
				_, _ = key.GetOrCreate(nil, cache, func() (int, error) { return 1, nil })
			})
			assert.PanicsWithValue(t, "cache: nil constructor", func() {
				_, _ = key.GetOrCreate(t.Context(), cache, nil)
			})
		})
	}
}

func TestCacheKey_GetOrCreate_RejectsAlreadyCancelledContextEvenOnHit(t *testing.T) {
	cache := axiom.NewCache()
	key := axiom.NewCacheKey[int]("value")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, state := range []string{"miss", "hit"} {
		t.Run(state, func(t *testing.T) {
			if state == "hit" {
				key.Set(cache, 1)
			}
			value, err := key.GetOrCreate(ctx, cache, func() (int, error) {
				t.Fatal("constructor must not run with an already cancelled context")
				return 1, nil
			})
			assert.Zero(t, value)
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
	value, ok := key.Get(cache)
	require.True(t, ok)
	assert.Equal(t, 1, value, "cancellation must not invalidate a ready value")
}
