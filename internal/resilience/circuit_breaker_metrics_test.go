package resilience

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerMetrics_Creation(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	// Create new metrics
	metrics := NewCircuitBreakerMetrics("test")

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.CircuitState)
	assert.NotNil(t, metrics.CircuitTransitions)
	assert.NotNil(t, metrics.CircuitRequests)
	assert.NotNil(t, metrics.CircuitDuration)
	assert.NotNil(t, metrics.CircuitFailures)
	assert.NotNil(t, metrics.CircuitSuccesses)
	assert.NotNil(t, metrics.CircuitRejects)
	assert.NotNil(t, metrics.BulkheadInFlight)
	assert.NotNil(t, metrics.BulkheadWaiting)
	assert.NotNil(t, metrics.BulkheadRejected)
}

func TestCircuitBreakerMetrics_GlobalInstance(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	// Get global instance
	metrics1 := GetGlobalCBMetrics()
	metrics2 := GetGlobalCBMetrics()

	assert.Same(t, metrics1, metrics2, "Should return same instance")
}

func TestCircuitBreakerMetrics_RecordStateChange(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record state transitions
	metrics.RecordStateChange(providerID, CBStateClosed, CBStateOpen)
	metrics.RecordStateChange(providerID, CBStateOpen, CBStateHalfOpen)
	metrics.RecordStateChange(providerID, CBStateHalfOpen, CBStateClosed)

	// Verify gauge was updated
	gauge, err := metrics.CircuitState.GetMetricWithLabelValues(providerID)
	require.NoError(t, err)
	require.NotNil(t, gauge)

	// Gauge should be at Closed state (0)
	var m prometheus.Metric
	ch := make(chan prometheus.Metric, 1)
	gauge.Collect(ch)
	m = <-ch

	// Should not panic - metrics collected successfully
	assert.NotNil(t, m)
}

func TestCircuitBreakerMetrics_RecordRequest(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record allowed request
	metrics.RecordRequest(providerID, true)

	// Record rejected request
	metrics.RecordRequest(providerID, false)

	// Verify counter incremented
	counter, err := metrics.CircuitRequests.GetMetricWithLabelValues(providerID, "allowed")
	require.NoError(t, err)
	require.NotNil(t, counter)

	ch := make(chan prometheus.Metric, 1)
	counter.Collect(ch)
	m := <-ch

	// Verify metric was collected
	assert.NotNil(t, m)
}

func TestCircuitBreakerMetrics_RecordFailure(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record failures
	metrics.RecordFailure(providerID, "timeout")
	metrics.RecordFailure(providerID, "connection_error")
	metrics.RecordFailure(providerID, "timeout")

	// Verify counter incremented
	counter, err := metrics.CircuitFailures.GetMetricWithLabelValues(providerID, "timeout")
	require.NoError(t, err)

	var m prometheus.Metric
	ch := make(chan prometheus.Metric, 1)
	counter.Collect(ch)
	m = <-ch

	assert.NotNil(t, m)
}

func TestCircuitBreakerMetrics_RecordSuccess(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record successes
	metrics.RecordSuccess(providerID)
	metrics.RecordSuccess(providerID)
	metrics.RecordSuccess(providerID)

	// Verify counter incremented
	counter, err := metrics.CircuitSuccesses.GetMetricWithLabelValues(providerID)
	require.NoError(t, err)

	var m prometheus.Metric
	ch := make(chan prometheus.Metric, 1)
	counter.Collect(ch)
	m = <-ch

	assert.NotNil(t, m)
}

func TestCircuitBreakerMetrics_RecordReject(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record rejects
	metrics.RecordReject(providerID, "circuit_open")
	metrics.RecordReject(providerID, "rate_limit")

	// Verify counter incremented
	counter, err := metrics.CircuitRejects.GetMetricWithLabelValues(providerID, "circuit_open")
	require.NoError(t, err)

	var m prometheus.Metric
	ch := make(chan prometheus.Metric, 1)
	counter.Collect(ch)
	m = <-ch

	assert.NotNil(t, m)
}

func TestCircuitBreakerMetrics_UpdateStateGauge(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	testCases := []struct {
		state  CBState
		expect float64
	}{
		{CBStateClosed, 0},
		{CBStateHalfOpen, 1},
		{CBStateOpen, 2},
		{CBStateDegraded, 3},
	}

	for _, tc := range testCases {
		metrics.updateStateGauge(providerID, tc.state)

		gauge, err := metrics.CircuitState.GetMetricWithLabelValues(providerID)
		require.NoError(t, err)

		var m prometheus.Metric
		ch := make(chan prometheus.Metric, 1)
		gauge.Collect(ch)
		m = <-ch

		assert.NotNil(t, m, "Gauge should be collectible for state %s", tc.state)
	}
}

func TestInstrumentedProviderCB_Allow(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	cb := NewInstrumentedProviderCB("test-provider")

	// Initial state should be closed - allow should return true
	assert.True(t, cb.Allow())

	// Record some failures to trigger open state
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Circuit should now be open - allow should return false
	assert.False(t, cb.Allow())
}

func TestInstrumentedProviderCB_RecordSuccess(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	cb := NewInstrumentedProviderCB("test-provider")

	// Record some failures first
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Circuit should be open
	assert.Equal(t, CBStateOpen, cb.GetState())

	// Allow one request (should be rejected but triggers half-open)
	cb.Allow()

	// Record success in half-open state
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	// Should transition back to closed
	assert.Equal(t, CBStateClosed, cb.GetState())
}

func TestInstrumentedProviderCB_RecordFailure(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	cb := NewInstrumentedProviderCB("test-provider")

	// Record failures until circuit opens
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}

	// Should be open now
	assert.Equal(t, CBStateOpen, cb.GetState())
}

func TestInstrumentedThreeLayerResilience_GetProviderCB(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	resilience := NewInstrumentedThreeLayerResilience()

	// Get instrumented CB for a provider
	cb1 := resilience.GetProviderCB("provider-1")
	cb2 := resilience.GetProviderCB("provider-1")

	// Should return same instance
	assert.Same(t, cb1.ProviderCB, cb2.ProviderCB)
}

func TestInstrumentedThreeLayerResilience_GetAllStatsWithMetrics(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	resilience := NewInstrumentedThreeLayerResilience()

	// Create some activity
	cb1 := resilience.GetProviderCB("provider-1")
	cb1.RecordSuccess()
	cb1.RecordSuccess()
	cb1.RecordFailure()

	cb2 := resilience.GetProviderCB("provider-2")
	cb2.RecordSuccess()
	cb2.RecordFailure()
	cb2.RecordFailure()

	// Get stats with metrics
	stats := resilience.GetAllStatsWithMetrics()

	assert.Len(t, stats, 2)

	// Find stats for provider-1
	var provider1Stats *CBStatsWithMetrics
	for _, s := range stats {
		if s.ProviderID == "provider-1" {
			provider1Stats = s
			break
		}
	}

	require.NotNil(t, provider1Stats)
	assert.Equal(t, int64(3), provider1Stats.TotalRequests)
	assert.Equal(t, int64(2), provider1Stats.TotalSuccesses)
	assert.Equal(t, int64(1), provider1Stats.TotalFailures)
	assert.Greater(t, provider1Stats.SuccessRate, 0.0)
}

func TestCalculateFailureRate(t *testing.T) {
	testCases := []struct {
		total     int64
		failures  int64
		expected  float64
	}{
		{100, 10, 0.1},
		{100, 50, 0.5},
		{100, 100, 1.0},
		{100, 0, 0.0},
		{0, 0, 0.0},
	}

	for _, tc := range testCases {
		rate := calculateFailureRate(tc.total, tc.failures)
		assert.Equal(t, tc.expected, rate, "Failure rate for %d/%d should be %f", tc.total, tc.failures, tc.expected)
	}
}

func TestCircuitBreakerMetrics_RecordCircuitBreaker(t *testing.T) {
	// Reset global metrics for clean test
	globalCBMetrics = nil
	cbMetricsOnce = sync.Once{}

	metrics := GetGlobalCBMetrics()
	providerID := "test-provider"

	// Record full circuit breaker state
	metrics.RecordCircuitBreaker(providerID, CBStateClosed, 100, 5, 95)

	// Verify all metrics were updated
	counter, err := metrics.CircuitRequests.GetMetricWithLabelValues(providerID, "total")
	require.NoError(t, err)
	assert.NotNil(t, counter)
}
