package observability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// OTELConfig holds OpenTelemetry configuration
type OTELConfig = Config

// OTELTracer implements the Tracer interface with OpenTelemetry backend
type OTELTracer struct {
	tracer trace.Tracer
}

// OTELSpan wraps OpenTelemetry span
type OTELSpan struct {
	span trace.Span
}

func (s *OTELSpan) End() {
	s.span.End()
}

func (s *OTELSpan) RecordError(err error) {
	s.span.RecordError(err)
}

func (s *OTELSpan) SetAttribute(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func (t *OTELTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := t.tracer.Start(ctx, name)
	return ctx, &OTELSpan{span: span}
}

// TracerProvider holds the OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	shutdown func(context.Context) error
}

// NewTracerProvider creates and configures an OpenTelemetry tracer provider
func NewTracerProvider(ctx context.Context, cfg Config) (*TracerProvider, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		slog.Info("OpenTelemetry disabled or no OTLP endpoint configured, using no-op tracer")
		// Set up a no-op tracer provider for consistency
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		return &TracerProvider{}, nil
	}

	// Create OTLP trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // Allow HTTP for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set W3C trace context propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("OpenTelemetry tracer provider initialized",
		slog.String("service", cfg.ServiceName),
		slog.String("endpoint", cfg.OTLPEndpoint),
	)

	return &TracerProvider{
		provider: tp,
		shutdown: tp.Shutdown,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.shutdown != nil {
		return tp.shutdown(ctx)
	}
	return nil
}

// InitGlobalTracer initializes the global tracer based on configuration
func InitGlobalTracer(ctx context.Context, cfg Config) error {
	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		return err
	}

	// Store for graceful shutdown
	// Note: In production, this should be stored in a variable accessible during shutdown
	_ = tp

	// Set global tracer
	GlobalTracer = &OTELTracer{
		tracer: otel.Tracer(cfg.ServiceName),
	}

	slog.Info("Global tracer initialized",
		slog.String("service", cfg.ServiceName),
		slog.Bool("enabled", cfg.Enabled),
	)

	return nil
}

// StartSpanWithAttributes starts a span with initial attributes
func StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, Span) {
	ctx, span := GlobalTracer.StartSpan(ctx, name)
	if span != nil {
		for k, v := range attrs {
			span.SetAttribute(k, v)
		}
	}
	return ctx, span
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(
			attribute.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
		))
		for k, v := range attrs {
			switch val := v.(type) {
			case string:
				span.SetAttributes(attribute.String(k, val))
			case int:
				span.SetAttributes(attribute.Int(k, val))
			case int64:
				span.SetAttributes(attribute.Int64(k, val))
			case float64:
				span.SetAttributes(attribute.Float64(k, val))
			case bool:
				span.SetAttributes(attribute.Bool(k, val))
			default:
				span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", val)))
			}
		}
	}
}

// InjectTraceContext injects trace context into a map for propagation
func InjectTraceContext(ctx context.Context) map[string]string {
	carrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(carrier))
	return carrier
}

// ExtractTraceContext extracts trace context from a map
func ExtractTraceContext(ctx context.Context, carrier map[string]string) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
}

// GetTraceID extracts the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
