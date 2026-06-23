package resilience

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CircuitBreakerMetrics holds Prometheus metrics for circuit breakers
type CircuitBreakerMetrics struct {
	// Circuit breaker state metrics
	CircuitState     *prometheus.GaugeVec
	CircuitTransitions *prometheus.CounterVec
	CircuitRequests  *prometheus.CounterVec
	CircuitDuration  *prometheus.HistogramVec
	CircuitFailures  *prometheus.CounterVec
	CircuitSuccesses *prometheus.CounterVec
	CircuitRejects  *prometheus.CounterVec

	// Bulkhead metrics
	BulkheadInFlight *prometheus.GaugeVec
	BulkheadWaiting  *prometheus.GaugeVec
	BulkheadRejected *prometheus.CounterVec

	mu sync.RWMutex
}

var (
	globalCBMetrics *CircuitBreakerMetrics
	cbMetricsOnce   sync.Once
)

// NewCircuitBreakerMetrics creates and registers circuit breaker Prometheus metrics
func NewCircuitBreakerMetrics(namespace string) *CircuitBreakerMetrics {
	m := &CircuitBreakerMetrics{
		CircuitState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_state",
				Help:      "Current state of circuit breaker (0=closed, 1=half-open, 2=open, 3=degraded)",
			},
			[]string{"provider_id"},
		),

		CircuitTransitions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_transitions_total",
				Help:      "Total number of circuit breaker state transitions",
			},
			[]string{"provider_id", "from_state", "to_state"},
		),

		CircuitRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_requests_total",
				Help:      "Total number of requests processed by circuit breaker",
			},
			[]string{"provider_id", "result"},
		),

		CircuitDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_duration_seconds",
				Help:      "Duration of circuit breaker operations",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
			},
			[]string{"provider_id", "operation"},
		),

		CircuitFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_failures_total",
				Help:      "Total number of failures recorded by circuit breaker",
			},
			[]string{"provider_id", "error_type"},
		),

		CircuitSuccesses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_successes_total",
				Help:      "Total number of successes recorded by circuit breaker",
			},
			[]string{"provider_id"},
		),

		CircuitRejects: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_rejects_total",
				Help:      "Total number of requests rejected due to open circuit",
			},
			[]string{"provider_id", "reason"},
		),

		BulkheadInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bulkhead_in_flight",
				Help:      "Number of requests currently in flight for bulkhead",
			},
			[]string{"bulkhead_name", "priority"},
		),

		BulkheadWaiting: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "bulkhead_waiting",
				Help:      "Number of requests currently waiting for bulkhead",
			},
			[]string{"bulkhead_name", "priority"},
		),

		BulkheadRejected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bulkhead_rejected_total",
				Help:      "Total number of requests rejected by bulkhead",
			},
			[]string{"bulkhead_name", "priority", "reason"},
		),
	}

	return m
}

// GetGlobalCBMetrics returns the global circuit breaker metrics instance
func GetGlobalCBMetrics() *CircuitBreakerMetrics {
	cbMetricsOnce.Do(func() {
		globalCBMetrics = NewCircuitBreakerMetrics("nexusai_gateway")
	})
	return globalCBMetrics
}

// RecordStateChange records a circuit breaker state transition
func (m *CircuitBreakerMetrics) RecordStateChange(providerID string, fromState, toState CBState) {
	m.CircuitTransitions.WithLabelValues(providerID, string(fromState), string(toState)).Inc()
	m.updateStateGauge(providerID, toState)
}

// updateStateGauge updates the state gauge based on current state
func (m *CircuitBreakerMetrics) updateStateGauge(providerID string, state CBState) {
	var value float64
	switch state {
	case CBStateClosed:
		value = 0
	case CBStateHalfOpen:
		value = 1
	case CBStateOpen:
		value = 2
	case CBStateDegraded:
		value = 3
	}
	m.CircuitState.WithLabelValues(providerID).Set(value)
}

// RecordRequest records a request processed by the circuit breaker
func (m *CircuitBreakerMetrics) RecordRequest(providerID string, allowed bool) {
	if allowed {
		m.CircuitRequests.WithLabelValues(providerID, "allowed").Inc()
	} else {
		m.CircuitRequests.WithLabelValues(providerID, "rejected").Inc()
	}
}

// RecordFailure records a failure for a circuit breaker
func (m *CircuitBreakerMetrics) RecordFailure(providerID string, errorType string) {
	m.CircuitFailures.WithLabelValues(providerID, errorType).Inc()
}

// RecordSuccess records a success for a circuit breaker
func (m *CircuitBreakerMetrics) RecordSuccess(providerID string) {
	m.CircuitSuccesses.WithLabelValues(providerID).Inc()
}

// RecordReject records a rejection due to open circuit
func (m *CircuitBreakerMetrics) RecordReject(providerID string, reason string) {
	m.CircuitRejects.WithLabelValues(providerID, reason).Inc()
}

// RecordBulkheadInFlight records in-flight requests for a bulkhead
func (m *CircuitBreakerMetrics) RecordBulkheadInFlight(bulkheadName, priority string, count float64) {
	m.BulkheadInFlight.WithLabelValues(bulkheadName, priority).Set(count)
}

// RecordBulkheadWaiting records waiting requests for a bulkhead
func (m *CircuitBreakerMetrics) RecordBulkheadWaiting(bulkheadName, priority string, count float64) {
	m.BulkheadWaiting.WithLabelValues(bulkheadName, priority).Set(count)
}

// RecordBulkheadRejected records a rejected request by bulkhead
func (m *CircuitBreakerMetrics) RecordBulkheadRejected(bulkheadName, priority, reason string) {
	m.BulkheadRejected.WithLabelValues(bulkheadName, priority, reason).Inc()
}

// InstrumentProviderCB wraps a ProviderCB with metrics collection
type InstrumentedProviderCB struct {
	*ProviderCB
	metrics *CircuitBreakerMetrics
}

// NewInstrumentedProviderCB creates a new instrumented circuit breaker
func NewInstrumentedProviderCB(providerID string) *InstrumentedProviderCB {
	return &InstrumentedProviderCB{
		ProviderCB: NewProviderCB(providerID),
		metrics:    GetGlobalCBMetrics(),
	}
}

// Allow checks if a request should be allowed and records metrics
func (cb *InstrumentedProviderCB) Allow() bool {
	allowed := cb.ProviderCB.Allow()
	cb.metrics.RecordRequest(cb.ProviderID, allowed)
	if !allowed {
		cb.metrics.RecordReject(cb.ProviderCB, "circuit_open")
	}
	return allowed
}

// RecordSuccess records a success and updates metrics
func (cb *InstrumentedProviderCB) RecordSuccess() {
	prevState := cb.ProviderCB.GetState()
	cb.ProviderCB.RecordSuccess()
	cb.metrics.RecordSuccess(cb.ProviderCB)
	
	newState := cb.ProviderCB.GetState()
	if prevState != newState {
		cb.metrics.RecordStateChange(cb.ProviderCB, prevState, newState)
	}
}

// RecordFailure records a failure and updates metrics
func (cb *InstrumentedProviderCB) RecordFailure() {
	prevState := cb.ProviderCB.GetState()
	cb.ProviderCB.RecordFailure()
	cb.metrics.RecordFailure(cb.ProviderCB, "provider_error")
	
	newState := cb.ProviderCB.GetState()
	if prevState != newState {
		cb.metrics.RecordStateChange(cb.ProviderCB, prevState, newState)
	}
}

// InstrumentedThreeLayerResilience wraps ThreeLayerResilience with metrics
type InstrumentedThreeLayerResilience struct {
	*ThreeLayerResilience
	metrics *CircuitBreakerMetrics
}

// NewInstrumentedThreeLayerResilience creates a new instrumented resilience system
func NewInstrumentedThreeLayerResilience() *InstrumentedThreeLayerResilience {
	return &InstrumentedThreeLayerResilience{
		ThreeLayerResilience: NewThreeLayerResilience(),
		metrics:              GetGlobalCBMetrics(),
	}
}

// GetProviderCB gets or creates an instrumented provider circuit breaker
func (t *InstrumentedThreeLayerResilience) GetProviderCB(providerID string) *InstrumentedProviderCB {
	// First get or create the underlying CB
	innerCB := t.ThreeLayerResilience.GetProviderCB(providerID)
	
	// Wrap with instrumentation
	return &InstrumentedProviderCB{
		ProviderCB: innerCB,
		metrics:    t.metrics,
	}
}

// GetAllStatsWithMetrics returns stats plus current metrics state
func (t *InstrumentedThreeLayerResilience) GetAllStatsWithMetrics() []*CBStatsWithMetrics {
	innerStats := t.ThreeLayerResilience.GetAllStats()
	result := make([]*CBStatsWithMetrics, len(innerStats))
	
	for i, stat := range innerStats {
		result[i] = &CBStatsWithMetrics{
			CBStats: stat,
			FailureRate: calculateFailureRate(stat.TotalRequests, stat.TotalFailures),
			SuccessRate: calculateFailureRate(stat.TotalRequests, stat.TotalSuccesses),
		}
	}
	
	return result
}

// CBStatsWithMetrics extends CBStats with calculated metrics
type CBStatsWithMetrics struct {
	*CBStats
	FailureRate float64 `json:"failure_rate"`
	SuccessRate float64 `json:"success_rate"`
}

func calculateFailureRate(total, failures int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(failures) / float64(total)
}

// MetricsCollector interface implementation for observability package integration
func (m *CircuitBreakerMetrics) RecordCircuitBreaker(providerID string, state CBState, requests, failures, successes int64) {
	m.updateStateGauge(providerID, state)
	
	if requests > 0 {
		m.CircuitRequests.WithLabelValues(providerID, "total").Add(float64(requests))
	}
	if failures > 0 {
		m.CircuitFailures.WithLabelValues(providerID, "total").Add(float64(failures))
	}
	if successes > 0 {
		m.CircuitSuccesses.WithLabelValues(providerID).Add(float64(successes))
	}
}
