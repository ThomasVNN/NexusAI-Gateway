package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
)

// DefaultKnowledgeClient calls the real NexusAI-Knowledge service for RAG context retrieval.
type DefaultKnowledgeClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDefaultKnowledgeClient creates a real HTTP client for the Knowledge service.
func NewDefaultKnowledgeClient(cfg *config.Config) *DefaultKnowledgeClient {
	baseURL := cfg.KnowledgeServiceURL
	if baseURL == "" {
		baseURL = "http://nexusai-knowledge:8080"
	}
	return &DefaultKnowledgeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// KnowledgeRetrieveRequest is the payload sent to the Knowledge service.
type KnowledgeRetrieveRequest struct {
	Message string `json:"message"`
	Tenant  string `json:"tenant,omitempty"`
}

// KnowledgeRetrieveResponse is the payload returned by the Knowledge service.
type KnowledgeRetrieveResponse struct {
	Success bool   `json:"success"`
	Context string `json:"context,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Retrieve calls the Knowledge service to retrieve relevant context for the given query.
// Falls back to an empty context string if the service is unavailable.
func (c *DefaultKnowledgeClient) Retrieve(ctx context.Context, query string) (string, error) {
	reqBody := KnowledgeRetrieveRequest{
		Message: query,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal knowledge request: %w", err)
	}

	// Try /api/chat first (Fastify server)
	url := c.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create knowledge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Service unavailable — return empty context (graceful degradation)
		return "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try alternative endpoint
		url = c.baseURL + "/api/knowledge/retrieve"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", nil
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return "", nil
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	var result struct {
		Context string `json:"context"`
		Data    struct {
			Context string `json:"context"`
		} `json:"data"`
		Message string `json:"message"`
		Answer  string `json:"answer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Non-JSON or parse error — return empty context
		return "", nil
	}

	context := result.Context
	if context == "" {
		context = result.Data.Context
	}
	if context == "" {
		context = result.Answer
	}
	return context, nil
}

// DefaultSkillsClient calls the real NexusAI-Skills service for skill execution.
type DefaultSkillsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDefaultSkillsClient creates a real HTTP client for the Skills service.
func NewDefaultSkillsClient(cfg *config.Config) *DefaultSkillsClient {
	baseURL := cfg.SkillsServiceURL
	if baseURL == "" {
		baseURL = "http://nexusai-skills:8083"
	}
	return &DefaultSkillsClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute calls the Skills service to execute a named skill.
// Falls back to a synthetic response if the service is unavailable.
func (c *DefaultSkillsClient) Execute(ctx context.Context, request *SkillRequest) (*SkillResponse, error) {
	reqBody := map[string]interface{}{
		"skillId":  request.SkillName,
		"toolName": request.SkillName,
		"input":    request.Input,
		"params":   request.Params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skill request: %w", err)
	}

	url := c.baseURL + "/api/skills/execute"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create skill request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Service unavailable — return a synthetic successful response
		return &SkillResponse{
			Output: fmt.Sprintf("[Skills service unavailable] Skill '%s' was requested. Service returned error: %v", request.SkillName, err),
		}, nil
	}
	defer resp.Body.Close()

	// Read body for debugging
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode != http.StatusOK {
		return &SkillResponse{
			Output: fmt.Sprintf("[Skills service error] HTTP %d for skill '%s': %s", resp.StatusCode, request.SkillName, string(respBody)),
		}, nil
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Output string `json:"output"`
			Result string `json:"result"`
		} `json:"data"`
		Output string `json:"output"`
		Error  string `json:"error"`
		Meta   struct {
			PolicyCheck struct {
				Allowed bool   `json:"allowed"`
				Reason  string `json:"reason"`
			} `json:"policyCheck"`
		} `json:"meta"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return &SkillResponse{
			Output: fmt.Sprintf("[Skills service parse error] Skill '%s' response: %s", request.SkillName, string(respBody)),
		}, nil
	}

	if !result.Success && result.Error != "" {
		return &SkillResponse{
			Output: fmt.Sprintf("[Skills service rejected] Skill '%s': %s", request.SkillName, result.Error),
		}, nil
	}

	output := result.Output
	if output == "" {
		output = result.Data.Output
	}
	if output == "" {
		output = result.Data.Result
	}
	if output == "" {
		output = fmt.Sprintf("[Skill '%s' completed successfully]", request.SkillName)
	}

	return &SkillResponse{
		Output: output,
	}, nil
}

// DefaultModelPlatformClient calls the real NexusAI-Platform service for model routing.
type DefaultModelPlatformClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDefaultModelPlatformClient creates a real HTTP client for the Model Platform service.
func NewDefaultModelPlatformClient(cfg *config.Config) *DefaultModelPlatformClient {
	baseURL := cfg.PlatformServiceURL
	if baseURL == "" {
		baseURL = "http://nexusai-platform:8084"
	}
	return &DefaultModelPlatformClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Route calls the Platform service to resolve the best model for a request.
// Falls back to OpenAI gpt-4o if the service is unavailable.
func (c *DefaultModelPlatformClient) Route(ctx context.Context, request *ModelRouteRequest) (*ModelRouteResponse, error) {
	// Build a routing hint based on the model name request
	modelName := request.ModelName
	if modelName == "" {
		modelName = "gpt-4o"
	}

	reqBody := map[string]interface{}{
		"model":     modelName,
		"prompt":    request.Prompt,
		"tenant_id": request.TenantID,
		"metadata":  request.Metadata,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return c.fallbackResponse(modelName)
	}

	url := c.baseURL + "/api/v1/route"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return c.fallbackResponse(modelName)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Service unavailable — use sensible defaults
		return c.fallbackResponse(modelName)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackResponse(modelName)
	}

	var result struct {
		ProviderID string `json:"provider_id"`
		ModelName  string `json:"model_name"`
		URL        string `json:"url"`
		Endpoint   string `json:"endpoint"`
		Provider   string `json:"provider"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.fallbackResponse(modelName)
	}

	if result.ProviderID == "" {
		result.ProviderID = result.Provider
	}
	if result.ModelName == "" {
		result.ModelName = modelName
	}
	if result.URL == "" {
		result.URL = result.Endpoint
	}
	if result.URL == "" {
		return c.fallbackResponse(result.ModelName)
	}

	return &ModelRouteResponse{
		ProviderID: result.ProviderID,
		ModelName:  result.ModelName,
		URL:        result.URL,
	}, nil
}

func (c *DefaultModelPlatformClient) fallbackResponse(modelName string) (*ModelRouteResponse, error) {
	// Normalize model name to provider
	switch {
	case contains(modelName, "gpt-4o"):
		return &ModelRouteResponse{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
			URL:        "https://api.openai.com/v1/chat/completions",
		}, nil
	case contains(modelName, "claude"):
		return &ModelRouteResponse{
			ProviderID: "anthropic",
			ModelName:  "claude-sonnet-4-20250514",
			URL:        "https://api.anthropic.com/v1/messages",
		}, nil
	case contains(modelName, "gemini"):
		return &ModelRouteResponse{
			ProviderID: "google",
			ModelName:  "gemini-2.0-flash",
			URL:        "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent",
		}, nil
	case contains(modelName, "llama") || contains(modelName, "mistral") || contains(modelName, "gemma"):
		return &ModelRouteResponse{
			ProviderID: "openrouter",
			ModelName:  modelName,
			URL:        "https://openrouter.ai/api/v1/chat/completions",
		}, nil
	default:
		return &ModelRouteResponse{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
			URL:        "https://api.openai.com/v1/chat/completions",
		}, nil
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsRest(s, substr)))
}

func containsRest(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
