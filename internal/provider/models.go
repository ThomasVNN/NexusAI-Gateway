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

// HealthStatus represents the current health state of a provider
type HealthStatus struct {
	ProviderID   string    `json:"provider_id"`
	IsHealthy    bool      `json:"is_healthy"`
	LastCheck    time.Time `json:"last_check"`
	LatencyMs    int       `json:"latency_ms"`
	ErrorRate    float64   `json:"error_rate"`
	SuccessRate  float64   `json:"success_rate"`
	CircuitState string    `json:"circuit_state"`
	FailureCount int       `json:"failure_count"`
	NextCheck    time.Time `json:"next_check"`
}

// HealthCheckConfig holds configuration for health checks
type HealthCheckConfig struct {
	Interval         time.Duration
	Timeout          time.Duration
	Retries          int
	FailureThreshold int
	SuccessThreshold int
	CircuitTimeout   time.Duration
}

// DefaultHealthCheckConfig returns the default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:         30 * time.Second,
		Timeout:          10 * time.Second,
		Retries:          2,
		FailureThreshold: 5,
		SuccessThreshold: 3,
		CircuitTimeout:   30 * time.Second,
	}
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ProviderID string
	Success    bool
	Latency    time.Duration
	Error      error
	Timestamp  time.Time
}

// ProviderHealthMetrics aggregates health metrics for reporting
type ProviderHealthMetrics struct {
	ProviderID       string  `json:"provider_id"`
	TotalChecks      int     `json:"total_checks"`
	SuccessfulChecks int     `json:"successful_checks"`
	FailedChecks     int     `json:"failed_checks"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	MinLatencyMs     int     `json:"min_latency_ms"`
	MaxLatencyMs     int     `json:"max_latency_ms"`
	P99LatencyMs     int     `json:"p99_latency_ms"`
	SuccessRate      float64 `json:"success_rate"`
	ErrorRate        float64 `json:"error_rate"`
	CurrentState     string  `json:"current_state"`
	LastError        string  `json:"last_error,omitempty"`
}
