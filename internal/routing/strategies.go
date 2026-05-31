package routing

import (
	"context"
	"math/rand"
	"sort"
	"sync"
)

// RoutingStrategy defines all available routing strategies
type RoutingStrategy string

const (
	// Priority-based strategies
	StrategyPriority  RoutingStrategy = "priority"   // Highest priority first
	StrategyFillFirst RoutingStrategy = "fill_first" // Drain subscription before paying
	StrategyLeastUsed RoutingStrategy = "least_used" // Use least-used account

	// Load distribution strategies
	StrategyRoundRobin RoutingStrategy = "round_robin" // Rotate through accounts
	StrategyWeighted   RoutingStrategy = "weighted"    // Weight-based distribution
	StrategyP2C        RoutingStrategy = "p2c"         // Power of Two Choices

	// Optimization strategies
	StrategyCostOptimized RoutingStrategy = "cost_optimized"    // Cheapest viable
	StrategyLatencyOpt    RoutingStrategy = "latency_optimized" // Lowest latency
	StrategyQualityFirst  RoutingStrategy = "quality_first"     // Best quality

	// Context-aware strategies
	StrategyContextRelay RoutingStrategy = "context_relay"     // Handoff with summary
	StrategyContextOpt   RoutingStrategy = "context_optimized" // Optimize context usage

	// Privacy strategies
	StrategyRandom       RoutingStrategy = "random"        // Random selection
	StrategyStrictRandom RoutingStrategy = "strict_random" // Strict privacy routing

	// Smart strategies
	StrategyAuto       RoutingStrategy = "auto"        // 9-factor scoring
	StrategyLKGP       RoutingStrategy = "lkgp"        // Last Known Good Provider
	StrategyResetAware RoutingStrategy = "reset_aware" // Prioritize by quota reset time
)

// StrategyConfig holds configuration for routing strategies
type StrategyConfig struct {
	Strategy     RoutingStrategy
	Weights      map[string]float64 // For weighted strategies
	MinLatencyMs int                // Latency constraint
	MaxCostPer1K float64            // Cost constraint
}

// RoutingChain represents a combo/routing chain
type RoutingChain struct {
	ID        string             `json:"id"`
	TenantID  string             `json:"tenant_id"`
	Name      string             `json:"name"`
	Strategy  RoutingStrategy    `json:"strategy"`
	Steps     []RoutingChainStep `json:"steps"`
	IsActive  bool               `json:"is_active"`
	CreatedAt int64              `json:"created_at"`
	UpdatedAt int64              `json:"updated_at"`
}

// RoutingChainStep represents a single step in a routing chain
type RoutingChainStep struct {
	ID            string   `json:"id"`
	ChainID       string   `json:"chain_id"`
	Order         int      `json:"order"`
	ProviderID    string   `json:"provider_id"`
	ModelID       string   `json:"model_id"`
	FallbackModel string   `json:"fallback_model,omitempty"`
	Weight        float64  `json:"weight"`
	MinLatencyMs  int      `json:"min_latency_ms,omitempty"`
	MaxCostPer1K  float64  `json:"max_cost_per_1k,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
}

// AutoComboConfig for smart model selection
type AutoComboConfig struct {
	Mode    string           `json:"mode"` // "balanced", "coding", "fast", "cheap", "offline", "smart"
	Weights AutoComboWeights `json:"weights"`
}

// AutoComboWeights defines weights for the 9-factor auto-combo scoring
type AutoComboWeights struct {
	Health      float64 `json:"health"`       // Provider health (0-1)
	Quota       float64 `json:"quota"`        // Remaining quota (0-1)
	Cost        float64 `json:"cost"`         // Cost score (inverse)
	Latency     float64 `json:"latency"`      // Latency score (inverse)
	SuccessRate float64 `json:"success_rate"` // Historical success (0-1)
	Freshness   float64 `json:"freshness"`    // Time since last use
	Quality     float64 `json:"quality"`      // Model capability
	Exploration float64 `json:"exploration"`  // Exploration bonus
}

// DefaultAutoComboWeights returns the default weight configuration
func DefaultAutoComboWeights() AutoComboWeights {
	return AutoComboWeights{
		Health:      1.0,
		Quota:       1.0,
		Cost:        0.5,
		Latency:     0.5,
		SuccessRate: 1.0,
		Freshness:   0.3,
		Quality:     1.0,
		Exploration: 0.1,
	}
}

// ModelScore represents a scored candidate model
type ModelScore struct {
	ModelID    string
	ProviderID string
	TotalScore float64
	Breakdown  map[string]float64
}

// ScoringEngine implements the 9-factor auto-combo scoring
type ScoringEngine struct {
	mu      sync.RWMutex
	history map[string]*ModelScore
	lastIdx map[string]int // For round-robin
}

// NewScoringEngine creates a new scoring engine
func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{
		history: make(map[string]*ModelScore),
		lastIdx: make(map[string]int),
	}
}

// ScoreModel calculates the composite score for a model
func (e *ScoringEngine) ScoreModel(ctx context.Context, model *ModelInfo, config *AutoComboConfig) *ModelScore {
	breakdown := make(map[string]float64)

	// Health score (0-1)
	healthScore := 1.0
	if model.HealthStatus == "degraded" {
		healthScore = 0.5
	} else if model.HealthStatus == "unhealthy" {
		healthScore = 0.1
	}
	breakdown["health"] = healthScore * config.Weights.Health

	// Quota score (0-1)
	quotaScore := 1.0
	if model.QuotaRemaining < 0.1 {
		quotaScore = 0.1
	} else if model.QuotaRemaining < 0.5 {
		quotaScore = 0.5
	}
	breakdown["quota"] = quotaScore * config.Weights.Quota

	// Cost score (inverse - lower is better)
	costScore := 1.0 - (model.CostPer1KInput+model.CostPer1KOutput)/10.0
	if costScore < 0 {
		costScore = 0
	}
	breakdown["cost"] = costScore * config.Weights.Cost

	// Latency score (inverse - lower is better)
	latencyScore := 1.0 - float64(model.AvgLatencyMs)/5000.0
	if latencyScore < 0 {
		latencyScore = 0
	}
	breakdown["latency"] = latencyScore * config.Weights.Latency

	// Success rate (0-1)
	successScore := model.SuccessRate
	breakdown["success_rate"] = successScore * config.Weights.SuccessRate

	// Freshness (time since last use)
	freshnessScore := 1.0
	if model.LastUsedAt > 0 {
		hoursSinceUse := float64(model.AvgLatencyMs) / 3600.0
		freshnessScore = hoursSinceUse / 168.0
		if freshnessScore > 1.0 {
			freshnessScore = 1.0
		}
	}
	breakdown["freshness"] = freshnessScore * config.Weights.Freshness

	// Quality (based on capabilities)
	qualityScore := float64(len(model.Capabilities)) / 10.0
	if qualityScore > 1.0 {
		qualityScore = 1.0
	}
	breakdown["quality"] = qualityScore * config.Weights.Quality

	// Exploration bonus (for less-used models)
	explorationBonus := config.Weights.Exploration * (1.0 - successScore)
	breakdown["exploration"] = explorationBonus

	// Calculate total
	var total float64
	for _, score := range breakdown {
		total += score
	}

	return &ModelScore{
		ModelID:    model.Name,
		ProviderID: model.Provider,
		TotalScore: total,
		Breakdown:  breakdown,
	}
}

// RouteByStrategy routes a request using the specified strategy
func (r *ModelRouter) RouteByStrategy(ctx context.Context, req *RoutingRequest, strategy RoutingStrategy) (*RouteTarget, error) {
	switch strategy {
	case StrategyPriority:
		return r.routeByPriority(ctx, req)
	case StrategyFillFirst:
		return r.routeByFillFirst(ctx, req)
	case StrategyLeastUsed:
		return r.routeByLeastUsed(ctx, req)
	case StrategyRoundRobin:
		return r.routeByRoundRobin(ctx, req)
	case StrategyWeighted:
		return r.routeByWeighted(ctx, req)
	case StrategyP2C:
		return r.routeByP2C(ctx, req)
	case StrategyCostOptimized:
		return r.routeByCostOptimized(ctx, req)
	case StrategyLatencyOpt:
		return r.routeByLatency(ctx, req)
	case StrategyQualityFirst:
		return r.routeByQuality(ctx, req)
	case StrategyAuto:
		return r.routeByAuto(ctx, req)
	case StrategyRandom:
		return r.routeByRandom(ctx, req)
	case StrategyStrictRandom:
		return r.routeByStrictRandom(ctx, req)
	case StrategyResetAware:
		return r.routeByResetAware(ctx, req)
	case StrategyLKGP:
		return r.routeByLKGP(ctx, req)
	default:
		return r.routeByPriority(ctx, req)
	}
}

// routeByPriority selects the highest priority model
func (r *ModelRouter) routeByPriority(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Sort by priority (lower number = higher priority)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})

	selected := candidates[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyPriority,
	}, nil
}

// routeByFillFirst prioritizes draining subscriptions before paying
func (r *ModelRouter) routeByFillFirst(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Sort by tier: subscription > apikey > cheap > free
	// Within same tier, prefer models with more quota
	sort.Slice(candidates, func(i, j int) bool {
		tierI := getProviderTier(candidates[i].Provider)
		tierJ := getProviderTier(candidates[j].Provider)
		if tierI != tierJ {
			return tierI < tierJ
		}
		return candidates[i].QuotaRemaining > candidates[j].QuotaRemaining
	})

	selected := candidates[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyFillFirst,
	}, nil
}

// routeByLeastUsed selects the model with lowest usage
func (r *ModelRouter) routeByLeastUsed(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Sort by total request count (lowest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RequestCount < candidates[j].RequestCount
	})

	selected := candidates[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyLeastUsed,
	}, nil
}

// routeByRoundRobin rotates through candidates
func (r *ModelRouter) routeByRoundRobin(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	key := req.TenantID + ":round_robin"
	idx := r.scoring.lastIdx[key]
	selected := candidates[idx%len(candidates)]
	r.scoring.lastIdx[key] = (idx + 1) % len(candidates)

	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyRoundRobin,
	}, nil
}

// routeByWeighted selects based on configured weights
func (r *ModelRouter) routeByWeighted(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Calculate total weight
	var totalWeight float64
	for _, m := range candidates {
		totalWeight += m.Weight
	}

	// Random selection based on weights
	rng := rand.Float64() * totalWeight
	var cumulative float64
	for _, m := range candidates {
		cumulative += m.Weight
		if rng <= cumulative {
			return &RouteTarget{
				ModelID:    m.Name,
				ProviderID: m.Provider,
				Strategy:   StrategyWeighted,
			}, nil
		}
	}

	// Fallback to first
	return &RouteTarget{
		ModelID:    candidates[0].Name,
		ProviderID: candidates[0].Provider,
		Strategy:   StrategyWeighted,
	}, nil
}

// routeByP2C implements Power of Two Choices algorithm
func (r *ModelRouter) routeByP2C(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) < 2 {
		return r.routeByPriority(ctx, req)
	}

	// Randomly select 2 candidates
	idx1 := rand.Intn(len(candidates))
	idx2 := (idx1 + 1 + rand.Intn(len(candidates)-1)) % len(candidates)
	c1, c2 := candidates[idx1], candidates[idx2]

	// Pick the one with lower latency
	if c2.AvgLatencyMs < c1.AvgLatencyMs {
		c1 = c2
	}

	return &RouteTarget{
		ModelID:    c1.Name,
		ProviderID: c1.Provider,
		Strategy:   StrategyP2C,
	}, nil
}

// routeByCostOptimized selects the cheapest viable model
func (r *ModelRouter) routeByCostOptimized(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Filter by required capabilities
	var viable []*ModelInfo
	for _, m := range candidates {
		if hasCapabilities(m, req.RequiredCapabilities) {
			viable = append(viable, m)
		}
	}

	if len(viable) == 0 {
		viable = candidates
	}

	// Sort by total cost per 1k tokens
	sort.Slice(viable, func(i, j int) bool {
		return (viable[i].CostPer1KInput+viable[i].CostPer1KOutput)*1000 <
			(viable[j].CostPer1KInput+viable[j].CostPer1KOutput)*1000
	})

	selected := viable[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyCostOptimized,
	}, nil
}

// routeByLatency selects the fastest model
func (r *ModelRouter) routeByLatency(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Filter by latency constraint
	var viable []*ModelInfo
	for _, m := range candidates {
		if req.MaxLatencyMs == 0 || m.AvgLatencyMs <= req.MaxLatencyMs {
			viable = append(viable, m)
		}
	}

	if len(viable) == 0 {
		viable = candidates
	}

	// Sort by latency (lowest first)
	sort.Slice(viable, func(i, j int) bool {
		return viable[i].AvgLatencyMs < viable[j].AvgLatencyMs
	})

	selected := viable[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyLatencyOpt,
	}, nil
}

// routeByQuality selects the most capable model
func (r *ModelRouter) routeByQuality(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Sort by number of capabilities (most first)
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].Capabilities) > len(candidates[j].Capabilities)
	})

	selected := candidates[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyQualityFirst,
	}, nil
}

// routeByAuto implements 9-factor auto-combo scoring
func (r *ModelRouter) routeByAuto(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Get auto-combo config
	config := r.getAutoComboConfig(req)

	// Score all candidates
	var scores []*ModelScore
	for _, m := range candidates {
		score := r.scoring.ScoreModel(ctx, m, config)
		scores = append(scores, score)
	}

	// Sort by total score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	selected := scores[0]
	return &RouteTarget{
		ModelID:    selected.ModelID,
		ProviderID: selected.ProviderID,
		Strategy:   StrategyAuto,
	}, nil
}

// routeByRandom selects a random model
func (r *ModelRouter) routeByRandom(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	selected := candidates[rand.Intn(len(candidates))]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyRandom,
	}, nil
}

// routeByStrictRandom selects a random model (pure random, no weighting)
func (r *ModelRouter) routeByStrictRandom(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	return r.routeByRandom(ctx, req)
}

// routeByResetAware prioritizes by quota reset time
func (r *ModelRouter) routeByResetAware(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Sort by quota reset time (soonest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].QuotaResetAt < candidates[j].QuotaResetAt
	})

	selected := candidates[0]
	return &RouteTarget{
		ModelID:    selected.Name,
		ProviderID: selected.Provider,
		Strategy:   StrategyResetAware,
	}, nil
}

// routeByLKGP uses Last Known Good Provider
func (r *ModelRouter) routeByLKGP(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	candidates := r.getCandidates(req)
	if len(candidates) == 0 {
		return nil, ErrNoModelsAvailable
	}

	// Find LKGP for this tenant
	lkgp := r.getLKGP(req.TenantID)
	if lkgp != "" {
		for _, m := range candidates {
			if m.Name == lkgp && m.IsActive && m.AvgLatencyMs < 5000 {
				return &RouteTarget{
					ModelID:    m.Name,
					ProviderID: m.Provider,
					Strategy:   StrategyLKGP,
				}, nil
			}
		}
	}

	// Fallback to priority
	return r.routeByPriority(ctx, req)
}

// Helper methods

func (r *ModelRouter) getCandidates(req *RoutingRequest) []*ModelInfo {
	var candidates []*ModelInfo
	for _, m := range r.models {
		if !m.IsActive {
			continue
		}
		if hasCapabilities(m, req.RequiredCapabilities) {
			candidates = append(candidates, m)
		}
	}
	return candidates
}

func getProviderTier(providerID string) int {
	// Map provider tiers: subscription=0, apikey=1, cheap=2, free=3
	tierMap := map[string]int{
		"openai":         1,
		"anthropic":      1,
		"google":         1,
		"claude-code":    0,
		"codex":          0,
		"github-copilot": 0,
		"groq":           3,
		"deepseek":       2,
		"ollama":         3,
		"lmstudio":       3,
	}
	if tier, ok := tierMap[providerID]; ok {
		return tier
	}
	return 1 // Default to apikey tier
}

func (r *ModelRouter) getAutoComboConfig(req *RoutingRequest) *AutoComboConfig {
	config := &AutoComboConfig{
		Mode:    "balanced",
		Weights: DefaultAutoComboWeights(),
	}

	// Adjust weights based on mode
	switch req.Strategy {
	case "fast":
		config.Weights.Latency = 2.0
		config.Weights.Cost = 0.3
		config.Mode = "fast"
	case "cheap":
		config.Weights.Cost = 2.0
		config.Weights.Latency = 0.3
		config.Mode = "cheap"
	case "quality":
		config.Weights.Quality = 2.0
		config.Weights.Exploration = 0.2
		config.Mode = "quality"
	}

	return config
}

func (r *ModelRouter) getLKGP(tenantID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return the last known good provider from history
	key := tenantID + ":lkgp"
	if score, ok := r.scoring.history[key]; ok {
		return score.ModelID
	}
	return ""
}
