package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/eventbus"
)

// MockEventBus implements eventbus.Bus for testing
type MockEventBus struct {
	publishedEvents   []*eventbus.Event
	subscriptions     map[string]*eventbus.Subscription
	dlqEntries        map[string]*eventbus.DLQEntry
	healthError       error
	publishError      error
	subscribeError    error
	unsubscribeError  error
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		publishedEvents: make([]*eventbus.Event, 0),
		subscriptions:   make(map[string]*eventbus.Subscription),
		dlqEntries:      make(map[string]*eventbus.DLQEntry),
	}
}

func (m *MockEventBus) Publish(ctx context.Context, event *eventbus.Event) error {
	if m.publishError != nil {
		return m.publishError
	}
	event.ID = "test-event-id"
	m.publishedEvents = append(m.publishedEvents, event)
	return nil
}

func (m *MockEventBus) Subscribe(ctx context.Context, sub *eventbus.Subscription) error {
	if m.subscribeError != nil {
		return m.subscribeError
	}
	if sub.ID == "" {
		sub.ID = "test-sub-id"
	}
	m.subscriptions[sub.ID] = sub
	return nil
}

func (m *MockEventBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	if m.unsubscribeError != nil {
		return m.unsubscribeError
	}
	delete(m.subscriptions, subscriptionID)
	return nil
}

func (m *MockEventBus) GetSubscriptions(ctx context.Context, clientID string) ([]*eventbus.Subscription, error) {
	var subs []*eventbus.Subscription
	for _, sub := range m.subscriptions {
		if sub.ClientID == clientID {
			subs = append(subs, sub)
		}
	}
	return subs, nil
}

func (m *MockEventBus) GetDLQEntries(ctx context.Context, status eventbus.DLQStatus, limit int) ([]*eventbus.DLQEntry, error) {
	var entries []*eventbus.DLQEntry
	for _, entry := range m.dlqEntries {
		if status == "" || entry.Status == status {
			entries = append(entries, entry)
			if limit > 0 && len(entries) >= limit {
				break
			}
		}
	}
	return entries, nil
}

func (m *MockEventBus) RetryDLQEntry(ctx context.Context, entryID string) error {
	delete(m.dlqEntries, entryID)
	return nil
}

func (m *MockEventBus) PurgeDLQEntry(ctx context.Context, entryID string) error {
	delete(m.dlqEntries, entryID)
	return nil
}

func (m *MockEventBus) HealthCheck(ctx context.Context) error {
	return m.healthError
}

func (m *MockEventBus) Close() error {
	return nil
}

func TestEventHandlerPublish(t *testing.T) {
	t.Run("publishes valid event", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		payload := map[string]interface{}{
			"action": "create",
			"resource": "/users",
		}
		payloadBytes, _ := json.Marshal(payload)

		reqBody := PublishRequest{
			Type:        eventbus.EventTypeIntent,
			SourceAgent: "agent-1",
			Payload:     payloadBytes,
			Priority:    eventbus.PriorityHigh,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
		}

		var resp PublishResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if !resp.Success {
			t.Error("expected success")
		}

		if len(mockBus.publishedEvents) != 1 {
			t.Errorf("expected 1 published event, got %d", len(mockBus.publishedEvents))
		}
	})

	t.Run("returns error for missing type", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := PublishRequest{
			SourceAgent: "agent-1",
			Payload:     json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("returns error for missing source_agent", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := PublishRequest{
			Type:    eventbus.EventTypeIntent,
			Payload: json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("returns error for publish failure", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.publishError = fmt.Errorf("publish failed")
		handler := NewEventHandler(mockBus)

		reqBody := PublishRequest{
			Type:        eventbus.EventTypeIntent,
			SourceAgent: "agent-1",
			Payload:     json.RawMessage(`{}`),
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestEventHandlerSubscribe(t *testing.T) {
	t.Run("creates valid subscription", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := SubscribeRequest{
			ClientID:   "client-1",
			EventTypes: []eventbus.EventType{eventbus.EventTypeIntent, eventbus.EventTypeDecision},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events/subscribe", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Subscribe(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}

		var resp SubscribeResponse
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if !resp.Success {
			t.Error("expected success")
		}

		if resp.SubscriptionID == "" {
			t.Error("expected subscription ID")
		}
	})

	t.Run("returns error for missing client_id", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := SubscribeRequest{
			EventTypes: []eventbus.EventType{eventbus.EventTypeIntent},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events/subscribe", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Subscribe(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("returns error for missing event_types", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := SubscribeRequest{
			ClientID: "client-1",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events/subscribe", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Subscribe(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestEventHandlerUnsubscribe(t *testing.T) {
	t.Run("removes subscription", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.subscriptions["sub-123"] = &eventbus.Subscription{
			ID:       "sub-123",
			ClientID: "client-1",
		}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodDelete, "/v1/events/subscribe/sub-123", nil)
		rec := httptest.NewRecorder()

		handler.Unsubscribe(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if _, exists := mockBus.subscriptions["sub-123"]; exists {
			t.Error("expected subscription to be removed")
		}
	})

	t.Run("returns 404 for unknown subscription", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodDelete, "/v1/events/subscribe/unknown", nil)
		rec := httptest.NewRecorder()

		handler.Unsubscribe(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestEventHandlerGetSubscriptions(t *testing.T) {
	t.Run("returns client subscriptions", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.subscriptions["sub-1"] = &eventbus.Subscription{
			ID:        "sub-1",
			ClientID:  "client-1",
			EventTypes: []eventbus.EventType{eventbus.EventTypeIntent},
		}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/subscriptions?client_id=client-1", nil)
		rec := httptest.NewRecorder()

		handler.GetSubscriptions(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if !resp["success"].(bool) {
			t.Error("expected success")
		}
	})

	t.Run("returns error for missing client_id", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/subscriptions", nil)
		rec := httptest.NewRecorder()

		handler.GetSubscriptions(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestEventHandlerGetDLQEntries(t *testing.T) {
	t.Run("returns DLQ entries", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.dlqEntries["entry-1"] = &eventbus.DLQEntry{
			ID:     "entry-1",
			Status: eventbus.DLQStatusPending,
		}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/dlq", nil)
		rec := httptest.NewRecorder()

		handler.GetDLQEntries(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.dlqEntries["entry-1"] = &eventbus.DLQEntry{
			ID:     "entry-1",
			Status: eventbus.DLQStatusPending,
		}
		mockBus.dlqEntries["entry-2"] = &eventbus.DLQEntry{
			ID:     "entry-2",
			Status: eventbus.DLQStatusDead,
		}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/dlq?status=pending", nil)
		rec := httptest.NewRecorder()

		handler.GetDLQEntries(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestEventHandlerRetryDLQEntry(t *testing.T) {
	t.Run("retries DLQ entry", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.dlqEntries["entry-1"] = &eventbus.DLQEntry{
			ID:        "entry-1",
			RetryCount: 0,
			MaxRetries: 3,
		}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodPost, "/v1/events/dlq/entry-1/retry", nil)
		rec := httptest.NewRecorder()

		handler.RetryDLQEntry(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestEventHandlerPurgeDLQEntry(t *testing.T) {
	t.Run("purges DLQ entry", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.dlqEntries["entry-1"] = &eventbus.DLQEntry{ID: "entry-1"}
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodDelete, "/v1/events/dlq/entry-1", nil)
		rec := httptest.NewRecorder()

		handler.PurgeDLQEntry(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if _, exists := mockBus.dlqEntries["entry-1"]; exists {
			t.Error("expected entry to be purged")
		}
	})

	t.Run("returns 404 for unknown entry", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodDelete, "/v1/events/dlq/unknown", nil)
		rec := httptest.NewRecorder()

		handler.PurgeDLQEntry(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestEventHandlerHealthCheck(t *testing.T) {
	t.Run("returns UP when healthy", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/health", nil)
		rec := httptest.NewRecorder()

		handler.HealthCheck(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp["status"] != "UP" {
			t.Errorf("expected status UP, got %s", resp["status"])
		}
	})

	t.Run("returns DOWN when unhealthy", func(t *testing.T) {
		mockBus := NewMockEventBus()
		mockBus.healthError = fmt.Errorf("connection lost")
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/health", nil)
		rec := httptest.NewRecorder()

		handler.HealthCheck(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}

		var resp map[string]string
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp["status"] != "DOWN" {
			t.Errorf("expected status DOWN, got %s", resp["status"])
		}
	})
}

func TestPublishRequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         PublishRequest
		expectError bool
	}{
		{
			name: "valid request",
			req: PublishRequest{
				Type:        eventbus.EventTypeIntent,
				SourceAgent: "agent-1",
				Payload:     json.RawMessage(`{}`),
			},
			expectError: false,
		},
		{
			name: "missing type",
			req: PublishRequest{
				SourceAgent: "agent-1",
				Payload:     json.RawMessage(`{}`),
			},
			expectError: true,
		},
		{
			name: "missing source",
			req: PublishRequest{
				Type:    eventbus.EventTypeIntent,
				Payload: json.RawMessage(`{}`),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBus := NewMockEventBus()
			handler := NewEventHandler(mockBus)

			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Publish(rec, req)

			if tt.expectError && rec.Code == http.StatusAccepted {
				t.Error("expected error but got success")
			}
			if !tt.expectError && rec.Code != http.StatusAccepted {
				t.Errorf("expected success but got error: %s", rec.Body.String())
			}
		})
	}
}

func TestSubscribeRequestValidation(t *testing.T) {
	tests := []struct {
		name        string
		req         SubscribeRequest
		expectError bool
	}{
		{
			name: "valid request",
			req: SubscribeRequest{
				ClientID:   "client-1",
				EventTypes: []eventbus.EventType{eventbus.EventTypeIntent},
			},
			expectError: false,
		},
		{
			name: "missing client_id",
			req: SubscribeRequest{
				EventTypes: []eventbus.EventType{eventbus.EventTypeIntent},
			},
			expectError: true,
		},
		{
			name: "missing event_types",
			req: SubscribeRequest{
				ClientID: "client-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBus := NewMockEventBus()
			handler := NewEventHandler(mockBus)

			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest(http.MethodPost, "/v1/events/subscribe", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Subscribe(rec, req)

			if tt.expectError && rec.Code == http.StatusCreated {
				t.Error("expected error but got success")
			}
			if !tt.expectError && rec.Code != http.StatusCreated {
				t.Errorf("expected success but got error: %s", rec.Body.String())
			}
		})
	}
}

func TestHTTPMethodValidation(t *testing.T) {
	t.Run("Publish rejects non-POST", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("Subscribe rejects non-POST", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodGet, "/v1/events/subscribe", nil)
		rec := httptest.NewRecorder()

		handler.Subscribe(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("Unsubscribe rejects non-DELETE", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		req := httptest.NewRequest(http.MethodPost, "/v1/events/subscribe/test", nil)
		rec := httptest.NewRecorder()

		handler.Unsubscribe(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})
}

func TestExtractPathParam(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"/v1/events/dlq/entry-123", "/v1/events/dlq/", "entry-123"},
		{"/v1/events/subscribe/sub-456", "/v1/events/subscribe/", "sub-456"},
		{"short", "/v1/events/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractPathParam(tt.path, tt.prefix)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEventHandlerWithCorrelationAndMetadata(t *testing.T) {
	t.Run("publishes event with correlation_id and metadata", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := PublishRequest{
			Type:          eventbus.EventTypeIntent,
			SourceAgent:   "agent-1",
			Payload:       json.RawMessage(`{"action": "test"}`),
			TargetAgent:   "agent-2",
			CorrelationID: "corr-123",
			Metadata:      map[string]interface{}{"key1": "value1", "key2": 42},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
		}

		if len(mockBus.publishedEvents) != 1 {
			t.Fatalf("expected 1 event, got %d", len(mockBus.publishedEvents))
		}

		event := mockBus.publishedEvents[0]
		if event.TargetAgent != "agent-2" {
			t.Errorf("expected target agent agent-2, got %s", event.TargetAgent)
		}
		if event.CorrelationID != "corr-123" {
			t.Errorf("expected correlation ID corr-123, got %s", event.CorrelationID)
		}
	})
}

func TestEventHandlerWithTTL(t *testing.T) {
	t.Run("handles TTL as string duration", func(t *testing.T) {
		mockBus := NewMockEventBus()
		handler := NewEventHandler(mockBus)

		reqBody := PublishRequest{
			Type:        eventbus.EventTypeIntent,
			SourceAgent: "agent-1",
			Payload:     json.RawMessage(`{}`),
			TTL:         "5m",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.Publish(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status %d, got %d", http.StatusAccepted, rec.Code)
		}
	})
}
