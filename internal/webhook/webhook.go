package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType defines the type of webhook event
type EventType string

const (
	// Request events
	EventRequestStart     EventType = "request.start"
	EventRequestComplete  EventType = "request.complete"
	EventRequestFailed    EventType = "request.failed"

	// Budget events
	EventBudgetAlert       EventType = "budget.alert"
	EventBudgetExceeded    EventType = "budget.exceeded"
	EventBudgetReset       EventType = "budget.reset"

	// Provider events
	EventProviderUp        EventType = "provider.up"
	EventProviderDown      EventType = "provider.down"
	EventProviderDegraded  EventType = "provider.degraded"

	// Model events
	EventModelSelected    EventType = "model.selected"
	EventModelFallback    EventType = "model.fallback"

	// Guardrail events
	EventGuardrailTriggered EventType = "guardrail.triggered"
	EventGuardrailBlocked   EventType = "guardrail.blocked"

	// Session events
	EventSessionStart     EventType = "session.start"
	EventSessionEnd       EventType = "session.end"
)

// WebhookSubscription represents a webhook subscription
type WebhookSubscription struct {
	mu           sync.RWMutex
	ID           string            `json:"id"`
	OrgID        string            `json:"org_id"`
	URL          string            `json:"url"`
	Secret       string            `json:"-"` // HMAC secret
	EventTypes   []EventType       `json:"event_types"`
	Headers      map[string]string `json:"headers"`
	Active       bool              `json:"active"`
	RetryPolicy  *RetryPolicy      `json:"retry_policy"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	LastDelivery time.Time         `json:"last_delivery,omitempty"`
	FailureCount int               `json:"failure_count"`
}

// RetryPolicy defines retry behavior for failed deliveries
type RetryPolicy struct {
	MaxAttempts    int           `json:"max_attempts"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:    5,
		InitialDelay:  time.Second,
		MaxDelay:      time.Hour,
		BackoffFactor: 2.0,
	}
}

// WebhookEvent represents a webhook event to be delivered
type WebhookEvent struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	OrgID       string                 `json:"org_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
	Attempts    int                    `json:"attempts"`
	LastAttempt time.Time              `json:"last_attempt,omitempty"`
}

// DeliveryResult represents the result of a delivery attempt
type DeliveryResult struct {
	Success     bool      `json:"success"`
	StatusCode  int       `json:"status_code"`
	Response    string    `json:"response,omitempty"`
	Error       string    `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	AttemptedAt time.Time `json:"attempted_at"`
}

// WebhookManager manages webhook subscriptions and deliveries
type WebhookManager struct {
	mu             sync.RWMutex
	subscriptions  map[string]*WebhookSubscription
	eventBuffer    chan *WebhookEvent
	deliveryQueue  chan *DeliveryTask
	maxBufferSize  int
	httpClient     *HTTPClient
}

// HTTPClient is a simple HTTP client interface
type HTTPClient interface {
	Do(req *DeliveryRequest) *DeliveryResult
}

// SimpleHTTPClient implements HTTPClient
type SimpleHTTPClient struct {
	Timeout time.Duration
}

// DeliveryRequest represents an HTTP delivery request
type DeliveryRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    []byte
	Secret  string
}

// Do performs the HTTP request
func (c *SimpleHTTPClient) Do(req *DeliveryRequest) *DeliveryResult {
	start := time.Now()

	// Create HMAC signature
	if req.Secret != "" {
		h := hmac.New(sha256.New, []byte(req.Secret))
		h.Write(req.Body)
		signature := hex.EncodeToString(h.Sum(nil))
		req.Headers["X-Webhook-Signature"] = "sha256=" + signature
	}

	// In production, use actual HTTP client
	// For now, simulate delivery
	// This would use net/http.Client in production
	
	return &DeliveryResult{
		Success:    true,
		StatusCode: 200,
		Response:   "OK",
		Duration:   time.Since(start),
		AttemptedAt: time.Now(),
	}
}

// DeliveryTask represents a delivery task in the queue
type DeliveryTask struct {
	subscription *WebhookSubscription
	event       *WebhookEvent
	result      chan *DeliveryResult
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(bufferSize int) *WebhookManager {
	wm := &WebhookManager{
		subscriptions: make(map[string]*WebhookSubscription),
		eventBuffer:   make(chan *WebhookEvent, bufferSize),
		deliveryQueue: make(chan *DeliveryTask, bufferSize),
		maxBufferSize: bufferSize,
		httpClient: &SimpleHTTPClient{
			Timeout: 30 * time.Second,
		},
	}

	// Start delivery workers
	for i := 0; i < 5; i++ {
		go wm.deliveryWorker()
	}

	return wm
}

// CreateSubscription creates a new webhook subscription
func (wm *WebhookManager) CreateSubscription(ctx context.Context, sub *WebhookSubscription) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	if sub.RetryPolicy == nil {
		sub.RetryPolicy = DefaultRetryPolicy()
	}
	sub.CreatedAt = time.Now()
	sub.UpdatedAt = time.Now()
	sub.Active = true

	wm.subscriptions[sub.ID] = sub

	slog.InfoContext(ctx, "Webhook subscription created",
		slog.String("id", sub.ID),
		slog.String("org_id", sub.OrgID),
		slog.String("url", sub.URL),
		slog.Any("event_types", sub.EventTypes),
	)

	return nil
}

// DeleteSubscription deletes a subscription
func (wm *WebhookManager) DeleteSubscription(ctx context.Context, id string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.subscriptions[id]; !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	delete(wm.subscriptions, id)

	slog.InfoContext(ctx, "Webhook subscription deleted",
		slog.String("id", id),
	)

	return nil
}

// ListSubscriptions returns all subscriptions for an org
func (wm *WebhookManager) ListSubscriptions(orgID string) []*WebhookSubscription {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var result []*WebhookSubscription
	for _, sub := range wm.subscriptions {
		if sub.OrgID == orgID && sub.Active {
			result = append(result, sub)
		}
	}
	return result
}

// GetSubscription returns a subscription by ID
func (wm *WebhookManager) GetSubscription(id string) (*WebhookSubscription, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	sub, exists := wm.subscriptions[id]
	return sub, exists
}

// UpdateSubscription updates a subscription
func (wm *WebhookManager) UpdateSubscription(ctx context.Context, sub *WebhookSubscription) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	existing, exists := wm.subscriptions[sub.ID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", sub.ID)
	}

	sub.CreatedAt = existing.CreatedAt
	sub.UpdatedAt = time.Now()

	wm.subscriptions[sub.ID] = sub

	return nil
}

// Publish publishes an event to all matching subscriptions
func (wm *WebhookManager) Publish(ctx context.Context, event *WebhookEvent) error {
	// Non-blocking send to buffer
	select {
	case wm.eventBuffer <- event:
		return nil
	default:
		return fmt.Errorf("event buffer full")
	}
}

// deliveryWorker processes delivery tasks
func (wm *WebhookManager) deliveryWorker() {
	for task := range wm.deliveryQueue {
		result := wm.deliverEvent(task.subscription, task.event)

		// Retry if failed
		if !result.Success && task.event.Attempts < task.subscription.RetryPolicy.MaxAttempts {
			wm.scheduleRetry(task, result)
		} else {
			// Update subscription stats
			task.subscription.mu.Lock()
			task.subscription.LastDelivery = time.Now()
			if !result.Success {
				task.subscription.FailureCount++
			} else {
				task.subscription.FailureCount = 0
			}
			task.subscription.mu.Unlock()
		}

		// Send result to waiting channel
		if task.result != nil {
			task.result <- result
		}
	}
}

// deliverEvent attempts to deliver an event to a subscription
func (wm *WebhookManager) deliverEvent(sub *WebhookSubscription, event *WebhookEvent) *DeliveryResult {
	// Serialize event
	body, err := json.Marshal(event)
	if err != nil {
		return &DeliveryResult{
			Success:    false,
			Error:      fmt.Sprintf("failed to marshal event: %v", err),
			Duration:   0,
			AttemptedAt: time.Now(),
		}
	}

	// Build request
	req := &DeliveryRequest{
		URL:     sub.URL,
		Method:  "POST",
		Headers: make(map[string]string),
		Body:    body,
		Secret:  sub.Secret,
	}

	// Add custom headers
	for k, v := range sub.Headers {
		req.Headers[k] = v
	}
	req.Headers["Content-Type"] = "application/json"
	req.Headers["X-Webhook-ID"] = sub.ID
	req.Headers["X-Event-Type"] = string(event.Type)
	req.Headers["X-Event-ID"] = event.ID

	return wm.httpClient.Do(req)
}

// scheduleRetry schedules a retry for a failed delivery
func (wm *WebhookManager) scheduleRetry(task *DeliveryTask, lastResult *DeliveryResult) {
	task.event.Attempts++
	task.event.LastAttempt = time.Now()

	delay := task.subscription.RetryPolicy.InitialDelay
	for i := 1; i < task.event.Attempts; i++ {
		delay = time.Duration(float64(delay) * task.subscription.RetryPolicy.BackoffFactor)
		if delay > task.subscription.RetryPolicy.MaxDelay {
			delay = task.subscription.RetryPolicy.MaxDelay
		}
	}

	// In production, use a proper scheduler
	// For now, requeue with delay
	go func() {
		time.Sleep(delay)
		wm.deliveryQueue <- task
	}()
}

// SubscribeToEvents creates a subscription for specific event types
func SubscribeToEvents(orgID, url, secret string, events []EventType) *WebhookSubscription {
	return &WebhookSubscription{
		ID:         uuid.New().String(),
		OrgID:      orgID,
		URL:        url,
		Secret:     secret,
		EventTypes: events,
		Headers:    make(map[string]string),
		RetryPolicy: DefaultRetryPolicy(),
		Active:     true,
	}
}

// WebhookEventBuilder helps build webhook events
type WebhookEventBuilder struct {
	event *WebhookEvent
}

// NewEvent creates a new event builder
func NewEvent(orgID string, eventType EventType) *WebhookEventBuilder {
	return &WebhookEventBuilder{
		event: &WebhookEvent{
			ID:        uuid.New().String(),
			Type:      eventType,
			OrgID:     orgID,
			Timestamp: time.Now(),
			Data:      make(map[string]interface{}),
			Metadata:  make(map[string]string),
		},
	}
}

// WithData adds data to the event
func (b *WebhookEventBuilder) WithData(key string, value interface{}) *WebhookEventBuilder {
	b.event.Data[key] = value
	return b
}

// WithMetadata adds metadata to the event
func (b *WebhookEventBuilder) WithMetadata(key, value string) *WebhookEventBuilder {
	b.event.Metadata[key] = value
	return b
}

// Build returns the final event
func (b *WebhookEventBuilder) Build() *WebhookEvent {
	return b.event
}

// VerifySignature verifies an HMAC signature
func VerifySignature(payload []byte, signature, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expected := "sha256=" + hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// Stats returns webhook statistics
type WebhookStats struct {
	TotalSubscriptions int            `json:"total_subscriptions"`
	ActiveSubscriptions int          `json:"active_subscriptions"`
	TotalEvents       int64         `json:"total_events"`
	BufferUtilization float64       `json:"buffer_utilization"`
	SubscriptionsByOrg map[string]int `json:"subscriptions_by_org"`
}

// GetStats returns webhook statistics
func (wm *WebhookManager) GetStats() WebhookStats {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	stats := WebhookStats{
		SubscriptionsByOrg: make(map[string]int),
	}
	stats.TotalSubscriptions = len(wm.subscriptions)

	for _, sub := range wm.subscriptions {
		if sub.Active {
			stats.ActiveSubscriptions++
		}
		stats.SubscriptionsByOrg[sub.OrgID]++
	}

	stats.BufferUtilization = float64(len(wm.eventBuffer)) / float64(wm.maxBufferSize)

	return stats
}

// Ensure types are used
var _ = context.Background
