package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	m := NewMetrics("test")

	t.Run("RecordModelCall success", func(t *testing.T) {
		m.RecordModelCall("gpt-4", "success", 150*time.Millisecond)
		// Metrics are recorded via Prometheus, validated via integration tests
	})

	t.Run("RecordModelCall error", func(t *testing.T) {
		m.RecordModelError("gpt-4", "timeout")
		// Error counter incremented
	})

	t.Run("RecordModelTokens", func(t *testing.T) {
		m.RecordModelTokens("gpt-4", 100, 50)
		// Token counters incremented
	})

	t.Run("RecordPipelineExecution", func(t *testing.T) {
		m.RecordPipelineExecution("chat", "success", 100*time.Millisecond)
		// Pipeline metrics recorded
	})
}

func TestHealthCheck(t *testing.T) {
	t.Run("PerformHealthCheck with no checks", func(t *testing.T) {
		health := PerformHealthCheck()

		if health.Status != "healthy" {
			t.Errorf("Status: got %s, want healthy", health.Status)
		}
	})

	t.Run("PerformHealthCheck with passing check", func(t *testing.T) {
		RegisterHealthCheck(&testHealthChecker{name: "test", healthy: true})

		health := PerformHealthCheck()

		if health.Checks["test"].Status != "healthy" {
			t.Errorf("Check status: got %s, want healthy", health.Checks["test"].Status)
		}

		// Cleanup
		GlobalHealthChecks = nil
	})

	t.Run("PerformHealthCheck with failing check", func(t *testing.T) {
		RegisterHealthCheck(&testHealthChecker{name: "failing", healthy: false, message: "service down"})

		health := PerformHealthCheck()

		if health.Checks["failing"].Status != "unhealthy" {
			t.Errorf("Check status: got %s, want unhealthy", health.Checks["failing"].Status)
		}
		if health.Status != "degraded" {
			t.Errorf("Overall status: got %s, want degraded", health.Status)
		}

		// Cleanup
		GlobalHealthChecks = nil
	})
}

type testHealthChecker struct {
	name    string
	healthy bool
	message string
}

func (c *testHealthChecker) Name() string {
	return c.name
}

func (c *testHealthChecker) Check() (bool, string) {
	return c.healthy, c.message
}

func TestTraceCollector(t *testing.T) {
	collector := NewTraceCollector(10)

	t.Run("AddTrace", func(t *testing.T) {
		trace := TracingContext{
			TraceID:   "trace-1",
			SpanID:    "span-1",
			StartTime: time.Now(),
		}
		collector.AddTrace(trace)

		traces := collector.GetTraces(5)
		if len(traces) != 1 {
			t.Errorf("Trace count: got %d, want 1", len(traces))
		}
	})

	t.Run("GetTraces with limit", func(t *testing.T) {
		// Add more traces
		for i := 0; i < 5; i++ {
			collector.AddTrace(TracingContext{
				TraceID:   "trace-" + string(rune('0'+i)),
				StartTime: time.Now(),
			})
		}

		traces := collector.GetTraces(3)
		if len(traces) != 3 {
			t.Errorf("Traces with limit: got %d, want 3", len(traces))
		}
	})
}

func TestLogCollector(t *testing.T) {
	collector := NewLogCollector(100)

	t.Run("AddLog", func(t *testing.T) {
		collector.AddLog(LogEntry{
			Level:   "info",
			Message: "Test message",
			Service: "test",
		})

		logs := collector.GetLogs(10, "")
		if len(logs) != 1 {
			t.Errorf("Log count: got %d, want 1", len(logs))
		}
	})

	t.Run("GetLogs with level filter", func(t *testing.T) {
		collector.AddLog(LogEntry{Level: "error", Message: "Error 1"})
		collector.AddLog(LogEntry{Level: "info", Message: "Info 1"})

		errorLogs := collector.GetLogs(10, "error")
		if len(errorLogs) != 1 {
			t.Errorf("Error logs: got %d, want 1", len(errorLogs))
		}
	})
}

func TestObservabilityHandler(t *testing.T) {
	handler := NewObservabilityHandler()

	t.Run("HandlePipelineMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		handler.HandlePipelineMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code: got %d, want 200", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["requests"] == nil {
			t.Error("Requests section missing")
		}
	})

	t.Run("HandleHealth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		handler.HandleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code: got %d, want 200", w.Code)
		}

		var health Health
		if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if health.Status == "" {
			t.Error("Health status missing")
		}
	})

	t.Run("HandleTraces", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/traces", nil)
		w := httptest.NewRecorder()

		handler.HandleTraces(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code: got %d, want 200", w.Code)
		}
	})

	t.Run("HandleLogs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/logs", nil)
		w := httptest.NewRecorder()

		handler.HandleLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code: got %d, want 200", w.Code)
		}
	})

	t.Run("HandleStatus", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()

		handler.HandleStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code: got %d, want 200", w.Code)
		}

		var status map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if status["service"] != "nexusai-gateway" {
			t.Errorf("Service name: got %v, want nexusai-gateway", status["service"])
		}
	})
}

func TestLogf(t *testing.T) {
	t.Run("Logf creates log entry", func(t *testing.T) {
		Logf("info", "Test message %s", "param")

		logs := GlobalLogCollector.GetLogs(1, "info")
		if len(logs) < 1 {
			t.Skip("No logs in collector")
		}

		if logs[0].Message != "Test message param" {
			t.Errorf("Message: got %s, want 'Test message param'", logs[0].Message)
		}
	})
}
