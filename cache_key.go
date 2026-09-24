package axiom

import (
	"context"
	"fmt"
)

// CacheKey identifies a typed entry by name and value type. Keys with the same
// name and type address the same entry in a Cache. Its zero value is invalid.
type CacheKey[T any] struct {
	name string
}

type cacheEntry[T any] struct {
	value T
	err   error
	done  chan struct{}
	ready bool
}

// NewCacheKey creates a typed key and panics if name is empty.
func NewCacheKey[T any](name string) CacheKey[T] {
	if name == "" {
		panic("cache: key name must not be empty")
	}
	return CacheKey[T]{name: name}
}

func (k CacheKey[T]) Name() string { return k.name }

// Get returns a completed cached value. An absent or in-progress entry is a
// miss and returns the zero value with false.
func (k CacheKey[T]) Get(cache *Cache) (T, bool) {
	k.validate(cache)
	cache.mu.Lock()
	defer cache.mu.Unlock()

	if raw, ok := cache.entries[k]; ok {
		entry := raw.(*cacheEntry[T])
		if entry.ready {
			return entry.value, true
		}
	}
	var zero T
	return zero, false
}

// Set replaces this key's entry without copying or closing value. A concurrent
// constructor can finish for its existing waiters, while the new value remains
// authoritative for subsequent lookups.
func (k CacheKey[T]) Set(cache *Cache, value T) {
	k.validate(cache)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storeLocked(k, &cacheEntry[T]{value: value, ready: true})
}

// Delete removes this key's entry without closing its value. A concurrent
// constructor can finish for its existing waiters without restoring the entry.
func (k CacheKey[T]) Delete(cache *Cache) {
	k.validate(cache)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.entries, k)
}

// GetOrCreate returns a cached value or coordinates construction for the key.
// Callers waiting on the same construction share its result; successful values
// are cached, while errors allow a later attempt. Cancellation is checked
// before lookup, so an already canceled waitCtx is honored even on a hit.
// Canceling a waiter affects only that caller; the constructor and other
// waiters continue. Set and Delete establish the value seen by subsequent
// lookups without interrupting existing waiters or allowing an older result to
// overwrite the new state. Inputs are validated before lookup, including on a
// hit. A panic or Goexit in create propagates to its caller and gives waiters a
// diagnostic error. Recursive creation of the same key is unsupported.
func (k CacheKey[T]) GetOrCreate(waitCtx context.Context, cache *Cache, create func() (T, error)) (T, error) {
	k.validate(cache)
	if waitCtx == nil {
		panic("cache: nil context")
	}
	if create == nil {
		panic("cache: nil constructor")
	}
	var zero T
	if err := waitCtx.Err(); err != nil {
		return zero, err
	}

	cache.mu.Lock()
	if raw, ok := cache.entries[k]; ok {
		entry := raw.(*cacheEntry[T])
		ready := entry.ready
		cache.mu.Unlock()
		if ready {
			return entry.value, nil
		}
		select {
		case <-entry.done:
			return entry.value, entry.err
		case <-waitCtx.Done():
			return zero, waitCtx.Err()
		}
	}

	entry := &cacheEntry[T]{done: make(chan struct{})}
	cache.storeLocked(k, entry)
	cache.mu.Unlock()
	return k.createValue(cache, entry, create)
}

func (k CacheKey[T]) createValue(cache *Cache, entry *cacheEntry[T], create func() (T, error)) (value T, err error) {
	returned := false
	defer func() {
		if !returned {
			err = fmt.Errorf("cache: creation aborted for key %q", k.name)
		}
		if err != nil {
			var zero T
			value = zero
		}

		cache.mu.Lock()
		defer cache.mu.Unlock()
		entry.value, entry.err, entry.ready = value, err, err == nil
		if err != nil && cache.entries[k] == entry {
			delete(cache.entries, k)
		}

		close(entry.done)
	}()

	value, err = create()
	returned = true
	return value, err
}

func (k CacheKey[T]) validate(cache *Cache) {
	if k.name == "" {
		panic("cache: key must be created with NewCacheKey")
	}
	if cache == nil {
		panic("cache: nil *Cache")
	}
}
