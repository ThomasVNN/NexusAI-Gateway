package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Bus defines the interface for the real-time agent event bus
type Bus interface {
	// Publish sends an event to the event bus
	Publish(ctx context.Context, event *Event) error

	// Subscribe creates a new subscription for receiving events
	Subscribe(ctx context.Context, sub *Subscription) error

	// Unsubscribe removes a subscription
	Unsubscribe(ctx context.Context, subscriptionID string) error

	// GetSubscriptions returns all active subscriptions
	GetSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error)

	// GetDLQEntries returns entries from the dead letter queue
	GetDLQEntries(ctx context.Context, status DLQStatus, limit int) ([]*DLQEntry, error)

	// RetryDLQEntry attempts to retry a DLQ entry
	RetryDLQEntry(ctx context.Context, entryID string) error

	// PurgeDLQEntry removes a DLQ entry permanently
	PurgeDLQEntry(ctx context.Context, entryID string) error

	// HealthCheck returns the health status of the event bus
	HealthCheck(ctx context.Context) error

	// Close gracefully shuts down the event bus
	Close() error
}

// EventHandler is a callback function for handling received events
type EventHandler func(ctx context.Context, event *Event) error

// BusConfig holds configuration for creating a new event bus
type BusConfig struct {
	// Config contains the event bus configuration
	Config *EventBusConfig
	// Handler is called when events are received (for subscribe mode)
	Handler EventHandler
	// Logger is the structured logger for the event bus
	Logger *slog.Logger
	// OnDLQError is called when an event fails processing and goes to DLQ
	OnDLQError func(event *Event, err error)
}

// DefaultBusConfig creates a default configuration for local/in-memory mode
func DefaultBusConfig() *BusConfig {
	return &BusConfig{
		Config: &EventBusConfig{
			Provider:         "memory",
			DLQMaxRetries:    3,
			DefaultTTL:       24 * time.Hour,
			EnableOrdering:   true,
			OrderingKeyField: "correlation_id",
		},
		Logger: slog.Default(),
	}
}

// localEventBus implements Bus using in-memory pub/sub
// Used for development and when external brokers are unavailable
type localEventBus struct {
	config     *BusConfig
	logger     *slog.Logger
	subs       map[string]*Subscription      // subscriptionID -> subscription
	subTopics  map[EventType]map[string]bool // eventType -> set of subscriptionIDs
	handlers   map[string]EventHandler       // subscriptionID -> handler
	dlq        map[string]*DLQEntry          // entryID -> DLQ entry
	eventOrder *EventSequencer               // for ordering guarantees
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewBus creates a new event bus based on configuration
// Falls back to in-memory implementation when external brokers are unavailable
func NewBus(ctx context.Context, cfg *BusConfig) (Bus, error) {
	if cfg == nil {
		cfg = DefaultBusConfig()
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	childCtx, cancel := context.WithCancel(ctx)

	bus := &localEventBus{
		config:     cfg,
		logger:     cfg.Logger,
		subs:       make(map[string]*Subscription),
		subTopics:  make(map[EventType]map[string]bool),
		handlers:   make(map[string]EventHandler),
		dlq:        make(map[string]*DLQEntry),
		eventOrder: NewEventSequencer(),
		ctx:        childCtx,
		cancel:     cancel,
	}

	// Initialize topic maps for all event types
	for _, eventType := range []EventType{
		EventTypeIntent,
		EventTypeDecision,
		EventTypeApproval,
		EventTypeRejection,
		EventTypeError,
	} {
		bus.subTopics[eventType] = make(map[string]bool)
	}

	bus.logger.Info("Event bus initialized",
		slog.String("provider", cfg.Config.Provider),
		slog.Int("dlq_max_retries", cfg.Config.DLQMaxRetries),
		slog.Bool("ordering_enabled", cfg.Config.EnableOrdering),
	)

	return bus, nil
}

// Publish sends an event to the event bus
func (b *localEventBus) Publish(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	if event.ID == "" {
		event.ID = generateEventID()
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Check TTL
	if event.IsExpired() {
		b.logger.Warn("Discarding expired event",
			slog.String("event_id", event.ID),
			slog.String("event_type", string(event.Type)),
			slog.Duration("ttl", event.TTL),
		)
		return fmt.Errorf("event has expired")
	}

	b.mu.Lock()

	// Assign sequence number if ordering is enabled
	if b.config.Config.EnableOrdering {
		key := event.CorrelationID
		if key == "" {
			key = event.SourceAgent
		}
		event.SequenceNumber = b.eventOrder.Next(key)
	}

	// Find matching subscriptions
	subscribers := b.findSubscribers(event)

	b.mu.Unlock()

	// Publish to all subscribers
	eventJSON, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	for _, subID := range subscribers {
		b.mu.RLock()
		handler, ok := b.handlers[subID]
		sub := b.subs[subID]
		b.mu.RUnlock()

		if !ok || !sub.Active {
			continue
		}

		// Apply filters
		if !b.matchesFilter(event, sub.Filter) {
			continue
		}

		// Deliver event asynchronously
		b.wg.Add(1)
		go func(subID string, handler EventHandler, evt *Event, data []byte) {
			defer b.wg.Done()

			parsedEvent, err := ParseEvent(data)
			if err != nil {
				b.logger.Error("Failed to parse event for delivery",
					slog.String("subscription_id", subID),
					slog.String("error", err.Error()),
				)
				b.sendToDLQ(evt, fmt.Errorf("failed to parse event: %w", err))
				return
			}

			deliverCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := handler(deliverCtx, parsedEvent); err != nil {
				b.logger.Error("Event handler failed",
					slog.String("subscription_id", subID),
					slog.String("event_id", evt.ID),
					slog.String("error", err.Error()),
				)
				b.sendToDLQ(evt, err)
			}
		}(subID, handler, event, eventJSON)
	}

	b.logger.Debug("Event published",
		slog.String("event_id", event.ID),
		slog.String("event_type", string(event.Type)),
		slog.Int("subscriber_count", len(subscribers)),
	)

	return nil
}

// findSubscribers returns all subscription IDs that match the event
func (b *localEventBus) findSubscribers(event *Event) []string {
	var subs []string

	// Direct match on event type
	for subID := range b.subTopics[event.Type] {
		subs = append(subs, subID)
	}

	// Check wildcard subscriptions
	for subID := range b.subTopics[EventType("")] {
		subs = append(subs, subID)
	}

	// Check target-specific subscriptions
	if event.TargetAgent != "" {
		for _, sub := range b.subs {
			if sub.TargetAgent == event.TargetAgent {
				for _, et := range sub.EventTypes {
					if et == event.Type || et == "" {
						subs = append(subs, sub.ID)
						break
					}
				}
			}
		}
	}

	return subs
}

// matchesFilter checks if an event matches subscription filters
func (b *localEventBus) matchesFilter(event *Event, filter *Filter) bool {
	if filter == nil {
		return true
	}

	// Check source agents
	if len(filter.SourceAgents) > 0 {
		found := false
		for _, agent := range filter.SourceAgents {
			if agent == event.SourceAgent {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check priority threshold
	if filter.PriorityThreshold > 0 && event.Priority < filter.PriorityThreshold {
		return false
	}

	// Check metadata filters
	for key, expected := range filter.MetadataFilters {
		actual, ok := event.Metadata[key]
		if !ok {
			return false
		}
		if actual != expected {
			return false
		}
	}

	return true
}

// sendToDLQ moves a failed event to the dead letter queue
func (b *localEventBus) sendToDLQ(event *Event, err error) {
	entry := NewDLQEntry(event, err, b.config.Config.DLQMaxRetries)

	b.mu.Lock()
	b.dlq[entry.ID] = entry
	b.mu.Unlock()

	b.logger.Warn("Event sent to DLQ",
		slog.String("entry_id", entry.ID),
		slog.String("event_id", event.ID),
		slog.String("event_type", string(event.Type)),
		slog.String("error", err.Error()),
	)

	// Call DLQ callback if configured
	if b.config.OnDLQError != nil {
		b.config.OnDLQError(event, err)
	}
}

// Subscribe creates a new subscription for receiving events
func (b *localEventBus) Subscribe(ctx context.Context, sub *Subscription) error {
	if sub == nil {
		return fmt.Errorf("subscription cannot be nil")
	}

	if sub.ID == "" {
		sub.ID = GenerateSubscriptionID()
	}

	if sub.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}

	if len(sub.EventTypes) == 0 {
		return fmt.Errorf("at least one event_type is required")
	}

	sub.CreatedAt = time.Now().UTC()
	sub.Active = true

	b.mu.Lock()
	defer b.mu.Unlock()

	// Register subscription
	b.subs[sub.ID] = sub

	// Register for each event type
	for _, eventType := range sub.EventTypes {
		if _, exists := b.subTopics[eventType]; !exists {
			b.subTopics[eventType] = make(map[string]bool)
		}
		b.subTopics[eventType][sub.ID] = true
	}

	// Register handler
	if b.config.Handler != nil {
		b.handlers[sub.ID] = b.config.Handler
	}

	b.logger.Info("Subscription created",
		slog.String("subscription_id", sub.ID),
		slog.String("client_id", sub.ClientID),
		slog.Any("event_types", sub.EventTypes),
	)

	return nil
}

// Unsubscribe removes a subscription
func (b *localEventBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subs[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	// Remove from topic mappings
	for _, eventType := range sub.EventTypes {
		delete(b.subTopics[eventType], subscriptionID)
	}

	// Remove handler
	delete(b.handlers, subscriptionID)

	// Mark as inactive (don't delete to preserve history)
	sub.Active = false

	b.logger.Info("Subscription removed",
		slog.String("subscription_id", subscriptionID),
	)

	return nil
}

// GetSubscriptions returns all active subscriptions for a client
func (b *localEventBus) GetSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var subs []*Subscription
	for _, sub := range b.subs {
		if sub.ClientID == clientID && sub.Active {
			subs = append(subs, sub)
		}
	}

	return subs, nil
}

// GetDLQEntries returns entries from the dead letter queue
func (b *localEventBus) GetDLQEntries(ctx context.Context, status DLQStatus, limit int) ([]*DLQEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var entries []*DLQEntry
	count := 0

	for _, entry := range b.dlq {
		if status != "" && entry.Status != status {
			continue
		}

		entries = append(entries, entry)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return entries, nil
}

// RetryDLQEntry attempts to retry a DLQ entry
func (b *localEventBus) RetryDLQEntry(ctx context.Context, entryID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, exists := b.dlq[entryID]
	if !exists {
		return fmt.Errorf("DLQ entry not found: %s", entryID)
	}

	if !entry.ShouldRetry() {
		return fmt.Errorf("entry has exceeded max retries (%d)", entry.MaxRetries)
	}

	entry.RecordRetry(nil)
	delete(b.dlq, entryID)

	b.logger.Info("DLQ entry will be retried",
		slog.String("entry_id", entryID),
		slog.Int("retry_count", entry.RetryCount),
	)

	return nil
}

// PurgeDLQEntry removes a DLQ entry permanently
func (b *localEventBus) PurgeDLQEntry(ctx context.Context, entryID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.dlq[entryID]; !exists {
		return fmt.Errorf("DLQ entry not found: %s", entryID)
	}

	delete(b.dlq, entryID)

	b.logger.Info("DLQ entry purged",
		slog.String("entry_id", entryID),
	)

	return nil
}

// HealthCheck returns the health status of the event bus
func (b *localEventBus) HealthCheck(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.ctx.Err() != nil {
		return fmt.Errorf("event bus is shutting down")
	}

	return nil
}

// Close gracefully shuts down the event bus
func (b *localEventBus) Close() error {
	b.logger.Info("Shutting down event bus...")

	b.cancel()

	// Wait for pending deliveries
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("Event bus shutdown complete")
	case <-time.After(30 * time.Second):
		b.logger.Warn("Event bus shutdown timed out with pending deliveries")
	}

	return nil
}
