package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	m := &Metrics{
		ModelRequests:  make(map[string]int64),
		ModelTokens:    make(map[string]int64),
		ModelErrors:    make(map[string]int64),
		TenantRequests: make(map[string]int64),
		StartTime:      time.Now(),
	}

	t.Run("RecordRequest success", func(t *testing.T) {
		m.RecordRequest("tenant-1", "gpt-4", true, 150)
		
		metrics := m.GetMetrics()
		requests := metrics["requests"].(map[string]interface{})
		
		if requests["total"].(int64) != 1 {
			t.Errorf("Total requests: got %d, want 1", requests["total"])
		}
		if requests["success"].(int64) != 1 {
			t.Errorf("Success requests: got %d, want 1", requests["success"])
		}
	})

	t.Run("RecordRequest error", func(t *testing.T) {
		m.RecordRequest("tenant-1", "gpt-4", false, 50)
		
		metrics := m.GetMetrics()
		requests := metrics["requests"].(map[string]interface{})
		
		if requests["errors"].(int64) != 1 {
			t.Errorf("Error requests: got %d, want 1", requests["errors"])
		}
	})

	t.Run("RecordRequest latency tracking", func(t *testing.T) {
		m.RecordRequest("tenant-1", "claude-3", true, 200)
		m.RecordRequest("tenant-1", "claude-3", true, 100)
		
		metrics := m.GetMetrics()
		latency := metrics["latency_ms"].(map[string]interface{})
		
		if latency["min"].(int64) != 50 {
			t.Errorf("Min latency: got %d, want 50", latency["min"])
		}
		if latency["max"].(int64) != 200 {
			t.Errorf("Max latency: got %d, want 200", latency["max"])
		}
	})

	t.Run("GetMetrics returns valid structure", func(t *testing.T) {
		metrics := m.GetMetrics()
		
		if metrics["timestamp"] == nil {
			t.Error("Timestamp should be present")
		}
		if metrics["uptime_seconds"] == nil {
			t.Error("Uptime should be present")
		}
		if metrics["requests"] == nil {
			t.Error("Requests should be present")
		}
		if metrics["latency_ms"] == nil {
			t.Error("Latency should be present")
		}
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

	t.Run("HandleMetrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		
		handler.HandleMetrics(w, req)
		
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
