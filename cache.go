package axiom

import "sync"

type Cache struct {
	mu      sync.Mutex
	entries map[any]any
}

func NewCache() *Cache { return &Cache{} }

func (c *Cache) storeLocked(key, entry any) {
	if c.entries == nil {
		c.entries = make(map[any]any)
	}
	c.entries[key] = entry
}
