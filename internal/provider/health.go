package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for provider health
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int
	successCount     int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given thresholds
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Allow checks if requests are allowed through the circuit
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			cb.failureCount = 0
			slog.Info("Circuit breaker transitioning to half-open")
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.successCount = 0
			slog.Info("Circuit breaker closed after successful requests")
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitOpen
			slog.Warn("Circuit breaker opened", slog.Int("failure_count", cb.failureCount))
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.failureCount = 1
		slog.Warn("Circuit breaker reopened after half-open failure")
	}
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":             cb.state.String(),
		"failure_count":     cb.failureCount,
		"success_count":     cb.successCount,
		"last_failure":      cb.lastFailureTime,
		"failure_threshold": cb.failureThreshold,
		"success_threshold": cb.successThreshold,
	}
}

// HealthChecker performs periodic health checks on providers
type HealthChecker struct {
	config          *HealthCheckConfig
	circuitBreakers map[string]*CircuitBreaker
	healthStatus    map[string]*HealthStatus
	metrics         map[string]*ProviderHealthMetrics
	mu              sync.RWMutex
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewHealthChecker creates a new health checker with the given configuration
func NewHealthChecker(config *HealthCheckConfig) *HealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	return &HealthChecker{
		config:          config,
		circuitBreakers: make(map[string]*CircuitBreaker),
		healthStatus:    make(map[string]*HealthStatus),
		metrics:         make(map[string]*ProviderHealthMetrics),
		stopCh:          make(chan struct{}),
	}
}

// Start begins the background health check scheduler
func (hc *HealthChecker) Start(providers []*Provider) {
	hc.wg.Add(1)
	go hc.scheduler(providers)
	slog.Info("Health checker started")
}

// Stop stops the health check scheduler
func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
	slog.Info("Health checker stopped")
}

// scheduler runs periodic health checks
func (hc *HealthChecker) scheduler(providers []*Provider) {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	// Initial check
	hc.checkAllProviders(providers)

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkAllProviders(providers)
		}
	}
}

// checkAllProviders performs health checks on all providers
func (hc *HealthChecker) checkAllProviders(providers []*Provider) {
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		hc.wg.Add(1)
		go func(provider *Provider) {
			defer hc.wg.Done()
			hc.CheckProvider(provider)
		}(p)
	}
}

// CheckProvider performs a health check on a single provider
func (hc *HealthChecker) CheckProvider(provider *Provider) *HealthCheckResult {
	result := &HealthCheckResult{
		ProviderID: provider.ID,
		Timestamp:  time.Now(),
	}

	// Check circuit breaker first
	cb := hc.getOrCreateCircuitBreaker(provider.ID)
	if !cb.Allow() {
		result.Success = false
		result.Error = fmt.Errorf("circuit breaker is open")
		hc.updateHealthStatus(provider.ID, result)
		return result
	}

	// Perform HTTP health check
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.Endpoint+"/health", nil)
	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("failed to create request: %w", err)
		cb.RecordFailure()
		hc.updateHealthStatus(provider.ID, result)
		return result
	}

	// Add auth header if available
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	client := &http.Client{Timeout: hc.config.Timeout}
	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Success = false
		result.Error = fmt.Errorf("health check failed: %w", err)
		cb.RecordFailure()
		hc.updateHealthStatus(provider.ID, result)
		return result
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
		cb.RecordSuccess()
	} else {
		result.Success = false
		result.Error = fmt.Errorf("health check returned status %d", resp.StatusCode)
		cb.RecordFailure()
	}

	hc.updateHealthStatus(provider.ID, result)
	return result
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a provider
func (hc *HealthChecker) getOrCreateCircuitBreaker(providerID string) *CircuitBreaker {
	hc.mu.RLock()
	cb, exists := hc.circuitBreakers[providerID]
	hc.mu.RUnlock()

	if exists {
		return cb
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = hc.circuitBreakers[providerID]; exists {
		return cb
	}

	cb = NewCircuitBreaker(
		hc.config.FailureThreshold,
		hc.config.SuccessThreshold,
		hc.config.CircuitTimeout,
	)
	hc.circuitBreakers[providerID] = cb
	return cb
}

// updateHealthStatus updates the health status for a provider
func (hc *HealthChecker) updateHealthStatus(providerID string, result *HealthCheckResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status, exists := hc.healthStatus[providerID]
	if !exists {
		status = &HealthStatus{
			ProviderID: providerID,
		}
		hc.healthStatus[providerID] = status
	}

	status.LastCheck = result.Timestamp
	status.LatencyMs = int(result.Latency.Milliseconds())

	metrics, exists := hc.metrics[providerID]
	if !exists {
		metrics = &ProviderHealthMetrics{
			ProviderID: providerID,
		}
		hc.metrics[providerID] = metrics
	}

	metrics.TotalChecks++
	if result.Success {
		metrics.SuccessfulChecks++
	} else {
		metrics.FailedChecks++
		metrics.LastError = ""
		if result.Error != nil {
			metrics.LastError = result.Error.Error()
		}
	}

	// Calculate rates
	if metrics.TotalChecks > 0 {
		metrics.SuccessRate = float64(metrics.SuccessfulChecks) / float64(metrics.TotalChecks)
		metrics.ErrorRate = float64(metrics.FailedChecks) / float64(metrics.TotalChecks)
	}

	// Update rolling average latency
	if metrics.TotalChecks == 1 {
		metrics.AvgLatencyMs = float64(result.Latency.Milliseconds())
		metrics.MinLatencyMs = int(result.Latency.Milliseconds())
		metrics.MaxLatencyMs = int(result.Latency.Milliseconds())
	} else {
		metrics.AvgLatencyMs = (metrics.AvgLatencyMs*float64(metrics.TotalChecks-1) + float64(result.Latency.Milliseconds())) / float64(metrics.TotalChecks)
		if result.Latency.Milliseconds() < int64(metrics.MinLatencyMs) {
			metrics.MinLatencyMs = int(result.Latency.Milliseconds())
		}
		if result.Latency.Milliseconds() > int64(metrics.MaxLatencyMs) {
			metrics.MaxLatencyMs = int(result.Latency.Milliseconds())
		}
	}

	// Determine health status based on circuit breaker
	cb := hc.circuitBreakers[providerID]
	if cb != nil {
		status.CircuitState = cb.State().String()
		status.FailureCount = func() int {
			cb.mu.RLock()
			defer cb.mu.RUnlock()
			return cb.failureCount
		}()
		status.IsHealthy = cb.State() != CircuitOpen
		status.SuccessRate = metrics.SuccessRate
		status.ErrorRate = metrics.ErrorRate
	} else {
		status.IsHealthy = result.Success
		status.CircuitState = "unknown"
	}

	status.NextCheck = time.Now().Add(hc.config.Interval)
	metrics.CurrentState = status.CircuitState
	metrics.P99LatencyMs = int(float64(metrics.MaxLatencyMs) * 0.99)
}

// GetHealthStatus returns the current health status for a provider
func (hc *HealthChecker) GetHealthStatus(providerID string) (*HealthStatus, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	status, exists := hc.healthStatus[providerID]
	return status, exists
}

// GetAllHealthStatus returns health status for all providers
func (hc *HealthChecker) GetAllHealthStatus() map[string]*HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]*HealthStatus, len(hc.healthStatus))
	for k, v := range hc.healthStatus {
		result[k] = v
	}
	return result
}

// GetMetrics returns health metrics for a provider
func (hc *HealthChecker) GetMetrics(providerID string) (*ProviderHealthMetrics, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	metrics, exists := hc.metrics[providerID]
	return metrics, exists
}

// GetAllMetrics returns health metrics for all providers
func (hc *HealthChecker) GetAllMetrics() map[string]*ProviderHealthMetrics {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]*ProviderHealthMetrics, len(hc.metrics))
	for k, v := range hc.metrics {
		result[k] = v
	}
	return result
}

// IsProviderHealthy checks if a specific provider is healthy
func (hc *HealthChecker) IsProviderHealthy(providerID string) bool {
	cb := hc.getOrCreateCircuitBreaker(providerID)
	return cb.State() != CircuitOpen
}
