package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		want      bool
	}{
		{
			name:      "valid signature",
			payload:   []byte(`{"test":"data"}`),
			signature: computeSignature([]byte(`{"test":"data"}`), "secret123"),
			secret:    "secret123",
			want:      true,
		},
		{
			name:      "invalid signature",
			payload:   []byte(`{"test":"data"}`),
			signature: "sha256=invalid",
			secret:    "secret123",
			want:      false,
		},
		{
			name:      "wrong secret",
			payload:   []byte(`{"test":"data"}`),
			signature: computeSignature([]byte(`{"test":"data"}`), "wrongsecret"),
			secret:    "secret123",
			want:      false,
		},
		{
			name:      "empty signature",
			payload:   []byte(`{"test":"data"}`),
			signature: "",
			secret:    "secret123",
			want:      false,
		},
		{
			name:      "empty secret",
			payload:   []byte(`{"test":"data"}`),
			signature: "sha256=abc",
			secret:    "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifySignature(tt.payload, tt.signature, tt.secret)
			assert.Equal(t, tt.want, got)
		})
	}
}

func computeSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestNew(t *testing.T) {
	cfg := DefaultConfig()
	receiver := New(cfg)

	require.NotNil(t, receiver)
	assert.NotNil(t, receiver.httpClient)
	assert.Equal(t, cfg.MaxRetries, receiver.config.MaxRetries)
}

func TestRegisterEndpoint(t *testing.T) {
	receiver := New(DefaultConfig())

	endpoint := &WebhookEndpoint{
		URL:    "https://example.com/webhook",
		Secret: "secret123",
		Events: []EventType{EventMessageSent},
		Active: true,
	}

	receiver.RegisterEndpoint(endpoint)

	// Verify endpoint was registered with generated ID
	assert.NotEmpty(t, endpoint.ID)
	assert.False(t, endpoint.CreatedAt.IsZero())

	// Verify we can retrieve it
	retrieved := receiver.GetEndpoint(endpoint.ID)
	require.NotNil(t, retrieved)
	assert.Equal(t, endpoint.URL, retrieved.URL)
	assert.Equal(t, endpoint.Secret, retrieved.Secret)
}

func TestUnregisterEndpoint(t *testing.T) {
	receiver := New(DefaultConfig())

	endpoint := &WebhookEndpoint{
		URL:    "https://example.com/webhook",
		Secret: "secret123",
		Events: []EventType{EventMessageSent},
		Active: true,
	}

	receiver.RegisterEndpoint(endpoint)
	receiver.UnregisterEndpoint(endpoint.ID)

	retrieved := receiver.GetEndpoint(endpoint.ID)
	assert.Nil(t, retrieved)
}

func TestListEndpoints(t *testing.T) {
	receiver := New(DefaultConfig())

	// Register multiple endpoints
	for i := range 3 {
		receiver.RegisterEndpoint(&WebhookEndpoint{
			URL:    fmt.Sprintf("https://example.com/webhook%d", i),
			Secret: fmt.Sprintf("secret%d", i),
			Events: []EventType{EventMessageSent},
			Active: true,
		})
	}

	endpoints := receiver.ListEndpoints()
	assert.Len(t, endpoints, 3)
}

func TestHandleWebhook_ValidRequest(t *testing.T) {
	receiver := New(DefaultConfig())

	// Register an endpoint
	endpoint := &WebhookEndpoint{
		URL:    "https://example.com/webhook",
		Secret: "secret123",
		Events: []EventType{EventMessageSent, EventMessageReceived},
		Active: true,
	}
	receiver.RegisterEndpoint(endpoint)

	event := WebhookEvent{
		ID:        "evt-123",
		Type:      EventMessageSent,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"message_id": "msg-456"},
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleWebhook(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "accepted", resp["status"])
	assert.NotEmpty(t, resp["event_id"])
}

func TestHandleWebhook_InvalidMethod(t *testing.T) {
	receiver := New(DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/api/webhooks/receive", nil)
	w := httptest.NewRecorder()

	receiver.HandleWebhook(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	receiver := New(DefaultConfig())

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/receive", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleWebhook_InvalidEventType(t *testing.T) {
	receiver := New(DefaultConfig())

	event := map[string]interface{}{
		"id":        "evt-123",
		"type":      "invalid.event",
		"timestamp": time.Now().Format(time.RFC3339),
		"data":      map[string]interface{}{},
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/receive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	receiver.HandleWebhook(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGenerateEvent(t *testing.T) {
	data := map[string]interface{}{
		"user_id": "user-123",
		"name":    "Test User",
	}

	event := GenerateEvent(EventUserCreated, data)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventUserCreated, event.Type)
	assert.False(t, event.Timestamp.IsZero())
	assert.Equal(t, data["user_id"], event.Data["user_id"])
	assert.Equal(t, data["name"], event.Data["name"])
}

func TestIsValidEventType(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      bool
	}{
		{EventMessageSent, true},
		{EventMessageReceived, true},
		{EventUserCreated, true},
		{EventUserDeleted, true},
		{EventType("invalid"), false},
		{EventType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			got := isValidEventType(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	receiver := New(Config{
		MaxRetries: 5,
		BaseDelay:  time.Second,
	})

	tests := []struct {
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{0, time.Second, time.Second * 2},
		{1, time.Second * 2, time.Second * 4},
		{2, time.Second * 4, time.Second * 8},
		{3, time.Second * 8, time.Minute}, // Should cap at 5 minutes
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := receiver.calculateBackoff(tt.attempt)
			assert.GreaterOrEqual(t, got, tt.wantMin)
			assert.Less(t, got, tt.wantMax)
		})
	}
}

func TestDeliver_Success(t *testing.T) {
	receiver := New(DefaultConfig())

	// Create a test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Event"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Event-ID"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Delivery"))
		assert.NotEmpty(t, r.Header.Get("X-Webhook-Attempt"))
		assert.Contains(t, r.Header.Get("X-Webhook-Signature-256"), "sha256=")

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint := &WebhookEndpoint{
		ID:     "ep-123",
		URL:    server.URL,
		Secret: "secret123",
		Events: []EventType{EventMessageSent},
		Active: true,
	}

	event := GenerateEvent(EventMessageSent, map[string]interface{}{"test": "data"})

	delivery := receiver.deliver(context.Background(), endpoint, event, 0)

	assert.Equal(t, http.StatusOK, delivery.StatusCode)
	assert.Nil(t, delivery.Error)
	assert.Equal(t, 1, delivery.Attempt)
	assert.NotEmpty(t, delivery.ID)
}

func TestDeliver_RetryOnFailure(t *testing.T) {
	receiver := New(Config{
		MaxRetries: 3,
		BaseDelay:  10 * time.Millisecond, // Short delay for tests
		Timeout:    time.Second,
	})

	attemptCount := 0
	var mu sync.Mutex

	// Create a test server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attemptCount++
		count := attemptCount
		mu.Unlock()

		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	endpoint := &WebhookEndpoint{
		ID:     "ep-123",
		URL:    server.URL,
		Secret: "secret123",
		Events: []EventType{EventMessageSent},
		Active: true,
	}

	event := GenerateEvent(EventMessageSent, map[string]interface{}{"test": "data"})

	// Deliver with retry
	receiver.deliverWithRetry(context.Background(), endpoint, event)

	mu.Lock()
	assert.GreaterOrEqual(t, attemptCount, 3) // Should have retried
	mu.Unlock()
}

func TestDispatch_ToMatchingEndpoints(t *testing.T) {
	receiver := New(DefaultConfig())

	// Register endpoint A that listens to message events
	endpointA := &WebhookEndpoint{
		ID:     "ep-a",
		URL:    "https://example.com/webhook-a",
		Secret: "secret-a",
		Events: []EventType{EventMessageSent},
		Active: true,
	}
	receiver.RegisterEndpoint(endpointA)

	// Register endpoint B that listens to user events
	endpointB := &WebhookEndpoint{
		ID:     "ep-b",
		URL:    "https://example.com/webhook-b",
		Secret: "secret-b",
		Events: []EventType{EventUserCreated},
		Active: true,
	}
	receiver.RegisterEndpoint(endpointB)

	// Register inactive endpoint
	endpointC := &WebhookEndpoint{
		ID:     "ep-c",
		URL:    "https://example.com/webhook-c",
		Secret: "secret-c",
		Events: []EventType{EventMessageSent},
		Active: false,
	}
	receiver.RegisterEndpoint(endpointC)

	// Verify only A and B are returned for their respective events
	messageEndpoints := receiver.getActiveEndpointsForEvent(EventMessageSent)
	userEndpoints := receiver.getActiveEndpointsForEvent(EventUserCreated)

	assert.Len(t, messageEndpoints, 1)
	assert.Equal(t, "ep-a", messageEndpoints[0].ID)

	assert.Len(t, userEndpoints, 1)
	assert.Equal(t, "ep-b", userEndpoints[0].ID)
}

func TestWebhookEvent_JSON(t *testing.T) {
	event := WebhookEvent{
		ID:        "evt-123",
		Type:      EventMessageSent,
		Timestamp: time.Date(2026, 6, 24, 10, 30, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"message_id": "msg-456",
			"user_id":    "user-789",
		},
	}

	// Marshal
	data, err := json.Marshal(event)
	require.NoError(t, err)

	// Unmarshal
	var decoded WebhookEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.ID, decoded.ID)
	assert.Equal(t, event.Type, decoded.Type)
	assert.Equal(t, event.Data["message_id"], decoded.Data["message_id"])
}

func TestWebhookEndpoint_Registration(t *testing.T) {
	receiver := New(DefaultConfig())

	// Test that duplicate IDs are handled
	endpoint1 := &WebhookEndpoint{
		ID:  "same-id",
		URL: "https://example.com/1",
		Events: []EventType{EventMessageSent},
		Active: true,
	}

	endpoint2 := &WebhookEndpoint{
		ID:  "same-id",
		URL: "https://example.com/2",
		Events: []EventType{EventUserCreated},
		Active: true,
	}

	receiver.RegisterEndpoint(endpoint1)
	receiver.RegisterEndpoint(endpoint2)

	// Second registration should overwrite
	retrieved := receiver.GetEndpoint("same-id")
	require.NotNil(t, retrieved)
	assert.Equal(t, "https://example.com/2", retrieved.URL)
}

func TestWebhookConfig_Validate(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "valid config",
			config: Config{MaxRetries: 5, BaseDelay: time.Second, Timeout: 10 * time.Second, BufferSize: 1000},
		},
		{
			name:   "default config",
			config: DefaultConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestStartDeliveryWorker(t *testing.T) {
	receiver := New(DefaultConfig())
	ctx, cancel := context.WithCancel(context.Background())

	// Start the worker
	receiver.StartDeliveryWorker(ctx)

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Cancel should stop the worker
	cancel()
}

func TestWebhookDelivery_Structure(t *testing.T) {
	delivery := WebhookDelivery{
		ID:         "del-123",
		EndpointID: "ep-456",
		Event: WebhookEvent{
			ID:   "evt-789",
			Type: EventMessageSent,
		},
		Attempt:     1,
		StatusCode:  200,
		Response:    `{"status":"ok"}`,
		DeliveredAt: time.Now(),
	}

	assert.Equal(t, "del-123", delivery.ID)
	assert.Equal(t, "ep-456", delivery.EndpointID)
	assert.Equal(t, 1, delivery.Attempt)
	assert.Equal(t, 200, delivery.StatusCode)
}

func TestGenerateEndpoint(t *testing.T) {
	endpoint := GenerateEndpoint("https://example.com/webhook", "secret", []EventType{EventMessageSent})

	assert.NotEmpty(t, endpoint.ID)
	assert.Equal(t, "https://example.com/webhook", endpoint.URL)
	assert.Equal(t, "secret", endpoint.Secret)
	assert.Equal(t, []EventType{EventMessageSent}, endpoint.Events)
	assert.True(t, endpoint.Active)
	assert.False(t, endpoint.CreatedAt.IsZero())
}

// GenerateEndpoint creates a new endpoint with generated ID
func GenerateEndpoint(url, secret string, events []EventType) *WebhookEndpoint {
	return &WebhookEndpoint{
		ID:        uuid.New().String(),
		URL:       url,
		Secret:    secret,
		Events:    events,
		Active:    true,
		CreatedAt: time.Now(),
	}
}

// Helper function to read request body
func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, fmt.Errorf("request body is nil")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	return body, nil
}
