package rate

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RateLimiter implements per-user and per-model rate limiting
type RateLimiter struct {
	storage RateLimitStorage
	config  *RateLimitConfig
	mu      sync.RWMutex
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled    bool
	DefaultRPM int // Default requests per minute
	DefaultRPD int // Default requests per day
	DefaultTPM int // Default tokens per minute
	DefaultTPD int // Default tokens per day
	RedisURL   string
}

// DefaultRateLimitConfig returns default configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:    true,
		DefaultRPM: 60,
		DefaultRPD: 10000,
		DefaultTPM: 100000,
		DefaultTPD: 1000000,
	}
}

// RateLimitStorage defines the interface for rate limit storage
type RateLimitStorage interface {
	// Check checks if a rate limit is exceeded
	Check(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, retryAfter time.Duration, err error)
	// Increment increments the counter for a key
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	// GetLimit gets the limit for a key
	GetLimit(ctx context.Context, key string) (int, error)
	// SetLimit sets the limit for a key
	SetLimit(ctx context.Context, key string, limit int, window time.Duration) error
}

// InMemoryRateLimitStorage is an in-memory rate limit storage
type InMemoryRateLimitStorage struct {
	mu       sync.RWMutex
	counters map[string]*rateCounter
	limits   map[string]*rateLimit
}

type rateCounter struct {
	count     int64
	windowEnd time.Time
}

type rateLimit struct {
	limit  int
	window time.Duration
}

// NewInMemoryRateLimitStorage creates a new in-memory rate limit storage
func NewInMemoryRateLimitStorage() *InMemoryRateLimitStorage {
	return &InMemoryRateLimitStorage{
		counters: make(map[string]*rateCounter),
		limits:   make(map[string]*rateLimit),
	}
}

// Check checks if a rate limit is exceeded
func (s *InMemoryRateLimitStorage) Check(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	counter, exists := s.counters[key]

	if !exists || now.After(counter.windowEnd) {
		// Start new window
		s.counters[key] = &rateCounter{
			count:     1,
			windowEnd: now.Add(window),
		}
		return true, limit - 1, 0, nil
	}

	if counter.count >= int64(limit) {
		retryAfter := counter.windowEnd.Sub(now)
		return false, 0, retryAfter, nil
	}

	counter.count++
	return true, limit - int(counter.count), 0, nil
}

// Increment increments the counter for a key
func (s *InMemoryRateLimitStorage) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	counter, exists := s.counters[key]

	if !exists || now.After(counter.windowEnd) {
		s.counters[key] = &rateCounter{
			count:     1,
			windowEnd: now.Add(window),
		}
		return 1, nil
	}

	counter.count++
	return counter.count, nil
}

// GetLimit gets the limit for a key
func (s *InMemoryRateLimitStorage) GetLimit(ctx context.Context, key string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit, exists := s.limits[key]; exists {
		return limit.limit, nil
	}

	return 0, nil
}

// SetLimit sets the limit for a key
func (s *InMemoryRateLimitStorage) SetLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.limits[key] = &rateLimit{
		limit:  limit,
		window: window,
	}

	return nil
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed    bool          `json:"allowed"`
	Remaining  int           `json:"remaining"`
	Limit      int           `json:"limit"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	ResetAt    time.Time     `json:"reset_at,omitempty"`
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(storage RateLimitStorage, config *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		storage: storage,
		config:  config,
	}
}

// CheckUserRPM checks rate limit for a user by requests per minute
func (rl *RateLimiter) CheckUserRPM(ctx context.Context, userID string) (*RateLimitResult, error) {
	key := fmt.Sprintf("rpm:user:%s", userID)
	limit, err := rl.storage.GetLimit(ctx, key)
	if err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = rl.config.DefaultRPM
	}

	allowed, remaining, retryAfter, err := rl.storage.Check(ctx, key, limit, time.Minute)
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		Limit:      limit,
		RetryAfter: retryAfter,
		ResetAt:    time.Now().Add(time.Minute),
	}, nil
}

// CheckUserRPD checks rate limit for a user by requests per day
func (rl *RateLimiter) CheckUserRPD(ctx context.Context, userID string) (*RateLimitResult, error) {
	key := fmt.Sprintf("rpd:user:%s", userID)
	limit, err := rl.storage.GetLimit(ctx, key)
	if err != nil {
		return nil, err
	}
	if limit == 0 {
		limit = rl.config.DefaultRPD
	}

	allowed, remaining, retryAfter, err := rl.storage.Check(ctx, key, limit, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		Limit:      limit,
		RetryAfter: retryAfter,
		ResetAt:    time.Now().Add(24 * time.Hour),
	}, nil
}

// CheckModelRPM checks rate limit for a model by requests per minute
func (rl *RateLimiter) CheckModelRPM(ctx context.Context, model string) (*RateLimitResult, error) {
	key := fmt.Sprintf("rpm:model:%s", model)

	allowed, remaining, retryAfter, err := rl.storage.Check(ctx, key, 120, time.Minute)
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		Limit:      120,
		RetryAfter: retryAfter,
		ResetAt:    time.Now().Add(time.Minute),
	}, nil
}

// CheckTokensPM checks rate limit by tokens per minute
func (rl *RateLimiter) CheckTokensPM(ctx context.Context, userID string, tokens int) (*RateLimitResult, error) {
	key := fmt.Sprintf("tpm:user:%s", userID)
	limit := rl.config.DefaultTPM

	// Increment token count
	count, err := rl.storage.Increment(ctx, key, time.Minute)
	if err != nil {
		return nil, err
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return &RateLimitResult{
		Allowed:   count <= int64(limit),
		Remaining: remaining,
		Limit:     limit,
		ResetAt:   time.Now().Add(time.Minute),
	}, nil
}

// SetUserLimit sets a custom rate limit for a user
func (rl *RateLimiter) SetUserLimit(ctx context.Context, userID string, rpm, rpd int) error {
	if rpm > 0 {
		if err := rl.storage.SetLimit(ctx, fmt.Sprintf("rpm:user:%s", userID), rpm, time.Minute); err != nil {
			return err
		}
	}
	if rpd > 0 {
		if err := rl.storage.SetLimit(ctx, fmt.Sprintf("rpd:user:%s", userID), rpd, 24*time.Hour); err != nil {
			return err
		}
	}

	slog.Info("Set user rate limits",
		slog.String("user_id", userID),
		slog.Int("rpm", rpm),
		slog.Int("rpd", rpd),
	)

	return nil
}

// Check checks a generic rate limit
func (rl *RateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	allowed, remaining, retryAfter, err := rl.storage.Check(ctx, key, limit, window)
	if err != nil {
		return nil, err
	}

	return &RateLimitResult{
		Allowed:    allowed,
		Remaining:  remaining,
		Limit:      limit,
		RetryAfter: retryAfter,
		ResetAt:    time.Now().Add(window),
	}, nil
}
