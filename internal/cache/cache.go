package cache

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// TTLCache implements a thread-safe TTL cache
type TTLCache struct {
	mu         sync.RWMutex
	items      map[string]*CacheEntry
	defaultTTL time.Duration
}

// NewTTLCache creates a new TTL cache
func NewTTLCache(defaultTTL time.Duration) *TTLCache {
	cache := &TTLCache{
		items:      make(map[string]*CacheEntry),
		defaultTTL: defaultTTL,
	}
	// Start cleanup goroutine
	go cache.cleanup()
	return cache
}

// Get retrieves a value from cache
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.Expiration) {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache
func (c *TTLCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL stores a value in cache with custom TTL
func (c *TTLCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from cache
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*CacheEntry)
}

// Size returns the number of items in cache
func (c *TTLCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// cleanup removes expired items periodically
func (c *TTLCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
