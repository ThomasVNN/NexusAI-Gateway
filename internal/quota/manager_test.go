package quota

import (
	"context"
	"testing"
	"time"
)

func TestDefaultQuotaPlans(t *testing.T) {
	plans := DefaultQuotaPlans()

	// Test free plan
	free, ok := plans["free"]
	if !ok {
		t.Fatal("Free plan not found")
	}
	if free.RequestsPerMin != 10 {
		t.Errorf("Free plan requests per min: got %d, want 10", free.RequestsPerMin)
	}
	if free.RequestsPerHour != 100 {
		t.Errorf("Free plan requests per hour: got %d, want 100", free.RequestsPerHour)
	}
	if free.RequestsPerDay != 1000 {
		t.Errorf("Free plan requests per day: got %d, want 1000", free.RequestsPerDay)
	}
	if free.Features["streaming"] != false {
		t.Error("Free plan should not have streaming")
	}

	// Test standard plan
	standard, ok := plans["standard"]
	if !ok {
		t.Fatal("Standard plan not found")
	}
	if standard.RequestsPerMin != 60 {
		t.Errorf("Standard plan requests per min: got %d, want 60", standard.RequestsPerMin)
	}
	if standard.Features["streaming"] != true {
		t.Error("Standard plan should have streaming")
	}

	// Test enterprise plan
	enterprise, ok := plans["enterprise"]
	if !ok {
		t.Fatal("Enterprise plan not found")
	}
	if enterprise.RequestsPerMin != 300 {
		t.Errorf("Enterprise plan requests per min: got %d, want 300", enterprise.RequestsPerMin)
	}
	if enterprise.Features["custom_models"] != true {
		t.Error("Enterprise plan should have custom_models")
	}
}

func TestInMemoryQuotaRepository(t *testing.T) {
	repo := NewInMemoryQuotaRepository()
	ctx := context.Background()

	t.Run("GetQuotaUsage for new tenant", func(t *testing.T) {
		usage, err := repo.GetQuotaUsage(ctx, "tenant-1")
		if err != nil {
			t.Fatalf("GetQuotaUsage failed: %v", err)
		}
		if usage.TenantID != "tenant-1" {
			t.Errorf("Tenant ID: got %s, want tenant-1", usage.TenantID)
		}
		if usage.Plan != "free" {
			t.Errorf("Default plan: got %s, want free", usage.Plan)
		}
	})

	t.Run("SaveQuotaUsage", func(t *testing.T) {
		usage := &QuotaUsage{
			TenantID:    "tenant-2",
			Plan:        "standard",
			DayRequests: 50,
			TokensToday: 5000,
		}
		err := repo.SaveQuotaUsage(ctx, usage)
		if err != nil {
			t.Fatalf("SaveQuotaUsage failed: %v", err)
		}

		retrieved, err := repo.GetQuotaUsage(ctx, "tenant-2")
		if err != nil {
			t.Fatalf("GetQuotaUsage failed: %v", err)
		}
		if retrieved.DayRequests != 50 {
			t.Errorf("Day requests: got %d, want 50", retrieved.DayRequests)
		}
		if retrieved.TokensToday != 5000 {
			t.Errorf("Tokens today: got %d, want 5000", retrieved.TokensToday)
		}
	})

	t.Run("ResetMinute", func(t *testing.T) {
		usage := &QuotaUsage{
			TenantID:        "tenant-3",
			Plan:            "enterprise",
			MinuteRequests:  10,
			LastMinuteReset: time.Now().Add(-time.Minute),
		}
		repo.SaveQuotaUsage(ctx, usage)

		err := repo.ResetMinute(ctx, "tenant-3")
		if err != nil {
			t.Fatalf("ResetMinute failed: %v", err)
		}

		retrieved, _ := repo.GetQuotaUsage(ctx, "tenant-3")
		if retrieved.MinuteRequests != 0 {
			t.Errorf("Minute requests after reset: got %d, want 0", retrieved.MinuteRequests)
		}
	})

	t.Run("ResetHour", func(t *testing.T) {
		usage := &QuotaUsage{
			TenantID:       "tenant-4",
			Plan:           "enterprise",
			MinuteRequests: 5,
			HourRequests:   50,
			LastHourReset:  time.Now().Add(-time.Hour),
		}
		repo.SaveQuotaUsage(ctx, usage)

		err := repo.ResetHour(ctx, "tenant-4")
		if err != nil {
			t.Fatalf("ResetHour failed: %v", err)
		}

		retrieved, _ := repo.GetQuotaUsage(ctx, "tenant-4")
		if retrieved.MinuteRequests != 0 || retrieved.HourRequests != 0 {
			t.Errorf("Counters not reset: minute=%d, hour=%d", retrieved.MinuteRequests, retrieved.HourRequests)
		}
	})

	t.Run("ResetDay", func(t *testing.T) {
		usage := &QuotaUsage{
			TenantID:       "tenant-5",
			Plan:           "enterprise",
			MinuteRequests: 5,
			HourRequests:   50,
			DayRequests:    500,
			TokensToday:    50000,
			LastDayReset:   time.Now().Add(-24 * time.Hour),
		}
		repo.SaveQuotaUsage(ctx, usage)

		err := repo.ResetDay(ctx, "tenant-5")
		if err != nil {
			t.Fatalf("ResetDay failed: %v", err)
		}

		retrieved, _ := repo.GetQuotaUsage(ctx, "tenant-5")
		if retrieved.MinuteRequests != 0 || retrieved.HourRequests != 0 || retrieved.DayRequests != 0 || retrieved.TokensToday != 0 {
			t.Errorf("Counters not reset: minute=%d, hour=%d, day=%d, tokens=%d",
				retrieved.MinuteRequests, retrieved.HourRequests, retrieved.DayRequests, retrieved.TokensToday)
		}
	})

	t.Run("GetPlan", func(t *testing.T) {
		plan, err := repo.GetPlan(ctx, "free")
		if err != nil {
			t.Fatalf("GetPlan failed: %v", err)
		}
		if plan.Name != "free" {
			t.Errorf("Plan name: got %s, want free", plan.Name)
		}

		_, err = repo.GetPlan(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent plan")
		}
	})

	t.Run("UpdatePlan", func(t *testing.T) {
		newPlan := &QuotaPlan{
			Name:            "custom",
			RequestsPerMin:  100,
			RequestsPerHour: 1000,
			RequestsPerDay:  10000,
			MaxTokensPerDay: 5000000,
			Features:        map[string]bool{"streaming": true},
		}
		err := repo.UpdatePlan(ctx, newPlan)
		if err != nil {
			t.Fatalf("UpdatePlan failed: %v", err)
		}

		retrieved, _ := repo.GetPlan(ctx, "custom")
		if retrieved.RequestsPerMin != 100 {
			t.Errorf("Custom plan requests per min: got %d, want 100", retrieved.RequestsPerMin)
		}
	})
}

func TestTenantRateLimiter(t *testing.T) {
	t.Run("Allow under limit", func(t *testing.T) {
		limiter := NewTenantRateLimiter("standard")
		tenantID := "test-tenant-1"

		// Should allow up to 60 requests per minute
		for i := 0; i < 60; i++ {
			if !limiter.Allow(tenantID) {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("Block over limit", func(t *testing.T) {
		limiter := NewTenantRateLimiter("free") // 10 per minute
		tenantID := "test-tenant-2"

		// Use up all requests
		for i := 0; i < 10; i++ {
			limiter.Allow(tenantID)
		}

		// Next one should be blocked
		if limiter.Allow(tenantID) {
			t.Error("Request over limit should be blocked")
		}
	})

	t.Run("GetRemaining", func(t *testing.T) {
		limiter := NewTenantRateLimiter("standard") // 60 per minute
		tenantID := "test-tenant-3"

		remaining := limiter.GetRemaining(tenantID)
		if remaining != 60 {
			t.Errorf("Initial remaining: got %d, want 60", remaining)
		}

		// Use 5 requests
		for i := 0; i < 5; i++ {
			limiter.Allow(tenantID)
		}

		remaining = limiter.GetRemaining(tenantID)
		if remaining != 55 {
			t.Errorf("Remaining after 5 requests: got %d, want 55", remaining)
		}
	})

	t.Run("GetResetTime", func(t *testing.T) {
		limiter := NewTenantRateLimiter("standard")
		tenantID := "test-tenant-4"

		limiter.Allow(tenantID)
		resetTime := limiter.GetResetTime(tenantID)

		// Reset should be about a minute from now
		expectedReset := time.Now().Add(time.Minute)
		diff := resetTime.Sub(expectedReset)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("Reset time off by %v", diff)
		}
	})

	t.Run("Reset", func(t *testing.T) {
		limiter := NewTenantRateLimiter("standard")
		tenantID := "test-tenant-5"

		// Use some requests
		for i := 0; i < 10; i++ {
			limiter.Allow(tenantID)
		}

		// Reset
		limiter.Reset(tenantID)

		// Should have full quota again
		if limiter.GetRemaining(tenantID) != 60 {
			t.Error("Remaining should be full after reset")
		}
	})
}

func TestCheckQuota(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryQuotaRepository()
	plans := DefaultQuotaPlans()
	plan := plans["free"]

	t.Run("Within limits", func(t *testing.T) {
		usage, _ := repo.GetQuotaUsage(ctx, "quota-test-1")
		usage.MinuteRequests = 5
		usage.HourRequests = 50
		usage.DayRequests = 500
		usage.TokensToday = 50000

		err := CheckQuota(usage, plan, 1000)
		if err != nil {
			t.Errorf("Should be within limits: %v", err)
		}
	})

	t.Run("Minute limit exceeded", func(t *testing.T) {
		usage, _ := repo.GetQuotaUsage(ctx, "quota-test-2")
		usage.MinuteRequests = 10
		usage.LastMinuteReset = time.Now()

		err := CheckQuota(usage, plan, 0)
		if err == nil {
			t.Error("Should fail minute limit check")
		}

		quotaErr, ok := err.(*QuotaExceededError)
		if !ok {
			t.Error("Should be QuotaExceededError")
		}
		if quotaErr.LimitType != "minute" {
			t.Errorf("Limit type: got %s, want minute", quotaErr.LimitType)
		}
	})

	t.Run("Day limit exceeded", func(t *testing.T) {
		usage, _ := repo.GetQuotaUsage(ctx, "quota-test-3")
		usage.DayRequests = 1000

		err := CheckQuota(usage, plan, 0)
		if err == nil {
			t.Error("Should fail day limit check")
		}
	})

	t.Run("Token limit exceeded", func(t *testing.T) {
		usage, _ := repo.GetQuotaUsage(ctx, "quota-test-4")
		usage.TokensToday = 99000

		err := CheckQuota(usage, plan, 20000) // Would exceed 100000 limit
		if err == nil {
			t.Error("Should fail token limit check")
		}
	})
}
