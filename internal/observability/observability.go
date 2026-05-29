package observability

import (
	"context"
	"time"
)

// Span represents an active telemetry span
type Span interface {
	End()
	RecordError(err error)
	SetAttribute(key string, value interface{})
}

// Tracer defines the interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// MetricsCollector defines the interface for system performance counters
type MetricsCollector interface {
	IncrementCounter(name string, tags map[string]string)
	ObserveHistogram(name string, value float64, tags map[string]string)
}

type NoOpSpan struct{}

func (s *NoOpSpan) End()                                       {}
func (s *NoOpSpan) RecordError(err error)                      {}
func (s *NoOpSpan) SetAttribute(key string, value interface{}) {}

type NoOpTracer struct{}

func (t *NoOpTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, &NoOpSpan{}
}

type NoOpMetricsCollector struct{}

func (m *NoOpMetricsCollector) IncrementCounter(name string, tags map[string]string)                {}
func (m *NoOpMetricsCollector) ObserveHistogram(name string, value float64, tags map[string]string) {}

// Global trackers
var (
	GlobalTracer  Tracer           = &NoOpTracer{}
	GlobalMetrics MetricsCollector = &NoOpMetricsCollector{}
)

// ObserveDuration records operation timings
func ObserveDuration(name string, startTime time.Time, tags map[string]string) {
	duration := time.Since(startTime).Seconds()
	GlobalMetrics.ObserveHistogram(name, duration, tags)
}
