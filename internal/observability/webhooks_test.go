package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookManager_RegisterWebhook(t *testing.T) {
	bus := NewDefaultEventBus(100)
	manager := NewWebhookManager(bus)
	defer manager.Close()

	t.Run("registers webhook with valid URL", func(t *testing.T) {
		webhook, err := manager.RegisterWebhook(
			"https://example.com/webhook",
			[]EventType{EventTypeRequestCompleted},
			"secret123",
		)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if webhook == nil {
			t.Fatal("Expected webhook to be returned")
		}

		if webhook.ID == "" {
			t.Error("Expected webhook ID to be set")
		}

		if webhook.URL != "https://example.com/webhook" {
			t.Errorf("Expected URL %s, got %s", "https://example.com/webhook", webhook.URL)
		}

		if webhook.Status != WebhookStatusActive {
			t.Errorf("Expected status %s, got %s", WebhookStatusActive, webhook.Status)
		}
	})

	t.Run("rejects webhook with empty URL", func(t *testing.T) {
		_, err := manager.RegisterWebhook("", []EventType{EventTypeRequestCompleted}, "")
		if err == nil {
			t.Error("Expected error for empty URL")
		}
	})

	t.Run("retrieves registered webhook", func(t *testing.T) {
		webhook, _ := manager.RegisterWebhook(
			"https://example.com/test",
			[]EventType{EventTypeRequestCompleted},
			"",
		)

		retrieved, err := manager.GetWebhook(webhook.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if retrieved.ID != webhook.ID {
			t.Errorf("Expected ID %s, got %s", webhook.ID, retrieved.ID)
		}
	})

	t.Run("lists all webhooks", func(t *testing.T) {
		initialCount := len(manager.ListWebhooks())

		manager.RegisterWebhook("https://example.com/webhook1", nil, "")
		manager.RegisterWebhook("https://example.com/webhook2", nil, "")

		webhooks := manager.ListWebhooks()
		if len(webhooks) != initialCount+2 {
			t.Errorf("Expected %d webhooks, got %d", initialCount+2, len(webhooks))
		}
	})

	t.Run("unregisters webhook", func(t *testing.T) {
		webhook, _ := manager.RegisterWebhook(
			"https://example.com/to-delete",
			nil,
			"",
		)

		err := manager.UnregisterWebhook(webhook.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		_, err = manager.GetWebhook(webhook.ID)
		if err == nil {
			t.Error("Expected error when getting deleted webhook")
		}
	})
}

func TestWebhookManager_Send(t *testing.T) {
	t.Run("HMAC signature verification", func(t *testing.T) {
		bus := NewDefaultEventBus(100)
		manager := NewWebhookManager(bus)
		defer manager.Close()

		secret := "my-secret-key"
		payload := []byte(`{"test":"data"}`)

		// Generate signature
		signature := manager.signPayload(payload, secret)

		// Verify valid signature
		if !manager.VerifySignature(payload, signature, secret) {
			t.Error("Expected signature to be valid")
		}

		// Verify invalid signature
		if manager.VerifySignature(payload, "invalid", secret) {
			t.Error("Expected invalid signature to fail verification")
		}

		// Verify wrong secret
		if manager.VerifySignature(payload, signature, "wrong-secret") {
			t.Error("Expected wrong secret to fail verification")
		}
	})
}

func TestWebhookManager_SendWithRetry(t *testing.T) {
	bus := NewDefaultEventBus(100)
	manager := NewWebhookManager(bus)
	defer manager.Close()

	webhook, _ := manager.RegisterWebhook(
		"https://example.com/original",
		[]EventType{EventTypeRequestCompleted},
		"original-secret",
	)

	err := manager.UpdateWebhook(
		webhook.ID,
		"https://example.com/updated",
		[]EventType{EventTypeRequestFailed},
		"updated-secret",
	)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updated, _ := manager.GetWebhook(webhook.ID)

	if updated.URL != "https://example.com/updated" {
		t.Errorf("Expected URL to be updated")
	}

	if len(updated.Events) != 1 || updated.Events[0] != EventTypeRequestFailed {
		t.Errorf("Expected events to be updated")
	}

	if updated.Secret != "updated-secret" {
		t.Errorf("Expected secret to be updated")
	}
}

func TestWebhookManager_GetDeliveryStats(t *testing.T) {
	bus := NewDefaultEventBus(100)
	manager := NewWebhookManager(bus)
	defer manager.Close()

	webhook, _ := manager.RegisterWebhook("https://example.com/test", nil, "")

	stats := manager.GetDeliveryStats(webhook.ID)

	if stats["total"] != 0 {
		t.Errorf("Expected 0 total deliveries, got %d", stats["total"])
	}
}

func bodyFromRequest(r *http.Request) ([]byte, error) {
	return []byte{}, nil
}

func TestRetryPolicy(t *testing.T) {
	t.Run("default retry policy has sensible values", func(t *testing.T) {
		policy := DefaultRetryPolicy()

		if policy.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries=3, got %d", policy.MaxRetries)
		}

		if policy.InitialDelay == 0 {
			t.Error("Expected InitialDelay to be non-zero")
		}

		if policy.BackoffFactor != 2.0 {
			t.Errorf("Expected BackoffFactor=2.0, got %f", policy.BackoffFactor)
		}
	})
}

func TestEventDeliveryOnPublish(t *testing.T) {
	t.Run("webhook receives event from event bus", func(t *testing.T) {
		var eventReceived bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventReceived = true
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		bus := NewDefaultEventBus(100)
		manager := NewWebhookManager(bus)
		defer manager.Close()

		_, err := manager.RegisterWebhook(
			server.URL,
			[]EventType{EventTypeRequestCompleted},
			"",
		)
		if err != nil {
			t.Fatalf("Failed to register webhook: %v", err)
		}

		// Publish an event directly to the bus
		event := NewEvent(EventTypeRequestCompleted)
		bus.Publish(context.Background(), event)

		// Give async processing time
		time.Sleep(200 * time.Millisecond)

		if !eventReceived {
			t.Error("Expected event to be delivered to webhook")
		}
	})
}
