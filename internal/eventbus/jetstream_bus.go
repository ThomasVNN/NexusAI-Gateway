package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// JetStreamConfig holds NATS JetStream-specific configuration
type JetStreamConfig struct {
	StreamName   string
	Subject      string
	ConsumerName string
	MaxBytes     int64
	MaxAge       time.Duration
	Storage      jetstream.StorageType
	Replicas     int
	Retention    jetstream.RetentionPolicy
	Duplicates   time.Duration
}

// jetStreamBus implements event publishing with JetStream persistence
type jetStreamBus struct {
	*natsBus
	js       jetstream.JetStream
	jsConfig *JetStreamConfig
}

// NewJetStreamBus creates a new event bus backed by NATS JetStream
func NewJetStreamBus(ctx context.Context, cfg *BusConfig, opts ...NATSOption) (Bus, error) {
	// First create the base NATS bus
	natsBusInstance, err := NewNATSBus(ctx, cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS bus: %w", err)
	}

	jsBus := &jetStreamBus{
		natsBus: natsBusInstance.(*natsBus),
		jsConfig: &JetStreamConfig{
			StreamName:   "nexusai-events",
			Subject:      "events.>",
			ConsumerName: "nexusai-consumer",
			MaxBytes:     1 * 1024 * 1024 * 1024, // 1GB
			MaxAge:       7 * 24 * time.Hour,     // 7 days
			Storage:      jetstream.FileStorage,
			Replicas:     1,
			Retention:    jetstream.InterestPolicy,
			Duplicates:   2 * time.Minute,
		},
	}

	// Initialize JetStream
	if err := jsBus.initJetStream(ctx); err != nil {
		jsBus.logger.Warn("JetStream initialization failed, falling back to core NATS",
			slog.String("error", err.Error()))
		return jsBus.natsBus, nil
	}

	jsBus.logger.Info("JetStream event bus initialized",
		slog.String("stream", jsBus.jsConfig.StreamName),
		slog.String("subject", jsBus.jsConfig.Subject),
	)

	return jsBus, nil
}

// initJetStream sets up JetStream stream and consumer
func (b *jetStreamBus) initJetStream(ctx context.Context) error {
	if b.conn == nil {
		return fmt.Errorf("NATS connection is required")
	}

	jsContext, err := jetstream.New(b.conn)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create stream if it doesn't exist
	_, err = jsContext.Stream(ctx, b.jsConfig.StreamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = jsContext.CreateStream(ctx, jetstream.StreamConfig{
			Name:        b.jsConfig.StreamName,
			Subjects:    []string{b.jsConfig.Subject},
			MaxBytes:    b.jsConfig.MaxBytes,
			MaxAge:      b.jsConfig.MaxAge,
			Storage:     b.jsConfig.Storage,
			Replicas:    b.jsConfig.Replicas,
			Retention:   b.jsConfig.Retention,
			Duplicates:  b.jsConfig.Duplicates,
			AllowDirect: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		b.logger.Info("Created JetStream stream",
			slog.String("stream", b.jsConfig.StreamName))
	}

	b.js = jsContext

	// Create consumer if it doesn't exist
	_, err = b.js.CreateOrUpdateConsumer(ctx, b.jsConfig.StreamName, jetstream.ConsumerConfig{
		Name:          b.jsConfig.ConsumerName,
		Durable:       b.jsConfig.ConsumerName,
		FilterSubject:  b.jsConfig.Subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    3,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		b.logger.Warn("Failed to create consumer (may already exist)",
			slog.String("error", err.Error()))
	}

	return nil
}

// Publish sends an event to JetStream with persistence
func (b *jetStreamBus) Publish(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Generate ID and timestamp
	if event.ID == "" {
		event.ID = GenerateEventID()
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

	// Determine subject
	subject := fmt.Sprintf("events.%s", event.Type)

	// Publish to JetStream for persistence
	if b.js != nil {
		_, err = b.js.Publish(ctx, subject, data, jetstream.WithMsgID(event.ID))
		if err != nil {
			b.sendToDLQ(event, err)
			return fmt.Errorf("failed to publish to JetStream: %w", err)
		}
	} else {
		// Fallback to core NATS
		err = b.conn.Publish(subject, data)
		if err != nil {
			b.sendToDLQ(event, err)
			return fmt.Errorf("failed to publish event: %w", err)
		}
	}

	b.logger.Debug("Event published to JetStream",
		slog.String("event_id", event.ID),
		slog.String("subject", subject),
	)

	return nil
}

// GetStreamInfo returns information about the JetStream stream
func (b *jetStreamBus) GetStreamInfo(ctx context.Context) (jetstream.Stream, error) {
	if b.js == nil {
		return nil, fmt.Errorf("JetStream is not initialized")
	}
	return b.js.Stream(ctx, b.jsConfig.StreamName)
}

// CreateConsumer creates a new consumer on the stream
func (b *jetStreamBus) CreateConsumer(ctx context.Context, name string, config jetstream.ConsumerConfig) error {
	if b.js == nil {
		return fmt.Errorf("JetStream is not initialized")
	}
	_, err := b.js.CreateOrUpdateConsumer(ctx, b.jsConfig.StreamName, config)
	return err
}
