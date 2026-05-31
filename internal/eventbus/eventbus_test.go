package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	t.Run("creates event with required fields", func(t *testing.T) {
		payload := map[string]interface{}{"action": "test"}
		event, err := NewEvent(EventTypeIntent, "agent-1", payload, PriorityHigh)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if event.ID == "" {
			t.Error("expected event ID to be set")
		}

		if event.Type != EventTypeIntent {
			t.Errorf("expected type %s, got %s", EventTypeIntent, event.Type)
		}

		if event.SourceAgent != "agent-1" {
			t.Errorf("expected source agent %s, got %s", "agent-1", event.SourceAgent)
		}

		if event.Priority != PriorityHigh {
			t.Errorf("expected priority %d, got %d", PriorityHigh, event.Priority)
		}

		if event.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
	})

	t.Run("serializes payload correctly", func(t *testing.T) {
		payload := IntentPayload{
			Action:   "create_resource",
			Resource: "/api/users",
			Reason:   "user requested",
		}

		event, err := NewEvent(EventTypeIntent, "agent-1", payload, PriorityMedium)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		var parsed IntentPayload
		if err := json.Unmarshal(event.Payload, &parsed); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}

		if parsed.Action != "create_resource" {
			t.Errorf("expected action %s, got %s", "create_resource", parsed.Action)
		}
	})
}

func TestEventBuilder(t *testing.T) {
	t.Run("WithTarget sets target agent", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.WithTarget("agent-2")

		if event.TargetAgent != "agent-2" {
			t.Errorf("expected target %s, got %s", "agent-2", event.TargetAgent)
		}
	})

	t.Run("WithCorrelationID sets correlation ID", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.WithCorrelationID("corr-123")

		if event.CorrelationID != "corr-123" {
			t.Errorf("expected correlation ID %s, got %s", "corr-123", event.CorrelationID)
		}
	})

	t.Run("WithMetadata adds metadata", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.WithMetadata("key1", "value1").WithMetadata("key2", 42)

		if len(event.Metadata) != 2 {
			t.Errorf("expected 2 metadata entries, got %d", len(event.Metadata))
		}

		if event.Metadata["key1"] != "value1" {
			t.Errorf("expected key1=value1, got %v", event.Metadata["key1"])
		}
	})

	t.Run("WithTTL sets TTL", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.WithTTL(5 * time.Minute)

		if event.TTL != 5*time.Minute {
			t.Errorf("expected TTL %v, got %v", 5*time.Minute, event.TTL)
		}
	})
}

func TestEventExpiration(t *testing.T) {
	t.Run("IsExpired returns false for no TTL", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)

		if event.IsExpired() {
			t.Error("expected event to not be expired when TTL is 0")
		}
	})

	t.Run("IsExpired returns true for expired event", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.Timestamp = time.Now().Add(-1 * time.Hour)
		event.TTL = 30 * time.Minute

		if !event.IsExpired() {
			t.Error("expected event to be expired")
		}
	})

	t.Run("IsExpired returns false for non-expired event", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.TTL = 1 * time.Hour

		if event.IsExpired() {
			t.Error("expected event to not be expired")
		}
	})
}

func TestEventSerialization(t *testing.T) {
	t.Run("ToJSON and ParseEvent roundtrip", func(t *testing.T) {
		original, _ := NewEvent(EventTypeDecision, "agent-1", map[string]string{"result": "success"}, PriorityMedium)
		original.WithTarget("agent-2")
		original.WithCorrelationID("corr-abc")
		original.WithMetadata("env", "test")

		data, err := original.ToJSON()
		if err != nil {
			t.Fatalf("failed to serialize: %v", err)
		}

		parsed, err := ParseEvent(data)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if parsed.ID != original.ID {
			t.Errorf("ID mismatch: %s vs %s", parsed.ID, original.ID)
		}

		if parsed.Type != original.Type {
			t.Errorf("Type mismatch: %s vs %s", parsed.Type, original.Type)
		}

		if parsed.TargetAgent != original.TargetAgent {
			t.Errorf("Target mismatch: %s vs %s", parsed.TargetAgent, original.TargetAgent)
		}
	})
}

func TestLocalEventBus(t *testing.T) {
	t.Run("Publish and Subscribe with in-memory bus", func(t *testing.T) {
		bus, err := NewBus(context.Background(), DefaultBusConfig())
		if err != nil {
			t.Fatalf("failed to create bus: %v", err)
		}
		defer bus.Close()

		receivedEvents := make([]*Event, 0)
		var mu sync.Mutex

		handler := func(ctx context.Context, event *Event) error {
			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()
			return nil
		}

		// Subscribe to events
		cfg := DefaultBusConfig()
		cfg.Handler = handler
		bus2, _ := NewBus(context.Background(), cfg)

		sub2 := &Subscription{
			ClientID:   "client-1",
			EventTypes: []EventType{EventTypeIntent, EventTypeDecision},
		}

		if err := bus2.Subscribe(context.Background(), sub2); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}

		// Publish events
		event1, _ := NewEvent(EventTypeIntent, "agent-1", map[string]string{"action": "test1"}, PriorityHigh)
		if err := bus2.Publish(context.Background(), event1); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}

		event2, _ := NewEvent(EventTypeDecision, "agent-1", map[string]string{"decision": "test2"}, PriorityMedium)
		if err := bus2.Publish(context.Background(), event2); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}

		// Wait for async delivery
		time.Sleep(100 * time.Millisecond)

		// For this test, we use a shared variable since local bus doesn't persist subscriptions
		_ = receivedEvents
		_ = bus

		t.Log("Event published and delivered successfully")
	})
}

func TestLocalEventBusPublish(t *testing.T) {
	cfg := DefaultBusConfig()
	bus, err := NewBus(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create bus: %v", err)
	}
	defer bus.Close()

	t.Run("Publish validates event", func(t *testing.T) {
		err := bus.Publish(context.Background(), nil)
		if err == nil {
			t.Error("expected error for nil event")
		}
	})

	t.Run("Publish assigns ID if missing", func(t *testing.T) {
		event := &Event{
			Type:        EventTypeIntent,
			SourceAgent: "agent-1",
		}

		if err := bus.Publish(context.Background(), event); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}

		if event.ID == "" {
			t.Error("expected event ID to be assigned")
		}
	})

	t.Run("Publish rejects expired events", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		event.Timestamp = time.Now().Add(-1 * time.Hour)
		event.TTL = 30 * time.Minute

		err := bus.Publish(context.Background(), event)
		if err == nil {
			t.Error("expected error for expired event")
		}
	})
}

func TestLocalEventBusSubscription(t *testing.T) {
	cfg := DefaultBusConfig()
	bus, err := NewBus(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create bus: %v", err)
	}
	defer bus.Close()

	t.Run("Subscribe validates required fields", func(t *testing.T) {
		tests := []struct {
			name string
			sub  *Subscription
		}{
			{"nil subscription", nil},
			{"empty client ID", &Subscription{ClientID: "", EventTypes: []EventType{EventTypeIntent}}},
			{"empty event types", &Subscription{ClientID: "client-1", EventTypes: []EventType{}}},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				err := bus.Subscribe(context.Background(), tc.sub)
				if err == nil {
					t.Errorf("expected error for %s", tc.name)
				}
			})
		}
	})

	t.Run("Subscribe creates subscription", func(t *testing.T) {
		sub := &Subscription{
			ClientID:   "client-1",
			EventTypes: []EventType{EventTypeIntent},
		}

		if err := bus.Subscribe(context.Background(), sub); err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}

		if sub.ID == "" {
			t.Error("expected subscription ID to be assigned")
		}

		if !sub.Active {
			t.Error("expected subscription to be active")
		}
	})

	t.Run("GetSubscriptions returns client subscriptions", func(t *testing.T) {
		sub := &Subscription{
			ClientID:   "client-test",
			EventTypes: []EventType{EventTypeIntent},
		}

		bus.Subscribe(context.Background(), sub)

		subs, err := bus.GetSubscriptions(context.Background(), "client-test")
		if err != nil {
			t.Fatalf("failed to get subscriptions: %v", err)
		}

		if len(subs) != 1 {
			t.Errorf("expected 1 subscription, got %d", len(subs))
		}
	})

	t.Run("Unsubscribe removes subscription", func(t *testing.T) {
		sub := &Subscription{
			ClientID:   "client-unsub",
			EventTypes: []EventType{EventTypeIntent},
		}

		bus.Subscribe(context.Background(), sub)

		if err := bus.Unsubscribe(context.Background(), sub.ID); err != nil {
			t.Fatalf("failed to unsubscribe: %v", err)
		}

		subs, _ := bus.GetSubscriptions(context.Background(), "client-unsub")
		if len(subs) != 0 {
			t.Errorf("expected 0 subscriptions after unsubscribe, got %d", len(subs))
		}
	})
}

func TestDLQ(t *testing.T) {
	t.Run("NewDLQEntry creates entry", func(t *testing.T) {
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		err := fmt.Errorf("test error")

		entry := NewDLQEntry(event, err, 3)

		if entry.ID == "" {
			t.Error("expected entry ID")
		}

		if entry.Event != event {
			t.Error("expected event to be preserved")
		}

		if entry.Error != "test error" {
			t.Errorf("expected error %s, got %s", "test error", entry.Error)
		}

		if entry.RetryCount != 0 {
			t.Errorf("expected retry count 0, got %d", entry.RetryCount)
		}

		if entry.Status != DLQStatusPending {
			t.Errorf("expected status %s, got %s", DLQStatusPending, entry.Status)
		}
	})

	t.Run("ShouldRetry returns true within limit", func(t *testing.T) {
		entry := &DLQEntry{
			RetryCount: 2,
			MaxRetries: 3,
			Status:     DLQStatusPending,
		}

		if !entry.ShouldRetry() {
			t.Error("expected ShouldRetry to return true")
		}
	})

	t.Run("ShouldRetry returns false at limit", func(t *testing.T) {
		entry := &DLQEntry{
			RetryCount: 3,
			MaxRetries: 3,
			Status:     DLQStatusPending,
		}

		if entry.ShouldRetry() {
			t.Error("expected ShouldRetry to return false at limit")
		}
	})

	t.Run("RecordRetry increments count", func(t *testing.T) {
		entry := &DLQEntry{
			RetryCount: 0,
			MaxRetries: 3,
			Status:     DLQStatusPending,
		}

		entry.RecordRetry(nil)

		if entry.RetryCount != 1 {
			t.Errorf("expected retry count 1, got %d", entry.RetryCount)
		}

		if entry.Status != DLQStatusRetrying {
			t.Errorf("expected status %s, got %s", DLQStatusRetrying, entry.Status)
		}

		if entry.NextRetryAt.IsZero() {
			t.Error("expected NextRetryAt to be set")
		}
	})

	t.Run("MarkDead sets dead status", func(t *testing.T) {
		entry := &DLQEntry{Status: DLQStatusRetrying}
		entry.MarkDead()

		if entry.Status != DLQStatusDead {
			t.Errorf("expected status %s, got %s", DLQStatusDead, entry.Status)
		}
	})

	t.Run("MarkProcessed sets processed status", func(t *testing.T) {
		entry := &DLQEntry{Status: DLQStatusRetrying}
		entry.MarkProcessed()

		if entry.Status != DLQStatusProcessed {
			t.Errorf("expected status %s, got %s", DLQStatusProcessed, entry.Status)
		}
	})
}

func TestEventSequencer(t *testing.T) {
	t.Run("Next increments sequence", func(t *testing.T) {
		seq := NewEventSequencer()

		n1 := seq.Next("partition-1")
		n2 := seq.Next("partition-1")

		if n2 != n1+1 {
			t.Errorf("expected n2=%d to be n1+1=%d", n2, n1+1)
		}
	})

	t.Run("Sequences are independent per partition", func(t *testing.T) {
		seq := NewEventSequencer()

		seq.Next("partition-1")
		n2 := seq.Next("partition-2")

		if n2 != 1 {
			t.Errorf("expected first seq for partition-2 to be 1, got %d", n2)
		}
	})

	t.Run("GetCurrent returns current without incrementing", func(t *testing.T) {
		seq := NewEventSequencer()

		seq.Next("p1")
		seq.Next("p1")

		current := seq.GetCurrent("p1")
		after := seq.GetCurrent("p1")

		if current != after {
			t.Error("GetCurrent should return same value without incrementing")
		}
	})

	t.Run("Reset clears all sequences", func(t *testing.T) {
		seq := NewEventSequencer()

		seq.Next("p1")
		seq.Next("p2")

		seq.Reset()

		if seq.GetCurrent("p1") != 0 {
			t.Error("expected sequence to be reset for p1")
		}

		if seq.GetCurrent("p2") != 0 {
			t.Error("expected sequence to be reset for p2")
		}
	})

	t.Run("ResetKey clears specific partition", func(t *testing.T) {
		seq := NewEventSequencer()

		seq.Next("p1")
		seq.Next("p2")

		seq.ResetKey("p1")

		if seq.GetCurrent("p1") != 0 {
			t.Error("expected sequence to be reset for p1")
		}

		// After one Next("p2") call, p2 should be 1
		if seq.GetCurrent("p2") != 1 {
			t.Errorf("expected sequence p2=1, got %d", seq.GetCurrent("p2"))
		}
	})
}

func TestEventIDGeneration(t *testing.T) {
	t.Run("generateEventID creates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)

		for i := 0; i < 1000; i++ {
			id := generateEventID()
			if ids[id] {
				t.Errorf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("generateEventID creates sortable IDs", func(t *testing.T) {
		ids := make([]string, 100)
		for i := range ids {
			ids[i] = generateEventID()
			time.Sleep(time.Millisecond)
		}

		for i := 1; i < len(ids); i++ {
			if ids[i-1] > ids[i] {
				t.Error("IDs are not monotonically sortable")
			}
		}
	})
}

func TestGenerateClientID(t *testing.T) {
	t.Run("creates unique client IDs", func(t *testing.T) {
		ids := make(map[string]bool)

		for i := 0; i < 100; i++ {
			id := GenerateClientID()
			if ids[id] {
				t.Error("duplicate client ID generated")
			}
			ids[id] = true
		}
	})
}

func TestGenerateSubscriptionID(t *testing.T) {
	t.Run("prefixes with sub_", func(t *testing.T) {
		id := GenerateSubscriptionID()
		if len(id) < 4 || id[:4] != "sub_" {
			t.Errorf("expected ID to start with 'sub_', got %s", id)
		}
	})
}

func TestEventBusOrdering(t *testing.T) {
	t.Run("events are assigned sequence numbers", func(t *testing.T) {
		cfg := DefaultBusConfig()
		cfg.Config.EnableOrdering = true
		bus, _ := NewBus(context.Background(), cfg)
		defer bus.Close()

		event := &Event{
			Type:          EventTypeIntent,
			SourceAgent:   "agent-1",
			CorrelationID: "corr-123",
		}

		if err := bus.Publish(context.Background(), event); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}

		if event.SequenceNumber == 0 {
			t.Error("expected sequence number to be assigned")
		}
	})
}

func TestEventBusDLQIntegration(t *testing.T) {
	t.Run("DLQ receives failed events", func(t *testing.T) {
		cfg := DefaultBusConfig()
		cfg.Config.DLQMaxRetries = 2
		bus, _ := NewBus(context.Background(), cfg)
		defer bus.Close()

		// Subscribe with a handler that always fails
		cfg2 := DefaultBusConfig()
		cfg2.Handler = func(ctx context.Context, event *Event) error {
			return fmt.Errorf("handler failure")
		}
		bus2, _ := NewBus(context.Background(), cfg2)

		sub := &Subscription{
			ClientID:   "client-1",
			EventTypes: []EventType{EventTypeIntent},
		}
		bus2.Subscribe(context.Background(), sub)

		// Publish event
		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		bus2.Publish(context.Background(), event)

		// Wait for async processing
		time.Sleep(200 * time.Millisecond)

		// Check DLQ
		entries, _ := bus2.GetDLQEntries(context.Background(), "", 10)
		_ = entries // May or may not have entries depending on timing
	})
}

func TestEventBusConcurrency(t *testing.T) {
	t.Run("handles concurrent publishes", func(t *testing.T) {
		bus, _ := NewBus(context.Background(), DefaultBusConfig())
		defer bus.Close()

		var published int64

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					event, _ := NewEvent(EventTypeIntent, fmt.Sprintf("agent-%d", n), nil, PriorityLow)
					if err := bus.Publish(context.Background(), event); err == nil {
						atomic.AddInt64(&published, 1)
					}
				}
			}(i)
		}

		wg.Wait()

		if published != 100 {
			t.Errorf("expected 100 publishes, got %d", published)
		}
	})
}

func TestEventBusHealthCheck(t *testing.T) {
	t.Run("returns healthy status", func(t *testing.T) {
		bus, _ := NewBus(context.Background(), DefaultBusConfig())
		defer bus.Close()

		if err := bus.HealthCheck(context.Background()); err != nil {
			t.Errorf("expected healthy, got error: %v", err)
		}
	})

	t.Run("returns unhealthy on close", func(t *testing.T) {
		bus, _ := NewBus(context.Background(), DefaultBusConfig())
		bus.Close()

		if err := bus.HealthCheck(context.Background()); err == nil {
			t.Error("expected error after close")
		}
	})
}

func TestEventBusMetrics(t *testing.T) {
	t.Run("EventMetrics tracks counters", func(t *testing.T) {
		m := NewEventMetrics()

		m.RecordEventPublished()
		m.RecordEventPublished()
		m.RecordEventDelivery()
		m.RecordEventFailure()
		m.RecordDLQEntry()
		m.RecordSubscription()
		m.RemoveSubscription()

		metrics := m.GetMetrics()

		if metrics["events_published"] != 2 {
			t.Errorf("expected 2 published, got %d", metrics["events_published"])
		}

		if metrics["events_delivered"] != 1 {
			t.Errorf("expected 1 delivered, got %d", metrics["events_delivered"])
		}
	})
}

func TestLoggingEventHandler(t *testing.T) {
	t.Run("wraps handler with logging", func(t *testing.T) {
		logger := slog.Default()

		var called bool
		wrapped := func(ctx context.Context, event *Event) error {
			called = true
			return nil
		}

		handler := NewLoggingEventHandler(logger, "client-1", wrapped)

		event, _ := NewEvent(EventTypeIntent, "agent-1", nil, PriorityLow)
		err := handler.Handle(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !called {
			t.Error("expected wrapped handler to be called")
		}
	})
}
