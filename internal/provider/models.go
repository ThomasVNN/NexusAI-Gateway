package provider

import (
	"database/sql"
	"time"
)

// ProviderType defines the type of AI provider
type ProviderType string

const (
	ProviderTypeOpenAI    ProviderType = "openai"
	ProviderTypeAnthropic ProviderType = "anthropic"
	ProviderTypeDeepSeek  ProviderType = "deepseek"
	ProviderTypeGoogle    ProviderType = "google"
	ProviderTypeAzure     ProviderType = "azure"
	ProviderTypeCohere    ProviderType = "cohere"
	ProviderTypeMistral   ProviderType = "mistral"
	ProviderTypeOllama    ProviderType = "ollama"
	ProviderTypeCustom    ProviderType = "custom"
)

// Provider represents an AI provider configuration
type Provider struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Type           ProviderType `json:"type"`
	Endpoint       string       `json:"endpoint"`
	Credentials    string       `json:"-"`        // Never expose in JSON (encrypted)
	Status         string       `json:"status"`   // "active", "inactive", "degraded"
	Priority       int          `json:"priority"` // Lower = higher priority
	Enabled        bool         `json:"enabled"`
	HealthCheckURL string       `json:"health_check_url,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// ProviderHealth tracks health metrics for a provider
type ProviderHealth struct {
	ProviderID      string    `json:"provider_id"`
	LastHealthCheck time.Time `json:"last_health_check"`
	ErrorCount      int       `json:"error_count"`
	LatencyP99      float64   `json:"latency_p99"`  // in milliseconds
	SuccessRate     float64   `json:"success_rate"` // percentage
	IsHealthy       bool      `json:"is_healthy"`
}

// ProviderWithHealth combines provider info with health data for responses
type ProviderWithHealth struct {
	Provider
	Health ProviderHealth `json:"health"`
}

// ProviderResponse is the API response type (credentials masked)
type ProviderResponse struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Type           ProviderType `json:"type"`
	Endpoint       string       `json:"endpoint"`
	Status         string       `json:"status"`
	Priority       int          `json:"priority"`
	Enabled        bool         `json:"enabled"`
	HealthCheckURL string       `json:"health_check_url,omitempty"`
	Credentials    string       `json:"credentials,omitempty"` // "***" if set
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// ToResponse converts Provider to API response (masks credentials)
func (p *Provider) ToResponse() *ProviderResponse {
	resp := &ProviderResponse{
		ID:             p.ID,
		Name:           p.Name,
		Type:           p.Type,
		Endpoint:       p.Endpoint,
		Status:         p.Status,
		Priority:       p.Priority,
		Enabled:        p.Enabled,
		HealthCheckURL: p.HealthCheckURL,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.Credentials != "" {
		resp.Credentials = "***"
	}
	return resp
}

// ProviderFilter provides filtering options for provider queries
type ProviderFilter struct {
	Type    ProviderType
	Enabled *bool
	Status  string
}

// Validate checks if the provider configuration is valid
func (p *Provider) Validate() error {
	if p.Name == "" {
		return ErrProviderNameRequired
	}
	if p.Type == "" {
		return ErrProviderTypeRequired
	}
	if p.Endpoint == "" {
		return ErrProviderEndpointRequired
	}
	return nil
}

// Custom errors
type ProviderError struct {
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}

var (
	ErrProviderNameRequired     = &ProviderError{Message: "provider name is required"}
	ErrProviderTypeRequired     = &ProviderError{Message: "provider type is required"}
	ErrProviderEndpointRequired = &ProviderError{Message: "provider endpoint is required"}
	ErrProviderNotFound         = &ProviderError{Message: "provider not found"}
	ErrProviderInactive         = &ProviderError{Message: "provider is inactive"}
	ErrNoHealthyProvider        = &ProviderError{Message: "no healthy provider available"}
)

// Helper to convert sql.NullTime to *time.Time
func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
