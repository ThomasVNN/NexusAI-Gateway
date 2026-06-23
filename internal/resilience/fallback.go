package resilience

import (
	"context"
	"log/slog"
	"time"
)

// ProviderTier represents the pricing tier
type ProviderTier int

const (
	TierSubscription ProviderTier = iota
	TierAPIKey
	TierCheap
	TierFree
)

func (t ProviderTier) String() string {
	switch t {
	case TierSubscription:
		return "subscription"
	case TierAPIKey:
		return "api_key"
	case TierCheap:
		return "cheap"
	case TierFree:
		return "free"
	default:
		return "unknown"
	}
}

// Provider represents a provider for fallback purposes
type Provider struct {
	ID             string
	Name           string
	Tier           ProviderTier
	Health         float64
	QuotaRemaining int64
	QuotaLimit     int64
	IsHealthy      bool
}

// FallbackTier represents a tier in the fallback hierarchy
type FallbackTier struct {
	Tier         ProviderTier
	Providers    []*Provider
	FallbackTo   *FallbackTier
}

// FallbackOrchestrator manages the 4-tier auto-fallback
type FallbackOrchestrator struct {
	tiers    []*FallbackTier
	maxDepth int
}

// NewFallbackOrchestrator creates a new fallback orchestrator
func NewFallbackOrchestrator() *FallbackOrchestrator {
	return &FallbackOrchestrator{
		tiers:    make([]*FallbackTier, 0),
		maxDepth: 4,
	}
}

// BuildTierHierarchy builds the 4-tier fallback hierarchy
func (f *FallbackOrchestrator) BuildTierHierarchy(providers []*Provider) {
	// Group providers by tier
	tiers := map[ProviderTier][]*Provider{}
	for _, p := range providers {
		tiers[p.Tier] = append(tiers[p.Tier], p)
	}

	// Build chain: Subscription -> APIKey -> Cheap -> Free
	var current *FallbackTier
	
	for _, tier := range []ProviderTier{TierSubscription, TierAPIKey, TierCheap, TierFree} {
		providers := tiers[tier]
		if len(providers) == 0 {
			continue
		}
		
		t := &FallbackTier{
			Tier:      tier,
			Providers: providers,
		}
		
		if current != nil {
			current.FallbackTo = t
		}
		
		f.tiers = append(f.tiers, t)
		current = t
	}
}

// ExecuteWithFallback attempts to execute with fallback through tiers
func (f *FallbackOrchestrator) ExecuteWithFallback(
	ctx context.Context,
	request *FallbackRequest,
	executor func(ctx context.Context, provider *Provider) (*FallbackResponse, error),
) (*FallbackResponse, error) {
	var lastErr error
	depth := 0

	// Start from the appropriate tier based on request preferences
	startTier := f.findStartTier(request)
	current := startTier

	for current != nil && depth < f.maxDepth {
		// Find the best provider in this tier
		provider := f.selectBestProvider(current.Providers, request)
		
		if provider != nil {
			slog.InfoContext(ctx, "Attempting provider",
				slog.String("provider", provider.Name),
				slog.String("tier", current.Tier.String()),
			)

			resp, err := executor(ctx, provider)
			if err == nil {
				slog.InfoContext(ctx, "Provider succeeded",
					slog.String("provider", provider.Name),
					slog.Int("attempts", depth+1),
				)
				resp.Attempts = depth + 1
				resp.SelectedProvider = provider.Name
				resp.SelectedTier = current.Tier.String()
				return resp, nil
			}
			
			lastErr = err
			nextTierName := ""
			if current.FallbackTo != nil {
				nextTierName = current.FallbackTo.Tier.String()
			}
			slog.WarnContext(ctx, "Provider failed, trying fallback",
				slog.String("provider", provider.Name),
				slog.String("error", err.Error()),
				slog.String("next_tier", nextTierName),
			)
		}

		current = current.FallbackTo
		depth++
	}

	// All tiers exhausted
	return &FallbackResponse{
		Error:         lastErr,
		Attempts:      depth,
		Fallbacked:    depth > 1,
	}, ErrAllProvidersExhausted
}

func (f *FallbackOrchestrator) findStartTier(request *FallbackRequest) *FallbackTier {
	if len(f.tiers) == 0 {
		return nil
	}

	// If subscription preferred, start there
	if request.PreferSubscription && f.tiers[0].Tier == TierSubscription {
		return f.tiers[0]
	}

	// If any preference, start from that tier
	for _, t := range f.tiers {
		if t.Tier == request.MinimumTier {
			return t
		}
	}

	return f.tiers[0]
}

func (f *FallbackOrchestrator) selectBestProvider(providers []*Provider, request *FallbackRequest) *Provider {
	var best *Provider
	var bestScore float64 = -1

	for _, p := range providers {
		if !p.IsHealthy {
			continue
		}

		score := f.calculateProviderScore(p, request)
		if score > bestScore {
			bestScore = score
			best = p
		}
	}

	return best
}

func (f *FallbackOrchestrator) calculateProviderScore(p *Provider, request *FallbackRequest) float64 {
	var score float64

	// Health is primary factor
	score += p.Health * 100

	// Quota availability
	if p.QuotaLimit > 0 {
		remaining := float64(p.QuotaRemaining) / float64(p.QuotaLimit)
		score += remaining * 50
	} else {
		score += 50 // Unlimited = max quota score
	}

	// Prefer higher tiers
	switch p.Tier {
	case TierSubscription:
		score += 100
	case TierAPIKey:
		score += 70
	case TierCheap:
		score += 40
	case TierFree:
		score += 10
	}

	return score
}

// FallbackRequest contains request parameters for fallback execution
type FallbackRequest struct {
	PreferredTier      ProviderTier
	MinimumTier       ProviderTier
	PreferSubscription bool
	AllowFree         bool
	MaxAttempts       int
	RequestID         string
}

// FallbackResponse contains the response from fallback execution
type FallbackResponse struct {
	Result            interface{}
	Error             error
	Attempts          int
	Fallbacked        bool
	SelectedProvider  string
	SelectedTier      string
	LatencyMs         int64
}

// ProviderHealthTracker tracks provider health for fallback decisions
type ProviderHealthTracker struct {
	health    map[string]*HealthSnapshot
	mu        int
}

type HealthSnapshot struct {
	ProviderID    string
	Health       float64
	LastSuccess  time.Time
	LastFailure  time.Time
	FailureCount int
	TotalCalls   int
}

func NewProviderHealthTracker() *ProviderHealthTracker {
	return &ProviderHealthTracker{
		health: make(map[string]*HealthSnapshot),
	}
}

func (t *ProviderHealthTracker) RecordSuccess(providerID string) {
	t.mu++
	snapshot := t.getOrCreate(providerID)
	snapshot.LastSuccess = time.Now()
	snapshot.TotalCalls++
	// Decay failure count
	if snapshot.FailureCount > 0 {
		snapshot.FailureCount--
	}
	t.mu--
}

func (t *ProviderHealthTracker) RecordFailure(providerID string) {
	t.mu++
	snapshot := t.getOrCreate(providerID)
	snapshot.LastFailure = time.Now()
	snapshot.FailureCount++
	snapshot.TotalCalls++
	
	// Calculate health based on recent performance
	successRate := 1.0
	if snapshot.TotalCalls > 0 {
		successRate = float64(snapshot.TotalCalls-snapshot.FailureCount) / float64(snapshot.TotalCalls)
	}
	snapshot.Health = successRate
	t.mu--
}

func (t *ProviderHealthTracker) GetHealth(providerID string) float64 {
	t.mu++
	defer func() { t.mu-- }()
	
	if snapshot, ok := t.health[providerID]; ok {
		return snapshot.Health
	}
	return 0.5 // Default to 50% health
}

func (t *ProviderHealthTracker) getOrCreate(providerID string) *HealthSnapshot {
	if _, ok := t.health[providerID]; !ok {
		t.health[providerID] = &HealthSnapshot{
			ProviderID: providerID,
			Health:    1.0,
		}
	}
	return t.health[providerID]
}

// AddProvider adds a provider to the fallback orchestrator
func (f *FallbackOrchestrator) AddProvider(provider *Provider) {
	// Find or create the tier
	var tier *FallbackTier
	for _, t := range f.tiers {
		if t.Tier == provider.Tier {
			tier = t
			break
		}
	}

	if tier == nil {
		// Create new tier
		tier = &FallbackTier{
			Tier:      provider.Tier,
			Providers: make([]*Provider, 0),
		}
		
		// Insert in correct order
		inserted := false
		for i, t := range f.tiers {
			if provider.Tier < t.Tier {
				// Insert before this tier
				f.tiers = append(f.tiers[:i], append([]*FallbackTier{tier}, f.tiers[i:]...)...)
				inserted = true
				break
			}
		}
		if !inserted {
			f.tiers = append(f.tiers, tier)
		}
	}

	tier.Providers = append(tier.Providers, provider)
}

// GetTiers returns all tiers
func (f *FallbackOrchestrator) GetTiers() []*FallbackTier {
	return f.tiers
}
