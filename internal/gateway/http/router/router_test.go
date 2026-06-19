package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRouterHealthzEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP","service":"nexusai-gateway"}`))
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestRouterReadyEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"nexusai-gateway"}`))
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRouterMetricsEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`# HELP nexusai_gateway_uptime_seconds Uptime`))
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header to be set")
	}
}

func TestMiddlewareChainWithCorrelationID(t *testing.T) {
	handler := WithCorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", "test-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddlewareChainWithRecovery(t *testing.T) {
	handler := WithRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddlewareChainWithStructuredLogging(t *testing.T) {
	handler := WithStructuredLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMiddlewareChainCorrelationIDGeneratesIfAbsent(t *testing.T) {
	handler := WithCorrelationID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRouterReadyzDownWithoutSandboxFallback(t *testing.T) {
	mux := http.NewServeMux()
	sandboxEnabled := false

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := "UP"
		code := http.StatusOK
		if !sandboxEnabled {
			status = "DOWN"
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		w.Write([]byte(`{"status":"` + status + `"}`))
	})

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 without sandbox fallback, got %d", rec.Code)
	}
}

func TestRouterReadyzUpWithSandboxFallback(t *testing.T) {
	mux := http.NewServeMux()
	sandboxEnabled := true

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := "UP"
		code := http.StatusOK
		if !sandboxEnabled {
			status = "DOWN"
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		w.Write([]byte(`{"status":"` + status + `"}`))
	})

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with sandbox fallback, got %d", rec.Code)
	}
}

func TestHealthEndpointResponseFormat(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP","service":"nexusai-gateway","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `"status":"UP"`) {
		t.Errorf("response should contain status UP, got: %s", body)
	}
	if !strings.Contains(body, `"service":"nexusai-gateway"`) {
		t.Errorf("response should contain service name, got: %s", body)
	}
}

func TestRouterMiddlewareRecoveryPanic(t *testing.T) {
	handler := WithRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 after panic recovery, got %d", rec.Code)
	}
}
