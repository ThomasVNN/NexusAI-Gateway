package resilience

import (
	"context"
	"testing"
	"time"
)

func TestFallbackOrchestrator(t *testing.T) {
	providers := []*Provider{
		{
			ID:       "openai",
			Name:     "OpenAI",
			Tier:     ProviderTier(TierSubscription),
			Health:   0.95,
			IsHealthy: true,
		},
		{
			ID:       "groq",
			Name:     "Groq",
			Tier:     ProviderTier(TierCheap),
			Health:   0.88,
			IsHealthy: true,
		},
		{
			ID:       "ollama",
			Name:     "Ollama",
			Tier:     ProviderTier(TierFree),
			Health:   0.75,
			IsHealthy: true,
		},
	}

	orchestrator := NewFallbackOrchestrator()
	orchestrator.BuildTierHierarchy(providers)

	if len(orchestrator.GetTiers()) != 3 {
		t.Errorf("Expected 3 tiers, got %d", len(orchestrator.GetTiers()))
	}
}

func TestFallbackExecution(t *testing.T) {
	providers := []*Provider{
		{
			ID:             "failing",
			Name:           "Failing Provider",
			Tier:           ProviderTier(TierSubscription),
			Health:         0.95,
			IsHealthy:      true,
			QuotaRemaining: 50000,
			QuotaLimit:     50000,
		},
		{
			ID:             "backup",
			Name:           "Backup Provider",
			Tier:           ProviderTier(TierAPIKey),
			Health:         0.90,
			IsHealthy:      true,
			QuotaRemaining: 8000,
			QuotaLimit:     10000,
		},
		{
			ID:             "free",
			Name:           "Free Provider",
			Tier:           ProviderTier(TierFree),
			Health:         0.70,
			IsHealthy:      true,
			QuotaRemaining: 0,
			QuotaLimit:     0,
		},
	}

	orchestrator := NewFallbackOrchestrator()
	orchestrator.BuildTierHierarchy(providers)

	callCount := make(map[string]int)

	executor := func(ctx context.Context, provider *Provider) (*FallbackResponse, error) {
		callCount[provider.ID]++
		if provider.ID == "failing" {
			return nil, context.DeadlineExceeded
		}
		return &FallbackResponse{
			Result:           "success",
			SelectedProvider: provider.Name,
		}, nil
	}

	resp, err := orchestrator.ExecuteWithFallback(context.Background(), &FallbackRequest{}, executor)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.SelectedProvider != "Backup Provider" {
		t.Errorf("Expected Backup Provider, got %s", resp.SelectedProvider)
	}

	if callCount["failing"] != 1 {
		t.Errorf("Expected failing to be called once, got %d", callCount["failing"])
	}

	if callCount["backup"] != 1 {
		t.Errorf("Expected backup to be called once, got %d", callCount["backup"])
	}
}

func TestFallbackAllExhausted(t *testing.T) {
	providers := []*Provider{
		{
			ID:        "fail1",
			Name:      "Fail 1",
			Tier:      ProviderTier(TierSubscription),
			Health:    0.95,
			IsHealthy: true,
		},
		{
			ID:        "fail2",
			Name:      "Fail 2",
			Tier:      ProviderTier(TierAPIKey),
			Health:    0.90,
			IsHealthy: true,
		},
	}

	orchestrator := NewFallbackOrchestrator()
	orchestrator.BuildTierHierarchy(providers)

	executor := func(ctx context.Context, provider *Provider) (*FallbackResponse, error) {
		return nil, context.DeadlineExceeded
	}

	resp, err := orchestrator.ExecuteWithFallback(context.Background(), &FallbackRequest{}, executor)

	if err != ErrAllProvidersExhausted {
		t.Errorf("Expected ErrAllProvidersExhausted, got %v", err)
	}

	if resp.Attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", resp.Attempts)
	}
}

func TestProviderCircuitBreaker(t *testing.T) {
	cb := NewProviderCB("test-provider")

	// Initial state should be CLOSED
	if cb.GetState() != CBStateClosed {
		t.Errorf("Expected initial state CLOSED, got %s", cb.GetState())
	}

	// Allow should return true when CLOSED
	if !cb.Allow() {
		t.Error("Expected Allow() to return true when CLOSED")
	}

	// Record failures until threshold
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Should now be OPEN
	if cb.GetState() != CBStateOpen {
		t.Errorf("Expected state OPEN after 5 failures, got %s", cb.GetState())
	}

	// Allow should return false when OPEN
	if cb.Allow() {
		t.Error("Expected Allow() to return false when OPEN")
	}
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	cb := NewProviderCB("test-provider")
	cb.CooldownDuration = 10 * time.Millisecond

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	if cb.GetState() != CBStateOpen {
		t.Errorf("Expected OPEN state, got %s", cb.GetState())
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Should transition to HALF_OPEN
	if !cb.Allow() {
		t.Error("Expected Allow() to return true after cooldown")
	}

	// Record success to close
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.GetState() != CBStateClosed {
		t.Errorf("Expected CLOSED after success threshold, got %s", cb.GetState())
	}
}

func TestCircuitBreakerHalfOpenFailure(t *testing.T) {
	cb := NewProviderCB("test-provider")
	cb.CooldownDuration = 10 * time.Millisecond

	// Open the circuit
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Allow first request
	cb.Allow()

	// Record failure in half-open
	cb.RecordFailure()

	// Should be back to OPEN
	if cb.GetState() != CBStateOpen {
		t.Errorf("Expected OPEN after failure in half-open, got %s", cb.GetState())
	}
}

func TestCircuitBreakerDegradedMode(t *testing.T) {
	cb := NewProviderCB("test-provider")
	cb.DegradedThreshold = 3

	// Record failures to reach degraded threshold
	for i := 0; i < 10; i++ {
		cb.RecordFailure()
	}

	// Should be in DEGRADED or OPEN state
	state := cb.GetState()
	if state != CBStateDegraded && state != CBStateOpen {
		t.Errorf("Expected DEGRADED or OPEN state, got %s", state)
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewProviderCB("test-provider")

	// Record some requests
	for i := 0; i < 3; i++ {
		cb.RecordSuccess()
	}
	for i := 0; i < 2; i++ {
		cb.RecordFailure()
	}

	stats := cb.Stats()

	if stats.TotalRequests != 5 {
		t.Errorf("Expected 5 total requests, got %d", stats.TotalRequests)
	}

	if stats.TotalSuccesses != 3 {
		t.Errorf("Expected 3 successes, got %d", stats.TotalSuccesses)
	}

	if stats.TotalFailures != 2 {
		t.Errorf("Expected 2 failures, got %d", stats.TotalFailures)
	}
}

func TestConnectionCooldown(t *testing.T) {
	cc := NewConnectionCooldown("account1", "provider1")

	// Should be able to use initially
	if !cc.CanUse() {
		t.Error("Expected CanUse() to return true initially")
	}

	// Set cooldown
	cc.SetCooldown(100*time.Millisecond, "rate_limit")

	// Should not be able to use during cooldown
	if cc.CanUse() {
		t.Error("Expected CanUse() to return false during cooldown")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Should be able to use now
	if !cc.CanUse() {
		t.Error("Expected CanUse() to return true after cooldown")
	}
}

func TestCooldownManager(t *testing.T) {
	manager := NewCooldownMgr()

	// Get cooldown for account/provider
	cc1 := manager.GetCooldown("account1", "provider1")
	cc2 := manager.GetCooldown("account1", "provider1")

	// Should return same instance
	if cc1 != cc2 {
		t.Error("Expected same cooldown instance for same account/provider")
	}

	// Get different cooldown
	cc3 := manager.GetCooldown("account2", "provider1")
	if cc1 == cc3 {
		t.Error("Expected different cooldown for different account")
	}

	// Remove cooldown
	manager.RemoveCooldown("account1", "provider1")

	// Should create new instance after removal
	cc4 := manager.GetCooldown("account1", "provider1")
	if cc4 == cc1 {
		t.Error("Expected new cooldown instance after removal")
	}
}

func TestModelLockout(t *testing.T) {
	ml := NewModelLockout("openai", "gpt-4")

	// Should not be locked initially
	if ml.IsLocked() {
		t.Error("Expected IsLocked() to return false initially")
	}

	// Lock for 100ms
	ml.Lock(100*time.Millisecond, "maintenance", "admin")

	// Should be locked
	if !ml.IsLocked() {
		t.Error("Expected IsLocked() to return true after lock")
	}

	// Wait for lock to expire
	time.Sleep(150 * time.Millisecond)

	// Should not be locked
	if ml.IsLocked() {
		t.Error("Expected IsLocked() to return false after lock expires")
	}
}

func TestLockoutManager(t *testing.T) {
	manager := NewLockoutMgr()

	// Get lockout
	ml1 := manager.GetLockout("openai", "gpt-4")
	ml2 := manager.GetLockout("openai", "gpt-4")

	// Should return same instance
	if ml1 != ml2 {
		t.Error("Expected same lockout instance")
	}

	// Check if locked
	if manager.IsLocked("openai", "gpt-4") {
		t.Error("Expected IsLocked() to return false initially")
	}

	// Lock
	ml1.Lock(100*time.Millisecond, "test", "system")

	if !manager.IsLocked("openai", "gpt-4") {
		t.Error("Expected IsLocked() to return true after lock")
	}

	// Wait and check
	time.Sleep(150 * time.Millisecond)

	if manager.IsLocked("openai", "gpt-4") {
		t.Error("Expected IsLocked() to return false after expiration")
	}
}

func TestThreeLayerResilience(t *testing.T) {
	lr := NewThreeLayerResilience()

	// Should allow routing initially
	if !lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return true initially")
	}

	// Record some failures to open circuit
	cb := lr.GetProviderCB("openai")
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Should not allow routing now
	if lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return false when circuit is open")
	}

	// Record success should transition to HALF_OPEN, allowing limited requests
	lr.RecordSuccess("openai", "gpt-4", "account1")
	// In HALF_OPEN state, requests are allowed
	if !lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return true in HALF_OPEN state")
	}
}

func TestThreeLayerCooldown(t *testing.T) {
	lr := NewThreeLayerResilience()

	// Set cooldown
	lr.SetCooldown("account1", "openai", 100*time.Millisecond, "rate_limit")

	// Should not allow routing during cooldown
	if lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return false during cooldown")
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Should allow routing now
	if !lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return true after cooldown")
	}
}

func TestThreeLayerLockout(t *testing.T) {
	lr := NewThreeLayerResilience()

	// Lock model
	lr.LockModel("openai", "gpt-4", 100*time.Millisecond, "test", "system")

	// Should not allow routing to locked model
	if lr.CanRoute("openai", "gpt-4", "account1") {
		t.Error("Expected CanRoute() to return false for locked model")
	}

	// Other models should still work
	if !lr.CanRoute("openai", "gpt-3.5", "account1") {
		t.Error("Expected CanRoute() to return true for unlocked model")
	}
}

func TestProviderHealthTracker(t *testing.T) {
	tracker := NewProviderHealthTracker()

	// Record some successes
	for i := 0; i < 10; i++ {
		tracker.RecordSuccess("provider1")
	}

	// Health should be high
	health := tracker.GetHealth("provider1")
	if health < 0.9 {
		t.Errorf("Expected health > 0.9 after 10 successes, got %f", health)
	}

	// Record some failures
	for i := 0; i < 5; i++ {
		tracker.RecordFailure("provider1")
	}

	// Health should decrease
	health = tracker.GetHealth("provider1")
	if health > 0.7 {
		t.Errorf("Expected health < 0.7 after failures, got %f", health)
	}
}

func TestTierOrdering(t *testing.T) {
	providers := []*Provider{
		{ID: "free", Name: "Free", Tier: TierFree, IsHealthy: true},
		{ID: "api", Name: "API Key", Tier: TierAPIKey, IsHealthy: true},
		{ID: "cheap", Name: "Cheap", Tier: TierCheap, IsHealthy: true},
		{ID: "sub", Name: "Subscription", Tier: TierSubscription, IsHealthy: true},
	}

	orchestrator := NewFallbackOrchestrator()
	orchestrator.BuildTierHierarchy(providers)

	tiers := orchestrator.GetTiers()

	// Verify tier order
	expectedOrder := []ProviderTier{TierSubscription, TierAPIKey, TierCheap, TierFree}
	for i, tier := range tiers {
		if tier.Tier != expectedOrder[i] {
			t.Errorf("Expected tier %v at position %d, got %v", expectedOrder[i], i, tier.Tier)
		}
	}
}

// Benchmark tests
func BenchmarkFallbackExecution(b *testing.B) {
	providers := []*Provider{
		{ID: "p1", Name: "P1", Tier: ProviderTier(TierSubscription), IsHealthy: true, Health: 0.95},
		{ID: "p2", Name: "P2", Tier: ProviderTier(TierAPIKey), IsHealthy: true, Health: 0.90},
		{ID: "p3", Name: "P3", Tier: ProviderTier(TierCheap), IsHealthy: true, Health: 0.85},
	}

	orchestrator := NewFallbackOrchestrator()
	orchestrator.BuildTierHierarchy(providers)

	executor := func(ctx context.Context, provider *Provider) (*FallbackResponse, error) {
		return &FallbackResponse{Result: "success"}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = orchestrator.ExecuteWithFallback(context.Background(), &FallbackRequest{}, executor)
	}
}

func BenchmarkCircuitBreakerAllow(b *testing.B) {
	cb := NewProviderCB("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Allow()
	}
}

func BenchmarkThreeLayerCanRoute(b *testing.B) {
	lr := NewThreeLayerResilience()
	lr.GetProviderCB("openai")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lr.CanRoute("openai", "gpt-4", "account1")
	}
}
