package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCircuitBreakerMiddlewareCreation(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)
	if middleware == nil {
		t.Error("Expected non-nil middleware")
	}
}

func TestCircuitBreakerMiddlewareAllowed(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestCircuitBreakerMiddlewareRecordsSuccess(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make several successful requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func TestCircuitBreakerMiddlewareRecordsFailure(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	// Make several failing requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func TestCircuitBreakerMiddlewareTimeout(t *testing.T) {
	middleware := NewCircuitBreakerMiddlewareWithTimeout(5, time.Minute, 5*time.Second)

	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestCircuitBreakerMiddlewareGetCircuitState(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	state := middleware.GetCircuitState()
	if state != "closed" && state != "open" && state != "half-open" {
		t.Errorf("Invalid circuit state: %s", state)
	}
}

func TestCircuitBreakerMiddlewareGetStats(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	// Make some requests
	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	stats := middleware.GetStats()
	if stats.Requests < 5 {
		t.Errorf("Expected at least 5 requests, got %d", stats.Requests)
	}
}

func TestCircuitBreakerMiddlewareReset(t *testing.T) {
	middleware := NewCircuitBreakerMiddleware(5, time.Minute)

	// Make some requests
	handler := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	// Reset
	middleware.Reset()

	stats := middleware.GetStats()
	if stats.Requests != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", stats.Requests)
	}
}
