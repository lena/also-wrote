package tmdb

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value   interface{}
	expires time.Time
}

// ttlCache is a thread-safe in-memory cache with TTL. Keys expire after the given duration.
type ttlCache struct {
	mu     sync.Mutex
	entries map[string]*cacheEntry
	ttl    time.Duration
}

func newTTLCache(ttl time.Duration) *ttlCache {
	return &ttlCache{
		entries: make(map[string]*cacheEntry),
		ttl:    ttl,
	}
}

func (c *ttlCache) get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expires) {
		if ok {
			delete(c.entries, key)
		}
		return nil, false
	}
	return e.value, true
}

func (c *ttlCache) set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
}
