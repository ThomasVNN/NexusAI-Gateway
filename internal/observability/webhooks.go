package observability

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

// WebhookStatus represents the status of a webhook
type WebhookStatus string

const (
	WebhookStatusActive   WebhookStatus = "active"
	WebhookStatusDisabled WebhookStatus = "disabled"
	WebhookStatusFailed   WebhookStatus = "failed"
)

// Webhook represents a webhook subscription
type Webhook struct {
	ID          string
	URL         string
	Events      []EventType
	Secret      string
	RetryPolicy *RetryPolicy
	Status      WebhookStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Metadata    map[string]string
}

// RetryPolicy defines how failed webhook deliveries are retried
type RetryPolicy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryPolicy returns the default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
	}
}

// WebhookDelivery represents a single webhook delivery attempt
type WebhookDelivery struct {
	ID           string
	WebhookID    string
	EventID      string
	EventType    EventType
	Payload      []byte
	Attempts     int
	Status       string
	StatusCode   int
	ResponseBody string
	Error        string
	CreatedAt    time.Time
	DeliveredAt  *time.Time
}

// WebhookDeliveryStatus represents the status of a delivery
type WebhookDeliveryStatus string

const (
	DeliveryStatusPending   WebhookDeliveryStatus = "pending"
	DeliveryStatusSuccess   WebhookDeliveryStatus = "success"
	DeliveryStatusFailed    WebhookDeliveryStatus = "failed"
	DeliveryStatusRetrying  WebhookDeliveryStatus = "retrying"
)

// WebhookManager manages webhook subscriptions and deliveries
type WebhookManager struct {
	webhooks      map[string]*Webhook
	deliveries    map[string][]*WebhookDelivery
	mu            sync.RWMutex
	httpClient    *http.Client
	retryPolicy   *RetryPolicy
	eventBus      EventBus
	logger        *slog.Logger
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(eventBus EventBus) *WebhookManager {
	manager := &WebhookManager{
		webhooks:    make(map[string]*Webhook),
		deliveries:  make(map[string][]*WebhookDelivery),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		retryPolicy: DefaultRetryPolicy(),
		eventBus:    eventBus,
		logger:      slog.Default(),
		stopChan:    make(chan struct{}),
	}

	if eventBus != nil {
		manager.subscribeToEvents()
	}

	return manager
}

// subscribeToEvents subscribes the webhook manager to relevant events
func (m *WebhookManager) subscribeToEvents() {
	m.eventBus.Subscribe(m.handleEvent,
		EventTypeRequestReceived,
		EventTypeRequestCompleted,
		EventTypeRequestFailed,
		EventTypeProviderCalled,
		EventTypeProviderFailed,
		EventTypeCircuitOpened,
		EventTypeCircuitClosed,
		EventTypeQuotaWarning,
		EventTypeQuotaExceeded,
		EventTypeCostAnomaly,
		EventTypeFallbackTriggered,
		EventTypeCacheHit,
		EventTypeCacheMiss,
	)
}

// handleEvent handles incoming events for webhook delivery
func (m *WebhookManager) handleEvent(ctx context.Context, event *Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, webhook := range m.webhooks {
		if webhook.Status != WebhookStatusActive {
			continue
		}

		if m.shouldDeliverToWebhook(webhook, event.Type) {
			m.wg.Add(1)
			go func(wh *Webhook, e *Event) {
				defer m.wg.Done()
				if err := m.Send(wh, e); err != nil {
					m.logger.Error("Failed to send webhook",
						slog.String("webhook_id", wh.ID),
						slog.String("event_id", e.ID),
						slog.Any("error", err))
				}
			}(webhook, event)
		}
	}
}

// shouldDeliverToWebhook checks if an event should be delivered to a webhook
func (m *WebhookManager) shouldDeliverToWebhook(webhook *Webhook, eventType EventType) bool {
	if len(webhook.Events) == 0 {
		return true
	}

	for _, et := range webhook.Events {
		if et == eventType {
			return true
		}
	}

	return false
}

// RegisterWebhook registers a new webhook
func (m *WebhookManager) RegisterWebhook(url string, events []EventType, secret string) (*Webhook, error) {
	if url == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	webhook := &Webhook{
		ID:          uuid.New().String(),
		URL:         url,
		Events:      events,
		Secret:      secret,
		RetryPolicy: DefaultRetryPolicy(),
		Status:      WebhookStatusActive,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Metadata:    make(map[string]string),
	}

	m.webhooks[webhook.ID] = webhook

	m.logger.Info("Registered webhook",
		slog.String("webhook_id", webhook.ID),
		slog.String("url", url),
		slog.Int("events", len(events)))

	return webhook, nil
}

// UnregisterWebhook removes a webhook by ID
func (m *WebhookManager) UnregisterWebhook(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.webhooks[id]; !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}

	delete(m.webhooks, id)
	delete(m.deliveries, id)

	m.logger.Info("Unregistered webhook",
		slog.String("webhook_id", id))

	return nil
}

// GetWebhook retrieves a webhook by ID
func (m *WebhookManager) GetWebhook(id string) (*Webhook, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	webhook, exists := m.webhooks[id]
	if !exists {
		return nil, fmt.Errorf("webhook not found: %s", id)
	}

	return webhook, nil
}

// ListWebhooks returns all registered webhooks
func (m *WebhookManager) ListWebhooks() []*Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()

	webhooks := make([]*Webhook, 0, len(m.webhooks))
	for _, wh := range m.webhooks {
		webhooks = append(webhooks, wh)
	}

	return webhooks
}

// UpdateWebhook updates an existing webhook
func (m *WebhookManager) UpdateWebhook(id string, url string, events []EventType, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	webhook, exists := m.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}

	if url != "" {
		webhook.URL = url
	}
	if events != nil {
		webhook.Events = events
	}
	if secret != "" {
		webhook.Secret = secret
	}
	webhook.UpdatedAt = time.Now().UTC()

	return nil
}

// Send delivers an event to a webhook
func (m *WebhookManager) Send(webhook *Webhook, event *Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	delivery := &WebhookDelivery{
		ID:        uuid.New().String(),
		WebhookID: webhook.ID,
		EventID:   event.ID,
		EventType: event.Type,
		Payload:   payload,
		Attempts:  0,
		Status:    string(DeliveryStatusPending),
		CreatedAt: time.Now().UTC(),
	}

	m.recordDelivery(delivery)

	return m.sendWithRetry(webhook, delivery)
}

// sendWithRetry sends a webhook with retry logic
func (m *WebhookManager) sendWithRetry(webhook *Webhook, delivery *WebhookDelivery) error {
	maxRetries := webhook.RetryPolicy.MaxRetries
	if maxRetries <= 0 {
		maxRetries = m.retryPolicy.MaxRetries
	}

	delay := webhook.RetryPolicy.InitialDelay
	if delay == 0 {
		delay = m.retryPolicy.InitialDelay
	}

	maxDelay := webhook.RetryPolicy.MaxDelay
	if maxDelay == 0 {
		maxDelay = m.retryPolicy.MaxDelay
	}

	backoffFactor := webhook.RetryPolicy.BackoffFactor
	if backoffFactor == 0 {
		backoffFactor = m.retryPolicy.BackoffFactor
	}

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		delivery.Attempts = attempt + 1

		if attempt > 0 {
			m.updateDeliveryStatus(delivery.ID, string(DeliveryStatusRetrying), 0, "", "")
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * backoffFactor)
			if delay > maxDelay {
				delay = maxDelay
			}
		}

		statusCode, responseBody, err := m.doSend(webhook, delivery.Payload)

		if err != nil {
			lastErr = err
			delivery.Error = err.Error()
			m.updateDeliveryStatus(delivery.ID, string(DeliveryStatusRetrying), statusCode, responseBody, err.Error())
			continue
		}

		if statusCode >= 200 && statusCode < 300 {
			now := time.Now().UTC()
			delivery.Status = string(DeliveryStatusSuccess)
			delivery.StatusCode = statusCode
			delivery.ResponseBody = responseBody
			delivery.DeliveredAt = &now
			m.updateDeliveryStatus(delivery.ID, string(DeliveryStatusSuccess), statusCode, responseBody, "")
			return nil
		}

		lastErr = fmt.Errorf("webhook returned non-success status: %d", statusCode)
		delivery.StatusCode = statusCode
		delivery.ResponseBody = responseBody
		delivery.Error = lastErr.Error()
		m.updateDeliveryStatus(delivery.ID, string(DeliveryStatusRetrying), statusCode, responseBody, lastErr.Error())
	}

	delivery.Status = string(DeliveryStatusFailed)
	m.updateDeliveryStatus(delivery.ID, string(DeliveryStatusFailed), delivery.StatusCode, delivery.ResponseBody, delivery.Error)

	m.logger.Error("Webhook delivery failed after all retries",
		slog.String("webhook_id", webhook.ID),
		slog.String("delivery_id", delivery.ID),
		slog.Int("attempts", delivery.Attempts),
		slog.Any("error", lastErr))

	return lastErr
}

// doSend performs the actual HTTP POST to the webhook URL
func (m *WebhookManager) doSend(webhook *Webhook, payload []byte) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return 0, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NexusAI-Gateway/1.0")
	req.Header.Set("X-Webhook-ID", webhook.ID)

	if len(webhook.Events) > 0 {
		req.Header.Set("X-Webhook-Event", string(webhook.Events[0]))
	}

	if webhook.Secret != "" {
		signature := m.signPayload(payload, webhook.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return resp.StatusCode, "", fmt.Errorf("failed to read response: %w", err)
	}

	return resp.StatusCode, string(body), nil
}

// signPayload creates an HMAC-SHA256 signature for the payload
func (m *WebhookManager) signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature verifies an HMAC signature
func (m *WebhookManager) VerifySignature(payload []byte, signature, secret string) bool {
	expected := m.signPayload(payload, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// recordDelivery records a new delivery attempt
func (m *WebhookManager) recordDelivery(delivery *WebhookDelivery) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deliveries[delivery.WebhookID] = append(m.deliveries[delivery.WebhookID], delivery)
}

// updateDeliveryStatus updates the status of a delivery
func (m *WebhookManager) updateDeliveryStatus(id, status string, statusCode int, responseBody, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, deliveries := range m.deliveries {
		for _, d := range deliveries {
			if d.ID == id {
				d.Status = status
				d.StatusCode = statusCode
				d.ResponseBody = responseBody
				d.Error = errorMsg
				return
			}
		}
	}
}

// GetDeliveries returns all deliveries for a webhook
func (m *WebhookManager) GetDeliveries(webhookID string, limit int) []*WebhookDelivery {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deliveries := m.deliveries[webhookID]
	if len(deliveries) == 0 {
		return nil
	}

	if limit > 0 && limit < len(deliveries) {
		return deliveries[len(deliveries)-limit:]
	}

	return deliveries
}

// GetDeliveryStats returns statistics about webhook deliveries
func (m *WebhookManager) GetDeliveryStats(webhookID string) map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     0,
		"success":   0,
		"failed":    0,
		"retrying":  0,
		"pending":   0,
	}

	deliveries := m.deliveries[webhookID]
	stats["total"] = len(deliveries)

	for _, d := range deliveries {
		switch d.Status {
		case string(DeliveryStatusSuccess):
			stats["success"]++
		case string(DeliveryStatusFailed):
			stats["failed"]++
		case string(DeliveryStatusRetrying):
			stats["retrying"]++
		case string(DeliveryStatusPending):
			stats["pending"]++
		}
	}

	return stats
}

// Close shuts down the webhook manager
func (m *WebhookManager) Close() {
	close(m.stopChan)
	m.wg.Wait()
}

// Global webhook manager instance
var globalWebhookManager *WebhookManager

// InitGlobalWebhookManager initializes the global webhook manager
func InitGlobalWebhookManager(eventBus EventBus) {
	globalWebhookManager = NewWebhookManager(eventBus)
}

// GetGlobalWebhookManager returns the global webhook manager
func GetGlobalWebhookManager() *WebhookManager {
	return globalWebhookManager
}
