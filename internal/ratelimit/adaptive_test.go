package ratelimit

import (
	"testing"
)

func TestAdaptiveRateLimiter_InitialState(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	stats := arl.GetStats()

	if stats["base_rate"].(int) != 100 {
		t.Errorf("Expected base rate 100, got %d", stats["base_rate"].(int))
	}

	if stats["current_rate"].(int) != 100 {
		t.Errorf("Expected current rate 100, got %d", stats["current_rate"].(int))
	}

	if stats["health_score"].(float64) != 1.0 {
		t.Errorf("Expected health score 1.0, got %f", stats["health_score"].(float64))
	}
}

func TestAdaptiveRateLimiter_Allow(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	// Should allow requests up to the rate
	for i := 0; i < 100; i++ {
		if !arl.Allow() {
			t.Errorf("Expected to allow request %d", i)
		}
	}

	// Should deny after rate limit
	if arl.Allow() {
		t.Error("Expected to deny after rate limit")
	}
}

func TestAdaptiveRateLimiter_UpdateHealth(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	// Update to 50% health
	arl.UpdateHealth(0.5)

	stats := arl.GetStats()
	healthScore := stats["health_score"].(float64)

	if healthScore != 0.5 {
		t.Errorf("Expected health score 0.5, got %f", healthScore)
	}
}

func TestAdaptiveRateLimiter_RecordFailure(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	// Get initial state
	initialStats := arl.GetStats()
	initialMultiplier := initialStats["backoff_multiplier"].(float64)

	arl.RecordFailure()

	stats := arl.GetStats()
	newMultiplier := stats["backoff_multiplier"].(float64)

	// Backoff multiplier should increase after failure
	if newMultiplier <= initialMultiplier {
		t.Errorf("Expected backoff multiplier to increase, got %f -> %f", initialMultiplier, newMultiplier)
	}
}

func TestAdaptiveRateLimiter_RecordSuccess(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	// Reduce rate first
	arl.UpdateHealth(0.1)
	lowRate := arl.GetCurrentRate()

	// Record success
	arl.RecordSuccess()

	// Rate should increase
	newRate := arl.GetCurrentRate()
	if newRate <= lowRate {
		t.Errorf("Expected rate to increase, got %d -> %d", lowRate, newRate)
	}
}

func TestAdaptiveRateLimiter_GetCurrentRate(t *testing.T) {
	arl := NewAdaptiveRateLimiter(50)

	rate := arl.GetCurrentRate()
	if rate != 50 {
		t.Errorf("Expected rate 50, got %d", rate)
	}
}

func TestAdaptiveRateLimiter_GetHealthScore(t *testing.T) {
	arl := NewAdaptiveRateLimiter(100)

	arl.UpdateHealth(0.75)

	score := arl.GetHealthScore()
	if score != 0.75 {
		t.Errorf("Expected health score 0.75, got %f", score)
	}
}

func TestManager_GetOrCreate(t *testing.T) {
	m := NewManager()

	limiter1 := m.GetOrCreate("test-key", 100)
	if limiter1 == nil {
		t.Error("Expected non-nil limiter")
	}

	// Same key should return same limiter
	limiter2 := m.GetOrCreate("test-key", 200)
	if limiter1 != limiter2 {
		t.Error("Expected same limiter for same key")
	}

	// Different key should return different limiter
	limiter3 := m.GetOrCreate("other-key", 100)
	if limiter1 == limiter3 {
		t.Error("Expected different limiter for different key")
	}
}

func TestManager_Delete(t *testing.T) {
	m := NewManager()

	m.GetOrCreate("test-key", 100)

	limiters := m.List()
	if len(limiters) != 1 {
		t.Errorf("Expected 1 limiter, got %d", len(limiters))
	}

	m.Delete("test-key")

	limiters = m.List()
	if len(limiters) != 0 {
		t.Errorf("Expected 0 limiters, got %d", len(limiters))
	}
}

func TestManager_List(t *testing.T) {
	m := NewManager()

	m.GetOrCreate("key1", 100)
	m.GetOrCreate("key2", 200)
	m.GetOrCreate("key3", 300)

	limiters := m.List()
	if len(limiters) != 3 {
		t.Errorf("Expected 3 limiters, got %d", len(limiters))
	}
}
