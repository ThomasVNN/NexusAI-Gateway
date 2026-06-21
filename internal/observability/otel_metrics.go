package observability

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// MeterProvider holds the OpenTelemetry meter provider for metrics
type MeterProvider struct {
	meter    metric.Meter
	provider *sdkmetric.MeterProvider
	shutdown func(context.Context) error
}

// OTELMetrics holds OpenTelemetry metrics instruments
type OTELMetrics struct {
	// Request metrics
	RequestCount    metric.Int64Counter
	RequestDuration metric.Float64Histogram
	ErrorRate       metric.Float64Counter

	// Latency percentiles (p50, p95, p99)
	LatencyP50 metric.Float64Histogram
	LatencyP95 metric.Float64Histogram
	LatencyP99 metric.Float64Histogram

	// Model-specific metrics
	ModelLatency   metric.Float64Histogram
	ModelTokensIn  metric.Int64Counter
	ModelTokensOut metric.Int64Counter
	ModelCalls     metric.Int64Counter
	ModelErrors    metric.Int64Counter

	// Cache metrics
	CacheHits    metric.Int64Counter
	CacheMisses  metric.Int64Counter
	CacheLatency metric.Float64Histogram

	// DB metrics
	DBQueryLatency metric.Float64Histogram
	DBQueryCount   metric.Int64Counter
}

var globalOTELMetrics *OTELMetrics
var meterProvider *MeterProvider
var metricsProvider *sdkmetric.MeterProvider

// Default latency buckets for histograms (in seconds)
var defaultLatencyBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
var p50Buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5}
var p95Buckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
var p99Buckets = []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// NewMeterProvider creates and configures an OpenTelemetry meter provider
func NewMeterProvider(ctx context.Context, cfg Config) (*MeterProvider, error) {
	if !cfg.Enabled || cfg.OTLPEndpoint == "" {
		slog.Info("OpenTelemetry metrics disabled or no OTLP endpoint configured")
		// Create a no-op meter provider
		provider := sdkmetric.NewMeterProvider()
		otel.SetMeterProvider(provider)
		return &MeterProvider{}, nil
	}

	// Create OTLP metric exporter
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithInsecure(), // Allow HTTP for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
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

	// Create meter provider with OTLP exporter
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(15*time.Second),
		)),
		sdkmetric.WithResource(res),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	slog.Info("OpenTelemetry meter provider initialized",
		slog.String("service", cfg.ServiceName),
		slog.String("endpoint", cfg.OTLPEndpoint),
	)

	return &MeterProvider{
		provider: provider,
		shutdown: provider.Shutdown,
	}, nil
}

// Shutdown gracefully shuts down the meter provider
func (mp *MeterProvider) Shutdown(ctx context.Context) error {
	if mp.shutdown != nil {
		return mp.shutdown(ctx)
	}
	return nil
}

// NewOTELMetrics creates and registers all OpenTelemetry metrics instruments
func NewOTELMetrics(meter metric.Meter) (*OTELMetrics, error) {
	m := &OTELMetrics{}
	var err error

	// Request metrics
	m.RequestCount, err = meter.Int64Counter("gateway.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	m.RequestDuration, err = meter.Float64Histogram("gateway.request.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(defaultLatencyBuckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration histogram: %w", err)
	}

	m.ErrorRate, err = meter.Float64Counter("gateway.errors.rate",
		metric.WithDescription("Error rate counter"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error rate counter: %w", err)
	}

	// Latency percentiles
	m.LatencyP50, err = meter.Float64Histogram("gateway.latency.p50",
		metric.WithDescription("Request latency p50 in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(p50Buckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p50 histogram: %w", err)
	}

	m.LatencyP95, err = meter.Float64Histogram("gateway.latency.p95",
		metric.WithDescription("Request latency p95 in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(p95Buckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p95 histogram: %w", err)
	}

	m.LatencyP99, err = meter.Float64Histogram("gateway.latency.p99",
		metric.WithDescription("Request latency p99 in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(p99Buckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p99 histogram: %w", err)
	}

	// Model metrics
	m.ModelLatency, err = meter.Float64Histogram("gateway.model.latency",
		metric.WithDescription("Model API call latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(defaultLatencyBuckets...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model latency histogram: %w", err)
	}

	m.ModelTokensIn, err = meter.Int64Counter("gateway.model.tokens.in",
		metric.WithDescription("Total prompt tokens consumed"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens in counter: %w", err)
	}

	m.ModelTokensOut, err = meter.Int64Counter("gateway.model.tokens.out",
		metric.WithDescription("Total completion tokens generated"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens out counter: %w", err)
	}

	m.ModelCalls, err = meter.Int64Counter("gateway.model.calls.total",
		metric.WithDescription("Total model API calls"),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model calls counter: %w", err)
	}

	m.ModelErrors, err = meter.Int64Counter("gateway.model.errors.total",
		metric.WithDescription("Total model API errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create model errors counter: %w", err)
	}

	// Cache metrics
	m.CacheHits, err = meter.Int64Counter("gateway.cache.hits",
		metric.WithDescription("Total cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache hits counter: %w", err)
	}

	m.CacheMisses, err = meter.Int64Counter("gateway.cache.misses",
		metric.WithDescription("Total cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache misses counter: %w", err)
	}

	m.CacheLatency, err = meter.Float64Histogram("gateway.cache.latency",
		metric.WithDescription("Cache lookup latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache latency histogram: %w", err)
	}

	// DB metrics
	m.DBQueryLatency, err = meter.Float64Histogram("gateway.db.query.latency",
		metric.WithDescription("Database query latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB query latency histogram: %w", err)
	}

	m.DBQueryCount, err = meter.Int64Counter("gateway.db.query.count",
		metric.WithDescription("Total database queries"),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB query count: %w", err)
	}

	return m, nil
}

// InitOTELMetrics initializes the global OTEL metrics with optional OTLP export
func InitOTELMetrics(ctx context.Context, cfg Config) error {
	// Create meter provider
	mp, err := NewMeterProvider(ctx, cfg)
	if err != nil {
		return err
	}
	meterProvider = mp

	// Create meter
	meter := otel.Meter(cfg.ServiceName)

	// Create metrics
	m, err := NewOTELMetrics(meter)
	if err != nil {
		return err
	}
	globalOTELMetrics = m

	slog.Info("OTEL metrics initialized",
		slog.String("service", cfg.ServiceName),
		slog.Bool("enabled", cfg.Enabled),
	)

	return nil
}

// GetOTELMetrics returns the global OTEL metrics instance
func GetOTELMetrics() *OTELMetrics {
	return globalOTELMetrics
}

// RecordRequest records HTTP request metrics with full OTEL instrumentation
func (m *OTELMetrics) RecordRequest(ctx context.Context, attrs map[string]string, duration time.Duration, statusCode int) {
	attrsMap := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		attrsMap = append(attrsMap, attribute.String(k, v))
	}

	// Record request count
	m.RequestCount.Add(ctx, 1, metric.WithAttributes(attrsMap...))

	// Record duration
	durationSeconds := duration.Seconds()
	m.RequestDuration.Record(ctx, durationSeconds, metric.WithAttributes(attrsMap...))

	// Record latency percentiles
	m.LatencyP50.Record(ctx, durationSeconds, metric.WithAttributes(attrsMap...))
	m.LatencyP95.Record(ctx, durationSeconds, metric.WithAttributes(attrsMap...))
	m.LatencyP99.Record(ctx, durationSeconds, metric.WithAttributes(attrsMap...))

	// Record errors if status code indicates failure
	if statusCode >= 400 {
		m.ErrorRate.Add(ctx, 1, metric.WithAttributes(attrsMap...))
	}
}

// RecordModelCall records model API call metrics
func (m *OTELMetrics) RecordModelCall(ctx context.Context, model string, provider string, duration time.Duration, success bool, promptTokens, completionTokens int64) {
	attrs := []attribute.KeyValue{
		attribute.String("model", model),
		attribute.String("provider", provider),
	}

	durationSeconds := duration.Seconds()

	m.ModelLatency.Record(ctx, durationSeconds, metric.WithAttributes(attrs...))
	m.ModelCalls.Add(ctx, 1, metric.WithAttributes(attrs...))

	if promptTokens > 0 {
		m.ModelTokensIn.Add(ctx, promptTokens, metric.WithAttributes(
			append(attrs, attribute.String("token_type", "prompt"))...,
		))
	}

	if completionTokens > 0 {
		m.ModelTokensOut.Add(ctx, completionTokens, metric.WithAttributes(
			append(attrs, attribute.String("token_type", "completion"))...,
		))
	}

	if !success {
		m.ModelErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordCacheAccess records cache lookup metrics
func (m *OTELMetrics) RecordCacheAccess(ctx context.Context, hit bool, keyType string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.key_type", keyType),
		attribute.Bool("cache.hit", hit),
	}

	if hit {
		m.CacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
	} else {
		m.CacheMisses.Add(ctx, 1, metric.WithAttributes(attrs...))
	}

	m.CacheLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordDBQuery records database query metrics
func (m *OTELMetrics) RecordDBQuery(ctx context.Context, queryType string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", queryType),
		attribute.Bool("db.success", success),
	}

	m.DBQueryLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.DBQueryCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// GetMetricsProvider returns the metrics provider for shutdown
func GetMetricsProvider() *sdkmetric.MeterProvider {
	return metricsProvider
}

// ShutdownOTELMetrics gracefully shuts down all OTEL metrics
func ShutdownOTELMetrics(ctx context.Context) error {
	if meterProvider != nil {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
	}
	if metricsProvider != nil {
		if err := metricsProvider.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
