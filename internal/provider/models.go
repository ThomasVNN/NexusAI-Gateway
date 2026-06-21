package provider

import (
	"time"
)

// Provider represents a configured AI provider
type Provider struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Endpoint        string    `json:"endpoint"`
	APIKey          string    `json:"-"`
	Status          string    `json:"status"`
	Priority        int       `json:"priority"`
	Enabled         bool      `json:"enabled"`
	LastHealthCheck time.Time `json:"last_health_check"`
	ErrorCount      int       `json:"error_count"`
	LatencyP99      int       `json:"latency_p99"`
	SuccessRate     float64   `json:"success_rate"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
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
