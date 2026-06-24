package resilience

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Tier 1-4 provider names from the spec
const (
	// Tier 1: Claude Code, Codex, Copilot quota
	Tier1ClaudeCode  = "claude-code"
	Tier1OpenAICode = "openai-codex"
	Tier1Copilot    = "github-copilot"

	// Tier 2: DeepSeek, Grok, xAI
	Tier2DeepSeek = "deepseek"
	Tier2Grok     = "xai-grok"
	Tier2XAI      = "xai"

	// Tier 3: GLM $0.5, MiniMax $0.2
	Tier3GLM     = "zhipu-glm"
	Tier3MiniMax = "minimax"

	// Tier 4: Kiro, Qoder, Pollinations (free)
	Tier4Kiro        = "kiro"
	Tier4Qoder       = "qoder"
	Tier4Pollinations = "pollinations"
)

// AutoFallbackTier defines the auto-fallback hierarchy
type AutoFallbackTier struct {
	Tier          int
	Name          string
	Providers    []*Provider
	LatencyBudget time.Duration
	IsHealthy     bool
}

// FourTierAutoFallback implements the 4-tier auto-fallback system
type FourTierAutoFallback struct {
	mu             sync.RWMutex
	tiers         []*AutoFallbackTier
	circuitBreaker *CircuitBreaker
	maxDepth      int
	requestCounts  map[string]int64
}

// CircuitBreaker tracks provider health for circuit breaking
type CircuitBreaker struct {
	mu           sync.RWMutex
	states      map[string]*CircuitState
	failureThreshold int
	timeout      time.Duration
}

// CircuitState tracks the state of a circuit breaker
type CircuitState struct {
	State       CircuitStateType
	FailureCount int
	LastFailure time.Time
	NextRetry   time.Time
}

// CircuitStateType defines circuit breaker states
type CircuitStateType int

const (
	CircuitClosed CircuitStateType = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitStateType) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// FallbackRequestOptions contains options for fallback requests
type FallbackRequestOptions struct {
	TenantID           string
	PreferredTier      int           // 1-4, 0 = no preference
	MinimumTier       int           // Minimum acceptable tier
	AllowFree         bool          // Allow free tier
	MaxAttempts       int           // Max total attempts
	MaxLatencyMs      int64         // Maximum allowed latency
	RequireVision     bool          // Require vision capability
	RequireStreaming  bool          // Require streaming capability
	ContextWindow     int           // Required context window size
	PreferredProvider string        // Preferred specific provider
	Strategy          string        // "speed", "cost", "quality", "balanced"
}

// FallbackResult contains the result of a fallback operation
type FallbackResult struct {
	Provider       *Provider
	ModelName      string
	Tier           int
	Attempts       int
	TotalLatencyMs int64
	Fallbacked     bool
	FallbackPath   []string
	Error          error
	Success        bool
}

// NewFourTierAutoFallback creates a new 4-tier auto-fallback system
func NewFourTierAutoFallback() *FourTierAutoFallback {
	f := &FourTierAutoFallback{
		tiers:         make([]*AutoFallbackTier, 4),
		circuitBreaker: NewCircuitBreaker(5, 30*time.Second),
		maxDepth:      4,
		requestCounts: make(map[string]int64),
	}

	f.initializeTiers()
	return f
}

// initializeTiers sets up the default 4-tier hierarchy
func (f *FourTierAutoFallback) initializeTiers() {
	// Tier 1: Premium (Claude Code, Codex, Copilot quota)
	f.tiers[0] = &AutoFallbackTier{
		Tier:          1,
		Name:          "premium",
		LatencyBudget: 2000 * time.Millisecond,
		IsHealthy:     true,
		Providers: []*Provider{
			{Name: Tier1ClaudeCode, Tier: TierSubscription, Priority: 1},
			{Name: Tier1OpenAICode, Tier: TierSubscription, Priority: 2},
			{Name: Tier1Copilot, Tier: TierSubscription, Priority: 3},
		},
	}

	// Tier 2: Standard API (DeepSeek, Grok, xAI)
	f.tiers[1] = &AutoFallbackTier{
		Tier:          2,
		Name:          "standard",
		LatencyBudget: 3000 * time.Millisecond,
		IsHealthy:     true,
		Providers: []*Provider{
			{Name: Tier2DeepSeek, Tier: TierAPIKey, Priority: 1},
			{Name: Tier2Grok, Tier: TierAPIKey, Priority: 2},
			{Name: Tier2XAI, Tier: TierAPIKey, Priority: 3},
		},
	}

	// Tier 3: Budget (GLM $0.5, MiniMax $0.2)
	f.tiers[2] = &AutoFallbackTier{
		Tier:          3,
		Name:          "budget",
		LatencyBudget: 5000 * time.Millisecond,
		IsHealthy:     true,
		Providers: []*Provider{
			{Name: Tier3GLM, Tier: TierCheap, Priority: 1},
			{Name: Tier3MiniMax, Tier: TierCheap, Priority: 2},
		},
	}

	// Tier 4: Free (Kiro, Qoder, Pollinations)
	f.tiers[3] = &AutoFallbackTier{
		Tier:          4,
		Name:          "free",
		LatencyBudget: 10000 * time.Millisecond,
		IsHealthy:     true,
		Providers: []*Provider{
			{Name: Tier4Kiro, Tier: TierFree, Priority: 1},
			{Name: Tier4Qoder, Tier: TierFree, Priority: 2},
			{Name: Tier4Pollinations, Tier: TierFree, Priority: 3},
		},
	}
}

// ExecuteWithFallback executes a request with automatic fallback through tiers
func (f *FourTierAutoFallback) ExecuteWithFallback(
	ctx context.Context,
	req *FallbackRequestOptions,
	executor func(ctx context.Context, provider *Provider, modelName string) (*FallbackResult, error),
) (*FallbackResult, error) {
	
	f.mu.Lock()
	var lastErr error
	var result *FallbackResult
	fallbackPath := make([]string, 0)
	attempts := 0

	// Determine starting tier
	startTier := f.determineStartTier(req)

	for tierIdx := startTier; tierIdx < f.maxDepth && attempts < req.MaxAttempts; tierIdx++ {
		tier := f.tiers[tierIdx]

		// Skip unhealthy tiers
		if !tier.IsHealthy {
			slog.WarnContext(ctx, "Skipping unhealthy tier", slog.Int("tier", tier.Tier))
			continue
		}

		// Check circuit breaker for all providers in tier
		availableProviders := f.getAvailableProviders(tier, req)
		if len(availableProviders) == 0 {
			continue
		}

		// Try each provider in this tier
		for _, provider := range availableProviders {
			if attempts >= req.MaxAttempts {
				break
			}

			// Check latency budget
			if req.MaxLatencyMs > 0 && int64(tier.LatencyBudget.Milliseconds()) > req.MaxLatencyMs {
				continue
			}

			// Check circuit breaker
			if f.circuitBreaker.IsOpen(provider.Name) {
				slog.WarnContext(ctx, "Provider circuit open, skipping",
					slog.String("provider", provider.Name),
					slog.Int("tier", tier.Tier))
				continue
			}

			// Select model for provider
			modelName := f.selectModelForProvider(provider, req)

			slog.InfoContext(ctx, "Attempting provider",
				slog.String("provider", provider.Name),
				slog.String("model", modelName),
				slog.Int("tier", tier.Tier),
				slog.Int("attempt", attempts+1),
			)

			startTime := time.Now()
			_, err := executor(ctx, provider, modelName)
			latencyMs := time.Since(startTime).Milliseconds()
			attempts++

			if err == nil {
				// Success
				f.circuitBreaker.RecordSuccess(provider.Name)
				f.requestCounts[provider.Name]++

				result = &FallbackResult{
					Provider:        provider,
					ModelName:       modelName,
					Tier:            tier.Tier,
					Attempts:        attempts,
					TotalLatencyMs:  latencyMs,
					Fallbacked:      len(fallbackPath) > 0,
					FallbackPath:    fallbackPath,
					Success:         true,
				}

				f.mu.Unlock()
				return result, nil
			}

			// Record failure
			lastErr = err
			f.circuitBreaker.RecordFailure(provider.Name)
			fallbackPath = append(fallbackPath, fmt.Sprintf("%s:%s", provider.Name, err.Error()))

			slog.WarnContext(ctx, "Provider failed, trying next",
				slog.String("provider", provider.Name),
				slog.String("error", err.Error()),
				slog.Int64("latency_ms", latencyMs),
			)
		}
	}

	f.mu.Unlock()

	// All providers exhausted
	return &FallbackResult{
		Attempts:     attempts,
		Fallbacked:   true,
		FallbackPath: fallbackPath,
		Error:        lastErr,
		Success:      false,
	}, fmt.Errorf("all providers exhausted after %d attempts", attempts)
}

// determineStartTier determines which tier to start from
func (f *FourTierAutoFallback) determineStartTier(req *FallbackRequestOptions) int {
	if req.PreferredTier > 0 && req.PreferredTier <= 4 {
		return req.PreferredTier - 1
	}
	if req.MinimumTier > 0 && req.MinimumTier <= 4 {
		return req.MinimumTier - 1
	}
	return 0 // Start from Tier 1
}

// getAvailableProviders returns providers that are healthy and have capacity
func (f *FourTierAutoFallback) getAvailableProviders(tier *AutoFallbackTier, req *FallbackRequestOptions) []*Provider {
	var available []*Provider

	for _, p := range tier.Providers {
		// Check if circuit is open
		if f.circuitBreaker.IsOpen(p.Name) {
			continue
		}

		// Skip free tier if not allowed
		if tier.Tier == 4 && !req.AllowFree {
			continue
		}

		// Check preferred provider
		if req.PreferredProvider != "" && p.Name != req.PreferredProvider {
			continue
		}

		available = append(available, p)
	}

	return available
}

// selectModelForProvider selects the best model for a provider based on requirements
func (f *FourTierAutoFallback) selectModelForProvider(provider *Provider, req *FallbackRequestOptions) string {
	// Provider-specific model selection
	switch provider.Name {
	case Tier1ClaudeCode:
		return "claude-3-5-sonnet-20240620"
	case Tier1OpenAICode:
		return "gpt-4o"
	case Tier1Copilot:
		return "gpt-4-turbo"
	case Tier2DeepSeek:
		return "deepseek-chat"
	case Tier2Grok, Tier2XAI:
		return "grok-2"
	case Tier3GLM:
		return "glm-4"
	case Tier3MiniMax:
		return "moonshot-v1-32k"
	case Tier4Kiro, Tier4Qoder, Tier4Pollinations:
		return "llama-3-70b-instruct"
	default:
		return "gpt-4o-mini"
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		states:           make(map[string]*CircuitState),
		failureThreshold: failureThreshold,
		timeout:          timeout,
	}
}

// RecordSuccess records a successful call
func (cb *CircuitBreaker) RecordSuccess(providerID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if state, exists := cb.states[providerID]; exists {
		if state.State == CircuitHalfOpen {
			state.State = CircuitClosed
			state.FailureCount = 0
		} else if state.FailureCount > 0 {
			state.FailureCount--
		}
	}
}

// RecordFailure records a failed call
func (cb *CircuitBreaker) RecordFailure(providerID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.getOrCreateState(providerID)
	state.FailureCount++
	state.LastFailure = time.Now()

	if state.FailureCount >= cb.failureThreshold {
		state.State = CircuitOpen
		state.NextRetry = time.Now().Add(cb.timeout)
		slog.Warn("Circuit breaker opened",
			slog.String("provider", providerID),
			slog.Int("failures", state.FailureCount),
		)
	}
}

// IsOpen checks if the circuit is open for a provider
func (cb *CircuitBreaker) IsOpen(providerID string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.getOrCreateState(providerID)

	if state.State == CircuitOpen {
		// Check if we should transition to half-open
		if time.Now().After(state.NextRetry) {
			state.State = CircuitHalfOpen
			return false
		}
		return true
	}

	return false
}

// getOrCreateState gets or creates a circuit state
func (cb *CircuitBreaker) getOrCreateState(providerID string) *CircuitState {
	if state, exists := cb.states[providerID]; exists {
		return state
	}

	state := &CircuitState{
		State:       CircuitClosed,
		FailureCount: 0,
	}
	cb.states[providerID] = state
	return state
}

// GetCircuitState returns the circuit state for a provider
func (cb *CircuitBreaker) GetCircuitState(providerID string) *CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.getOrCreateState(providerID)
}

// GetAllCircuitStates returns all circuit states
func (cb *CircuitBreaker) GetAllCircuitStates() map[string]*CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	result := make(map[string]*CircuitState)
	for k, v := range cb.states {
		result[k] = v
	}
	return result
}

// SetTierHealth sets the health status of a tier
func (f *FourTierAutoFallback) SetTierHealth(tier int, healthy bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if tier >= 1 && tier <= 4 {
		f.tiers[tier-1].IsHealthy = healthy
	}
}

// GetTiers returns all tiers
func (f *FourTierAutoFallback) GetTiers() []*AutoFallbackTier {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]*AutoFallbackTier, len(f.tiers))
	copy(result, f.tiers)
	return result
}

// GetProviderHealth returns health info for all providers
func (f *FourTierAutoFallback) GetProviderHealth() map[string]*ProviderHealth {
	f.mu.RLock()
	defer f.mu.RUnlock()

	health := make(map[string]*ProviderHealth)

	for _, tier := range f.tiers {
		for _, p := range tier.Providers {
			circuitState := f.circuitBreaker.GetCircuitState(p.Name)
			health[p.Name] = &ProviderHealth{
				ProviderName: p.Name,
				Tier:         tier.Tier,
				TierName:     tier.Name,
				IsHealthy:    tier.IsHealthy && circuitState.State != CircuitOpen,
				CircuitState: circuitState.State.String(),
				RequestCount: f.requestCounts[p.Name],
			}
		}
	}

	return health
}

// ProviderHealth contains health information for a provider
type ProviderHealth struct {
	ProviderName string `json:"provider_name"`
	Tier         int    `json:"tier"`
	TierName     string `json:"tier_name"`
	IsHealthy    bool   `json:"is_healthy"`
	CircuitState string `json:"circuit_state"`
	RequestCount int64  `json:"request_count"`
}

// FourTierStats returns statistics about the fallback system
type FourTierStats struct {
	Tiers           []TierStats        `json:"tiers"`
	CircuitBreakers map[string]CircuitStats `json:"circuit_breakers"`
}

// TierStats contains statistics for a tier
type TierStats struct {
	Tier         int      `json:"tier"`
	Name         string   `json:"name"`
	ProviderCount int      `json:"provider_count"`
	Healthy      bool     `json:"healthy"`
	LatencyBudgetMs int64 `json:"latency_budget_ms"`
}

// CircuitStats contains circuit breaker statistics
type CircuitStats struct {
	State         string `json:"state"`
	FailureCount int    `json:"failure_count"`
	LastFailure  string `json:"last_failure,omitempty"`
	NextRetry    string `json:"next_retry,omitempty"`
}

// GetStats returns fallback statistics
func (f *FourTierAutoFallback) GetStats() FourTierStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	tierStats := make([]TierStats, len(f.tiers))
	for i, tier := range f.tiers {
		tierStats[i] = TierStats{
			Tier:           tier.Tier,
			Name:           tier.Name,
			ProviderCount: len(tier.Providers),
			Healthy:       tier.IsHealthy,
			LatencyBudgetMs: int64(tier.LatencyBudget.Milliseconds()),
		}
	}

	circuitStats := make(map[string]CircuitStats)
	for name, state := range f.circuitBreaker.GetAllCircuitStates() {
		stats := CircuitStats{
			State:         state.State.String(),
			FailureCount: state.FailureCount,
		}
		if !state.LastFailure.IsZero() {
			stats.LastFailure = state.LastFailure.Format(time.RFC3339)
		}
		if !state.NextRetry.IsZero() {
			stats.NextRetry = state.NextRetry.Format(time.RFC3339)
		}
		circuitStats[name] = stats
	}

	return FourTierStats{
		Tiers:           tierStats,
		CircuitBreakers: circuitStats,
	}
}
