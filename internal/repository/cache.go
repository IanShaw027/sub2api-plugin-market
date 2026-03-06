package repository

import (
	"sync"
	"time"
)

// TTLCache is a simple in-memory cache with per-item TTL expiration.
type TTLCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewTTLCache creates a new TTLCache with the given default TTL.
func NewTTLCache(ttl time.Duration) *TTLCache {
	return &TTLCache{
		items: make(map[string]cacheItem),
		ttl:   ttl,
	}
}

// Get retrieves a value by key. Returns nil, false if missing or expired.
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.value, true
}

// Set stores a value with the default TTL.
func (c *TTLCache) Set(key string, value interface{}) {
	c.mu.Lock()
	c.items[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes all cached items.
func (c *TTLCache) Invalidate() {
	c.mu.Lock()
	c.items = make(map[string]cacheItem)
	c.mu.Unlock()
}
