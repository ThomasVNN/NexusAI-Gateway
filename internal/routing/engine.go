package routing

import (
	"context"
	"log/slog"
	"sync"
)

// RoutingStrategy defines how to select the optimal model
type RoutingStrategy string

const (
	// StrategyCostOptimized prioritizes cheaper models
	StrategyCostOptimized RoutingStrategy = "cost_optimized"
	// StrategyLatencyOptimized prioritizes faster models
	StrategyLatencyOptimized RoutingStrategy = "latency_optimized"
	// StrategyCapabilityOptimized prioritizes more capable models
	StrategyCapabilityOptimized RoutingStrategy = "capability_optimized"
)

// ModelInfo holds metadata about a model
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
	},
}

// ModelRouter is the enhanced routing engine
type ModelRouter struct {
	mu         sync.RWMutex
	models     map[string]*ModelInfo
	strategies map[string]RoutingStrategy
}

// NewModelRouter creates a new enhanced model router
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		models:     DefaultModels,
		strategies: make(map[string]RoutingStrategy),
	}
}

// Route selects the optimal model based on request characteristics and strategy
func (r *ModelRouter) Route(ctx context.Context, req *RoutingRequest) (*RouteTarget, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Determine routing strategy
	strategy := r.getStrategy(req.TenantID, req.Strategy)

	// Find candidates that support the requested capabilities
	candidates := r.findCandidates(req)

	if len(candidates) == 0 {
		slog.WarnContext(ctx, "No model candidates found",
			slog.String("requested_model", req.RequestedModel),
			slog.Any("capabilities", req.RequiredCapabilities),
		)
		return nil, ErrNoModelAvailable
	}

	// Select best candidate based on strategy
	selected := r.selectBest(candidates, strategy)

	slog.InfoContext(ctx, "Model routed",
		slog.String("strategy", string(strategy)),
		slog.String("selected_model", selected.Name),
		slog.String("provider", selected.Provider),
		slog.String("tenant_id", req.TenantID),
	)

	return &RouteTarget{
		ProviderID: selected.Provider,
		ModelName:  selected.Name,
		Weight:     100,
		Priority:   selected.Priority,
	}, nil
}

// RoutingRequest contains parameters for model selection
type RoutingRequest struct {
	RequestedModel       string
	TenantID             string
	Strategy             RoutingStrategy
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
			if r.hasCapabilities(model, req.RequiredCapabilities) {
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
		if r.hasCapabilities(model, req.RequiredCapabilities) {
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
func (r *ModelRouter) hasCapabilities(model *ModelInfo, required []string) bool {
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
	case StrategyLatencyOptimized:
		return r.selectByLatency(candidates)
	case StrategyCapabilityOptimized:
		return r.selectByCapability(candidates)
	default:
		return r.selectByCost(candidates)
	}
}

// selectByCost picks the cheapest model
func (r *ModelRouter) selectByCost(candidates []*ModelInfo) *ModelInfo {
	best := candidates[0]
	minCost := best.CostPer1KInput
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

// ErrNoModelAvailable is returned when no suitable model is found
var ErrNoModelAvailable = &RoutingError{Message: "no suitable model available for the requested parameters"}

type RoutingError struct {
	Message string
}

func (e *RoutingError) Error() string {
	return e.Message
}
