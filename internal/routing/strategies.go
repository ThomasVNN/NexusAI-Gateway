package routing

import (
	"context"
	"math/rand"
	"sort"
	"sync"
)

// Extended routing strategies (additions to existing constants in engine.go)
const (
	// Basic strategies (aligning with engine.go naming)
	StrategyQualityOptimized RoutingStrategy = "quality"

	// Advanced strategies (beyond basic)
	StrategyFallback   RoutingStrategy = "fallback"
	StrategyRoundRobin RoutingStrategy = "round_robin"
	StrategyWeighted   RoutingStrategy = "weighted"
	StrategyAffinity   RoutingStrategy = "affinity"
	StrategyBurst      RoutingStrategy = "burst"
	StrategyDegraded   RoutingStrategy = "degraded"
	StrategyGuardian   RoutingStrategy = "guardian"
	StrategyParallel   RoutingStrategy = "parallel"
	StrategyCascade    RoutingStrategy = "cascade"
	StrategyGeo        RoutingStrategy = "geo"
	StrategyFreeTier   RoutingStrategy = "free_tier"
	StrategyComposite  RoutingStrategy = "composite"
)

// StrategyConfig holds configuration for a routing strategy
type StrategyConfig struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Strategy    RoutingStrategy `json:"strategy"`
	Description string          `json:"description"`
	Priority    int             `json:"priority"` // Lower = higher priority
	Enabled     bool            `json:"enabled"`
	Config      map[string]any  `json:"config,omitempty"`
	Weights     map[string]int  `json:"weights,omitempty"`   // For weighted routing
	Fallbacks   []string        `json:"fallbacks,omitempty"` // For fallback chain
}

// GeoLocation represents a geographic location
type GeoLocation struct {
	Region  string `json:"region"` // us-east, us-west, eu-west, asia-east, etc.
	Country string `json:"country"`
	City    string `json:"city,omitempty"`
}

// RouteTargetExtended includes additional metadata for routing decisions
type RouteTargetExtended struct {
	ModelInfo
	Score         float64 `json:"score"`
	GeoMatch      bool    `json:"geo_match,omitempty"`
	FreeTierAvail bool    `json:"free_tier_available"`
	HealthScore   float64 `json:"health_score"`
}

// StrategySelector manages routing strategies
type StrategySelector struct {
	mu          sync.RWMutex
	strategies  map[RoutingStrategy]*StrategyConfig
	activeIndex map[RoutingStrategy]int // For round-robin
	affinity    map[string]string       // tenant -> model mapping
	health      map[string]float64      // model -> health score
}

func NewStrategySelector() *StrategySelector {
	return &StrategySelector{
		strategies:  make(map[RoutingStrategy]*StrategyConfig),
		activeIndex: make(map[RoutingStrategy]int),
		affinity:    make(map[string]string),
		health:      make(map[string]float64),
	}
}

// Select implements the routing strategy selection
func (s *StrategySelector) Select(ctx context.Context, candidates []*ModelInfo, req *RoutingRequest, strategy RoutingStrategy) (*RouteTargetExtended, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update health scores
	for _, c := range candidates {
		if _, ok := s.health[c.Name]; !ok {
			s.health[c.Name] = 100.0
		}
	}

	// Score candidates
	scored := s.scoreCandidates(candidates, req)

	switch strategy {
	case StrategyLatencyOptimized:
		return s.selectLatency(scored)
	case StrategyCostOptimized:
		return s.selectCost(scored)
	case StrategyQualityOptimized:
		return s.selectQuality(scored)
	case StrategyFallback:
		return s.selectFallback(candidates, req)
	case StrategyRoundRobin:
		return s.selectRoundRobin(scored)
	case StrategyWeighted:
		return s.selectWeighted(scored, req.TenantID)
	case StrategyAffinity:
		return s.selectAffinity(candidates, req.TenantID)
	case StrategyBurst:
		return s.selectBurst(scored)
	case StrategyDegraded:
		return s.selectDegraded(candidates, req)
	case StrategyGuardian:
		return s.selectGuardian(candidates, req)
	case StrategyParallel:
		return s.selectParallel(candidates, req)
	case StrategyCascade:
		return s.selectCascade(candidates, req)
	case StrategyGeo:
		return s.selectGeo(scored, req)
	case StrategyFreeTier:
		return s.selectFreeTier(scored)
	case StrategyComposite:
		return s.selectComposite(candidates, req)
	default:
		return s.selectCost(scored)
	}
}

// scoreCandidates calculates scores for all candidates
func (s *StrategySelector) scoreCandidates(candidates []*ModelInfo, req *RoutingRequest) []*RouteTargetExtended {
	result := make([]*RouteTargetExtended, 0, len(candidates))

	for _, c := range candidates {
		if !c.IsActive {
			continue
		}

		target := &RouteTargetExtended{
			ModelInfo:     *c,
			Score:         0,
			GeoMatch:      false,
			FreeTierAvail: s.isFreeTier(c),
			HealthScore:   s.health[c.Name],
		}

		// Calculate composite score
		target.Score = s.calculateScore(c, req)
		result = append(result, target)
	}

	return result
}

// calculateScore computes a weighted score for a model
func (s *StrategySelector) calculateScore(model *ModelInfo, req *RoutingRequest) float64 {
	// Default weights
	latencyWeight := 0.2
	costWeight := 0.3
	qualityWeight := 0.2
	healthWeight := 0.15
	freeTierWeight := 0.15

	// Calculate individual scores (0-100)
	latencyScore := s.scoreLatency(model.AvgLatencyMs)
	costScore := s.scoreCost(model.CostPer1KInput + model.CostPer1KOutput)
	qualityScore := s.scoreQuality(model)
	healthScore := s.health[model.Name]
	freeTierScore := 0.0
	if s.isFreeTier(model) {
		freeTierScore = 100.0
	}

	// Weighted composite score
	score := latencyScore*latencyWeight +
		costScore*costWeight +
		qualityScore*qualityWeight +
		healthScore*healthWeight +
		freeTierScore*freeTierWeight

	return score
}

func (s *StrategySelector) scoreLatency(latencyMs int64) float64 {
	// Lower latency = higher score
	// 100ms = 100, 2000ms = 0
	score := 100.0 - float64(latencyMs)/20.0
	if score < 0 {
		return 0
	}
	return score
}

func (s *StrategySelector) scoreCost(costPer1K float64) float64 {
	// Lower cost = higher score
	// $0.0001 = 100, $0.10 = 0
	score := 100.0 - costPer1K*10000.0
	if score < 0 {
		return 0
	}
	return score
}

func (s *StrategySelector) scoreQuality(model *ModelInfo) float64 {
	// More capabilities = higher score
	base := 50.0
	base += float64(len(model.Capabilities) * 10)
	base += float64(100 - model.Priority*10) // Lower priority number = higher score
	if base > 100 {
		return 100
	}
	return base
}

func (s *StrategySelector) isFreeTier(model *ModelInfo) bool {
	// Check if model has free tier
	freeModels := map[string]bool{
		"gpt-4o-mini":      true,
		"claude-3-5-haiku": true,
		"gemini-1.5-flash": true,
	}
	return freeModels[model.Name]
}

// Strategy selection methods
func (s *StrategySelector) selectLatency(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].AvgLatencyMs < candidates[j].AvgLatencyMs
	})

	return candidates[0], nil
}

func (s *StrategySelector) selectCost(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	sort.Slice(candidates, func(i, j int) bool {
		iCost := candidates[i].CostPer1KInput + candidates[i].CostPer1KOutput
		jCost := candidates[j].CostPer1KInput + candidates[j].CostPer1KOutput
		return iCost < jCost
	})

	return candidates[0], nil
}

func (s *StrategySelector) selectQuality(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})

	return candidates[0], nil
}

func (s *StrategySelector) selectFallback(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Use first candidate as primary, rest as fallbacks
	return &RouteTargetExtended{
		ModelInfo: *candidates[0],
		Score:     100,
	}, nil
}

func (s *StrategySelector) selectRoundRobin(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	s.activeIndex[StrategyRoundRobin] = (s.activeIndex[StrategyRoundRobin] + 1) % len(candidates)
	return candidates[s.activeIndex[StrategyRoundRobin]], nil
}

func (s *StrategySelector) selectWeighted(candidates []*RouteTargetExtended, tenantID string) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Random weighted selection based on priority
	totalWeight := 0
	for _, c := range candidates {
		totalWeight += 100 - c.Priority*10
	}

	r := rand.Intn(totalWeight)
	current := 0
	for _, c := range candidates {
		current += 100 - c.Priority*10
		if r <= current {
			return c, nil
		}
	}

	return candidates[0], nil
}

func (s *StrategySelector) selectAffinity(candidates []*ModelInfo, tenantID string) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Check if tenant has an affinity model
	if modelName, ok := s.affinity[tenantID]; ok {
		for _, c := range candidates {
			if c.Name == modelName {
				return &RouteTargetExtended{
					ModelInfo: *c,
					Score:     100,
				}, nil
			}
		}
	}

	// Set new affinity
	s.affinity[tenantID] = candidates[0].Name
	return &RouteTargetExtended{
		ModelInfo: *candidates[0],
		Score:     100,
	}, nil
}

func (s *StrategySelector) selectBurst(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// For burst traffic, select lowest latency models
	return s.selectLatency(candidates)
}

func (s *StrategySelector) selectDegraded(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Sort by cost (cheapest first for degraded mode)
	sort.Slice(candidates, func(i, j int) bool {
		return (candidates[i].CostPer1KInput + candidates[i].CostPer1KOutput) <
			(candidates[j].CostPer1KInput + candidates[j].CostPer1KOutput)
	})

	return &RouteTargetExtended{
		ModelInfo: *candidates[0],
		Score:     50, // Lower score for degraded mode
	}, nil
}

func (s *StrategySelector) selectGuardian(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Guardian mode: prioritize health and reliability
	sorted := make([]*RouteTargetExtended, len(candidates))
	for i, c := range candidates {
		sorted[i] = &RouteTargetExtended{
			ModelInfo:   *c,
			Score:       s.health[c.Name],
			HealthScore: s.health[c.Name],
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].HealthScore > sorted[j].HealthScore
	})

	return sorted[0], nil
}

func (s *StrategySelector) selectParallel(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// For parallel, select the best candidate (others used in parallel)
	return s.selectComposite(candidates, req)
}

func (s *StrategySelector) selectCascade(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Cascade: use the highest quality primary
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})

	return &RouteTargetExtended{
		ModelInfo: *candidates[0],
		Score:     80, // Leave room for cascade fallbacks
	}, nil
}

func (s *StrategySelector) selectGeo(candidates []*RouteTargetExtended, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Sort by score (already includes geo considerations)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].GeoMatch != candidates[j].GeoMatch {
			return candidates[i].GeoMatch
		}
		return candidates[i].Score > candidates[j].Score
	})

	return candidates[0], nil
}

func (s *StrategySelector) selectFreeTier(candidates []*RouteTargetExtended) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Find free tier candidates first
	for _, c := range candidates {
		if c.FreeTierAvail {
			return c, nil
		}
	}

	// Fall back to cheapest if no free tier available
	return s.selectCost(candidates)
}

func (s *StrategySelector) selectComposite(candidates []*ModelInfo, req *RoutingRequest) (*RouteTargetExtended, error) {
	if len(candidates) == 0 {
		return nil, ErrNoModelAvailable
	}

	// Composite: use best scoring candidate
	scored := s.scoreCandidates(candidates, req)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored[0], nil
}

// RecordHealth records health metrics for a model
func (s *StrategySelector) RecordHealth(modelName string, healthScore float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health[modelName] = healthScore
}

// ============================================================
// ComboEngine provides the auto-combo functionality with 9-factor scoring
// ============================================================
type ComboEngine struct {
	selector *StrategySelector
	Config   *ComboConfig
}

type ComboConfig struct {
	LatencyWeight         float64 `json:"latency_weight"`
	CostWeight            float64 `json:"cost_weight"`
	QualityWeight         float64 `json:"quality_weight"`
	AvailabilityWeight    float64 `json:"availability_weight"`
	FreeTierWeight        float64 `json:"free_tier_weight"`
	ContextWindowWeight   float64 `json:"context_window_weight"`
	TokenEfficiencyWeight float64 `json:"token_efficiency_weight"`
	ReliabilityWeight     float64 `json:"reliability_weight"`
	PreferenceWeight      float64 `json:"preference_weight"`
}

func NewComboEngine() *ComboEngine {
	return &ComboEngine{
		selector: NewStrategySelector(),
		Config: &ComboConfig{
			LatencyWeight:         0.15,
			CostWeight:            0.20,
			QualityWeight:         0.15,
			AvailabilityWeight:    0.10,
			FreeTierWeight:        0.10,
			ContextWindowWeight:   0.10,
			TokenEfficiencyWeight: 0.10,
			ReliabilityWeight:     0.05,
			PreferenceWeight:      0.05,
		},
	}
}

// ComboScore represents a scored route target with all 9 factors
type ComboScore struct {
	ModelName         string  `json:"model_name"`
	Provider          string  `json:"provider"`
	TotalScore        float64 `json:"total_score"`
	LatencyScore      float64 `json:"latency_score"`
	CostScore         float64 `json:"cost_score"`
	QualityScore      float64 `json:"quality_score"`
	AvailabilityScore float64 `json:"availability_score"`
	FreeTierScore     float64 `json:"free_tier_score"`
	ContextScore      float64 `json:"context_window_score"`
	EfficiencyScore   float64 `json:"token_efficiency_score"`
	ReliabilityScore  float64 `json:"reliability_score"`
	PreferenceScore   float64 `json:"preference_score"`
}

// ScoreModels calculates combo scores for all models
func (c *ComboEngine) ScoreModels(ctx context.Context, candidates []*ModelInfo, req *RoutingRequest) []*ComboScore {
	results := make([]*ComboScore, 0, len(candidates))

	for _, model := range candidates {
		if !model.IsActive {
			continue
		}

		score := c.calculateComboScore(model, req)
		results = append(results, score)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	return results
}

func (c *ComboEngine) calculateComboScore(model *ModelInfo, req *RoutingRequest) *ComboScore {
	latencyScore := c.scoreLatency(model.AvgLatencyMs)
	costScore := c.scoreCost(model.CostPer1KInput + model.CostPer1KOutput)
	qualityScore := c.scoreQualityModel(model)
	availabilityScore := c.scoreAvailability(model)
	freeTierScore := c.scoreFreeTier(model)
	contextScore := c.scoreContextWindow(model.MaxTokens)
	efficiencyScore := c.scoreTokenEfficiency(model)
	reliabilityScore := c.scoreReliability(model)
	preferenceScore := c.scorePreference(model, req)

	total := latencyScore*c.Config.LatencyWeight +
		costScore*c.Config.CostWeight +
		qualityScore*c.Config.QualityWeight +
		availabilityScore*c.Config.AvailabilityWeight +
		freeTierScore*c.Config.FreeTierWeight +
		contextScore*c.Config.ContextWindowWeight +
		efficiencyScore*c.Config.TokenEfficiencyWeight +
		reliabilityScore*c.Config.ReliabilityWeight +
		preferenceScore*c.Config.PreferenceWeight

	return &ComboScore{
		ModelName:         model.Name,
		Provider:          model.Provider,
		TotalScore:        total,
		LatencyScore:      latencyScore,
		CostScore:         costScore,
		QualityScore:      qualityScore,
		AvailabilityScore: availabilityScore,
		FreeTierScore:     freeTierScore,
		ContextScore:      contextScore,
		EfficiencyScore:   efficiencyScore,
		ReliabilityScore:  reliabilityScore,
		PreferenceScore:   preferenceScore,
	}
}

func (c *ComboEngine) scoreLatency(latencyMs int64) float64 {
	score := 100.0 - float64(latencyMs)/20.0
	if score < 0 {
		return 0
	}
	return score
}

func (c *ComboEngine) scoreCost(costPer1K float64) float64 {
	score := 100.0 - costPer1K*1000.0
	if score < 0 {
		return 0
	}
	return score
}

func (c *ComboEngine) scoreQualityModel(model *ModelInfo) float64 {
	base := 50.0 + float64(len(model.Capabilities)*10)
	base += float64(100 - model.Priority*10)
	if base > 100 {
		return 100
	}
	return base
}

func (c *ComboEngine) scoreAvailability(model *ModelInfo) float64 {
	return 100.0
}

func (c *ComboEngine) scoreFreeTier(model *ModelInfo) float64 {
	freeModels := map[string]bool{
		"gpt-4o-mini":      true,
		"claude-3-5-haiku": true,
		"gemini-1.5-flash": true,
	}
	if freeModels[model.Name] {
		return 100.0
	}
	return 0.0
}

func (c *ComboEngine) scoreContextWindow(maxTokens int) float64 {
	ratio := float64(maxTokens) / 1000000.0 * 100.0
	if ratio > 100 {
		return 100
	}
	return ratio
}

func (c *ComboEngine) scoreTokenEfficiency(model *ModelInfo) float64 {
	base := 50.0 + float64(len(model.Capabilities))*10
	if base > 100 {
		return 100
	}
	return base
}

func (c *ComboEngine) scoreReliability(model *ModelInfo) float64 {
	providers := map[string]float64{
		"openai":    95,
		"anthropic": 95,
		"google":    90,
		"mistral":   85,
		"groq":      85,
	}
	if score, ok := providers[model.Provider]; ok {
		return score
	}
	return 70.0
}

func (c *ComboEngine) scorePreference(model *ModelInfo, req *RoutingRequest) float64 {
	if len(req.RequiredCapabilities) == 0 {
		return 100.0
	}

	for _, cap := range req.RequiredCapabilities {
		for _, modelCap := range model.Capabilities {
			if cap == modelCap {
				return 100.0
			}
		}
	}
	return 0.0
}

// GetFallbackTiers returns the 4-tier fallback sequence
func (c *ComboEngine) GetFallbackTiers(scores []*ComboScore) [][]*ComboScore {
	tiers := make([][]*ComboScore, 4)

	if len(scores) == 0 {
		return tiers
	}

	tiers[0] = []*ComboScore{scores[0]}

	if len(scores) > 1 {
		tiers[1] = scores[1:minInt(4, len(scores))]
	}

	var tier3 []*ComboScore
	for _, s := range scores {
		if s.FreeTierScore > 0 {
			tier3 = append(tier3, s)
		}
	}
	tiers[2] = tier3

	if len(scores) > 4 {
		tiers[3] = scores[4:]
	}

	return tiers
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
