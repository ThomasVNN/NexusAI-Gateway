package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
)

// sandboxTestKey is a well-known test key that sandbox mode should accept
const sandboxTestKey = "test-key-1"

// newTestRouter builds a router with nil DB (sandbox fallback mode) for route
// registration validation. No live database or upstream services are required.
func newTestRouter() http.Handler {
	cfg := &config.Config{
		Port:                  "0",
		AppEnv:                "development",
		EnableSandboxFallback: true,
		InitialPassword:       "test-password",
	}
	return New(nil, cfg)
}

// TestChatCompletionRouteRegistered verifies that POST /v1/chat/completions is
// routed to the runtime chat handler and does NOT return 404.
func TestChatCompletionRouteRegistered(t *testing.T) {
	handler := newTestRouter()

	payload := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sandboxTestKey)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code == http.StatusNotFound {
		t.Fatalf("POST /v1/chat/completions returned 404; route is not registered. Body: %s", rr.Body.String())
	}

	// Route is registered - verify we get a valid response.
	// In sandbox fallback mode, responses can be 200 (success), 401 (auth error in pipeline),
	// or 500 (pipeline error) depending on configuration.
	if rr.Code == http.StatusOK {
		var resp map[string]interface{}
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response JSON: %v", err)
		}

		if success, ok := resp["success"].(bool); !ok || !success {
			t.Errorf("expected success=true in response, got %v", resp["success"])
		}

		if resp["data"] == nil {
			t.Errorf("expected data field in response, got nil")
		}
	}

	t.Logf("POST /v1/chat/completions returned status %d with runtime pipeline response", rr.Code)
}

// TestChatCompletionRejectsNonPostMethods verifies that non-POST methods on
// /v1/chat/completions do not reach the chat handler. With Go 1.22+
// method-aware routing, unmatched methods fall through to the static catch-all.
func TestChatCompletionRejectsNonPostMethods(t *testing.T) {
	handler := newTestRouter()

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/v1/chat/completions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// Non-POST methods must NOT return 200 from the chat handler
		if rr.Code == http.StatusOK {
			ct := rr.Header().Get("Content-Type")
			if ct == "application/json" {
				t.Errorf("%s /v1/chat/completions unexpectedly returned 200 with JSON; chat handler should not serve non-POST", method)
			}
		}
	}
}

// TestModelsRouteRegistered verifies that GET /v1/models is routed correctly.
func TestModelsRouteRegistered(t *testing.T) {
	handler := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code == http.StatusNotFound {
		t.Fatalf("GET /v1/models returned 404; route is not registered. Body: %s", rr.Body.String())
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /v1/models expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

// TestHealthzRouteRegistered verifies the health check endpoint responds.
func TestHealthzRouteRegistered(t *testing.T) {
	handler := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /healthz expected 200, got %d", rr.Code)
	}
}
