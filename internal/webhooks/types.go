package webhooks

import (
	"time"
)

// EventType represents webhook event types
type EventType string

const (
	EventMessageSent     EventType = "message.sent"
	EventMessageReceived EventType = "message.received"
	EventUserCreated     EventType = "user.created"
	EventUserDeleted     EventType = "user.deleted"
)

// WebhookEvent is the payload sent to webhooks
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      EventType               `json:"type"`
	Timestamp time.Time               `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WebhookEndpoint represents a registered webhook endpoint
type WebhookEndpoint struct {
	ID        string     `json:"id"`
	URL       string     `json:"url"`
	Secret    string     `json:"secret"`
	Events    []EventType `json:"events"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
}

// WebhookDelivery tracks delivery attempts
type WebhookDelivery struct {
	ID          string        `json:"id"`
	EndpointID  string        `json:"endpoint_id"`
	Event       WebhookEvent  `json:"event"`
	Attempt     int           `json:"attempt"`
	StatusCode  int           `json:"status_code"`
	Response    string        `json:"response"`
	Error       error         `json:"error,omitempty"`
	DeliveredAt time.Time     `json:"delivered_at"`
	NextRetryAt time.Time     `json:"next_retry_at"`
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	Timeout    time.Duration `json:"timeout"`
}

// DefaultWebhookConfig returns sensible defaults
func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		MaxRetries: 5,
		BaseDelay:  time.Second,
		Timeout:    10 * time.Second,
	}
}
