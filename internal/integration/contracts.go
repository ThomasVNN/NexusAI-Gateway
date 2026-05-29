package integration

import (
	"context"
)

// KnowledgeClient defines the contract for retrieving relevant context from the Knowledge service.
type KnowledgeClient interface {
	Retrieve(ctx context.Context, query string) (string, error)
}

// SkillRequest represents the payload passed to Skills service.
type SkillRequest struct {
	SkillName string                 `json:"skill_name"`
	Input     string                 `json:"input"`
	Params    map[string]interface{} `json:"params"`
}

// SkillResponse represents the payload returned by Skills service.
type SkillResponse struct {
	Output string `json:"output"`
}

// SkillsClient defines the contract for executing skill workflows in the Skills service.
type SkillsClient interface {
	Execute(ctx context.Context, request *SkillRequest) (*SkillResponse, error)
}

// ModelRouteRequest represents the model resolution/routing request.
type ModelRouteRequest struct {
	ModelName string                 `json:"model_name"`
	Prompt    string                 `json:"prompt"`
	TenantID  string                 `json:"tenant_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ModelRouteResponse represents the target model platform routing mapping.
type ModelRouteResponse struct {
	ProviderID string `json:"provider_id"`
	ModelName  string `json:"model_name"`
	URL        string `json:"url"`
}

// ModelPlatformClient defines the contract for routing and dispatching requests to model providers.
type ModelPlatformClient interface {
	Route(ctx context.Context, request *ModelRouteRequest) (*ModelRouteResponse, error)
}
