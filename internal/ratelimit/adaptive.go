package ratelimit

import (
	"sync"
	"time"
)

// AdaptiveRateLimiter implements rate limiting that adapts based on provider health
type AdaptiveRateLimiter struct {
	mu          sync.RWMutex
	baseRate    int
	currentRate int
	minRate     int
	maxRate     int

	// Provider health tracking
	healthScore float64 // 0.0 to 1.0

	// Sliding window tracking
	windowSize time.Duration
	requests   []time.Time

	// Backoff state
	backoffUntil      time.Time
	backoffMultiplier float64
}

// NewAdaptiveRateLimiter creates an adaptive rate limiter
func NewAdaptiveRateLimiter(baseRate int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseRate:          baseRate,
		currentRate:       baseRate,
		minRate:           baseRate / 10, // 10% of base rate minimum
		maxRate:           baseRate * 2,  // 2x base rate maximum
		healthScore:       1.0,
		windowSize:        time.Minute,
		requests:          make([]time.Time, 0),
		backoffMultiplier: 1.0, // Start at 1.0
	}
}

// Allow checks if a request should be allowed
func (arl *AdaptiveRateLimiter) Allow() bool {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	now := time.Now()

	// Check backoff
	if now.Before(arl.backoffUntil) {
		return false
	}

	// Clean old requests
	cutoff := now.Add(-arl.windowSize)
	newRequests := make([]time.Time, 0)
	for _, req := range arl.requests {
		if req.After(cutoff) {
			newRequests = append(newRequests, req)
		}
	}
	arl.requests = newRequests

	// Check if we're at the rate limit
	if len(arl.requests) >= arl.currentRate {
		return false
	}

	// Record this request
	arl.requests = append(arl.requests, now)
	return true
}

// UpdateHealth updates the health score and adjusts rate accordingly
func (arl *AdaptiveRateLimiter) UpdateHealth(score float64) {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Clamp score to 0.0-1.0
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	arl.healthScore = score

	// Adjust rate based on health
	// Health score of 1.0 = full rate
	// Health score of 0.0 = minimum rate
	newRate := arl.minRate + int(float64(arl.maxRate-arl.minRate)*score)

	// Don't increase rate too fast
	if newRate > arl.currentRate {
		increase := (newRate - arl.currentRate) / 2
		if increase < 1 {
			increase = 1
		}
		arl.currentRate += increase
	} else {
		arl.currentRate = newRate
	}

	// Clear backoff if health improves
	if score > 0.7 {
		if now := time.Now(); now.After(arl.backoffUntil) {
			arl.backoffUntil = time.Time{}
			arl.backoffMultiplier = 1.0
		}
	}
}

// RecordFailure records a failure and may trigger backoff
func (arl *AdaptiveRateLimiter) RecordFailure() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Increase backoff multiplier
	arl.backoffMultiplier *= 1.5
	if arl.backoffMultiplier > 8 {
		arl.backoffMultiplier = 8
	}

	// Apply backoff
	backoffDuration := time.Duration(float64(time.Second) * arl.backoffMultiplier)
	arl.backoffUntil = time.Now().Add(backoffDuration)

	// Reduce rate
	arl.currentRate = (arl.currentRate * 8) / 10
	if arl.currentRate < arl.minRate {
		arl.currentRate = arl.minRate
	}
}

// RecordSuccess records a successful request
func (arl *AdaptiveRateLimiter) RecordSuccess() {
	arl.mu.Lock()
	defer arl.mu.Unlock()

	// Slowly increase rate after success
	if arl.currentRate < arl.baseRate {
		arl.currentRate += (arl.baseRate - arl.currentRate) / 4
		if arl.currentRate > arl.baseRate {
			arl.currentRate = arl.baseRate
		}
	}
}

// GetStats returns current statistics
func (arl *AdaptiveRateLimiter) GetStats() map[string]interface{} {
	arl.mu.RLock()
	defer arl.mu.RUnlock()

	return map[string]interface{}{
		"base_rate":          arl.baseRate,
		"current_rate":       arl.currentRate,
		"min_rate":           arl.minRate,
		"max_rate":           arl.maxRate,
		"health_score":       arl.healthScore,
		"requests_in_window": len(arl.requests),
		"backoff_until":      arl.backoffUntil,
		"backoff_multiplier": arl.backoffMultiplier,
	}
}

// GetCurrentRate returns the current rate limit
func (arl *AdaptiveRateLimiter) GetCurrentRate() int {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	return arl.currentRate
}

// GetHealthScore returns the current health score
func (arl *AdaptiveRateLimiter) GetHealthScore() float64 {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	return arl.healthScore
}

// Manager manages multiple adaptive rate limiters
type Manager struct {
	mu       sync.RWMutex
	limiters map[string]*AdaptiveRateLimiter
}

// NewManager creates a new rate limiter manager
func NewManager() *Manager {
	return &Manager{
		limiters: make(map[string]*AdaptiveRateLimiter),
	}
}

// GetOrCreate gets or creates a rate limiter for a key
func (m *Manager) GetOrCreate(key string, baseRate int) *AdaptiveRateLimiter {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limiter, exists := m.limiters[key]; exists {
		return limiter
	}

	limiter := NewAdaptiveRateLimiter(baseRate)
	m.limiters[key] = limiter
	return limiter
}

// Delete removes a rate limiter
func (m *Manager) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.limiters, key)
}

// List returns all rate limiters
func (m *Manager) List() map[string]*AdaptiveRateLimiter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*AdaptiveRateLimiter)
	for k, v := range m.limiters {
		result[k] = v
	}
	return result
}
