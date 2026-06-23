package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTelemetryData_SetResponseHeaders(t *testing.T) {
	data := &TelemetryData{
		ResponseCost:     0.005,
		TokensIn:        150,
		TokensOut:       50,
		Model:           "gpt-4",
		Provider:        "openai",
		LatencyMs:       1234,
		CacheHit:        true,
		FallbackAttempts: 0,
		RequestID:       "req-123",
		TraceID:         "trace-456",
		SpanID:          "span-789",
	}

	t.Run("sets all headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		data.SetResponseHeaders(w)

		if w.Header().Get(HeaderResponseCost) == "" {
			t.Error("Expected X-NexusAI-Response-Cost header to be set")
		}

		if w.Header().Get(HeaderTokensIn) == "" {
			t.Error("Expected X-NexusAI-Tokens-In header to be set")
		}

		if w.Header().Get(HeaderTokensOut) == "" {
			t.Error("Expected X-NexusAI-Tokens-Out header to be set")
		}

		if w.Header().Get(HeaderModel) == "" {
			t.Error("Expected X-NexusAI-Model header to be set")
		}

		if w.Header().Get(HeaderProvider) == "" {
			t.Error("Expected X-NexusAI-Provider header to be set")
		}

		if w.Header().Get(HeaderLatencyMs) == "" {
			t.Error("Expected X-NexusAI-Latency-Ms header to be set")
		}

		if w.Header().Get(HeaderCacheHit) == "" {
			t.Error("Expected X-NexusAI-Cache-Hit header to be set")
		}

		if w.Header().Get(HeaderFallbackAttempts) == "" {
			t.Error("Expected X-NexusAI-Fallback-Attempts header to be set")
		}
	})

	t.Run("sets correlation headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		data.SetResponseHeaders(w)

		if w.Header().Get(HeaderRequestID) != "req-123" {
			t.Errorf("Expected X-NexusAI-Request-ID=req-123, got %s", w.Header().Get(HeaderRequestID))
		}

		if w.Header().Get(HeaderTraceID) != "trace-456" {
			t.Errorf("Expected X-NexusAI-Trace-ID=trace-456, got %s", w.Header().Get(HeaderTraceID))
		}

		if w.Header().Get(HeaderSpanID) != "span-789" {
			t.Errorf("Expected X-NexusAI-Span-ID=span-789, got %s", w.Header().Get(HeaderSpanID))
		}
	})
}

func TestTelemetryData_GetResponseHeaders(t *testing.T) {
	data := &TelemetryData{
		ResponseCost:     0.005,
		TokensIn:        150,
		TokensOut:       50,
		Model:           "gpt-4",
		Provider:        "openai",
		LatencyMs:       1234,
		CacheHit:        true,
		FallbackAttempts: 0,
	}

	headers := data.GetResponseHeaders()

	if headers[HeaderResponseCost] == "" {
		t.Error("Expected response cost header")
	}

	if headers[HeaderTokensIn] != "150" {
		t.Errorf("Expected tokens in 150, got %s", headers[HeaderTokensIn])
	}

	if headers[HeaderCacheHit] != "true" {
		t.Errorf("Expected cache hit true, got %s", headers[HeaderCacheHit])
	}
}

func TestTelemetryData_ToMap(t *testing.T) {
	data := &TelemetryData{
		ResponseCost:     0.005,
		TokensIn:        150,
		TokensOut:       50,
		Model:           "gpt-4",
		Provider:        "openai",
		LatencyMs:       1234,
		CacheHit:        true,
		FallbackAttempts: 2,
		RequestID:       "req-123",
	}

	m := data.ToMap()

	if m["response_cost"] != 0.005 {
		t.Errorf("Expected response_cost 0.005, got %v", m["response_cost"])
	}

	if m["tokens_in"] != 150 {
		t.Errorf("Expected tokens_in 150, got %v", m["tokens_in"])
	}

	if m["model"] != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %v", m["model"])
	}

	if m["cache_hit"] != true {
		t.Errorf("Expected cache_hit true, got %v", m["cache_hit"])
	}

	if m["fallback_attempts"] != 2 {
		t.Errorf("Expected fallback_attempts 2, got %v", m["fallback_attempts"])
	}
}

func TestTelemetryData_SetFromLatency(t *testing.T) {
	data := &TelemetryData{}
	startTime := time.Now().Add(-1500 * time.Millisecond)

	data.SetFromLatency(startTime)

	if data.LatencyMs < 1400 || data.LatencyMs > 1600 {
		t.Errorf("Expected latency around 1500ms, got %d", data.LatencyMs)
	}
}

func TestNewTelemetryData(t *testing.T) {
	data := NewTelemetryData()

	if data.FallbackAttempts != 0 {
		t.Errorf("Expected FallbackAttempts=0, got %d", data.FallbackAttempts)
	}

	if data.CacheHit != false {
		t.Error("Expected CacheHit=false")
	}

	if data.TokensIn != 0 {
		t.Errorf("Expected TokensIn=0, got %d", data.TokensIn)
	}

	if data.ResponseCost != 0.0 {
		t.Errorf("Expected ResponseCost=0, got %f", data.ResponseCost)
	}
}

func TestTelemetryMiddleware(t *testing.T) {
	m := NewTelemetryMiddleware()

	if m == nil {
		t.Fatal("Expected middleware to be created")
	}
}

func TestCostCalculator(t *testing.T) {
	calc := NewCostCalculator()

	t.Run("calculates cost correctly", func(t *testing.T) {
		calc.SetModelPricing("gpt-4", 0.03, 0.06) // $0.03/1K in, $0.06/1K out

		cost := calc.CalculateCost("gpt-4", 1000, 500)
		// Expected: (1000/1000)*0.03 + (500/1000)*0.06 = 0.03 + 0.03 = 0.06
		expected := 0.06
		if cost != expected {
			t.Errorf("Expected cost %f, got %f", expected, cost)
		}
	})

	t.Run("returns 0 for unknown model", func(t *testing.T) {
		cost := calc.CalculateCost("unknown-model", 1000, 500)
		if cost != 0.0 {
			t.Errorf("Expected cost 0 for unknown model, got %f", cost)
		}
	})

	t.Run("calculates with large token counts", func(t *testing.T) {
		calc.SetModelPricing("claude-3", 0.015, 0.075)

		cost := calc.CalculateCost("claude-3", 100000, 50000)
		// Expected: (100000/1000)*0.015 + (50000/1000)*0.075 = 1.5 + 3.75 = 5.25
		expected := 5.25
		if cost != expected {
			t.Errorf("Expected cost %f, got %f", expected, cost)
		}
	})
}

func TestGlobalCostCalculator(t *testing.T) {
	if GlobalCostCalculator == nil {
		t.Fatal("Expected GlobalCostCalculator to be initialized")
	}

	GlobalCostCalculator.SetModelPricing("test-model", 0.01, 0.02)
	cost := GlobalCostCalculator.CalculateCost("test-model", 1000, 1000)

	expected := 0.03 // (1*0.01 + 1*0.02)
	if cost != expected {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestTelemetryHeaders(t *testing.T) {
	t.Run("all header constants are defined", func(t *testing.T) {
		headers := []string{
			HeaderResponseCost,
			HeaderTokensIn,
			HeaderTokensOut,
			HeaderModel,
			HeaderProvider,
			HeaderLatencyMs,
			HeaderCacheHit,
			HeaderFallbackAttempts,
			HeaderRequestID,
			HeaderTraceID,
			HeaderSpanID,
		}

		for _, h := range headers {
			if h == "" {
				t.Error("Expected header constant to be non-empty")
			}

			if !strings.HasPrefix(h, "X-NexusAI") {
				t.Errorf("Expected header to start with X-NexusAI, got %s", h)
			}
		}
	})
}
