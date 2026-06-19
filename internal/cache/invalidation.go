package cache

import (
	"sync"
	"time"
)

// InvalidationStrategy defines how cache entries are invalidated
type InvalidationStrategy string

const (
	// InvalidateAll clears the entire cache
	InvalidateAll InvalidationStrategy = "all"
	// InvalidateKeys removes specific keys
	InvalidateKeys InvalidationStrategy = "keys"
	// InvalidatePattern removes keys matching a pattern
	InvalidatePattern InvalidationStrategy = "pattern"
)

// InvalidationRequest represents a cache invalidation request
type InvalidationRequest struct {
	Strategy InvalidationStrategy
	Keys     []string
	Pattern  string
}

// CacheManager extends TTLCache with invalidation capabilities
type CacheManager struct {
	*TTLCache
	mu          sync.RWMutex
	subscribers map[string]chan InvalidationRequest
	keys        []string // Track keys for pattern matching
}

// NewCacheManager creates a new cache manager
func NewCacheManager(defaultTTL time.Duration) *CacheManager {
	return &CacheManager{
		TTLCache:    NewTTLCache(defaultTTL),
		subscribers: make(map[string]chan InvalidationRequest),
		keys:        make([]string, 0),
	}
}

// Set stores a value and tracks the key
func (m *CacheManager) Set(key string, value interface{}) {
	m.mu.Lock()
	m.keys = append(m.keys, key)
	m.mu.Unlock()
	m.TTLCache.Set(key, value)
}

// SetWithTTL stores a value with custom TTL and tracks the key
func (m *CacheManager) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	m.keys = append(m.keys, key)
	m.mu.Unlock()
	m.TTLCache.SetWithTTL(key, value, ttl)
}

// Subscribe adds a subscriber for invalidation events
func (m *CacheManager) Subscribe(name string) chan InvalidationRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan InvalidationRequest, 10)
	m.subscribers[name] = ch
	return ch
}

// Unsubscribe removes a subscriber
func (m *CacheManager) Unsubscribe(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, ok := m.subscribers[name]; ok {
		close(ch)
		delete(m.subscribers, name)
	}
}

// notifySubscribers notifies all subscribers of an invalidation
func (m *CacheManager) notifySubscribers(req InvalidationRequest) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, ch := range m.subscribers {
		select {
		case ch <- req:
		default:
		}
	}
}

// InvalidateAll clears the entire cache and notifies subscribers
func (m *CacheManager) InvalidateAll() {
	m.mu.Lock()
	m.keys = make([]string, 0)
	m.mu.Unlock()
	m.Clear()
	m.notifySubscribers(InvalidationRequest{Strategy: InvalidateAll})
}

// InvalidateKeys removes specific keys from the cache
func (m *CacheManager) InvalidateKeys(keys []string) {
	for _, key := range keys {
		m.Delete(key)
	}
	m.notifySubscribers(InvalidationRequest{
		Strategy: InvalidateKeys,
		Keys:     keys,
	})
}

// InvalidatePattern removes keys matching a pattern
func (m *CacheManager) InvalidatePattern(pattern string) {
	m.mu.Lock()
	var toDelete []string

	for _, key := range m.keys {
		if matchesPattern(key, pattern) {
			toDelete = append(toDelete, key)
		}
	}

	// Remove from tracking
	newKeys := make([]string, 0)
	for _, key := range m.keys {
		if !contains(toDelete, key) {
			newKeys = append(newKeys, key)
		}
	}
	m.keys = newKeys
	m.mu.Unlock()

	for _, key := range toDelete {
		m.Delete(key)
	}

	m.notifySubscribers(InvalidationRequest{
		Strategy: InvalidatePattern,
		Pattern:  pattern,
	})
}

// matchesPattern checks if a key matches a simple pattern
func matchesPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if len(pattern) == 0 || len(key) == 0 {
		return false
	}
	// Check prefix match
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return key == pattern
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ManagerStats provides cache statistics
type ManagerStats struct {
	Items int
}

func (m *CacheManager) Stats() ManagerStats {
	return ManagerStats{
		Items: m.Size(),
	}
}
