package routing

import (
	"context"
)

// RouteTarget holds information about which provider and model to dispatch a request to
type RouteTarget struct {
	ProviderID string
	ModelName  string
	Weight     int
	Priority   int
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
	// Seed standard models with fallback options
	return &FailoverRouter{
		targets: map[string][]RouteTarget{
			"gpt-4": {
				{ProviderID: "openai", ModelName: "gpt-4o", Weight: 100, Priority: 1},
				{ProviderID: "azure-openai", ModelName: "gpt-4-turbo", Weight: 0, Priority: 2},
			},
			"claude-3-opus": {
				{ProviderID: "anthropic", ModelName: "claude-3-5-sonnet", Weight: 100, Priority: 1},
				{ProviderID: "aws-bedrock", ModelName: "anthropic.claude-v3", Weight: 0, Priority: 2},
			},
		},
	}
}

func (r *FailoverRouter) Route(ctx context.Context, requestedModel string, tenantID string) (*RouteTarget, error) {
	targets, ok := r.targets[requestedModel]
	if !ok || len(targets) == 0 {
		// Default generic route
		return &RouteTarget{
			ProviderID: "openai",
			ModelName:  "gpt-4o-mini",
			Weight:     100,
			Priority:   1,
		}, nil
	}

	// Always select highest priority (primary) target first
	return &targets[0], nil
}

func (r *FailoverRouter) RecordFailure(ctx context.Context, target *RouteTarget, err error) {
	// Telemetry updates go here for passive circuit breaking
}

func (r *FailoverRouter) RecordLatency(ctx context.Context, target *RouteTarget, durationMs int64) {
	// Tracking latency patterns to support latency-based routing
}
