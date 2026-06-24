package routing

import (
	"context"
	"log/slog"
	"math"
	"sync"
)

// Extended routing strategies
const (
	// Base strategies
	StrategyCostOptimized RoutingStrategy = "cost_optimized"
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
	StrategyCapabilityOptimized RoutingStrategy = "capability_optimized"

	// Extended strategies (9-14)
	StrategyLeastLatency     RoutingStrategy = "least_latency"
	StrategyCheapest         RoutingStrategy = "cheapest"
	StrategyQualityFirst     RoutingStrategy = "quality_first"
	StrategyBalanced         RoutingStrategy = "balanced"
	StrategyCostAware        RoutingStrategy = "cost_aware"
	StrategyThroughput      RoutingStrategy = "throughput"
	StrategyReliability      RoutingStrategy = "reliability"
	StrategyCustomWeighted   RoutingStrategy = "custom_weighted"
	StrategyAdaptive         RoutingStrategy = "adaptive"
	StrategyLoadBalanced     RoutingStrategy = "load_balanced"
	StrategyGeographic       RoutingStrategy = "geographic"
	StrategyCompliance       RoutingStrategy = "compliance"
	StrategySecurityFirst    RoutingStrategy = "security_first"
	StrategyOfflineFriendly  RoutingStrategy = "offline_friendly"
)

// Strategy weights for 9-factor scoring (must sum to 1.0)
var DefaultScoringWeights = ScoringWeights{
	Latency:       0.22,
	Cost:          0.17,
	Quality:       0.17,
	Reliability:   0.13,
	Capabilities:  0.08,
	Throughput:    0.08,
	Compliance:    0.05,
	Security:      0.05,
	Geographic:    0.05,
}

// ScoringWeights defines the weights for 9-factor scoring
type ScoringWeights struct {
	Latency      float64 `json:"latency"`
	Cost         float64 `json:"cost"`
	Quality      float64 `json:"quality"`
	Reliability  float64 `json:"reliability"`
	Capabilities float64 `json:"capabilities"`
	Throughput   float64 `json:"throughput"`
	Compliance   float64 `json:"compliance"`
	Security     float64 `json:"security"`
	Geographic   float64 `json:"geographic"`
}

// Validate checks that weights sum to 1.0
func (w ScoringWeights) Validate() bool {
	sum := w.Latency + w.Cost + w.Quality + w.Reliability +
		w.Capabilities + w.Throughput + w.Compliance +
		w.Security + w.Geographic
	return math.Abs(sum-1.0) < 0.001
}

// ModePack defines a preset mode configuration
type ModePack struct {
	Name    string          `json:"name"`
	Weights ScoringWeights `json:"weights"`
	Strategy RoutingStrategy `json:"strategy"`
}

// ModePacks provides predefined mode configurations
var ModePacks = map[string]*ModePack{
	"ship_fast": {
		Name: "ship_fast",
		Weights: ScoringWeights{
			Latency:      0.35,
			Cost:         0.15,
			Quality:      0.10,
			Reliability:  0.10,
			Capabilities: 0.10,
			Throughput:   0.10,
			Compliance:   0.03,
			Security:     0.02,
			Geographic:   0.05,
		},
		Strategy: StrategyLeastLatency,
	},
	"cost_saver": {
		Name: "cost_saver",
		Weights: ScoringWeights{
			Latency:      0.10,
			Cost:         0.40,
			Quality:      0.10,
			Reliability:  0.10,
			Capabilities: 0.10,
			Throughput:   0.05,
			Compliance:   0.05,
			Security:     0.05,
			Geographic:   0.05,
		},
		Strategy: StrategyCheapest,
	},
	"quality_first": {
		Name: "quality_first",
		Weights: ScoringWeights{
			Latency:      0.10,
			Cost:         0.10,
			Quality:      0.35,
			Reliability:  0.15,
			Capabilities: 0.10,
			Throughput:   0.05,
			Compliance:   0.05,
			Security:     0.05,
			Geographic:   0.05,
		},
		Strategy: StrategyQualityFirst,
	},
	"offline_friendly": {
		Name: "offline_friendly",
		Weights: ScoringWeights{
			Latency:      0.15,
			Cost:         0.20,
			Quality:      0.15,
			Reliability:  0.20,
			Capabilities: 0.10,
			Throughput:   0.05,
			Compliance:   0.05,
			Security:     0.05,
			Geographic:   0.05,
		},
		Strategy: StrategyOfflineFriendly,
	},
	"balanced": {
		Name: "balanced",
		Weights: ScoringWeights{
			Latency:      0.15,
			Cost:         0.15,
			Quality:      0.15,
			Reliability:  0.15,
			Capabilities: 0.12,
			Throughput:   0.10,
			Compliance:   0.06,
			Security:     0.06,
			Geographic:   0.06,
		},
		Strategy: StrategyBalanced,
	},
}

// ExtendedModelInfo extends ModelInfo with additional scoring factors
type ExtendedModelInfo struct {
	Name            string   `json:"name"`
	Provider        string   `json:"provider"`
	CostPer1KInput  float64 `json:"cost_per_1k_input"`
	CostPer1KOutput float64 `json:"cost_per_1k_output"`
	MaxTokens       int      `json:"max_tokens"`
	Capabilities    []string `json:"capabilities"`
	AvgLatencyMs    int64    `json:"avg_latency_ms"`
	Priority        int      `json:"priority"`
	IsActive        bool     `json:"is_active"`

	// Extended fields for 9-factor scoring
	SuccessRate      float64 `json:"success_rate"`      // Reliability factor
	RequestsPerMin    int     `json:"requests_per_min"`  // Throughput factor
	Region           string  `json:"region"`             // Geographic factor
	ComplianceScore  float64 `json:"compliance_score"`  // Compliance factor (0-1)
	SecurityLevel    string  `json:"security_level"`     // Security factor
	ContextWindow    int     `json:"context_window"`     // Quality factor
	LastHealthCheck  int64   `json:"last_health_check"` // Unix timestamp
}

// ScoringFactors contains normalized scores for each factor
type ScoringFactors struct {
	Latency      float64 // Lower latency = higher score
	Cost         float64 // Lower cost = higher score
	Quality      float64 // Higher quality = higher score
	Reliability  float64 // Higher reliability = higher score
	Capabilities float64 // More capabilities = higher score
	Throughput   float64 // Higher throughput = higher score
	Compliance   float64 // Higher compliance = higher score
	Security     float64 // Higher security = higher score
	Geographic   float64 // Geographic proximity = higher score
}

// ScoringResult contains the final score and breakdown
type ScoringResult struct {
	ModelName   string         `json:"model_name"`
	ProviderID  string         `json:"provider_id"`
	TotalScore float64        `json:"total_score"`
	Factors     ScoringFactors `json:"factors"`
	Breakdown   map[string]float64 `json:"breakdown"`
	Rank        int            `json:"rank"`
}

// ExtendedModelRouter is the enhanced routing engine with 14 strategies
type ExtendedModelRouter struct {
	mu              sync.RWMutex
	models          map[string]*ExtendedModelInfo
	strategies      map[string]RoutingStrategy
	customWeights   map[string]ScoringWeights
	modePacks       map[string]*ModePack
	requestCounts   map[string]int64 // For load balancing
	latencyTracker  *LatencyTracker
}

// LatencyTracker tracks rolling latency metrics
type LatencyTracker struct {
	mu       sync.RWMutex
	latencies map[string][]int64 // Recent latency samples
	window    int
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker(window int) *LatencyTracker {
	if window <= 0 {
		window = 100
	}
	return &LatencyTracker{
		latencies: make(map[string][]int64),
		window:    window,
	}
}

// RecordLatency records a latency sample
func (t *LatencyTracker) RecordLatency(modelName string, latencyMs int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	samples := t.latencies[modelName]
	samples = append(samples, latencyMs)

	// Keep only recent samples
	if len(samples) > t.window {
		samples = samples[len(samples)-t.window:]
	}

	t.latencies[modelName] = samples
}

// GetAverageLatency returns the average latency for a model
func (t *LatencyTracker) GetAverageLatency(modelName string) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	samples := t.latencies[modelName]
	if len(samples) == 0 {
		return 0
	}

	var sum int64
	for _, s := range samples {
		sum += s
	}
	return sum / int64(len(samples))
}

// GetPercentileLatency returns the Nth percentile latency
func (t *LatencyTracker) GetPercentileLatency(modelName string, percentile float64) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	samples := t.latencies[modelName]
	if len(samples) == 0 {
		return 0
	}

	// Sort copies
	sorted := make([]int64, len(samples))
	copy(sorted, samples)
	for i := 0; i < len(sorted)/2; i++ {
		j := len(sorted) - 1 - i
		sorted[i], sorted[j] = sorted[j], sorted[i]
	}

	idx := int(float64(len(sorted)) * percentile / 100.0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// NewExtendedModelRouter creates a new extended model router
func NewExtendedModelRouter() *ExtendedModelRouter {
	return &ExtendedModelRouter{
		models:        make(map[string]*ExtendedModelInfo),
		strategies:    make(map[string]RoutingStrategy),
		customWeights: make(map[string]ScoringWeights),
		modePacks:     ModePacks,
		requestCounts: make(map[string]int64),
		latencyTracker: NewLatencyTracker(100),
	}
}

// RegisterModel registers a model with extended info
func (r *ExtendedModelRouter) RegisterModel(model *ExtendedModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.Name] = model
}

// RegisterModelFromInfo converts ModelInfo to ExtendedModelInfo and registers
func (r *ExtendedModelRouter) RegisterModelFromInfo(model *ModelInfo) {
	extModel := &ExtendedModelInfo{
		Name:            model.Name,
		Provider:        model.Provider,
		CostPer1KInput:  model.CostPer1KInput,
		CostPer1KOutput: model.CostPer1KOutput,
		MaxTokens:       model.MaxTokens,
		Capabilities:    model.Capabilities,
		AvgLatencyMs:    model.AvgLatencyMs,
		Priority:        model.Priority,
		IsActive:        model.IsActive,
		SuccessRate:     99.0,    // Default 99%
		RequestsPerMin:   1000,    // Default 1000 RPM
		Region:          "us-east",
		ComplianceScore: 0.9,
		SecurityLevel:   "standard",
		ContextWindow:   model.MaxTokens,
		LastHealthCheck: 0,
	}
	r.RegisterModel(extModel)
}

// RouteExtended routes using the extended 14-strategy + 9-factor scoring
func (r *ExtendedModelRouter) RouteExtended(ctx context.Context, req *ExtendedRoutingRequest) (*ScoringResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Determine strategy
	strategy := r.getStrategy(req.TenantID, req.Strategy)

	// Get weights - either custom, mode pack, or default
	weights := r.getWeights(req.TenantID, req.ModePack, strategy)

	// Find candidates
	candidates := r.findCandidates(req)

	if len(candidates) == 0 {
		slog.WarnContext(ctx, "No model candidates found for routing",
			slog.String("strategy", string(strategy)),
			slog.String("tenant_id", req.TenantID),
		)
		return nil, ErrNoModelAvailable
	}

	// Score all candidates
	results := r.scoreAllCandidates(candidates, weights, req)

	// Sort by score
	r.sortByScore(results)

	// Assign ranks
	for i := range results {
		results[i].Rank = i + 1
	}

	// Track request for load balancing
	modelName := results[0].ModelName
	r.requestCounts[modelName]++

	// Log routing decision
	slog.InfoContext(ctx, "Model routed with extended scoring",
		slog.String("strategy", string(strategy)),
		slog.String("mode_pack", req.ModePack),
		slog.String("selected_model", results[0].ModelName),
		slog.String("provider", results[0].ProviderID),
		slog.Float64("score", results[0].TotalScore),
		slog.Int("candidates", len(candidates)),
	)

	return results[0], nil
}

// ExtendedRoutingRequest extends RoutingRequest with mode pack and preferences
type ExtendedRoutingRequest struct {
	RoutingRequest
	ModePack         string          `json:"mode_pack"`         // e.g., "ship_fast", "cost_saver"
	CustomWeights    ScoringWeights  `json:"custom_weights"`    // Override weights
	PreferredRegion  string          `json:"preferred_region"`  // Geographic preference
	MinSuccessRate   float64        `json:"min_success_rate"`  // Reliability filter
	RequiredSecurity string          `json:"required_security"` // Security level filter
}

// findCandidates returns models matching requirements
func (r *ExtendedModelRouter) findCandidates(req *ExtendedRoutingRequest) []*ExtendedModelInfo {
	var candidates []*ExtendedModelInfo

	// If specific model requested, return it
	if req.RequestedModel != "" {
		if model, ok := r.models[req.RequestedModel]; ok && model.IsActive {
			candidates = append(candidates, model)
		}
		return candidates
	}

	for _, model := range r.models {
		if !model.IsActive {
			continue
		}

		// Check capabilities
		if len(req.RequiredCapabilities) > 0 {
			if !r.hasCapabilities(model, req.RequiredCapabilities) {
				continue
			}
		}

		// Check latency constraint
		if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
			continue
		}

		// Check success rate
		if req.MinSuccessRate > 0 && model.SuccessRate < req.MinSuccessRate {
			continue
		}

		// Check security level
		if req.RequiredSecurity != "" && model.SecurityLevel != req.RequiredSecurity {
			// Allow higher security levels
			if !r.securitySatisfies(model.SecurityLevel, req.RequiredSecurity) {
				continue
			}
		}

		candidates = append(candidates, model)
	}

	return candidates
}

// securitySatisfies checks if a security level meets requirements
func (r *ExtendedModelRouter) securitySatisfies(actual, required string) bool {
	levels := map[string]int{
		"standard": 1,
		"enhanced": 2,
		"high":     3,
		"maximum":  4,
	}

	actualLevel, ok1 := levels[actual]
	requiredLevel, ok2 := levels[required]

	if !ok1 || !ok2 {
		return actual == required
	}

	return actualLevel >= requiredLevel
}

// hasCapabilities checks if model has required capabilities
func (r *ExtendedModelRouter) hasCapabilities(model *ExtendedModelInfo, required []string) bool {
	if len(required) == 0 {
		return true
	}

	modelCaps := make(map[string]bool)
	for _, cap := range model.Capabilities {
		modelCaps[cap] = true
	}

	for _, req := range required {
		if !modelCaps[req] {
			return false
		}
	}
	return true
}

// scoreAllCandidates scores all candidates using 9-factor model
func (r *ExtendedModelRouter) scoreAllCandidates(candidates []*ExtendedModelInfo, weights ScoringWeights, req *ExtendedRoutingRequest) []*ScoringResult {
	results := make([]*ScoringResult, len(candidates))

	// Calculate min/max for normalization
	latencyMin, latencyMax := r.getMinMax(candidates, func(m *ExtendedModelInfo) float64 { return float64(m.AvgLatencyMs) })
	costMin, costMax := r.getMinMax(candidates, func(m *ExtendedModelInfo) float64 { return m.CostPer1KInput + m.CostPer1KOutput })
	qualityMin, qualityMax := r.getMinMax(candidates, func(m *ExtendedModelInfo) float64 { return float64(m.ContextWindow) })

	for i, model := range candidates {
		// Calculate individual factor scores (0-1 scale)
		factors := ScoringFactors{
			Latency:     r.normalizeLatency(float64(model.AvgLatencyMs), latencyMin, latencyMax),
			Cost:        r.normalizeCost(model.CostPer1KInput+model.CostPer1KOutput, costMin, costMax),
			Quality:     r.normalizeQuality(float64(model.ContextWindow), qualityMin, qualityMax),
			Reliability: model.SuccessRate / 100.0,
			Capabilities: float64(len(model.Capabilities)) / 10.0, // Normalize to 0-1 assuming max 10 caps
			Throughput:  float64(model.RequestsPerMin) / 10000.0, // Normalize to 0-1 assuming max 10000 RPM
			Compliance:  model.ComplianceScore,
			Security:    r.securityToScore(model.SecurityLevel),
			Geographic:  r.geographicScore(model.Region, req.PreferredRegion),
		}

		// Calculate weighted total score
		totalScore := factors.Latency*weights.Latency +
			factors.Cost*weights.Cost +
			factors.Quality*weights.Quality +
			factors.Reliability*weights.Reliability +
			factors.Capabilities*weights.Capabilities +
			factors.Throughput*weights.Throughput +
			factors.Compliance*weights.Compliance +
			factors.Security*weights.Security +
			factors.Geographic*weights.Geographic

		// Build breakdown
		breakdown := map[string]float64{
			"latency":      factors.Latency * weights.Latency,
			"cost":         factors.Cost * weights.Cost,
			"quality":      factors.Quality * weights.Quality,
			"reliability":  factors.Reliability * weights.Reliability,
			"capabilities": factors.Capabilities * weights.Capabilities,
			"throughput":   factors.Throughput * weights.Throughput,
			"compliance":   factors.Compliance * weights.Compliance,
			"security":     factors.Security * weights.Security,
			"geographic":   factors.Geographic * weights.Geographic,
		}

		results[i] = &ScoringResult{
			ModelName:  model.Name,
			ProviderID: model.Provider,
			TotalScore: totalScore,
			Factors:    factors,
			Breakdown:  breakdown,
		}
	}

	return results
}

// getMinMax returns min and max values from a slice
func (r *ExtendedModelRouter) getMinMax(candidates []*ExtendedModelInfo, extractor func(*ExtendedModelInfo) float64) (min, max float64) {
	min = math.MaxFloat64
	max = -math.MaxFloat64

	for _, c := range candidates {
		val := extractor(c)
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	if min == max {
		min = 0
	}

	return
}

// normalizeLatency lower is better
func (r *ExtendedModelRouter) normalizeLatency(value, min, max float64) float64 {
	if max <= min {
		return 1.0
	}
	return 1.0 - (value-min)/(max-min)
}

// normalizeCost lower is better
func (r *ExtendedModelRouter) normalizeCost(value, min, max float64) float64 {
	if max <= min {
		return 1.0
	}
	return 1.0 - (value-min)/(max-min)
}

// normalizeQuality higher is better
func (r *ExtendedModelRouter) normalizeQuality(value, min, max float64) float64 {
	if max <= min {
		return 1.0
	}
	return (value - min) / (max - min)
}

// securityToScore converts security level to score
func (r *ExtendedModelRouter) securityToScore(level string) float64 {
	switch level {
	case "maximum":
		return 1.0
	case "high":
		return 0.85
	case "enhanced":
		return 0.7
	case "standard":
		return 0.5
	default:
		return 0.5
	}
}

// geographicScore calculates geographic proximity score
func (r *ExtendedModelRouter) geographicScore(modelRegion, preferredRegion string) float64 {
	if preferredRegion == "" {
		return 0.5 // Neutral if no preference
	}

	if modelRegion == preferredRegion {
		return 1.0 // Exact match
	}

	// Check region proximity (simplified)
	regionGroups := map[string][]string{
		"us":    {"us-east", "us-west", "us-central"},
		"eu":    {"eu-west", "eu-central"},
		"asia":  {"asia-east", "asia-west"},
	}

	for group, regions := range regionGroups {
		prefGroup := ""
		modelGroup := ""

		for _, r := range regions {
			if r == preferredRegion {
				prefGroup = group
			}
			if r == modelRegion {
				modelGroup = group
			}
		}

		if prefGroup != "" && prefGroup == modelGroup {
			return 0.7 // Same region group
		}
	}

	return 0.3 // Different regions
}

// sortByScore sorts results by total score descending
func (r *ExtendedModelRouter) sortByScore(results []*ScoringResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].TotalScore > results[i].TotalScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// getStrategy returns strategy for tenant
func (r *ExtendedModelRouter) getStrategy(tenantID string, preferred RoutingStrategy) RoutingStrategy {
	if preferred != "" {
		return preferred
	}
	if strategy, ok := r.strategies[tenantID]; ok {
		return strategy
	}
	return StrategyBalanced // Default strategy
}

// getWeights returns scoring weights
func (r *ExtendedModelRouter) getWeights(tenantID, modePack string, strategy RoutingStrategy) ScoringWeights {
	// Check custom weights first
	if weights, ok := r.customWeights[tenantID]; ok {
		return weights
	}

	// Check mode pack
	if modePack != "" {
		if pack, ok := r.modePacks[modePack]; ok {
			return pack.Weights
		}
	}

	// Strategy-based defaults
	switch strategy {
	case StrategyLeastLatency:
		return ScoringWeights{Latency: 0.40, Cost: 0.15, Quality: 0.15, Reliability: 0.10, Capabilities: 0.05, Throughput: 0.05, Compliance: 0.03, Security: 0.02, Geographic: 0.05}
	case StrategyCheapest:
		return ScoringWeights{Latency: 0.10, Cost: 0.45, Quality: 0.10, Reliability: 0.10, Capabilities: 0.05, Throughput: 0.05, Compliance: 0.05, Security: 0.05, Geographic: 0.05}
	case StrategyQualityFirst:
		return ScoringWeights{Latency: 0.10, Cost: 0.10, Quality: 0.40, Reliability: 0.15, Capabilities: 0.10, Throughput: 0.05, Compliance: 0.03, Security: 0.02, Geographic: 0.05}
	case StrategyBalanced:
		return DefaultScoringWeights
	case StrategyCostAware:
		return ScoringWeights{Latency: 0.15, Cost: 0.35, Quality: 0.15, Reliability: 0.10, Capabilities: 0.08, Throughput: 0.05, Compliance: 0.05, Security: 0.02, Geographic: 0.05}
	case StrategyThroughput:
		return ScoringWeights{Latency: 0.15, Cost: 0.15, Quality: 0.10, Reliability: 0.10, Capabilities: 0.10, Throughput: 0.30, Compliance: 0.03, Security: 0.02, Geographic: 0.05}
	case StrategyReliability:
		return ScoringWeights{Latency: 0.10, Cost: 0.10, Quality: 0.10, Reliability: 0.40, Capabilities: 0.08, Throughput: 0.08, Compliance: 0.05, Security: 0.04, Geographic: 0.05}
	case StrategyCompliance:
		return ScoringWeights{Latency: 0.08, Cost: 0.08, Quality: 0.10, Reliability: 0.10, Capabilities: 0.08, Throughput: 0.05, Compliance: 0.40, Security: 0.06, Geographic: 0.05}
	case StrategySecurityFirst:
		return ScoringWeights{Latency: 0.05, Cost: 0.05, Quality: 0.10, Reliability: 0.15, Capabilities: 0.10, Throughput: 0.05, Compliance: 0.10, Security: 0.35, Geographic: 0.05}
	case StrategyGeographic:
		return ScoringWeights{Latency: 0.15, Cost: 0.10, Quality: 0.10, Reliability: 0.10, Capabilities: 0.08, Throughput: 0.05, Compliance: 0.05, Security: 0.05, Geographic: 0.32}
	case StrategyOfflineFriendly:
		return ScoringWeights{Latency: 0.15, Cost: 0.20, Quality: 0.15, Reliability: 0.25, Capabilities: 0.08, Throughput: 0.05, Compliance: 0.03, Security: 0.04, Geographic: 0.05}
	case StrategyLoadBalanced:
		return ScoringWeights{Latency: 0.12, Cost: 0.12, Quality: 0.12, Reliability: 0.12, Capabilities: 0.12, Throughput: 0.25, Compliance: 0.05, Security: 0.05, Geographic: 0.05}
	default:
		return DefaultScoringWeights
	}
}

// SetTenantStrategy sets the routing strategy for a tenant
func (r *ExtendedModelRouter) SetTenantStrategy(tenantID string, strategy RoutingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[tenantID] = strategy
}

// SetTenantWeights sets custom weights for a tenant
func (r *ExtendedModelRouter) SetTenantWeights(tenantID string, weights ScoringWeights) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.customWeights[tenantID] = weights
}

// ListModels returns all registered models
func (r *ExtendedModelRouter) ListModels() []*ExtendedModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ExtendedModelInfo, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	return result
}

// GetModel returns model info by name
func (r *ExtendedModelRouter) GetModel(name string) (*ExtendedModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[name]
	return m, ok
}

// GetLoadBalancedModel returns the model with lowest request count
func (r *ExtendedModelRouter) GetLoadBalancedModel(candidates []*ExtendedModelInfo) *ExtendedModelInfo {
	if len(candidates) == 0 {
		return nil
	}

	var best *ExtendedModelInfo
	var minCount int64 = math.MaxInt64

	for _, c := range candidates {
		count := r.requestCounts[c.Name]
		if count < minCount {
			minCount = count
			best = c
		}
	}

	return best
}

// RecordLatency records latency for a model
func (r *ExtendedModelRouter) RecordLatency(modelName string, latencyMs int64) {
	r.latencyTracker.RecordLatency(modelName, latencyMs)

	r.mu.Lock()
	defer r.mu.Unlock()
	if model, ok := r.models[modelName]; ok {
		// Update with EMA
		ema := float64(model.AvgLatencyMs)*0.9 + float64(latencyMs)*0.1
		model.AvgLatencyMs = int64(ema)
	}
}

// Ensure types are used
var _ = NewLatencyTracker
