package strategies

import (
	"math/rand"
	"sort"
	"time"
)

// ResetAwareStrategy prioritizes providers with quota resets far in the future
type ResetAwareStrategy struct{}

func (s *ResetAwareStrategy) Name() string {
	return "reset-aware"
}

func (s *ResetAwareStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *ResetAwareStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	type providerScore struct {
		provider *Provider
		score    float64
	}

	scores := make([]providerScore, 0)
	
	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}

		// Calculate score: prefer providers with high remaining quota
		// and quota resets far in the future
		var quotaScore float64
		if p.QuotaLimit > 0 {
			remaining := float64(p.QuotaLimit - p.QuotaUsed) / float64(p.QuotaLimit)
			if remaining < 0 {
				remaining = 0
			}
			quotaScore = remaining
		} else {
			quotaScore = 1.0 // Unlimited
		}

		scores = append(scores, providerScore{
			provider: p,
			score:    quotaScore,
		})
	}

	if len(scores) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].provider, nil
}

// StrictRandomStrategy selects a random provider without deduplication
type StrictRandomStrategy struct{}

func (s *StrictRandomStrategy) Name() string {
	return "strict-random"
}

func (s *StrictRandomStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *StrictRandomStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Simple random selection without health check
	return ctx.Providers[rand.Intn(len(ctx.Providers))], nil
}

// AutoComboStrategy implements the 9-factor scoring algorithm
type AutoComboStrategy struct {
	mode string
}

func (s *AutoComboStrategy) Name() string {
	return "auto"
}

func (s *AutoComboStrategy) Configure(config map[string]interface{}) error {
	if mode, ok := config["mode"].(string); ok {
		s.mode = mode
	} else {
		s.mode = "ship-fast"
	}
	return nil
}

func (s *AutoComboStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Weights for 9 factors
	weights := s.getWeights()

	type scoredProvider struct {
		provider *Provider
		score    float64
	}

	scores := make([]scoredProvider, 0)

	for _, p := range ctx.Providers {
		if !p.IsHealthy {
			continue
		}

		// Calculate composite score
		var totalScore float64

		// 1. Health score (0.22)
		healthScore := ctx.HealthScores[p.ID]
		totalScore += weights.health * healthScore

		// 2. Quota score (0.17)
		quotaScore := ctx.QuotaScores[p.ID]
		totalScore += weights.quota * quotaScore

		// 3. Cost score inverse (0.17)
		costScore := ctx.CostScores[p.ID]
		totalScore += weights.costInv * costScore

		// 4. Latency score inverse (0.13)
		latencyScore := ctx.LatencyScores[p.ID]
		totalScore += weights.latencyInv * latencyScore

		// 5. Task fit score (0.08)
		taskFitScore := ctx.TaskFitScores[p.ID]
		totalScore += weights.taskFit * taskFitScore

		// 6. Specificity match score (0.08)
		specificityScore := ctx.SpecificityScores[p.ID]
		totalScore += weights.specificityMatch * specificityScore

		// 7. Stability score (0.05)
		stabilityScore := ctx.StabilityScores[p.ID]
		totalScore += weights.stability * stabilityScore

		// 8. Tier priority (0.05)
		tierPriorityScore := 1.0 / float64(p.Tier+1)
		totalScore += weights.tierPriority * tierPriorityScore

		// 9. Tier affinity (0.05)
		tierAffinityScore := ctx.TierAffinities[p.ID]
		totalScore += weights.tierAffinity * tierAffinityScore

		scores = append(scores, scoredProvider{
			provider: p,
			score:    totalScore,
		})
	}

	if len(scores) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].provider, nil
}

type factorWeights struct {
	health          float64
	quota           float64
	costInv         float64
	latencyInv      float64
	taskFit         float64
	specificityMatch float64
	stability       float64
	tierPriority    float64
	tierAffinity    float64
}

func (s *AutoComboStrategy) getWeights() factorWeights {
	switch s.mode {
	case "cost-saver":
		return factorWeights{
			health:          0.15,
			quota:           0.15,
			costInv:         0.30,
			latencyInv:      0.10,
			taskFit:         0.08,
			specificityMatch: 0.08,
			stability:       0.05,
			tierPriority:    0.05,
			tierAffinity:    0.04,
		}
	case "quality-first":
		return factorWeights{
			health:          0.30,
			quota:           0.12,
			costInv:         0.10,
			latencyInv:      0.15,
			taskFit:         0.10,
			specificityMatch: 0.10,
			stability:       0.08,
			tierPriority:    0.03,
			tierAffinity:    0.02,
		}
	case "offline-friendly":
		return factorWeights{
			health:          0.20,
			quota:           0.20,
			costInv:         0.15,
			latencyInv:      0.08,
			taskFit:         0.08,
			specificityMatch: 0.08,
			stability:       0.10,
			tierPriority:    0.08,
			tierAffinity:    0.03,
		}
	default: // "ship-fast" (default)
		return factorWeights{
			health:          0.22,
			quota:           0.17,
			costInv:         0.17,
			latencyInv:      0.13,
			taskFit:         0.08,
			specificityMatch: 0.08,
			stability:       0.05,
			tierPriority:    0.05,
			tierAffinity:    0.05,
		}
	}
}

// LKGPStrategy implements Last Known Good Provider
type LKGPStrategy struct {
	lastProvider string
	lockUntil    time.Time
}

func (s *LKGPStrategy) Name() string {
	return "lkgp"
}

func (s *LKGPStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *LKGPStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Check if we have a last known good provider that's still valid
	if s.lastProvider != "" && time.Now().Before(s.lockUntil) {
		for _, p := range ctx.Providers {
			if p.ID == s.lastProvider && p.IsHealthy {
				return p, nil
			}
		}
	}

	// Find the best provider and remember it
	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Select based on health
	var best *Provider
	var bestScore float64 = -1

	for _, p := range healthyProviders {
		score := p.Health * p.StabilityScore
		if score > bestScore {
			bestScore = score
			best = p
		}
	}

	if best != nil {
		s.lastProvider = best.ID
		s.lockUntil = time.Now().Add(5 * time.Minute)
	}

	return best, nil
}

// ContextOptimizedStrategy selects provider based on context size fit
type ContextOptimizedStrategy struct{}

func (s *ContextOptimizedStrategy) Name() string {
	return "context-optimized"
}

func (s *ContextOptimizedStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *ContextOptimizedStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Select provider that best fits the context size
	// Prefer providers with context window that accommodates the request
	var best *Provider
	var bestFitScore float64 = -1

	for _, p := range healthyProviders {
		// Calculate fit score based on context size
		// Higher score = better fit (not too big, not too small)
		fitScore := s.calculateFitScore(p, ctx.ContextSize)
		
		if fitScore > bestFitScore {
			bestFitScore = fitScore
			best = p
		}
	}

	return best, nil
}

func (s *ContextOptimizedStrategy) calculateFitScore(p *Provider, contextSize int) float64 {
	// Assume models have max context of 128K tokens (128000)
	// and minimum useful context of 4K tokens
	maxContext := 128000
	minContext := 4000

	// If context fits well, high score
	if contextSize >= minContext && contextSize <= maxContext {
		// Prefer providers with context closer to request size
		midPoint := float64(maxContext) / 2
		distance := abs(float64(contextSize) - midPoint)
		maxDistance := midPoint
		return 1.0 - (distance / maxDistance)
	}

	// If request is too large, low score
	if contextSize > maxContext {
		return 0.1
	}

	// If request is very small, moderate score
	return 0.5
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ContextRelayStrategy implements session handoff for long conversations
type ContextRelayStrategy struct {
	sessionProviders map[string]string
}

func NewContextRelayStrategy() *ContextRelayStrategy {
	return &ContextRelayStrategy{
		sessionProviders: make(map[string]string),
	}
}

func (s *ContextRelayStrategy) Name() string {
	return "context-relay"
}

func (s *ContextRelayStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *ContextRelayStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Check if we have a provider assigned to this session
	if sessionProvider, ok := s.sessionProviders[ctx.SessionID]; ok {
		for _, p := range ctx.Providers {
			if p.ID == sessionProvider && p.IsHealthy {
				return p, nil
			}
		}
		// Provider no longer healthy, remove from session
		delete(s.sessionProviders, ctx.SessionID)
	}

	// Select a new provider for this session
	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Use auto-combo logic for selection
	auto := &AutoComboStrategy{mode: "ship-fast"}
	autoCtx := &RoutingContext{
		Providers:          healthyProviders,
		HealthScores:       ctx.HealthScores,
		QuotaScores:        ctx.QuotaScores,
		CostScores:         ctx.CostScores,
		LatencyScores:      ctx.LatencyScores,
		TaskFitScores:      ctx.TaskFitScores,
		SpecificityScores:  ctx.SpecificityScores,
		StabilityScores:    ctx.StabilityScores,
		TierPriorities:     ctx.TierPriorities,
		TierAffinities:     ctx.TierAffinities,
	}
	
	selected, err := auto.SelectProvider(autoCtx)
	if err != nil {
		return healthyProviders[0], nil
	}

	// Assign to session
	s.sessionProviders[ctx.SessionID] = selected.ID
	return selected, nil
}

// FillFirstStrategy fills one provider's quota before moving to the next
type FillFirstStrategy struct {
	currentIndex int
	mu           int
}

func (s *FillFirstStrategy) Name() string {
	return "fill-first"
}

func (s *FillFirstStrategy) Configure(config map[string]interface{}) error {
	return nil
}

func (s *FillFirstStrategy) SelectProvider(ctx *RoutingContext) (*Provider, error) {
	if len(ctx.Providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	healthyProviders := make([]*Provider, 0)
	for _, p := range ctx.Providers {
		if p.IsHealthy {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProvider
	}

	// Find provider with most remaining quota
	var best *Provider
	var highestRemaining int64 = -1

	for _, p := range healthyProviders {
		var remaining int64
		if p.QuotaLimit > 0 {
			remaining = p.QuotaLimit - p.QuotaUsed
		} else {
			remaining = int64(^uint64(0) >> 1) // Max int64 for unlimited
		}

		if remaining > highestRemaining {
			highestRemaining = remaining
			best = p
		}
	}

	return best, nil
}
