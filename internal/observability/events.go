package observability

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of an event
type EventType string

// Event type constants
const (
	EventTypeRequestReceived    EventType = "request.received"
	EventTypeRequestCompleted   EventType = "request.completed"
	EventTypeRequestFailed      EventType = "request.failed"
	EventTypeProviderCalled     EventType = "provider.called"
	EventTypeProviderFailed     EventType = "provider.failed"
	EventTypeCircuitOpened      EventType = "circuit.opened"
	EventTypeCircuitClosed      EventType = "circuit.closed"
	EventTypeCircuitHalfOpen    EventType = "circuit.half_open"
	EventTypeQuotaWarning       EventType = "quota.warning"
	EventTypeQuotaExceeded      EventType = "quota.exceeded"
	EventTypeCostAnomaly       EventType = "cost.anomaly"
	EventTypeFallbackTriggered  EventType = "fallback.triggered"
	EventTypeCacheHit          EventType = "cache.hit"
	EventTypeCacheMiss         EventType = "cache.miss"
	EventTypeEvalSuiteRun      EventType = "eval.suite.run"
	EventTypeEvalAssertionPass EventType = "eval.assertion.pass"
	EventTypeEvalAssertionFail EventType = "eval.assertion.fail"
	EventTypeWebhookDelivered  EventType = "webhook.delivered"
	EventTypeWebhookFailed     EventType = "webhook.failed"
	EventTypeAuditLogEntry     EventType = "audit.log_entry"
	EventTypeHealthCheck       EventType = "health.check"
)

// Event represents a system event
type Event struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	Data      map[string]interface{}
	Metadata  map[string]string
}

// NewEvent creates a new event with generated ID and timestamp
func NewEvent(eventType EventType) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      make(map[string]interface{}),
		Metadata:  make(map[string]string),
	}
}

// NewEventWithData creates a new event with initial data
func NewEventWithData(eventType EventType, data map[string]interface{}) *Event {
	event := NewEvent(eventType)
	event.Data = data
	return event
}

// SetMetadata sets a metadata key-value pair
func (e *Event) SetMetadata(key, value string) {
	e.Metadata[key] = value
}

// SetData sets a data key-value pair
func (e *Event) SetData(key string, value interface{}) {
	e.Data[key] = value
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event *Event)

// EventBus provides publish-subscribe functionality for events
type EventBus interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(handler EventHandler, events ...EventType) (string, error)
	Unsubscribe(id string) error
	UnsubscribeAll()
}

// Subscription represents an active event subscription
type Subscription struct {
	ID      string
	Handler EventHandler
	Events  []EventType
}

// DefaultEventBus is the default in-memory event bus implementation
type DefaultEventBus struct {
	subscribers map[string]*Subscription
	mu          sync.RWMutex
	eventChan   chan *Event
	stopChan    chan struct{}
	wg          sync.WaitGroup
	logger      *slog.Logger
}

// NewDefaultEventBus creates a new in-memory event bus
func NewDefaultEventBus(bufferSize int) *DefaultEventBus {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	bus := &DefaultEventBus{
		subscribers: make(map[string]*Subscription),
		eventChan:   make(chan *Event, bufferSize),
		stopChan:    make(chan struct{}),
		logger:      slog.Default(),
	}

	bus.wg.Add(1)
	go bus.processEvents()

	return bus
}

// Publish publishes an event to the bus
func (b *DefaultEventBus) Publish(ctx context.Context, event *Event) error {
	select {
	case b.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		b.logger.Warn("Event bus buffer full, dropping event",
			slog.String("event_type", string(event.Type)),
			slog.String("event_id", event.ID))
		return nil
	}
}

// Subscribe registers a handler for specific event types
func (b *DefaultEventBus) Subscribe(handler EventHandler, events ...EventType) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New().String()
	sub := &Subscription{
		ID:      id,
		Handler: handler,
		Events:  events,
	}

	b.subscribers[id] = sub

	b.logger.Debug("Subscribed handler",
		slog.String("subscription_id", id),
		slog.Int("event_types", len(events)))

	return id, nil
}

// Unsubscribe removes a subscription by ID
func (b *DefaultEventBus) Unsubscribe(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[id]; !exists {
		return nil
	}

	delete(b.subscribers, id)

	b.logger.Debug("Unsubscribed handler",
		slog.String("subscription_id", id))

	return nil
}

// UnsubscribeAll removes all subscriptions
func (b *DefaultEventBus) UnsubscribeAll() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers = make(map[string]*Subscription)

	b.logger.Debug("Unsubscribed all handlers")
}

// processEvents processes events from the channel
func (b *DefaultEventBus) processEvents() {
	defer b.wg.Done()

	for {
		select {
		case event := <-b.eventChan:
			b.dispatchEvent(event)
		case <-b.stopChan:
			return
		}
	}
}

// dispatchEvent dispatches an event to all matching subscribers
func (b *DefaultEventBus) dispatchEvent(event *Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sub := range b.subscribers {
		if b.shouldDeliver(sub, event.Type) {
			go func(handler EventHandler) {
				defer func() {
					if r := recover(); r != nil {
						b.logger.Error("Event handler panicked",
							slog.Any("panic", r),
							slog.String("event_type", string(event.Type)))
					}
				}()
				handler(context.Background(), event)
			}(sub.Handler)
		}
	}
}

// shouldDeliver checks if an event should be delivered to a subscription
func (b *DefaultEventBus) shouldDeliver(sub *Subscription, eventType EventType) bool {
	if len(sub.Events) == 0 {
		return true
	}

	for _, et := range sub.Events {
		if et == eventType {
			return true
		}
	}

	return false
}

// Close shuts down the event bus
func (b *DefaultEventBus) Close() {
	close(b.stopChan)
	b.wg.Wait()
	close(b.eventChan)
}

// SubscriberCount returns the number of active subscribers
func (b *DefaultEventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Global event bus instance
var globalEventBus EventBus

// InitGlobalEventBus initializes the global event bus
func InitGlobalEventBus(bufferSize int) {
	globalEventBus = NewDefaultEventBus(bufferSize)
}

// GetGlobalEventBus returns the global event bus
func GetGlobalEventBus() EventBus {
	if globalEventBus == nil {
		InitGlobalEventBus(1000)
	}
	return globalEventBus
}

// PublishEvent publishes an event to the global event bus
func PublishEvent(ctx context.Context, event *Event) error {
	return GetGlobalEventBus().Publish(ctx, event)
}

// SubscribeToEvents subscribes to events on the global event bus
func SubscribeToEvents(handler EventHandler, events ...EventType) (string, error) {
	return GetGlobalEventBus().Subscribe(handler, events...)
}

// UnsubscribeFromEvents unsubscribes from the global event bus
func UnsubscribeFromEvents(id string) error {
	return GetGlobalEventBus().Unsubscribe(id)
}
