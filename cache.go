package axiom

import "sync"

// Cache is a concurrent typed store accessed through [CacheKey] values. Its
// lifetime is independent of test attempts and runner cleanup; the caller
// chooses where to share it and owns cleanup of stored values. Cache protects
// its entries, not mutable values returned to callers. Do not copy a Cache
// after first use.
type Cache struct {
	mu      sync.Mutex
	entries map[any]any
}

// NewCache returns an empty Cache. The zero value of Cache is also ready to use.
func NewCache() *Cache { return &Cache{} }

func (c *Cache) storeLocked(key, entry any) {
	if c.entries == nil {
		c.entries = make(map[any]any)
	}
	c.entries[key] = entry
}
