package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// ResponseCache provides caching for API responses
type ResponseCache struct {
	store   CacheStore
	ttl     time.Duration
	enabled bool
}

// CacheStore defines the interface for cache storage
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// InMemoryCache is a simple in-memory cache implementation
type InMemoryCache struct {
	data map[string]*cacheEntry
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		data: make(map[string]*cacheEntry),
	}
}

// Get retrieves a value from the cache
func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	entry, ok := c.data[key]
	if !ok {
		return nil, nil
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, nil
	}

	return entry.value, nil
}

// Set stores a value in the cache
func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.data[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete removes a value from the cache
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

// Exists checks if a key exists in the cache
func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	entry, ok := c.data[key]
	if !ok {
		return false, nil
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return false, nil
	}

	return true, nil
}

// CachedResponse represents a cached API response
type CachedResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	CachedAt   time.Time         `json:"cached_at"`
	CacheHit   bool              `json:"cache_hit"`
}

// NewResponseCache creates a new response cache
func NewResponseCache(store CacheStore, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		store:   store,
		ttl:     ttl,
		enabled: true,
	}
}

// Enable enables the cache
func (c *ResponseCache) Enable() {
	c.enabled = true
	slog.Info("Response cache enabled")
}

// Disable disables the cache
func (c *ResponseCache) Disable() {
	c.enabled = false
	slog.Info("Response cache disabled")
}

// IsEnabled returns whether the cache is enabled
func (c *ResponseCache) IsEnabled() bool {
	return c.enabled
}

// GenerateCacheKey creates a cache key from request parameters
func GenerateCacheKey(model string, messages []map[string]interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})

	hash := sha256.Sum256(data)
	return fmt.Sprintf("cache:%s", hex.EncodeToString(hash[:]))
}

// Get retrieves a cached response
func (c *ResponseCache) Get(ctx context.Context, key string) (*CachedResponse, error) {
	if !c.enabled {
		return nil, nil
	}

	data, err := c.store.Get(ctx, key)
	if err != nil {
		slog.Warn("Cache get error", slog.Any("error", err))
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	var response CachedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	response.CacheHit = true
	slog.Debug("Cache hit", slog.String("key", key))

	return &response, nil
}

// Set stores a response in the cache
func (c *ResponseCache) Set(ctx context.Context, key string, response *CachedResponse) error {
	if !c.enabled {
		return nil
	}

	response.CachedAt = time.Now()
	response.CacheHit = false

	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	if err := c.store.Set(ctx, key, data, c.ttl); err != nil {
		slog.Warn("Cache set error", slog.Any("error", err))
		return err
	}

	slog.Debug("Cache set", slog.String("key", key), slog.Duration("ttl", c.ttl))
	return nil
}

// Delete removes a cached response
func (c *ResponseCache) Delete(ctx context.Context, key string) error {
	if err := c.store.Delete(ctx, key); err != nil {
		slog.Warn("Cache delete error", slog.Any("error", err))
		return err
	}

	slog.Debug("Cache delete", slog.String("key", key))
	return nil
}

// Clear clears all cached responses
func (c *ResponseCache) Clear(ctx context.Context) error {
	slog.Info("Cache cleared")
	return nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
}

// DefaultResponseCache returns a default configured cache
func DefaultResponseCache() *ResponseCache {
	store := NewInMemoryCache()
	return NewResponseCache(store, 5*time.Minute)
}
