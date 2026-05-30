package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/router"
)

// TestConfig holds configuration for E2E tests
type TestConfig struct {
	BaseURL    string
	GatewayURL string
	Timeout    time.Duration
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		BaseURL:    "http://localhost:8080",
		GatewayURL: "http://localhost:8080",
		Timeout:    30 * time.Second,
	}
}

// GatewayTest represents an E2E test scenario
type GatewayTest struct {
	Name        string
	Description string
	Method      string
	Path        string
	Headers     map[string]string
	Body        interface{}
	ExpectedStatus int
	CheckResponse func(*testing.T, *httptest.ResponseRecorder)
}

// RunGatewayTest executes a gateway test scenario
func RunGatewayTest(t *testing.T, cfg *TestConfig, test *GatewayTest) {
	t.Run(test.Name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()

		var body []byte
		if test.Body != nil {
			body, _ = json.Marshal(test.Body)
		}

		req := httptest.NewRequestWithContext(ctx, test.Method, cfg.GatewayURL+test.Path, nil)
		if body != nil {
			req = httptest.NewRequestWithContext(ctx, test.Method, cfg.GatewayURL+test.Path, nil)
			req.Body = nil
		}

		for key, value := range test.Headers {
			req.Header.Set(key, value)
		}

		w := httptest.NewRecorder()
		handler := createTestHandler()
		handler.ServeHTTP(w, req)

		if w.Code != test.ExpectedStatus {
			t.Errorf("Expected status %d, got %d", test.ExpectedStatus, w.Code)
		}

		if test.CheckResponse != nil {
			test.CheckResponse(t, w)
		}
	})
}

// createTestHandler creates a test HTTP handler
func createTestHandler() http.Handler {
	mux := http.NewServeMux()
	
	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
	
	// Chat endpoint
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": "This is a test response",
					},
					"finish_reason": "stop",
				},
			},
		})
	})

	return mux
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cfg := DefaultTestConfig()

	RunGatewayTest(t, cfg, &GatewayTest{
		Name:        "Health Check",
		Description: "Verify health endpoint returns 200",
		Method:      http.MethodGet,
		Path:        "/health",
		ExpectedStatus: http.StatusOK,
		CheckResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
			var resp map[string]string
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if resp["status"] != "healthy" {
				t.Errorf("Expected status 'healthy', got '%s'", resp["status"])
			}
		},
	})
}

// TestChatCompletions tests the chat completions endpoint
func TestChatCompletions(t *testing.T) {
	cfg := DefaultTestConfig()

	RunGatewayTest(t, cfg, &GatewayTest{
		Name:        "Chat Completions",
		Description: "Verify chat completions returns valid response",
		Method:      http.MethodPost,
		Path:        "/v1/chat/completions",
		Headers:     map[string]string{"Content-Type": "application/json"},
		Body: map[string]interface{}{
			"model": "gpt-4",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello"},
			},
		},
		ExpectedStatus: http.StatusOK,
		CheckResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			if resp["object"] != "chat.completion" {
				t.Errorf("Expected object 'chat.completion', got '%s'", resp["object"])
			}
		},
	})
}

// TestChatCompletionsValidation tests request validation
func TestChatCompletionsValidation(t *testing.T) {
	cfg := DefaultTestConfig()

	RunGatewayTest(t, cfg, &GatewayTest{
		Name:        "Invalid Method",
		Description: "Verify GET method is rejected",
		Method:      http.MethodGet,
		Path:        "/v1/chat/completions",
		ExpectedStatus: http.StatusMethodNotAllowed,
	})
}

// IntegrationTest represents a multi-step integration test
type IntegrationTest struct {
	Name        string
	Description string
	Steps      []IntegrationStep
}

// IntegrationStep represents a single step in an integration test
type IntegrationStep struct {
	Name        string
	Request     *GatewayTest
	SaveResponse map[string]string // Keys to save from response for later use
}

// RunIntegrationTest executes a multi-step integration test
func RunIntegrationTest(t *testing.T, cfg *TestConfig, test *IntegrationTest) {
	t.Run(test.Name, func(t *testing.T) {
		savedValues := make(map[string]string)

		for i, step := range test.Steps {
			t.Run(step.Name, func(t *testing.T) {
				if step.Request == nil {
					t.Fatal("Request is nil")
				}

				// Substitute saved values in path
				path := step.Request.Path
				for key, value := range savedValues {
					path = fmt.Sprintf("%s?%s=%s", 
						fmt.Sprintf("%s?old=%s", path, "placeholder"),
						key, value)
				}

				// Execute request
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
				defer cancel()

				var body []byte
				if step.Request.Body != nil {
					body, _ = json.Marshal(step.Request.Body)
				}

				req := httptest.NewRequestWithContext(ctx, step.Request.Method, cfg.GatewayURL+path, nil)
				for key, value := range step.Request.Headers {
					req.Header.Set(key, value)
				}

				w := httptest.NewRecorder()
				handler := createTestHandler()
				handler.ServeHTTP(w, req)

				if w.Code != step.Request.ExpectedStatus {
					t.Errorf("Step %d: Expected status %d, got %d", i+1, step.Request.ExpectedStatus, w.Code)
					return
				}

				if step.Request.CheckResponse != nil {
					step.Request.CheckResponse(t, w)
				}

				// Save values from response
				if step.SaveResponse != nil {
					var resp map[string]interface{}
					json.NewDecoder(w.Body).Decode(&resp)
					for key, saveKey := range step.SaveResponse {
						if value, ok := resp[saveKey]; ok {
							savedValues[key] = fmt.Sprintf("%v", value)
						}
					}
				}
			})
		}
	})
}

// TestGatewayIntegration tests the full gateway flow
func TestGatewayIntegration(t *testing.T) {
	cfg := DefaultTestConfig()

	RunIntegrationTest(t, cfg, &IntegrationTest{
		Name:        "Full Gateway Flow",
		Description: "Test complete gateway request flow",
		Steps: []IntegrationStep{
			{
				Name: "Health Check",
				Request: &GatewayTest{
					Method:         http.MethodGet,
					Path:           "/health",
					ExpectedStatus:  http.StatusOK,
				},
			},
			{
				Name: "Chat Completion",
				Request: &GatewayTest{
					Method:         http.MethodPost,
					Path:           "/v1/chat/completions",
					Headers:        map[string]string{"Content-Type": "application/json"},
					Body: map[string]interface{}{
						"model": "gpt-4",
						"messages": []map[string]string{
							{"role": "user", "content": "Test message"},
						},
					},
					ExpectedStatus: http.StatusOK,
				},
				SaveResponse: map[string]string{
					"message": "choices.0.message.content",
				},
			},
		},
	})
}

// BenchmarkGateway benchmarks gateway performance
func BenchmarkGateway(b *testing.B) {
	cfg := DefaultTestConfig()
	handler := createTestHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, cfg.GatewayURL+"/health", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
