# 📘 Cache

`Cache` is a standalone, concurrent, in-memory store with typed keys and coordinated lazy creation.
It is independent of Axiom's execution model: whoever holds the same `*Cache` shares its values.

---

## 📑 Table of Contents

- [API](#api)
- [Example](#example)
- [Using Cache with Axiom](#using-cache-with-axiom)
- [Behavior](#behavior)
- [Ownership and lifecycle](#ownership-and-lifecycle)

---

## API

```go
func NewCache() *Cache
func NewCacheKey[T any](name string) CacheKey[T]

func (key CacheKey[T]) Name() string
func (key CacheKey[T]) Get(cache *Cache) (T, bool)
func (key CacheKey[T]) Set(cache *Cache, value T)
func (key CacheKey[T]) Delete(cache *Cache)
func (key CacheKey[T]) GetOrCreate(
    waitCtx context.Context,
    cache *Cache,
    create func() (T, error),
) (T, error)
```

A key is identified by its name and value type. Keys created with the same name and type address the same entry;
the same name with another type addresses a different entry.

The zero value of `Cache` is usable, but a cache must not be copied after first use. Pass `*Cache` instead.
Cached values may be any type, including maps, slices, and nil values.

## Example

Create the cache where its intended sharing scope begins, then pass the same pointer to its consumers.

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/require"
)

func TestCacheExample(t *testing.T) {
	type Settings struct {
		Environment string
	}

	// This instance defines the sharing scope.
	cache := axiom.NewCache()
	settingsKey := axiom.NewCacheKey[Settings]("service.settings")

	// The constructor is called only on the first cache miss.
	loads := 0
	loadSettings := func() (Settings, error) {
		loads++
		return Settings{Environment: "staging"}, nil
	}

	first, err := settingsKey.GetOrCreate(t.Context(), cache, loadSettings)
	require.NoError(t, err)
	second, err := settingsKey.GetOrCreate(t.Context(), cache, loadSettings)
	require.NoError(t, err)

	// Both calls observe the value produced by the first call.
	require.Equal(t, first, second)
	require.Equal(t, 1, loads)
}
```

## Using Cache with Axiom

A Resource is a convenient owner when one cache must be shared by a runner:

```go
var SharedCache = axiom.DefineResource(
    "shared-cache",
    func(_ *axiom.Runner) (*axiom.Cache, func(), error) {
        return axiom.NewCache(), nil, nil
    },
)
```

An ordinary fixture can then create a value with the current test's `Config`:

```go
var schemaKey = axiom.NewCacheKey[Schema]("schema.current")

var SchemaFixture = axiom.DefineFixture("schema", func(cfg *axiom.Config) (Schema, func(), error) {
    cache := SharedCache.Get(cfg.Runner)

    schema, err := schemaKey.GetOrCreate(cfg.Context.Raw, cache, func() (Schema, error) {
        loader := SchemaLoaderFixture.Get(cfg)
        return loader.Load()
    })

    return schema, nil, err
})
```

Resolve creation-only dependencies inside `GetOrCreate`, so a cache hit does not initialize them.
Register the resource and fixtures through the usual runner options.

## Behavior

- `Get` returns only completed values. An absent or pending entry is a miss.
- `GetOrCreate` runs one constructor at a time per key. Callers of other keys proceed independently.
- Concurrent callers of the same key wait for and receive the same result.
- Successful results are cached. Errors and partial values are not; a later call may retry.
- `waitCtx` can cancel waiting, but it does not stop a constructor that is already running.
- If a constructor panics or exits through `runtime.Goexit`, its caller keeps that control flow and waiters receive
  a diagnostic error.
- Recursive loading of the same key and dependency cycles are unsupported.
- `Set` and `Delete` affect subsequent lookups but do not cancel work already in progress.

Use `GetOrCreate` instead of a separate `Get -> create -> Set` sequence when creation must be coordinated.

## Ownership and lifecycle

Cache has no automatic cleanup, expiration, eviction, persistence, or reporting. It stores values until they are
replaced, deleted, or the cache itself becomes unreachable.

| Owner | Effective scope |
| --- | --- |
| Resource | The runner using that resource instance |
| Context value | Every Config or joined runner inheriting the same pointer |
| Local value | One execution attempt |
| Application object | Every consumer receiving that pointer |

Cache synchronizes its entries, not the contents of returned values. Shared mutable values need their own
synchronization. If a cached value needs cleanup, the component that owns the cache must also own that cleanup.
