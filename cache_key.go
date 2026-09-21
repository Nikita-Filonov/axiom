package axiom

import (
	"context"
	"fmt"
)

type CacheKey[T any] struct {
	name string
}

type cacheEntry[T any] struct {
	value T
	err   error
	done  chan struct{}
	ready bool
}

func NewCacheKey[T any](name string) CacheKey[T] {
	if name == "" {
		panic("cache: key name must not be empty")
	}
	return CacheKey[T]{name: name}
}

func (k CacheKey[T]) Name() string { return k.name }

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

func (k CacheKey[T]) Set(cache *Cache, value T) {
	k.validate(cache)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.storeLocked(k, &cacheEntry[T]{value: value, ready: true})
}

func (k CacheKey[T]) Delete(cache *Cache) {
	k.validate(cache)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	delete(cache.entries, k)
}

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
