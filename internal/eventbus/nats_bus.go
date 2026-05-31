package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// natsBus implements Bus using NATS JetStream for production deployments
type natsBus struct {
	config     *BusConfig
	logger     *slog.Logger
	conn       *nats.Conn
	js         jetstream.JetStream
	ctx        context.Context
	cancel     context.CancelFunc
	subs       map[string]*Subscription
	handlers   map[string]EventHandler
	jsCons     map[string]jetstream.Consumer
	dlq        map[string]*DLQEntry
	eventOrder *EventSequencer
	mu         sync.RWMutex
	wg         sync.WaitGroup
}

// NATSOption is a functional option for NATS bus configuration
type NATSOption func(*natsBus) error

// WithNATSConnection sets a pre-configured NATS connection
func WithNATSConnection(conn *nats.Conn) NATSOption {
	return func(b *natsBus) error {
		b.conn = conn
		return nil
	}
}

// NewNATSBus creates a new event bus backed by NATS JetStream
func NewNATSBus(ctx context.Context, cfg *BusConfig, opts ...NATSOption) (Bus, error) {
	if cfg == nil || cfg.Config == nil {
		return nil, fmt.Errorf("configuration is required")
	}

	natsConfig := cfg.Config.NATSConfig
	if natsConfig == nil {
		return nil, fmt.Errorf("NATS configuration is required")
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	childCtx, cancel := context.WithCancel(ctx)

	bus := &natsBus{
		config:     cfg,
		logger:     cfg.Logger,
		subs:       make(map[string]*Subscription),
		handlers:   make(map[string]EventHandler),
		jsCons:     make(map[string]jetstream.Consumer),
		dlq:        make(map[string]*DLQEntry),
		eventOrder: NewEventSequencer(),
		ctx:       childCtx,
		cancel:    cancel,
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(bus); err != nil {
			cancel()
			return nil, err
		}
	}

	// Connect to NATS if not provided
	if bus.conn == nil {
		nc, err := nats.Connect(natsConfig.URLs[0],
			nats.Name("nexusai-event-bus"),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(2*time.Second),
		)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to connect to NATS: %w", err)
		}
		bus.conn = nc
	}

	// Create JetStream context
	js, err := jetstream.New(bus.conn)
	if err != nil {
		bus.conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	bus.js = js

	// Ensure stream exists
	if err := bus.ensureStream(ctx, natsConfig); err != nil {
		bus.Close()
		return nil, err
	}

	bus.logger.Info("NATS JetStream event bus initialized",
		slog.String("url", natsConfig.URLs[0]),
		slog.String("stream", natsConfig.StreamName),
	)

	return bus, nil
}

// ensureStream creates the JetStream stream if it doesn't exist
func (b *natsBus) ensureStream(ctx context.Context, cfg *NATSConfig) error {
	spec := jetstream.StreamConfig{
		Name:        cfg.StreamName,
		Description: "NexusAI Agent Event Bus Stream",
		Subjects:    []string{"events.>"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:     24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	}

	_, err := b.js.CreateOrUpdateStream(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	return nil
}

// Publish sends an event to NATS
func (b *natsBus) Publish(ctx context.Context, event *Event) error {
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
		return fmt.Errorf("event has expired")
	}

	// Assign sequence number for ordering
	if b.config.Config.EnableOrdering {
		key := event.CorrelationID
		if key == "" {
			key = event.SourceAgent
		}
		event.SequenceNumber = b.eventOrder.Next(key)
	}

	// Serialize event
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine subject based on event type
	subject := fmt.Sprintf("events.%s", event.Type)

	// Publish with headers
	hdrs := nats.Header{
		"X-Event-ID":        []string{event.ID},
		"X-Event-Type":      []string{string(event.Type)},
		"X-Event-Priority":  []string{fmt.Sprintf("%d", event.Priority)},
		"X-Source-Agent":    []string{event.SourceAgent},
		"X-Sequence-Number": []string{fmt.Sprintf("%d", event.SequenceNumber)},
	}

	if event.CorrelationID != "" {
		hdrs["X-Correlation-ID"] = []string{event.CorrelationID}
	}

	_, err = b.js.Publish(ctx, subject, data, jetstream.WithHeaders(hdrs))
	if err != nil {
		b.sendToDLQ(event, err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	b.logger.Debug("Event published to NATS",
		slog.String("event_id", event.ID),
		slog.String("subject", subject),
	)

	return nil
}

// sendToDLQ moves a failed event to the dead letter queue
func (b *natsBus) sendToDLQ(event *Event, err error) {
	entry := NewDLQEntry(event, err, b.config.Config.DLQMaxRetries)

	b.mu.Lock()
	b.dlq[entry.ID] = entry
	b.mu.Unlock()

	b.logger.Warn("Event sent to DLQ",
		slog.String("entry_id", entry.ID),
		slog.String("event_id", event.ID),
	)

	if b.config.OnDLQError != nil {
		b.config.OnDLQError(event, err)
	}
}

// Subscribe creates a NATS subscription for receiving events
func (b *natsBus) Subscribe(ctx context.Context, sub *Subscription) error {
	if sub == nil {
		return fmt.Errorf("subscription cannot be nil")
	}

	if sub.ID == "" {
		sub.ID = GenerateSubscriptionID()
	}

	sub.CreatedAt = time.Now().UTC()
	sub.Active = true

	b.mu.Lock()
	b.subs[sub.ID] = sub
	b.handlers[sub.ID] = b.config.Handler
	b.mu.Unlock()

	// Create durable consumer for this subscription
	consumerName := fmt.Sprintf("consumer_%s", sub.ID)

	// Build subject filter
	var subjects []string
	if len(sub.EventTypes) == 0 || (len(sub.EventTypes) == 1 && sub.EventTypes[0] == "") {
		subjects = []string{"events.>"}
	} else {
		for _, et := range sub.EventTypes {
			subjects = append(subjects, fmt.Sprintf("events.%s", et))
		}
	}

	cons, err := b.js.AddConsumer(ctx, b.config.Config.NATSConfig.StreamName, jetstream.ConsumerConfig{
		Name:         consumerName,
		Durable:      consumerName,
		Description:  fmt.Sprintf("Consumer for client %s", sub.ClientID),
		FilterSubject: subjects[0],
		AckWait:     30 * time.Second,
		MaxDeliver:  b.config.Config.DLQMaxRetries,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	b.mu.Lock()
	b.jsCons[sub.ID] = cons
	b.mu.Unlock()

	// Start consuming in background
	b.wg.Add(1)
	go b.consumeSubscription(ctx, sub.ID, cons)

	b.logger.Info("NATS subscription created",
		slog.String("subscription_id", sub.ID),
		slog.String("consumer", consumerName),
	)

	return nil
}

// consumeSubscription processes messages for a subscription
func (b *natsBus) consumeSubscription(ctx context.Context, subID string, cons jetstream.Consumer) {
	defer b.wg.Done()

	b.mu.RLock()
	sub := b.subs[subID]
	handler := b.handlers[subID]
	b.mu.RUnlock()

	if sub == nil || handler == nil {
		return
	}

	msgs, err := cons.Consume(b.ctx)
	if err != nil {
		b.logger.Error("Failed to start consumer",
			slog.String("subscription_id", subID),
			slog.String("error", err.Error()),
		)
		return
	}

	for {
		select {
		case <-b.ctx.Done():
			return
		case msg, ok := <-msgs.MsgChan():
			if !ok {
				return
			}

			event, err := ParseEvent(msg.Data())
			if err != nil {
				b.logger.Error("Failed to parse event",
					slog.String("subscription_id", subID),
					slog.String("error", err.Error()),
				)
				msg.Nak()
				continue
			}

			// Apply filters
			if !b.matchesFilter(event, sub.Filter) {
				msg.Ack()
				continue
			}

			// Deliver to handler
			deliverCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := handler(deliverCtx, event); err != nil {
				b.logger.Error("Handler failed for event",
					slog.String("subscription_id", subID),
					slog.String("event_id", event.ID),
					slog.String("error", err.Error()),
				)
				msg.Nak()
			} else {
				msg.Ack()
			}
			cancel()
		}
	}
}

// matchesFilter checks if an event matches subscription filters
func (b *natsBus) matchesFilter(event *Event, filter *Filter) bool {
	if filter == nil {
		return true
	}

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

	if filter.PriorityThreshold > 0 && event.Priority < filter.PriorityThreshold {
		return false
	}

	for key, expected := range filter.MetadataFilters {
		actual, ok := event.Metadata[key]
		if !ok || actual != expected {
			return false
		}
	}

	return true
}

// Unsubscribe removes a subscription
func (b *natsBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, exists := b.subs[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	// Delete consumer
	if cons, ok := b.jsCons[subscriptionID]; ok {
		_ = cons.Delete(ctx)
		delete(b.jsCons, subscriptionID)
	}

	delete(b.handlers, subscriptionID)
	sub.Active = false

	b.logger.Info("NATS subscription removed",
		slog.String("subscription_id", subscriptionID),
	)

	return nil
}

// GetSubscriptions returns all subscriptions for a client
func (b *natsBus) GetSubscriptions(ctx context.Context, clientID string) ([]*Subscription, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var subs []*Subscription
	for _, sub := range b.subs {
		if sub.ClientID == clientID {
			subs = append(subs, sub)
		}
	}

	return subs, nil
}

// GetDLQEntries returns entries from the dead letter queue
func (b *natsBus) GetDLQEntries(ctx context.Context, status DLQStatus, limit int) ([]*DLQEntry, error) {
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
func (b *natsBus) RetryDLQEntry(ctx context.Context, entryID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, exists := b.dlq[entryID]
	if !exists {
		return fmt.Errorf("DLQ entry not found: %s", entryID)
	}

	if !entry.ShouldRetry() {
		return fmt.Errorf("entry has exceeded max retries")
	}

	entry.RecordRetry(nil)
	delete(b.dlq, entryID)

	// Re-publish the event
	_, err := b.js.Publish(ctx, fmt.Sprintf("events.%s", entry.Event.Type), entry.Event.Payload)
	if err != nil {
		b.dlq[entry.ID] = entry
		return fmt.Errorf("failed to republish event: %w", err)
	}

	return nil
}

// PurgeDLQEntry removes a DLQ entry
func (b *natsBus) PurgeDLQEntry(ctx context.Context, entryID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.dlq, entryID)

	b.logger.Info("DLQ entry purged",
		slog.String("entry_id", entryID),
	)

	return nil
}

// HealthCheck returns the health status of the NATS bus
func (b *natsBus) HealthCheck(ctx context.Context) error {
	if !b.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not established")
	}

	if b.ctx.Err() != nil {
		return fmt.Errorf("event bus is shutting down")
	}

	return nil
}

// Close shuts down the NATS bus
func (b *natsBus) Close() error {
	b.logger.Info("Shutting down NATS event bus...")
	b.cancel()

	b.wg.Wait()

	if b.conn != nil {
		b.conn.Close()
	}

	return nil
}

// PublishBatch publishes multiple events in a single batch
func (b *natsBus) PublishBatch(ctx context.Context, events []*Event) error {
	var pubOpts []jetstream.PublishOpt

	for _, event := range events {
		if event.ID == "" {
			event.ID = generateEventID()
		}
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now().UTC()
		}

		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %s: %w", event.ID, err)
		}

		subject := fmt.Sprintf("events.%s", event.Type)
		if _, err := b.js.Publish(ctx, subject, data, pubOpts...); err != nil {
			return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
		}
	}

	return nil
}
