package routing

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

// ModelInfo holds metadata about a model (extended from routing.go)
type ModelInfo struct {
	Name            string
	Provider        string
	CostPer1KInput  float64
	CostPer1KOutput float64
	MaxTokens       int
	Capabilities    []string // e.g., "vision", "function_calling", "streaming"
	AvgLatencyMs    int64
	Priority        int
	IsActive        bool
	Weight          float64
	RequestCount    int64
	QuotaRemaining  float64
	QuotaResetAt    int64
	SuccessRate     float64
	LastUsedAt      int64
	HealthStatus    string
}

// DefaultModels provides the standard model catalog
var DefaultModels = map[string]*ModelInfo{
	"gpt-4o": {
		Name:            "gpt-4o",
		Provider:        "openai",
		CostPer1KInput:  0.005,
		CostPer1KOutput: 0.015,
		MaxTokens:       128000,
		Capabilities:    []string{"vision", "function_calling", "streaming"},
		AvgLatencyMs:    1000,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.95,
		HealthStatus:    "healthy",
	},
	"gpt-4o-mini": {
		Name:            "gpt-4o-mini",
		Provider:        "openai",
		CostPer1KInput:  0.00015,
		CostPer1KOutput: 0.0006,
		MaxTokens:       128000,
		Capabilities:    []string{"function_calling", "streaming"},
		AvgLatencyMs:    500,
		Priority:        2,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.97,
		HealthStatus:    "healthy",
	},
	"gpt-4-turbo": {
		Name:            "gpt-4-turbo",
		Provider:        "openai",
		CostPer1KInput:  0.01,
		CostPer1KOutput: 0.03,
		MaxTokens:       128000,
		Capabilities:    []string{"vision", "function_calling", "streaming"},
		AvgLatencyMs:    1500,
		Priority:        3,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.94,
		HealthStatus:    "healthy",
	},
	"claude-3-5-sonnet": {
		Name:            "claude-3-5-sonnet",
		Provider:        "anthropic",
		CostPer1KInput:  0.003,
		CostPer1KOutput: 0.015,
		MaxTokens:       200000,
		Capabilities:    []string{"vision", "function_calling", "streaming"},
		AvgLatencyMs:    1200,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.96,
		HealthStatus:    "healthy",
	},
	"claude-3-5-haiku": {
		Name:            "claude-3-5-haiku",
		Provider:        "anthropic",
		CostPer1KInput:  0.0008,
		CostPer1KOutput: 0.004,
		MaxTokens:       200000,
		Capabilities:    []string{"function_calling", "streaming"},
		AvgLatencyMs:    400,
		Priority:        2,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.98,
		HealthStatus:    "healthy",
	},
	"gemini-1.5-pro": {
		Name:            "gemini-1.5-pro",
		Provider:        "google",
		CostPer1KInput:  0.00125,
		CostPer1KOutput: 0.005,
		MaxTokens:       1000000,
		Capabilities:    []string{"vision", "function_calling", "streaming", "long_context"},
		AvgLatencyMs:    800,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.95,
		HealthStatus:    "healthy",
	},
	"gemini-1.5-flash": {
		Name:            "gemini-1.5-flash",
		Provider:        "google",
		CostPer1KInput:  0.000075,
		CostPer1KOutput: 0.0003,
		MaxTokens:       1000000,
		Capabilities:    []string{"function_calling", "streaming", "long_context"},
		AvgLatencyMs:    300,
		Priority:        2,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.97,
		HealthStatus:    "healthy",
	},
	"deepseek-chat": {
		Name:            "deepseek-chat",
		Provider:        "deepseek",
		CostPer1KInput:  0.00027,
		CostPer1KOutput: 0.0011,
		MaxTokens:       64000,
		Capabilities:    []string{"chat", "function_calling", "streaming"},
		AvgLatencyMs:    600,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.93,
		HealthStatus:    "healthy",
	},
	"groq-llama-3.1-8b-instant": {
		Name:            "groq-llama-3.1-8b-instant",
		Provider:        "groq",
		CostPer1KInput:  0,
		CostPer1KOutput: 0,
		MaxTokens:       8192,
		Capabilities:    []string{"chat", "function_calling", "streaming"},
		AvgLatencyMs:    150,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.98,
		HealthStatus:    "healthy",
	},
	"mistral-large": {
		Name:            "mistral-large",
		Provider:        "mistral",
		CostPer1KInput:  0.002,
		CostPer1KOutput: 0.006,
		MaxTokens:       128000,
		Capabilities:    []string{"chat", "function_calling", "streaming"},
		AvgLatencyMs:    900,
		Priority:        1,
		IsActive:        true,
		Weight:          1.0,
		QuotaRemaining:  1.0,
		SuccessRate:     0.94,
		HealthStatus:    "healthy",
	},
}

// ModelRouter is the enhanced routing engine
type ModelRouter struct {
	mu         sync.RWMutex
	models     map[string]*ModelInfo
	scoring    *ScoringEngine
	strategies map[string]RoutingStrategy
}

// NewModelRouter creates a new enhanced model router
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		models:     DefaultModels,
		scoring:    NewScoringEngine(),
		strategies: make(map[string]RoutingStrategy),
	}
}

// Route selects the optimal model based on request characteristics and strategy
func (r *ModelRouter) Route(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// If specific model requested and available, always prioritize it
	if req.RequestedModel != "" {
		if model, ok := r.models[req.RequestedModel]; ok && model.IsActive {
			if hasCapabilities(model, req.RequiredCapabilities) {
				// Check latency constraint if specified
				if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
					slog.WarnContext(ctx, "Requested model exceeds latency constraint",
						slog.String("model", model.Name),
						slog.Int64("latency", model.AvgLatencyMs),
						slog.Int64("max_latency", req.MaxLatencyMs),
					)
				} else {
					slog.InfoContext(ctx, "Model routed to explicitly requested",
						slog.String("requested_model", req.RequestedModel),
						slog.String("provider", model.Provider),
						slog.String("tenant_id", req.TenantID),
					)
					return &RouteTarget{
						ModelID:    model.Name,
						ProviderID: model.Provider,
						Strategy:   StrategyPriority,
					}, nil
				}
			}
		}
		// If requested model not available, continue with fallback selection
	}

	// Determine routing strategy
	var strategy RoutingStrategy
	if req.Strategy != "" {
		strategy = RoutingStrategy(req.Strategy)
	}

	// Route using the appropriate strategy
	target, err := r.RouteByStrategy(ctx, req, strategy)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Model routed",
		slog.String("strategy", string(target.Strategy)),
		slog.String("selected_model", target.ModelID),
		slog.String("provider", target.ProviderID),
		slog.String("tenant_id", req.TenantID),
	)

	return target, nil
}

// RoutingRequest contains parameters for model selection
type RoutingRequest struct {
	TenantID             string
	RequestedModel       string
	Strategy             string
	RequiredCapabilities []string
	PromptTokens         int
	MaxLatencyMs         int64
	MaxCost              float64
}

// findCandidates returns models that match the requirements
func (r *ModelRouter) findCandidates(req *RoutingRequest) []*ModelInfo {
	var candidates []*ModelInfo

	// If specific model requested and available, prioritize it
	if req.RequestedModel != "" {
		if model, ok := r.models[req.RequestedModel]; ok && model.IsActive {
			// Check capabilities
			if hasCapabilities(model, req.RequiredCapabilities) {
				candidates = append(candidates, model)
			}
		}
	}

	// Find fallback models with similar capabilities
	for _, model := range r.models {
		if !model.IsActive {
			continue
		}
		if model.Name == req.RequestedModel {
			continue
		}
		if hasCapabilities(model, req.RequiredCapabilities) {
			// Check latency constraint
			if req.MaxLatencyMs > 0 && model.AvgLatencyMs > req.MaxLatencyMs {
				continue
			}
			candidates = append(candidates, model)
		}
	}

	return candidates
}

// hasCapabilities checks if a model supports required capabilities
func hasCapabilities(model *ModelInfo, required []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, req := range required {
		found := false
		for _, cap := range model.Capabilities {
			if cap == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// selectBest chooses the optimal model based on strategy
func (r *ModelRouter) selectBest(candidates []*ModelInfo, strategy RoutingStrategy) *ModelInfo {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[0]
	}

	switch strategy {
	case StrategyCostOptimized:
		return r.selectByCost(candidates)
	case StrategyLatencyOpt:
		return r.selectByLatency(candidates)
	case StrategyQualityFirst:
		return r.selectByCapability(candidates)
	default:
		return r.selectByCost(candidates)
	}
}

// selectByCost picks the cheapest model
func (r *ModelRouter) selectByCost(candidates []*ModelInfo) *ModelInfo {
	best := candidates[0]
	minCost := best.CostPer1KInput + best.CostPer1KOutput
	for _, c := range candidates[1:] {
		cost := c.CostPer1KInput + c.CostPer1KOutput
		if cost < minCost {
			minCost = cost
			best = c
		}
	}
	return best
}

// selectByLatency picks the fastest model
func (r *ModelRouter) selectByLatency(candidates []*ModelInfo) *ModelInfo {
	best := candidates[0]
	minLatency := best.AvgLatencyMs
	for _, c := range candidates[1:] {
		if c.AvgLatencyMs < minLatency {
			minLatency = c.AvgLatencyMs
			best = c
		}
	}
	return best
}

// selectByCapability picks the most capable model (highest priority)
func (r *ModelRouter) selectByCapability(candidates []*ModelInfo) *ModelInfo {
	best := candidates[0]
	maxCaps := len(best.Capabilities)
	maxPriority := best.Priority
	for _, c := range candidates[1:] {
		caps := len(c.Capabilities)
		// Prefer more capabilities, then lower priority number (higher priority)
		if caps > maxCaps || (caps == maxCaps && c.Priority < maxPriority) {
			maxCaps = caps
			maxPriority = c.Priority
			best = c
		}
	}
	return best
}

// getStrategy returns the routing strategy for a tenant
func (r *ModelRouter) getStrategy(tenantID string, preferred RoutingStrategy) RoutingStrategy {
	if preferred != "" {
		return preferred
	}
	if strategy, ok := r.strategies[tenantID]; ok {
		return strategy
	}
	return StrategyCostOptimized // Default strategy
}

// SetTenantStrategy sets the routing strategy for a tenant
func (r *ModelRouter) SetTenantStrategy(tenantID string, strategy RoutingStrategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies[tenantID] = strategy
}

// AddModel registers a new model in the catalog
func (r *ModelRouter) AddModel(model *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.Name] = model
}

// GetModel returns model info by name
func (r *ModelRouter) GetModel(name string) (*ModelInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[name]
	return m, ok
}

// ListModels returns all registered models
func (r *ModelRouter) ListModels() []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ModelInfo, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	return result
}

// RecordLatency updates the latency metrics for a model
func (r *ModelRouter) RecordLatency(ctx context.Context, modelName string, latencyMs int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if model, ok := r.models[modelName]; ok {
		// Exponential moving average
		ema := float64(model.AvgLatencyMs)*0.9 + float64(latencyMs)*0.1
		model.AvgLatencyMs = int64(ema)
	}
}

// DeactivateModel marks a model as inactive
func (r *ModelRouter) DeactivateModel(modelName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if model, ok := r.models[modelName]; ok {
		model.IsActive = false
	}
}

// ErrNoModelsAvailable is returned when no suitable model is found
var ErrNoModelsAvailable = errors.New("no suitable model available for the requested parameters")
