package eventbus

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// LoggingEventHandler wraps an EventHandler with structured logging
type LoggingEventHandler struct {
	logger   *slog.Logger
	wrapped  EventHandler
	clientID string
}

// NewLoggingEventHandler creates a new logging wrapper for event handlers
func NewLoggingEventHandler(logger *slog.Logger, clientID string, handler EventHandler) *LoggingEventHandler {
	return &LoggingEventHandler{
		logger:   logger,
		wrapped:  handler,
		clientID: clientID,
	}
}

// Handle implements EventHandler with logging
func (h *LoggingEventHandler) Handle(ctx context.Context, event *Event) error {
	h.logger.Info("Event received by handler",
		slog.String("client_id", h.clientID),
		slog.String("event_id", event.ID),
		slog.String("event_type", string(event.Type)),
		slog.String("source_agent", event.SourceAgent),
	)

	if h.wrapped != nil {
		return h.wrapped(ctx, event)
	}

	return nil
}

// EventMetrics tracks event bus metrics
type EventMetrics struct {
	EventsPublished    atomic.Int64
	EventsDelivered    atomic.Int64
	EventsFailed       atomic.Int64
	DLQEntries        atomic.Int64
	ActiveSubscriptions atomic.Int64
}

// NewEventMetrics creates a new metrics tracker
func NewEventMetrics() *EventMetrics {
	return &EventMetrics{}
}

// RecordEventPublished increments the published counter
func (m *EventMetrics) RecordEventPublished() {
	m.EventsPublished.Add(1)
}

// RecordEventDelivery increments the delivered counter
func (m *EventMetrics) RecordEventDelivery() {
	m.EventsDelivered.Add(1)
}

// RecordEventFailure increments the failed counter
func (m *EventMetrics) RecordEventFailure() {
	m.EventsFailed.Add(1)
}

// RecordDLQEntry increments the DLQ counter
func (m *EventMetrics) RecordDLQEntry() {
	m.DLQEntries.Add(1)
}

// RecordSubscription registers an active subscription
func (m *EventMetrics) RecordSubscription() {
	m.ActiveSubscriptions.Add(1)
}

// RemoveSubscription unregisters a subscription
func (m *EventMetrics) RemoveSubscription() {
	m.ActiveSubscriptions.Add(-1)
}

// GetMetrics returns a snapshot of current metrics
func (m *EventMetrics) GetMetrics() map[string]int64 {
	return map[string]int64{
		"events_published":     m.EventsPublished.Load(),
		"events_delivered":     m.EventsDelivered.Load(),
		"events_failed":        m.EventsFailed.Load(),
		"dlq_entries":          m.DLQEntries.Load(),
		"active_subscriptions": m.ActiveSubscriptions.Load(),
	}
}
