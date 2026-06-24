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
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// VerifySignature verifies HMAC-SHA256 signature
func VerifySignature(payload []byte, signature, secret string) bool {
	if signature == "" || secret == "" {
		return false
	}
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
	
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// WebhookReceiver handles incoming webhook requests and delivery
type WebhookReceiver struct {
	config     Config
	endpoints  map[string]*WebhookEndpoint
	deliveries chan WebhookDelivery
	mu         sync.RWMutex
	httpClient *http.Client
}

// New creates a new WebhookReceiver
func New(config Config) *WebhookReceiver {
	if err := config.Validate(); err != nil {
		slog.Warn("webhook config validation failed, using defaults", slog.Any("error", err))
		config = DefaultConfig()
	}

	return &WebhookReceiver{
		config:    config,
		endpoints: make(map[string]*WebhookEndpoint),
		deliveries: make(chan WebhookDelivery, config.BufferSize),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// RegisterEndpoint adds a new webhook endpoint
func (r *WebhookReceiver) RegisterEndpoint(endpoint *WebhookEndpoint) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if endpoint.ID == "" {
		endpoint.ID = uuid.New().String()
	}
	if endpoint.CreatedAt.IsZero() {
		endpoint.CreatedAt = time.Now()
	}

	r.endpoints[endpoint.ID] = endpoint
	slog.Info("webhook endpoint registered",
		slog.String("endpoint_id", endpoint.ID),
		slog.String("url", endpoint.URL),
		slog.Any("events", endpoint.Events),
	)
}

// UnregisterEndpoint removes a webhook endpoint
func (r *WebhookReceiver) UnregisterEndpoint(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.endpoints[id]; exists {
		delete(r.endpoints, id)
		slog.Info("webhook endpoint unregistered", slog.String("endpoint_id", id))
	}
}

// GetEndpoint retrieves an endpoint by ID
func (r *WebhookReceiver) GetEndpoint(id string) *WebhookEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.endpoints[id]
}

// ListEndpoints returns all registered endpoints
func (r *WebhookReceiver) ListEndpoints() []*WebhookEndpoint {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoints := make([]*WebhookEndpoint, 0, len(r.endpoints))
	for _, ep := range r.endpoints {
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// HandleWebhook processes incoming webhook requests
func (r *WebhookReceiver) HandleWebhook(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the request body
	body, err := fmtRequestsBody(req)
	if err != nil {
		slog.Error("failed to read request body", slog.Any("error", err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Parse the webhook event
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("failed to parse webhook event", slog.Any("error", err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate event type
	if !isValidEventType(event.Type) {
		slog.Warn("invalid event type received", slog.String("type", string(event.Type)))
		http.Error(w, "Invalid event type", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	slog.Info("webhook received",
		slog.String("event_id", event.ID),
		slog.String("event_type", string(event.Type)),
	)

	// Dispatch to all matching endpoints
	go r.Dispatch(req.Context(), event)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"accepted","event_id":"%s"}`, event.ID)))
}

// Dispatch sends event to all matching endpoints with exponential backoff retry
func (r *WebhookReceiver) Dispatch(ctx context.Context, event WebhookEvent) {
	r.mu.RLock()
	endpoints := r.getActiveEndpointsForEvent(event.Type)
	r.mu.RUnlock()

	for _, endpoint := range endpoints {
		go r.deliverWithRetry(ctx, endpoint, event)
	}
}

// getActiveEndpointsForEvent returns all active endpoints subscribed to the event type
func (r *WebhookReceiver) getActiveEndpointsForEvent(eventType EventType) []*WebhookEndpoint {
	var matching []*WebhookEndpoint

	for _, ep := range r.endpoints {
		if !ep.Active {
			continue
		}
		for _, et := range ep.Events {
			if et == eventType {
				matching = append(matching, ep)
				break
			}
		}
	}

	return matching
}

// deliverWithRetry attempts to deliver an event with exponential backoff
func (r *WebhookReceiver) deliverWithRetry(ctx context.Context, endpoint *WebhookEndpoint, event WebhookEvent) {
	var lastErr error
	attempt := 0

	for attempt <= r.config.MaxRetries {
		delivery := r.deliver(ctx, endpoint, event, attempt)

		if delivery.StatusCode >= 200 && delivery.StatusCode < 300 {
			slog.Info("webhook delivered successfully",
				slog.String("endpoint_id", endpoint.ID),
				slog.String("event_id", event.ID),
				slog.Int("attempt", attempt+1),
			)
			return
		}

		lastErr = delivery.Error
		attempt++

		if attempt <= r.config.MaxRetries {
			delay := r.calculateBackoff(attempt)
			delivery.NextRetryAt = time.Now().Add(delay)

			slog.Warn("webhook delivery failed, retrying",
				slog.String("endpoint_id", endpoint.ID),
				slog.String("event_id", event.ID),
				slog.Int("attempt", attempt),
				slog.Int("status_code", delivery.StatusCode),
				slog.Duration("next_retry", delay),
				slog.Any("error", lastErr),
			)

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}

	slog.Error("webhook delivery exhausted all retries",
		slog.String("endpoint_id", endpoint.ID),
		slog.String("event_id", event.ID),
		slog.Int("total_attempts", attempt),
		slog.Any("error", lastErr),
	)
}

// deliver performs a single delivery attempt
func (r *WebhookReceiver) deliver(ctx context.Context, endpoint *WebhookEndpoint, event WebhookEvent, attempt int) WebhookDelivery {
	delivery := WebhookDelivery{
		ID:         uuid.New().String(),
		EndpointID: endpoint.ID,
		Event:      event,
		Attempt:    attempt + 1,
		DeliveredAt: time.Now(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		delivery.Error = fmt.Errorf("failed to marshal event: %w", err)
		return delivery
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewBuffer(payload))
	if err != nil {
		delivery.Error = fmt.Errorf("failed to create request: %w", err)
		return delivery
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", string(event.Type))
	req.Header.Set("X-Webhook-Event-ID", event.ID)
	req.Header.Set("X-Webhook-Delivery", delivery.ID)
	req.Header.Set("X-Webhook-Attempt", fmt.Sprintf("%d", attempt+1))

	// Sign the payload
	signature := r.signPayload(payload, endpoint.Secret)
	req.Header.Set("X-Webhook-Signature-256", signature)

	// Send the request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		delivery.Error = fmt.Errorf("request failed: %w", err)
		return delivery
	}
	defer resp.Body.Close()

	delivery.StatusCode = resp.StatusCode

	// Read response body (limited to 1KB)
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	delivery.Response = string(body[:n])

	return delivery
}

// signPayload creates HMAC-SHA256 signature for the payload
func (r *WebhookReceiver) signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// calculateBackoff calculates exponential backoff delay
func (r *WebhookReceiver) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt, capped at 5 minutes
	delay := r.config.BaseDelay * time.Duration(1<<uint(attempt))
	maxDelay := 5 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// StartDeliveryWorker begins the delivery retry worker (for monitoring deliveries)
func (r *WebhookReceiver) StartDeliveryWorker(ctx context.Context) {
	slog.Info("webhook delivery worker started")
	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("webhook delivery worker stopped")
				return
			case delivery := <-r.deliveries:
				slog.Debug("delivery tracked",
					slog.String("delivery_id", delivery.ID),
					slog.String("endpoint_id", delivery.EndpointID),
					slog.Int("status_code", delivery.StatusCode),
					slog.Int("attempt", delivery.Attempt),
				)
			}
		}
	}()
}

// GetDeliveryChannel returns the channel for tracking deliveries
func (r *WebhookReceiver) GetDeliveryChannel() <-chan WebhookDelivery {
	return r.deliveries
}

// formatRequestBody safely reads and returns the request body
func fmtRequestsBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, fmt.Errorf("request body is nil")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	return body, nil
}

// isValidEventType checks if the event type is valid
func isValidEventType(eventType EventType) bool {
	switch eventType {
	case EventMessageSent, EventMessageReceived, EventUserCreated, EventUserDeleted:
		return true
	default:
		return false
	}
}

// GenerateEvent creates a new webhook event with generated ID and timestamp
func GenerateEvent(eventType EventType, data map[string]interface{}) WebhookEvent {
	return WebhookEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}
