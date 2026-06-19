package telemetry

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TraceEvent represents a single trace event
type TraceEvent struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	Service    string            `json:"service"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	DurationMs float64           `json:"duration_ms"`
	Status     string            `json:"status"`
	Tags       map[string]string `json:"tags,omitempty"`
}

// Tracer provides distributed tracing
type Tracer struct {
	mu       sync.RWMutex
	spans    map[string]*TraceEvent
	exporter TraceExporter
}

// TraceExporter exports traces to a collector
type TraceExporter interface {
	Export(spans []*TraceEvent) error
}

// NoopExporter is a no-op trace exporter
type NoopExporter struct{}

func (e *NoopExporter) Export(spans []*TraceEvent) error {
	return nil
}

// NewTracer creates a new tracer
func NewTracer() *Tracer {
	return &Tracer{
		spans:    make(map[string]*TraceEvent),
		exporter: &NoopExporter{},
	}
}

// SetExporter sets the trace exporter
func (t *Tracer) SetExporter(exporter TraceExporter) {
	t.exporter = exporter
}

// Span represents an active trace span
type Span struct {
	tracer   *Tracer
	id       string
	parentID string
	name     string
	service  string
	start    time.Time
	tags     map[string]string
	status   string
}

// StartSpan starts a new trace span
func (t *Tracer) StartSpan(ctx context.Context, name, service string) (context.Context, *Span) {
	span := &Span{
		tracer:  t,
		id:      generateID(),
		name:    name,
		service: service,
		start:   time.Now(),
		tags:    make(map[string]string),
		status:  "ok",
	}

	// Get parent span from context if available
	if parentSpan, ok := ctx.Value("span").(*Span); ok {
		span.parentID = parentSpan.id
	}

	// Add span to context
	ctx = context.WithValue(ctx, "span", span)

	return ctx, span
}

// SetTag sets a tag on the span
func (s *Span) SetTag(key, value string) {
	s.tags[key] = value
}

// SetStatus sets the span status
func (s *Span) SetStatus(status string) {
	s.status = status
}

// End ends the span
func (s *Span) End() {
	span := &TraceEvent{
		TraceID:    s.id,
		SpanID:     s.id,
		ParentID:   s.parentID,
		Name:       s.name,
		Service:    s.service,
		StartTime:  s.start,
		EndTime:    time.Now(),
		DurationMs: float64(time.Since(s.start).Milliseconds()),
		Status:     s.status,
		Tags:       s.tags,
	}

	s.tracer.mu.Lock()
	s.tracer.spans[s.id] = span
	s.tracer.mu.Unlock()

	// Export span
	go func() {
		if err := s.tracer.exporter.Export([]*TraceEvent{span}); err != nil {
			log.Printf("Failed to export span: %v", err)
		}
	}()
}

// GetSpans returns all recorded spans
func (t *Tracer) GetSpans() []*TraceEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	spans := make([]*TraceEvent, 0, len(t.spans))
	for _, span := range t.spans {
		spans = append(spans, span)
	}
	return spans
}

// Flush exports all pending spans
func (t *Tracer) Flush() error {
	spans := t.GetSpans()
	return t.exporter.Export(spans)
}

// generateID generates a random ID
func generateID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

// Global tracer instance
var globalTracer *Tracer
var tracerOnce sync.Once

// GetTracer returns the global tracer
func GetTracer() *Tracer {
	tracerOnce.Do(func() {
		globalTracer = NewTracer()
	})
	return globalTracer
}
