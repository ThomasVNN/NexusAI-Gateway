package health

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// HealthChecker is a component that can be checked
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status    Status `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
	Details   string `json:"details,omitempty"`
}

// Manager manages health checks
type Manager struct {
	mu      sync.RWMutex
	checks  map[string]HealthChecker
	timeout time.Duration
}

// NewManager creates a new health manager
func NewManager(timeout time.Duration) *Manager {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return &Manager{
		checks:  make(map[string]HealthChecker),
		timeout: timeout,
	}
}

// Register registers a health checker
func (m *Manager) Register(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checks[checker.Name()] = checker
}

// Unregister removes a health checker
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.checks, name)
}

// Check runs all health checks
func (m *Manager) Check(ctx context.Context) *HealthStatus {
	m.mu.RLock()
	checks := make([]HealthChecker, 0, len(m.checks))
	for _, c := range m.checks {
		checks = append(checks, c)
	}
	m.mu.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	for _, checker := range checks {
		result := m.checkSingle(ctx, checker)
		results[checker.Name()] = result

		// Update overall status
		if result.Status == StatusUnhealthy && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		}
	}

	return &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
	}
}

// checkSingle runs a single health check
func (m *Manager) checkSingle(ctx context.Context, checker HealthChecker) CheckResult {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	err := checker.Check(checkCtx)
	latencyMs := time.Since(start).Milliseconds()

	if err == nil {
		return CheckResult{
			Status:    StatusHealthy,
			LatencyMs: latencyMs,
		}
	}

	return CheckResult{
		Status:    StatusUnhealthy,
		LatencyMs: latencyMs,
		Error:     err.Error(),
	}
}

// Handler returns an HTTP handler for health checks
func (m *Manager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := m.Check(r.Context())

		statusCode := http.StatusOK
		if status.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		// Note: In production, use json.Marshal properly
		w.Write([]byte(`{"status":"` + string(status.Status) + `","timestamp":"` + status.Timestamp.Format(time.RFC3339) + `"}`))
	})
}

// ReadyHandler returns an HTTP handler for readiness checks
func (m *Manager) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := m.Check(r.Context())

		// Only fail if unhealthy, degraded is OK for readiness
		statusCode := http.StatusOK
		if status.Status == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(`{"ready":` + boolToJSON(status.Status != StatusUnhealthy) + `}`))
	})
}

func boolToJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// SimpleHealthChecker provides a simple health check function
type SimpleHealthChecker struct {
	name    string
	checkFn func(ctx context.Context) error
}

// Name returns the checker name
func (c *SimpleHealthChecker) Name() string {
	return c.name
}

// Check runs the health check
func (c *SimpleHealthChecker) Check(ctx context.Context) error {
	return c.checkFn(ctx)
}

// NewSimpleChecker creates a simple health checker
func NewSimpleChecker(name string, checkFn func(ctx context.Context) error) *SimpleHealthChecker {
	return &SimpleHealthChecker{
		name:    name,
		checkFn: checkFn,
	}
}
