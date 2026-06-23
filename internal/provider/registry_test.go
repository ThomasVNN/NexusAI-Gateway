package provider

import (
	"testing"
	"time"
)

func TestProviderCreation(t *testing.T) {
	p := NewExtendedProvider("openai", "OpenAI", TierSubscription, true)

	if p.ID != "openai" {
		t.Errorf("Expected ID 'openai', got '%s'", p.ID)
	}

	if p.Name != "OpenAI" {
		t.Errorf("Expected Name 'OpenAI', got '%s'", p.Name)
	}

	if p.Tier != TierSubscription {
		t.Errorf("Expected TierSubscription, got %v", p.Tier)
	}

	if !p.OAuthSupport {
		t.Error("Expected OAuthSupport to be true")
	}

	if p.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", p.Status)
	}
}

func TestAccountCreation(t *testing.T) {
	account := NewAccount("openai", "acc1", "sk-123")

	if account.ProviderID != "openai" {
		t.Errorf("Expected ProviderID 'openai', got '%s'", account.ProviderID)
	}

	if account.AccountID != "acc1" {
		t.Errorf("Expected AccountID 'acc1', got '%s'", account.AccountID)
	}

	if account.APIKey != "sk-123" {
		t.Errorf("Expected APIKey 'sk-123', got '%s'", account.APIKey)
	}

	if account.Status != AccountActive {
		t.Errorf("Expected AccountActive, got %v", account.Status)
	}

	if account.QuotaLimit != 0 {
		t.Errorf("Expected QuotaLimit 0, got %d", account.QuotaLimit)
	}
}

func TestAccountQuotaRemaining(t *testing.T) {
	account := &Account{
		ProviderID: "openai",
		AccountID:  "acc1",
		QuotaLimit: 1000,
		QuotaUsed:  300,
	}

	remaining := account.QuotaRemaining()
	if remaining != 700 {
		t.Errorf("Expected 700 remaining, got %d", remaining)
	}
}

func TestAccountUnlimitedQuota(t *testing.T) {
	account := &Account{
		ProviderID: "openai",
		AccountID:  "acc1",
		QuotaLimit: 0, // Unlimited
		QuotaUsed:  500,
	}

	remaining := account.QuotaRemaining()
	if remaining != -1 {
		t.Errorf("Expected -1 for unlimited, got %d", remaining)
	}
}

func TestAccountConsumeQuota(t *testing.T) {
	account := &Account{
		ProviderID: "openai",
		AccountID:  "acc1",
		QuotaLimit: 1000,
		QuotaUsed:  0,
	}

	account.ConsumeQuota(100)
	if account.QuotaUsed != 100 {
		t.Errorf("Expected QuotaUsed 100, got %d", account.QuotaUsed)
	}

	account.ConsumeQuota(200)
	if account.QuotaUsed != 300 {
		t.Errorf("Expected QuotaUsed 300, got %d", account.QuotaUsed)
	}
}

func TestAccountCanUse(t *testing.T) {
	account := &Account{
		ProviderID: "openai",
		AccountID:  "acc1",
		QuotaLimit: 1000,
		QuotaUsed:  500,
		Status:     AccountActive,
	}

	// Should be able to use
	if !account.CanUse() {
		t.Error("Expected CanUse() to return true")
	}

	// Exhaust quota
	account.QuotaUsed = 1000
	if account.CanUse() {
		t.Error("Expected CanUse() to return false when quota exhausted")
	}

	// Reset and test inactive
	account.QuotaUsed = 500
	account.Status = AccountInactive
	if account.CanUse() {
		t.Error("Expected CanUse() to return false when inactive")
	}
}

func TestProviderAddAccount(t *testing.T) {
	p := NewExtendedProvider("openai", "OpenAI", TierSubscription, true)

	account1 := NewAccount("openai", "acc1", "sk-1")
	account2 := NewAccount("openai", "acc2", "sk-2")

	p.AddAccount(account1)
	p.AddAccount(account2)

	if len(p.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(p.Accounts))
	}
}

func TestProviderGetAvailableAccount(t *testing.T) {
	p := NewExtendedProvider("openai", "OpenAI", TierSubscription, true)

	// Add an inactive account first
	inactiveAccount := &Account{
		ProviderID: "openai",
		AccountID:  "inactive",
		Status:     AccountInactive,
	}

	// Add an active account
	activeAccount := NewAccount("openai", "active", "sk-active")
	activeAccount.QuotaLimit = 1000
	activeAccount.QuotaUsed = 500

	p.AddAccount(inactiveAccount)
	p.AddAccount(activeAccount)

	account, err := p.GetAvailableAccount()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.AccountID != "active" {
		t.Errorf("Expected 'active' account, got '%s'", account.AccountID)
	}
}

func TestProviderGetAvailableAccountExhausted(t *testing.T) {
	p := NewExtendedProvider("openai", "OpenAI", TierSubscription, true)

	// Add account with exhausted quota
	exhaustedAccount := &Account{
		ProviderID: "openai",
		AccountID:  "exhausted",
		Status:     AccountActive,
		QuotaLimit: 1000,
		QuotaUsed:  1000,
	}

	p.AddAccount(exhaustedAccount)

	_, err := p.GetAvailableAccount()
	if err != ErrNoAvailableAccount {
		t.Errorf("Expected ErrNoAvailableAccount, got %v", err)
	}
}

func TestProviderIsHealthy(t *testing.T) {
	p := NewExtendedProvider("openai", "OpenAI", TierSubscription, true)
	p.Health = NewExtendedHealthStatus("openai")
	p.Health.IsHealthy = true
	p.Status = "active"

	if !p.IsHealthy() {
		t.Error("Expected IsHealthy() to return true")
	}

	// Test with unhealthy
	p.Health.IsHealthy = false
	if p.IsHealthy() {
		t.Error("Expected IsHealthy() to return false when health is false")
	}

	// Test with inactive status
	p.Health.IsHealthy = true
	p.Status = "inactive"
	if p.IsHealthy() {
		t.Error("Expected IsHealthy() to return false when status is inactive")
	}
}

func TestProviderTierPriority(t *testing.T) {
	tests := []struct {
		tier    ProviderTier
		expected int
	}{
		{TierSubscription, 1},
		{TierAPIKey, 2},
		{TierCheap, 3},
		{TierFree, 4},
	}

	for _, tt := range tests {
		priority := TierPriority(tt.tier)
		if priority != tt.expected {
			t.Errorf("Expected tier %v priority %d, got %d", tt.tier, tt.expected, priority)
		}
	}
}

func TestProviderTierString(t *testing.T) {
	tests := []struct {
		tier    ProviderTier
		expected string
	}{
		{TierSubscription, "subscription"},
		{TierAPIKey, "api_key"},
		{TierCheap, "cheap"},
		{TierFree, "free"},
		{ProviderTier(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.tier.String() != tt.expected {
			t.Errorf("Expected %v.String() to be '%s', got '%s'", tt.tier, tt.expected, tt.tier.String())
		}
	}
}

func TestAccountStatusString(t *testing.T) {
	tests := []struct {
		status   AccountStatus
		expected string
	}{
		{AccountActive, "active"},
		{AccountInactive, "inactive"},
		{AccountSuspended, "suspended"},
		{AccountRateLimited, "rate_limited"},
		{AccountQuotaExceeded, "quota_exceeded"},
		{AccountStatus(99), "unknown"},
	}

	for _, tt := range tests {
		if tt.status.String() != tt.expected {
			t.Errorf("Expected %v.String() to be '%s', got '%s'", tt.status, tt.expected, tt.status.String())
		}
	}
}

func TestRegistryCreation(t *testing.T) {
	registry := NewProviderRegistry()

	providers := registry.ListProviders()
	if len(providers) == 0 {
		t.Error("Expected registry to be initialized with well-known providers")
	}

	// Should have at least 40 providers (we have 41)
	if len(providers) < 40 {
		t.Errorf("Expected at least 40 providers, got %d", len(providers))
	}
}

func TestRegistryGetProvider(t *testing.T) {
	registry := NewProviderRegistry()

	provider, err := registry.GetProvider("openai")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if provider.ID != "openai" {
		t.Errorf("Expected ID 'openai', got '%s'", provider.ID)
	}
}

func TestRegistryGetProviderNotFound(t *testing.T) {
	registry := NewProviderRegistry()

	_, err := registry.GetProvider("nonexistent")
	if err != ErrProviderNotFound {
		t.Errorf("Expected ErrProviderNotFound, got %v", err)
	}
}

func TestRegistryListProvidersByTier(t *testing.T) {
	registry := NewProviderRegistry()

	subscription := registry.ListProvidersByTier(TierSubscription)
	if len(subscription) == 0 {
		t.Error("Expected subscription tier providers")
	}

	// Verify all are subscription tier
	for _, p := range subscription {
		if p.Tier != TierSubscription {
			t.Errorf("Expected TierSubscription, got %v for provider %s", p.Tier, p.ID)
		}
	}
}

func TestRegistryListHealthyProviders(t *testing.T) {
	registry := NewProviderRegistry()

	// All well-known providers should be healthy initially
	healthy := registry.ListHealthyProviders()
	if len(healthy) == 0 {
		t.Error("Expected at least some healthy providers")
	}
}

func TestRegistryAddProvider(t *testing.T) {
	registry := NewProviderRegistry()

	newProvider := NewExtendedProvider("custom", "Custom Provider", TierAPIKey, false)
	err := registry.AddProvider(newProvider)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify it was added
	provider, err := registry.GetProvider("custom")
	if err != nil {
		t.Fatalf("Expected to get custom provider, got error: %v", err)
	}

	if provider.Name != "Custom Provider" {
		t.Errorf("Expected 'Custom Provider', got '%s'", provider.Name)
	}
}

func TestRegistryUpdateProvider(t *testing.T) {
	registry := NewProviderRegistry()

	provider, _ := registry.GetProvider("openai")
	originalStatus := provider.Status

	// Update
	provider.Status = "degraded"
	err := registry.UpdateProvider(provider)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify update
	updated, _ := registry.GetProvider("openai")
	if updated.Status != "degraded" {
		t.Errorf("Expected status 'degraded', got '%s'", updated.Status)
	}

	// Restore
	provider.Status = originalStatus
	registry.UpdateProvider(provider)
}

func TestHealthStatusCreation(t *testing.T) {
	status := NewExtendedHealthStatus("openai")

	if status.ProviderID != "openai" {
		t.Errorf("Expected ProviderID 'openai', got '%s'", status.ProviderID)
	}

	if !status.IsHealthy {
		t.Error("Expected IsHealthy to be true")
	}

	if status.SuccessRate != 100 {
		t.Errorf("Expected SuccessRate 100, got %f", status.SuccessRate)
	}

	if status.CircuitState != "CLOSED" {
		t.Errorf("Expected CircuitState 'CLOSED', got '%s'", status.CircuitState)
	}
}

func TestWellKnownProviders(t *testing.T) {
	providers := WellKnownProviders()

	// Should have at least 40 providers
	if len(providers) < 40 {
		t.Errorf("Expected at least 40 providers, got %d", len(providers))
	}

	// Verify structure of first provider
	if len(providers) > 0 {
		p := providers[0]
		if p.ID == "" {
			t.Error("Expected provider ID to be set")
		}
		if p.Name == "" {
			t.Error("Expected provider Name to be set")
		}
	}
}

func TestDBProviderConversion(t *testing.T) {
	now := time.Now()
	db := &DBProviderRow{
		ID:            "openai",
		Name:          "OpenAI",
		Tier:          0,
		OAuthSupport:  true,
		Endpoint:      "https://api.openai.com",
		AuthType:      "oauth",
		Status:        "active",
		Priority:      100,
		CostPer1KIn:  0.005,
		CostPer1KOut: 0.015,
		AvgLatencyMs: 1000,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	p := db.ToExtendedProvider()

	if p.ID != "openai" {
		t.Errorf("Expected ID 'openai', got '%s'", p.ID)
	}

	if p.Tier != TierSubscription {
		t.Errorf("Expected TierSubscription, got %v", p.Tier)
	}

	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("Expected BaseURL 'https://api.openai.com', got '%s'", p.BaseURL)
	}
}

// Benchmark tests
func BenchmarkRegistryListProviders(b *testing.B) {
	registry := NewProviderRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.ListProviders()
	}
}

func BenchmarkRegistryGetProvider(b *testing.B) {
	registry := NewProviderRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.GetProvider("openai")
	}
}

func BenchmarkWellKnownProviders(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WellKnownProviders()
	}
}
