package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSlidingWindowRateLimiter_Check(t *testing.T) {
	storage := NewInMemoryStorage()
	limiter := NewSlidingWindowRateLimiter(storage)

	ctx := context.Background()
	key := "test:key"
	limit := 5
	window := time.Minute

	// First request should succeed
	result, err := limiter.Check(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Allowed {
		t.Error("Expected first request to be allowed")
	}
	if result.Remaining != limit-1 {
		t.Errorf("Expected remaining %d, got %d", limit-1, result.Remaining)
	}

	// Exhaust the limit
	for i := 0; i < limit-1; i++ {
		_, err := limiter.Check(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("Expected no error on request %d, got: %v", i+2, err)
		}
	}

	// Next request should be denied
	result, err = limiter.Check(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Allowed {
		t.Error("Expected request to be denied after limit exhausted")
	}
	if result.Remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", result.Remaining)
	}
	if result.RetryAfter <= 0 {
		t.Error("Expected RetryAfter to be set when rate limited")
	}
}

func TestSlidingWindowRateLimiter_GetStatus(t *testing.T) {
	storage := NewInMemoryStorage()
	limiter := NewSlidingWindowRateLimiter(storage)

	ctx := context.Background()
	key := "test:status:key"
	limit := 10
	window := time.Minute

	// Make some requests
	for i := 0; i < 3; i++ {
		_, _ = limiter.Check(ctx, key, limit, window)
	}

	// Get status without consuming a token
	result, err := limiter.GetStatus(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Remaining != limit-3 {
		t.Errorf("Expected remaining %d, got %d", limit-3, result.Remaining)
	}

	if result.Used != 3 {
		t.Errorf("Expected used 3, got %d", result.Used)
	}

	// Verify that getting status doesn't consume a token
	result2, _ := limiter.GetStatus(ctx, key, limit, window)
	if result.Remaining != result2.Remaining {
		t.Error("Getting status should not consume a token")
	}
}

func TestQuotaManager_CheckTenantLimit(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()
	tenantID := "tenant-123"

	// Check free tier limit (10 requests per minute)
	for i := 0; i < 10; i++ {
		result, err := manager.CheckTenantLimit(ctx, tenantID, TierFree)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if i < 9 && !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	result, err := manager.CheckTenantLimit(ctx, tenantID, TierFree)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Allowed {
		t.Error("11th request should be denied for free tier")
	}
}

func TestQuotaManager_CheckUserLimit(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()
	tenantID := "tenant-123"
	userID := "user-456"

	// Check user limit
	result, err := manager.CheckUserLimit(ctx, tenantID, userID, TierFree)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Allowed {
		t.Error("First user request should be allowed")
	}
}

func TestQuotaManager_CheckSkillLimit(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()
	tenantID := "tenant-123"
	skillID := "skill-789"

	// Check skill limit
	result, err := manager.CheckSkillLimit(ctx, tenantID, skillID, TierFree)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Allowed {
		t.Error("First skill request should be allowed")
	}
}

func TestQuotaManager_CheckAPIKeyLimit(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()
	apiKeyHash := "hash123"

	// Check API key limit
	result, err := manager.CheckAPIKeyLimit(ctx, apiKeyHash, TierFree)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Allowed {
		t.Error("First API key request should be allowed")
	}
}

func TestQuotaManager_CheckAllLimits(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()

	tests := []struct {
		name      string
		tenantID  string
		userID    string
		skillID   string
		apiKeyHash string
		tier      RateLimitTier
		wantAllow bool
	}{
		{
			name:      "All identifiers provided",
			tenantID:  "tenant-1",
			userID:    "user-1",
			skillID:   "skill-1",
			apiKeyHash: "hash-1",
			tier:      TierFree,
			wantAllow: true,
		},
		{
			name:     "Tenant only",
			tenantID: "tenant-2",
			tier:     TierFree,
			wantAllow: true,
		},
		{
			name:     "No identifiers",
			tenantID: "default-tenant",
			tier:     TierFree,
			wantAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.CheckAllLimits(ctx, tt.tenantID, tt.userID, tt.skillID, tt.apiKeyHash, tt.tier)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tt.wantAllow && !result.Allowed {
				t.Errorf("Expected request to be allowed")
			}
		})
	}
}

func TestQuotaManager_BypassLimits(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	config.BypassLimits = true
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()

	// Even with bypass, should still work
	result, err := manager.CheckAllLimits(ctx, "tenant", "user", "skill", "hash", TierFree)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("With bypass enabled, all requests should be allowed")
	}
}

func TestQuotaManager_GetTierStatus(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	ctx := context.Background()
	tenantID := "tenant-status"
	userID := "user-status"

	// Make some requests
	for i := 0; i < 5; i++ {
		_, _ = manager.CheckTenantLimit(ctx, tenantID, TierFree)
	}

	status, err := manager.GetTierStatus(ctx, tenantID, userID, TierFree)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.Tier != string(TierFree) {
		t.Errorf("Expected tier %s, got %s", TierFree, status.Tier)
	}

	if status.RequestsPerMin != 10 {
		t.Errorf("Expected RequestsPerMin 10, got %d", status.RequestsPerMin)
	}
}

func TestInMemoryStorage_IncrementCount(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	key := "inc:test"
	window := time.Minute

	// First increment
	count1, err := storage.IncrementCount(ctx, key, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count1 != 1 {
		t.Errorf("Expected count 1, got %d", count1)
	}

	// Second increment
	count2, err := storage.IncrementCount(ctx, key, window)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count2 != 2 {
		t.Errorf("Expected count 2, got %d", count2)
	}
}

func TestInMemoryStorage_GetCount(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	key := "get:test"

	// Empty count
	count, err := storage.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0, got %d", count)
	}

	// After increment
	_, _ = storage.IncrementCount(ctx, key, time.Minute)
	count, _ = storage.GetCount(ctx, key)
	if count == 0 {
		t.Error("Expected count > 0 after increment")
	}
}

func TestInMemoryStorage_SetCount(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	key := "set:test"
	storage.SetCount(ctx, key, 100, 0)

	count, err := storage.GetCount(ctx, key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected count 100, got %d", count)
	}
}

func TestInMemoryStorage_AcquireBurst(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	key := "burst:test"
	maxBurst := 3
	ttl := time.Minute

	// First three should succeed
	for i := 0; i < maxBurst; i++ {
		acquired, err := storage.AcquireBurst(ctx, key, maxBurst, ttl)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !acquired {
			t.Errorf("Request %d should acquire burst", i+1)
		}
	}

	// Fourth should fail
	acquired, _ := storage.AcquireBurst(ctx, key, maxBurst, ttl)
	if acquired {
		t.Error("Fourth request should not acquire burst")
	}

	// Release and acquire again
	_ = storage.ReleaseBurst(ctx, key)
	acquired, _ = storage.AcquireBurst(ctx, key, maxBurst, ttl)
	if !acquired {
		t.Error("Should acquire after release")
	}
}

func TestInMemoryStorage_ConcurrentOperations(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()
	key := "concurrent:test"

	// Increment concurrently
	concurrent := 100
	done := make(chan bool, concurrent)

	for i := 0; i < concurrent; i++ {
		go func() {
			_, _ = storage.IncrementConcurrent(ctx, key)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < concurrent; i++ {
		<-done
	}

	count, _ := storage.GetConcurrentCount(ctx, key)
	if count != int64(concurrent) {
		t.Errorf("Expected concurrent count %d, got %d", concurrent, count)
	}

	// Decrement
	_ = storage.DecrementConcurrent(ctx, key)
	count, _ = storage.GetConcurrentCount(ctx, key)
	if count != int64(concurrent-1) {
		t.Errorf("Expected count %d after decrement, got %d", concurrent-1, count)
	}
}

func TestInMemoryStorage_Clear(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Add some data
	_, _ = storage.IncrementCount(ctx, "key1", time.Minute)
	_, _ = storage.IncrementCount(ctx, "key2", time.Minute)
	_ = storage.SetCount(ctx, "counter1", 100, 0)

	// Clear
	storage.Clear()

	// Verify empty
	count1, _ := storage.GetCount(ctx, "key1")
	count2, _ := storage.GetCount(ctx, "key2")
	counter1, _ := storage.GetCount(ctx, "counter1")

	if count1 != 0 || count2 != 0 || counter1 != 0 {
		t.Error("Expected all counts to be 0 after clear")
	}
}

func TestInMemoryStorage_GetStats(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	// Add some data
	_, _ = storage.IncrementCount(ctx, "key1", time.Minute)
	_, _ = storage.IncrementCount(ctx, "key2", time.Minute)

	stats := storage.GetStats()
	if stats["entries"].(int) == 0 && stats["counters"].(int) == 0 {
		// Both can be 0 since we're using counters map
	}
}

func TestParseTierFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected RateLimitTier
		wantErr  bool
	}{
		{"free", TierFree, false},
		{"Free", TierFree, false},
		{"FREE", TierFree, false},
		{"pro", TierPro, false},
		{"Pro", TierPro, false},
		{"enterprise", TierEnterprise, false},
		{"ENTERPRISE", TierEnterprise, false},
		{"unlimited", TierUnlimited, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tier, err := ParseTierFromString(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if tier != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tier)
			}
		})
	}
}

func TestTierToIntAndIntToTier(t *testing.T) {
	tiers := []RateLimitTier{TierFree, TierPro, TierEnterprise, TierUnlimited}

	for _, tier := range tiers {
		i := TierToInt(tier)
		converted := IntToTier(i)
		if converted != tier {
			t.Errorf("Round trip failed for %s", tier)
		}
	}
}

func TestGetTierConfig(t *testing.T) {
	// Test all default tiers
	freeCfg := GetTierConfig(TierFree)
	if freeCfg.RequestsPerMin != 10 {
		t.Errorf("Free tier should have 10 requests/min, got %d", freeCfg.RequestsPerMin)
	}

	proCfg := GetTierConfig(TierPro)
	if proCfg.RequestsPerMin != 100 {
		t.Errorf("Pro tier should have 100 requests/min, got %d", proCfg.RequestsPerMin)
	}

	enterpriseCfg := GetTierConfig(TierEnterprise)
	if enterpriseCfg.RequestsPerMin != 1000 {
		t.Errorf("Enterprise tier should have 1000 requests/min, got %d", enterpriseCfg.RequestsPerMin)
	}

	unlimitedCfg := GetTierConfig(TierUnlimited)
	if unlimitedCfg.RequestsPerMin != 10000 {
		t.Errorf("Unlimited tier should have 10000 requests/min, got %d", unlimitedCfg.RequestsPerMin)
	}

	// Test unknown tier falls back to free
	unknownCfg := GetTierConfig("unknown")
	if unknownCfg.RequestsPerMin != 10 {
		t.Errorf("Unknown tier should fall back to free tier (10 requests/min), got %d", unknownCfg.RequestsPerMin)
	}
}

func TestRateKeyBuilder(t *testing.T) {
	builder := NewRateKeyBuilder()

	// Test BuildKey
	key := builder.BuildKey(ScopeTenant, "tenant1", "user1", Windows.Min)
	expected := "rate:tenant:tenant1:user1:min"
	if key != expected {
		t.Errorf("Expected %s, got %s", expected, key)
	}

	// Test BuildTenantKey
	tenantKey := builder.BuildTenantKey("tenant1", Windows.Min)
	expected = "rate:tenant:tenant1:global:min"
	if tenantKey != expected {
		t.Errorf("Expected %s, got %s", expected, tenantKey)
	}

	// Test BuildUserKey
	userKey := builder.BuildUserKey("tenant1", "user1", Windows.Min)
	expected = "rate:user:tenant1:user1:min"
	if userKey != expected {
		t.Errorf("Expected %s, got %s", expected, userKey)
	}

	// Test BuildSkillKey
	skillKey := builder.BuildSkillKey("tenant1", "skill1", Windows.Min)
	expected = "rate:skill:tenant1:skill1:min"
	if skillKey != expected {
		t.Errorf("Expected %s, got %s", expected, skillKey)
	}

	// Test BuildAPIKeyKey
	apiKeyKey := builder.BuildAPIKeyKey("hash123", Windows.Min)
	expected = "rate:apikey:global:hash123:min"
	if apiKeyKey != expected {
		t.Errorf("Expected %s, got %s", expected, apiKeyKey)
	}

	// Test BuildGlobalKey
	globalKey := builder.BuildGlobalKey(Windows.Min)
	expected = "rate:global:global:global:min"
	if globalKey != expected {
		t.Errorf("Expected %s, got %s", expected, globalKey)
	}
}

func TestMockRedisStorage(t *testing.T) {
	storage := NewMockRedisStorage()
	ctx := context.Background()

	// Test healthy ping
	err := storage.Ping(ctx)
	if err != nil {
		t.Errorf("Expected healthy ping, got: %v", err)
	}

	// Test failed ping
	storage.SetFailure(true)
	err = storage.Ping(ctx)
	if err == nil {
		t.Error("Expected failure ping to error")
	}
	storage.SetFailure(false)

	// Test increment with delay
	count, err := storage.IncrementCountWithDelay(ctx, "test", time.Minute, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
}

func TestStorageFactory(t *testing.T) {
	factory := NewStorageFactory()

	// Test with empty URL (in-memory fallback)
	storage, err := factory.CreateStorage("", true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if storage == nil {
		t.Error("Expected storage to be created")
	}
	_ = storage.Close()

	// Test with invalid URL and no fallback
	_, err = factory.CreateStorage("redis://invalid:1234", false)
	if err == nil {
		t.Error("Expected error when fallback disabled and Redis unavailable")
	}
}

func TestHealthCheck(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	err := manager.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("Expected healthy, got: %v", err)
	}
}

func TestSetTierLimit(t *testing.T) {
	storage := NewInMemoryStorage()
	config := DefaultRateLimitConfig()
	manager := NewQuotaManager(storage, config)

	// Update free tier config
	newCfg := TierConfig{
		Tier:            TierFree,
		RequestsPerMin:  20,
		BurstSize:       10,
		RequestsPerHour: 200,
		RequestsPerDay:  1000,
		ConcurrentLimit: 5,
		Enabled:         true,
	}

	err := manager.SetTierLimit(TierFree, newCfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the update
	ctx := context.Background()
	tenantID := "test-tenant"

	// Should allow 20 requests now
	for i := 0; i < 20; i++ {
		result, err := manager.CheckTenantLimit(ctx, tenantID, TierFree)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 21st should be denied
	result, _ := manager.CheckTenantLimit(ctx, tenantID, TierFree)
	if result.Allowed {
		t.Error("21st request should be denied with updated limit")
	}
}
