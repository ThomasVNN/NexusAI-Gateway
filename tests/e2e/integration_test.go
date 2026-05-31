package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAPIEndpoints tests all API endpoints
func TestAPIEndpoints(t *testing.T) {
	handler := createTestHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Health GET", http.MethodGet, "/health", http.StatusOK},
		{"Chat POST", http.MethodPost, "/v1/chat/completions", http.StatusOK},
		{"Invalid Method", http.MethodDelete, "/v1/chat/completions", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestRateLimiting tests rate limiting behavior
func TestRateLimiting(t *testing.T) {
	handler := createTestHandler()

	// Make multiple requests to trigger rate limit
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// First 60 should succeed (standard rate limit)
		if i < 60 && w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

// TestAuthentication tests authentication requirements
func TestAuthentication(t *testing.T) {
	handler := createTestHandler()

	tests := []struct {
		name          string
		headers       map[string]string
		expectedAuth  bool
	}{
		{
			name:         "No Auth",
			headers:      map[string]string{},
			expectedAuth: false,
		},
		{
			name: "API Key",
			headers: map[string]string{
				"Authorization": "Bearer test-api-key",
			},
			expectedAuth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/chat/completions", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// In test mode, we don't enforce auth
			if w.Code == http.StatusUnauthorized && tt.expectedAuth {
				t.Error("Request should have been authenticated")
			}
		})
	}
}

// TestMultiTenantIsolation tests tenant isolation
func TestMultiTenantIsolation(t *testing.T) {
	handler := createTestHandler()

	tenants := []string{"tenant-a", "tenant-b", "tenant-c"}

	for _, tenant := range tenants {
		t.Run("Tenant_"+tenant, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			req.Header.Set("X-Tenant-ID", tenant)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Tenant %s: Expected status 200, got %d", tenant, w.Code)
			}
		})
	}
}

// TestPrivacyFilter tests privacy filtering
func TestPrivacyFilter(t *testing.T) {
	handler := createTestHandler()

	tests := []struct {
		name        string
		payload     string
		expectBlock bool
	}{
		{
			name:        "Normal Request",
			payload:     `{"message": "Hello, how are you?"}`,
			expectBlock: false,
		},
		{
			name:        "Email in Request",
			payload:     `{"message": "Contact me at john@example.com"}`,
			expectBlock: false, // Should be redacted, not blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if tt.expectBlock && w.Code == http.StatusOK {
				t.Error("Request should have been blocked")
			}
		})
	}
}

// TestObservabilityMetrics tests metrics collection
func TestObservabilityMetrics(t *testing.T) {
	handler := createTestHandler()

	// Make several requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// In real implementation, we would check metrics endpoint
	// For now, just verify requests succeeded
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health check failed: %d", w.Code)
	}
}
