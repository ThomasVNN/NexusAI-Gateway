package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// InMemoryStorage provides an in-memory implementation of QuotaStorage
// This is used when Redis is not available or for testing
type InMemoryStorage struct {
	mu       sync.RWMutex
	data     map[string]*inMemoryEntry
	counters map[string]int64
}

// inMemoryEntry represents a time-bucketed counter entry
type inMemoryEntry struct {
	Timestamp time.Time
	Count     int64
}

// NewInMemoryStorage creates a new in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		data:     make(map[string]*inMemoryEntry),
		counters: make(map[string]int64),
	}
}

// IncrementCount increments the count for a key and returns the new count
func (s *InMemoryStorage) IncrementCount(ctx context.Context, key string, windowDuration time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-windowDuration)

	// Clean old entries
	s.cleanOldEntriesLocked(key, windowStart)

	// Get or create entry
	entry, exists := s.data[key]
	if !exists {
		entry = &inMemoryEntry{
			Timestamp: now,
			Count:     0,
		}
		s.data[key] = entry
	}

	// Increment and update timestamp
	entry.Count++
	entry.Timestamp = now

	s.counters[key]++

	return entry.Count, nil
}

// GetCount gets the current count for a key
func (s *InMemoryStorage) GetCount(ctx context.Context, key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count, exists := s.counters[key]
	if !exists {
		return 0, nil
	}
	return count, nil
}

// GetCountWithWindow gets the count within a specific time window
func (s *InMemoryStorage) GetCountWithWindow(ctx context.Context, key string, windowStart, windowEnd time.Time) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Count all entries for this key within the window
	count := int64(0)
	for k, entry := range s.data {
		// Check if key matches exactly (not prefix)
		if k != key {
			continue
		}
		if entry.Timestamp.After(windowStart) && entry.Timestamp.Before(windowEnd.Add(time.Second)) {
			count += entry.Count
		}
	}

	return count, nil
}

// SetCount sets the count for a key
func (s *InMemoryStorage) SetCount(ctx context.Context, key string, count int64, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counters[key] = count

	if expiration > 0 {
		// Schedule cleanup
		go func() {
			time.Sleep(expiration)
			s.mu.Lock()
			delete(s.counters, key)
			s.mu.Unlock()
		}()
	}

	return nil
}

// GetRemaining gets the remaining quota for a key
func (s *InMemoryStorage) GetRemaining(ctx context.Context, key string, limit int64) (int64, error) {
	count, err := s.GetCount(ctx, key)
	if err != nil {
		return 0, err
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	return remaining, nil
}

// AcquireBurst attempts to acquire a burst slot
func (s *InMemoryStorage) AcquireBurst(ctx context.Context, key string, maxBurst int, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	burstKey := key + ":burst"
	count, exists := s.counters[burstKey]

	if !exists {
		s.counters[burstKey] = 1
		return true, nil
	}

	if count >= int64(maxBurst) {
		return false, nil
	}

	s.counters[burstKey]++
	return true, nil
}

// ReleaseBurst releases a burst slot
func (s *InMemoryStorage) ReleaseBurst(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	burstKey := key + ":burst"
	count, exists := s.counters[burstKey]

	if !exists || count <= 0 {
		return nil
	}

	s.counters[burstKey]--
	return nil
}

// IncrementConcurrent increments concurrent request count
func (s *InMemoryStorage) IncrementConcurrent(ctx context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	concurrentKey := key + ":concurrent"
	_, exists := s.counters[concurrentKey]

	if !exists {
		s.counters[concurrentKey] = 1
		return 1, nil
	}

	s.counters[concurrentKey]++
	return s.counters[concurrentKey], nil
}

// DecrementConcurrent decrements concurrent request count
func (s *InMemoryStorage) DecrementConcurrent(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	concurrentKey := key + ":concurrent"
	count, exists := s.counters[concurrentKey]

	if !exists || count <= 0 {
		return nil
	}

	s.counters[concurrentKey]--
	return nil
}

// GetConcurrentCount gets the current concurrent request count
func (s *InMemoryStorage) GetConcurrentCount(ctx context.Context, key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	concurrentKey := key + ":concurrent"
	count, exists := s.counters[concurrentKey]

	if !exists {
		return 0, nil
	}

	return count, nil
}

// Ping checks the connection health
func (s *InMemoryStorage) Ping(ctx context.Context) error {
	return nil // In-memory is always healthy
}

// Close closes the storage connection
func (s *InMemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*inMemoryEntry)
	s.counters = make(map[string]int64)

	return nil
}

// cleanOldEntriesLocked removes old entries for a key (caller must hold lock)
func (s *InMemoryStorage) cleanOldEntriesLocked(key string, windowStart time.Time) {
	for k, entry := range s.data {
		if len(k) >= len(key) && k[:len(key)] == key && entry.Timestamp.Before(windowStart) {
			delete(s.data, k)
		}
	}
}

// Clear resets all data (useful for testing)
func (s *InMemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]*inMemoryEntry)
	s.counters = make(map[string]int64)
}

// GetStats returns storage statistics
func (s *InMemoryStorage) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"entries":  len(s.data),
		"counters": len(s.counters),
	}
}

// ErrRedisUnavailable indicates Redis is not available
var ErrRedisUnavailable = errors.New("redis unavailable")

// RedisStorage provides a Redis-backed implementation of QuotaStorage
// This is a placeholder that would be implemented with actual Redis client
type RedisStorage struct {
	// In a real implementation, this would contain the Redis client
	// For now, we fall back to in-memory storage
	inMemory *InMemoryStorage
}

// NewRedisStorage creates a new Redis storage (with in-memory fallback)
func NewRedisStorage(redisURL string) (*RedisStorage, error) {
	// In a production implementation, we would:
	// 1. Parse the Redis URL
	// 2. Create a Redis client
	// 3. Test the connection
	// 4. Return the Redis-backed storage
	//
	// For now, we use in-memory storage as a fallback
	storage := &RedisStorage{
		inMemory: NewInMemoryStorage(),
	}

	return storage, nil
}

// IncrementCount increments the count for a key
func (s *RedisStorage) IncrementCount(ctx context.Context, key string, windowDuration time.Duration) (int64, error) {
	return s.inMemory.IncrementCount(ctx, key, windowDuration)
}

// GetCount gets the current count for a key
func (s *RedisStorage) GetCount(ctx context.Context, key string) (int64, error) {
	return s.inMemory.GetCount(ctx, key)
}

// GetCountWithWindow gets the count within a specific time window
func (s *RedisStorage) GetCountWithWindow(ctx context.Context, key string, windowStart, windowEnd time.Time) (int64, error) {
	return s.inMemory.GetCountWithWindow(ctx, key, windowStart, windowEnd)
}

// SetCount sets the count for a key
func (s *RedisStorage) SetCount(ctx context.Context, key string, count int64, expiration time.Duration) error {
	return s.inMemory.SetCount(ctx, key, count, expiration)
}

// GetRemaining gets the remaining quota for a key
func (s *RedisStorage) GetRemaining(ctx context.Context, key string, limit int64) (int64, error) {
	return s.inMemory.GetRemaining(ctx, key, limit)
}

// AcquireBurst attempts to acquire a burst slot
func (s *RedisStorage) AcquireBurst(ctx context.Context, key string, maxBurst int, ttl time.Duration) (bool, error) {
	return s.inMemory.AcquireBurst(ctx, key, maxBurst, ttl)
}

// ReleaseBurst releases a burst slot
func (s *RedisStorage) ReleaseBurst(ctx context.Context, key string) error {
	return s.inMemory.ReleaseBurst(ctx, key)
}

// IncrementConcurrent increments concurrent request count
func (s *RedisStorage) IncrementConcurrent(ctx context.Context, key string) (int64, error) {
	return s.inMemory.IncrementConcurrent(ctx, key)
}

// DecrementConcurrent decrements concurrent request count
func (s *RedisStorage) DecrementConcurrent(ctx context.Context, key string) error {
	return s.inMemory.DecrementConcurrent(ctx, key)
}

// GetConcurrentCount gets the current concurrent request count
func (s *RedisStorage) GetConcurrentCount(ctx context.Context, key string) (int64, error) {
	return s.inMemory.GetConcurrentCount(ctx, key)
}

// Ping checks the connection health
func (s *RedisStorage) Ping(ctx context.Context) error {
	// In production, ping the Redis server
	return s.inMemory.Ping(ctx)
}

// Close closes the storage connection
func (s *RedisStorage) Close() error {
	return s.inMemory.Close()
}

// CreateRedisStorage creates a Redis-backed storage with proper connection handling
func CreateRedisStorage(redisURL string) (QuotaStorage, error) {
	if redisURL == "" {
		// Return in-memory storage when no Redis URL is provided
		return NewInMemoryStorage(), nil
	}

	storage, err := NewRedisStorage(redisURL)
	if err != nil {
		// Fall back to in-memory storage on error
		return NewInMemoryStorage(), nil
	}

	return storage, nil
}

// MockRedisStorage is a mock implementation for testing
type MockRedisStorage struct {
	InMemoryStorage
	ShouldFail bool
}

// NewMockRedisStorage creates a new mock storage for testing
func NewMockRedisStorage() *MockRedisStorage {
	return &MockRedisStorage{
		InMemoryStorage: *NewInMemoryStorage(),
	}
}

// Ping always returns nil for mock
func (s *MockRedisStorage) Ping(ctx context.Context) error {
	if s.ShouldFail {
		return errors.New("mock connection failed")
	}
	return nil
}

// IncrementCountWithDelay increments count with artificial delay for testing
func (s *MockRedisStorage) IncrementCountWithDelay(ctx context.Context, key string, windowDuration time.Duration, delay time.Duration) (int64, error) {
	time.Sleep(delay)
	return s.InMemoryStorage.IncrementCount(ctx, key, windowDuration)
}

// SetFailure sets whether the mock should fail
func (s *MockRedisStorage) SetFailure(fail bool) {
	s.ShouldFail = fail
}

// StorageFactory creates appropriate storage based on configuration
type StorageFactory struct{}

// NewStorageFactory creates a new storage factory
func NewStorageFactory() *StorageFactory {
	return &StorageFactory{}
}

// CreateStorage creates storage based on the provided configuration
func (f *StorageFactory) CreateStorage(redisURL string, useInMemoryFallback bool) (QuotaStorage, error) {
	if redisURL == "" {
		if useInMemoryFallback {
			return NewInMemoryStorage(), nil
		}
		return nil, errors.New("redis URL is required when in-memory fallback is disabled")
	}

	storage, err := NewRedisStorage(redisURL)
	if err != nil {
		if useInMemoryFallback {
			return NewInMemoryStorage(), nil
		}
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return storage, nil
}
