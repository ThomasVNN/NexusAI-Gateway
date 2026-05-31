package routing

import (
	"context"
)

// RouteTarget holds information about which provider and model to dispatch a request to
type RouteTarget struct {
	ModelID    string          `json:"model_id"`
	ProviderID string          `json:"provider_id"`
	Strategy   RoutingStrategy `json:"strategy"`
}

// RouteResolver defines the interface for model routing
type RouteResolver interface {
	Route(ctx context.Context, requestedModel string, tenantID string) (*RouteTarget, error)
	RecordFailure(ctx context.Context, target *RouteTarget, err error)
	RecordLatency(ctx context.Context, target *RouteTarget, durationMs int64)
}

type FailoverRouter struct {
	targets map[string][]RouteTarget
}

func NewFailoverRouter() *FailoverRouter {
	return &FailoverRouter{
		targets: map[string][]RouteTarget{
			"gpt-4": {
				{ModelID: "gpt-4o", ProviderID: "openai", Strategy: StrategyPriority},
			},
			"claude-3-opus": {
				{ModelID: "claude-3-5-sonnet", ProviderID: "anthropic", Strategy: StrategyPriority},
			},
		},
	}
}

func (r *FailoverRouter) Route(ctx context.Context, requestedModel string, tenantID string) (*RouteTarget, error) {
	targets, ok := r.targets[requestedModel]
	if !ok || len(targets) == 0 {
		return &RouteTarget{
			ModelID:    "gpt-4o-mini",
			ProviderID: "openai",
			Strategy:   StrategyPriority,
		}, nil
	}

	return &targets[0], nil
}

func (r *FailoverRouter) RecordFailure(ctx context.Context, target *RouteTarget, err error) {
	// Telemetry updates go here for passive circuit breaking
}

func (r *FailoverRouter) RecordLatency(ctx context.Context, target *RouteTarget, durationMs int64) {
	// Tracking latency patterns to support latency-based routing
}
