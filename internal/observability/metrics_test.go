package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	// Create a new metrics instance
	m := NewMetrics("test_gateway")

	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}

	if m.RequestsTotal == nil {
		t.Error("RequestsTotal is nil")
	}

	if m.RequestDuration == nil {
		t.Error("RequestDuration is nil")
	}

	if m.ActiveConnections == nil {
		t.Error("ActiveConnections is nil")
	}

	if m.ModelCallsTotal == nil {
		t.Error("ModelCallsTotal is nil")
	}
}

func TestGlobalMetrics(t *testing.T) {
	// Get global metrics
	m := GetGlobalMetrics()

	if m == nil {
		t.Fatal("GetGlobalMetrics() returned nil")
	}

	// Second call should return the same instance
	m2 := GetGlobalMetrics()
	if m != m2 {
		t.Error("GetGlobalMetrics() returned different instances")
	}
}

func TestObserveRequest(t *testing.T) {
	m := NewMetrics("test_gateway_observe")

	// Record some requests
	m.ObserveRequest("GET", "/api/test", 200, 50*time.Millisecond)
	m.ObserveRequest("POST", "/api/test", 201, 100*time.Millisecond)
	m.ObserveRequest("GET", "/api/test", 400, 20*time.Millisecond)
	m.ObserveRequest("GET", "/api/test", 500, 200*time.Millisecond)

	// Verify counters were incremented
	counter, err := m.RequestsTotal.GetMetricWithLabelValues("GET", "/api/test", "2xx")
	if err != nil {
		t.Fatalf("GetMetricWithLabelValues() error = %v", err)
	}

	value := testutil.ToFloat64(counter)
	if value < 1 {
		t.Errorf("Request counter = %v, want >= 1", value)
	}
}

func TestIncrementDecrementRequestsInFlight(t *testing.T) {
	m := NewMetrics("test_gateway_inflight")

	// Increment
	m.IncrementRequestsInFlight()
	m.IncrementRequestsInFlight()

	// Get value
	value := testutil.ToFloat64(m.RequestsInFlight)
	if value < 2 {
		t.Errorf("RequestsInFlight = %v, want >= 2", value)
	}

	// Decrement
	m.DecrementRequestsInFlight()
	value = testutil.ToFloat64(m.RequestsInFlight)
	if value < 1 {
		t.Errorf("RequestsInFlight after decrement = %v, want >= 1", value)
	}
}

func TestIncrementDecrementActiveConnections(t *testing.T) {
	m := NewMetrics("test_gateway_connections")

	m.IncrementActiveConnections()
	m.IncrementActiveConnections()

	value := testutil.ToFloat64(m.ActiveConnections)
	if value < 2 {
		t.Errorf("ActiveConnections = %v, want >= 2", value)
	}

	m.DecrementActiveConnections()
	value = testutil.ToFloat64(m.ActiveConnections)
	if value < 1 {
		t.Errorf("ActiveConnections after decrement = %v, want >= 1", value)
	}
}

func TestRecordModelCall(t *testing.T) {
	m := NewMetrics("test_gateway_model")

	m.RecordModelCall("gpt-4", "success", 1500*time.Millisecond)
	m.RecordModelCall("gpt-4", "error", 500*time.Millisecond)
	m.RecordModelCall("claude-3", "success", 2000*time.Millisecond)

	// Verify counters
	successCounter, _ := m.ModelCallsTotal.GetMetricWithLabelValues("gpt-4", "success")
	if testutil.ToFloat64(successCounter) < 1 {
		t.Error("Expected at least 1 successful gpt-4 call")
	}

	errorCounter, _ := m.ModelCallsTotal.GetMetricWithLabelValues("gpt-4", "error")
	if testutil.ToFloat64(errorCounter) < 1 {
		t.Error("Expected at least 1 error gpt-4 call")
	}
}

func TestRecordModelTokens(t *testing.T) {
	m := NewMetrics("test_gateway_tokens")

	m.RecordModelTokens("gpt-4", 100, 200)
	m.RecordModelTokens("gpt-4", 150, 250)

	// Verify token counters
	promptCounter, _ := m.ModelTokensTotal.GetMetricWithLabelValues("gpt-4", "prompt")
	promptValue := testutil.ToFloat64(promptCounter)
	if promptValue < 250 {
		t.Errorf("Prompt tokens = %v, want >= 250", promptValue)
	}

	completionCounter, _ := m.ModelTokensTotal.GetMetricWithLabelValues("gpt-4", "completion")
	completionValue := testutil.ToFloat64(completionCounter)
	if completionValue < 450 {
		t.Errorf("Completion tokens = %v, want >= 450", completionValue)
	}
}

func TestRecordModelError(t *testing.T) {
	m := NewMetrics("test_gateway_errors")

	m.RecordModelError("gpt-4", "timeout")
	m.RecordModelError("gpt-4", "timeout")
	m.RecordModelError("gpt-4", "rate_limit")

	timeoutCounter, _ := m.ModelErrorsTotal.GetMetricWithLabelValues("gpt-4", "timeout")
	if testutil.ToFloat64(timeoutCounter) < 2 {
		t.Error("Expected at least 2 timeout errors")
	}
}

func TestRecordPipelineExecution(t *testing.T) {
	m := NewMetrics("test_gateway_pipeline")

	m.RecordPipelineExecution("auth", "success", 100*time.Millisecond)
	m.RecordPipelineExecution("auth", "error", 50*time.Millisecond)
	m.RecordPipelineExecution("privacy", "success", 200*time.Millisecond)

	successCounter, _ := m.PipelineExecutionsTotal.GetMetricWithLabelValues("auth", "success")
	if testutil.ToFloat64(successCounter) < 1 {
		t.Error("Expected at least 1 successful auth pipeline")
	}
}

func TestSetUptime(t *testing.T) {
	m := NewMetrics("test_gateway_uptime")

	m.SetUptime(3600) // 1 hour

	value := testutil.ToFloat64(m.UptimeSeconds)
	if value < 3600 {
		t.Errorf("Uptime = %v, want >= 3600", value)
	}
}

func TestSetDatabaseConnected(t *testing.T) {
	m := NewMetrics("test_gateway_db")

	// Test connected state
	m.SetDatabaseConnected(true)
	value := testutil.ToFloat64(m.DatabaseConnected)
	if value != 1 {
		t.Errorf("DatabaseConnected (true) = %v, want 1", value)
	}

	// Test disconnected state
	m.SetDatabaseConnected(false)
	value = testutil.ToFloat64(m.DatabaseConnected)
	if value != 0 {
		t.Errorf("DatabaseConnected (false) = %v, want 0", value)
	}
}

func TestStatusCodeToString(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{200, "2xx"},
		{201, "2xx"},
		{299, "2xx"},
		{301, "3xx"},
		{304, "3xx"},
		{400, "4xx"},
		{401, "4xx"},
		{404, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
		{600, "5xx"},
		{100, "unknown"},
	}

	for _, tt := range tests {
		got := statusCodeToString(tt.code)
		if got != tt.want {
			t.Errorf("statusCodeToString(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestPrometheusHandler(t *testing.T) {
	handler := PrometheusHandler()

	if handler == nil {
		t.Fatal("PrometheusHandler() returned nil")
	}

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Serve the handler
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("Expected Content-Type header to be set")
	}
}

func TestPrometheusMetricsCollector_IncrementCounter(t *testing.T) {
	collector := NewPrometheusMetricsCollector()

	// These should not panic
	collector.IncrementCounter("requests_total", map[string]string{
		"method": "GET",
		"path":   "/test",
		"status": "2xx",
	})

	collector.IncrementCounter("model_calls_total", map[string]string{
		"model":  "gpt-4",
		"status": "success",
	})
}

func TestPrometheusMetricsCollector_ObserveHistogram(t *testing.T) {
	collector := NewPrometheusMetricsCollector()

	// These should not panic
	collector.ObserveHistogram("request_duration_seconds", 0.05, map[string]string{
		"method": "GET",
		"path":   "/test",
	})

	collector.ObserveHistogram("model_call_duration_seconds", 1.5, map[string]string{
		"model":    "gpt-4",
		"provider": "openai",
	})
}

func TestInitGlobalMetricsCollector(t *testing.T) {
	// This should initialize the global metrics
	InitGlobalMetricsCollector()

	// GetGlobalMetrics should return the initialized instance
	metrics := GetGlobalMetrics()
	if metrics == nil {
		t.Error("GetGlobalMetrics is nil after InitGlobalMetricsCollector()")
	}
}
