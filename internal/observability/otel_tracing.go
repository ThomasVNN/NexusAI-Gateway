package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Span names for different operations
const (
	SpanGatewayRequest = "gateway.request"
	SpanProviderCall   = "provider.call"
	SpanCacheLookup    = "cache.lookup"
	SpanDBQuery        = "db.query"
)

// Attribute keys
const (
	AttrRequestID   = "request.id"
	AttrUserID      = "user.id"
	AttrModel       = "model"
	AttrTier        = "tier"
	AttrLatencyMS   = "latency_ms"
	AttrTokensIn    = "tokens_in"
	AttrTokensOut   = "tokens_out"
	AttrRoute       = "route"
	AttrStatusCode  = "http.status_code"
	AttrMethod      = "http.method"
	AttrPath        = "http.path"
	AttrProvider    = "provider"
	AttrCacheHit    = "cache.hit"
	AttrDBOperation = "db.operation"
)

// TracerMiddleware provides HTTP middleware with OpenTelemetry tracing
type TracerMiddleware struct {
	tracer trace.Tracer
}

// NewTracerMiddleware creates a new tracer middleware
func NewTracerMiddleware(serviceName string) *TracerMiddleware {
	return &TracerMiddleware{
		tracer: otel.Tracer(serviceName),
	}
}

// Middleware returns an HTTP handler that traces requests
func (t *TracerMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Extract trace context from incoming request headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// Start a new span for this request
		spanName := SpanGatewayRequest
		ctx, span := t.tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String(AttrMethod, r.Method),
				attribute.String(AttrPath, r.URL.Path),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.client_ip", getClientIP(r)),
			),
		)
		defer span.End()

		// Create response writer wrapper to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		r = r.WithContext(ctx)

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Calculate duration
		duration := time.Since(startTime)

		// Add response attributes to span
		span.SetAttributes(
			attribute.Int(AttrStatusCode, wrappedWriter.statusCode),
			attribute.Int64(AttrLatencyMS, duration.Milliseconds()),
		)

		// Record error if status code indicates failure
		if wrappedWriter.statusCode >= http.StatusInternalServerError {
			span.SetAttributes(attribute.Bool("error", true))
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// StartNamedSpan starts a new span with the given name
func StartNamedSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	tracer := otel.Tracer("nexusai-gateway")
	return tracer.Start(ctx, name)
}

// StartSpanWithAttrs starts a new span with attributes
func StartSpanWithAttrs(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, trace.Span) {
	tracer := otel.Tracer("nexusai-gateway")

	otelAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		otelAttrs = append(otelAttrs, toAttribute(k, v))
	}

	return tracer.Start(ctx, name,
		trace.WithAttributes(otelAttrs...),
	)
}

// StartProviderSpan starts a span for provider/model calls
func StartProviderSpan(ctx context.Context, model, provider string) (context.Context, trace.Span) {
	ctx, span := StartNamedSpan(ctx, SpanProviderCall)
	span.SetAttributes(
		attribute.String(AttrModel, model),
		attribute.String(AttrProvider, provider),
	)
	return ctx, span
}

// StartCacheSpan starts a span for cache operations
func StartCacheSpan(ctx context.Context, key string, keyType string) (context.Context, trace.Span) {
	ctx, span := StartNamedSpan(ctx, SpanCacheLookup)
	span.SetAttributes(
		attribute.String("cache.key", key),
		attribute.String("cache.key_type", keyType),
	)
	return ctx, span
}

// StartDBSpan starts a span for database operations
func StartDBSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	ctx, span := StartNamedSpan(ctx, SpanDBQuery)
	span.SetAttributes(
		attribute.String(AttrDBOperation, operation),
	)
	return ctx, span
}

// RecordTokens records token usage on the current span
func RecordTokens(ctx context.Context, promptTokens, completionTokens int64) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Int64(AttrTokensIn, promptTokens),
			attribute.Int64(AttrTokensOut, completionTokens),
			attribute.Int64("tokens.total", promptTokens+completionTokens),
		)
	}
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, message string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.message", message),
			attribute.Bool("error", true),
		)
	}
}

// RecordCacheHit records cache hit/miss on the current span
func RecordCacheHit(ctx context.Context, hit bool) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(
			attribute.Bool(AttrCacheHit, hit),
		)
	}
}

// toAttribute converts a key-value pair to an OpenTelemetry attribute
func toAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	default:
		return attribute.String(key, toString(value))
	}
}

// toString converts any value to string
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return string(rune(val))
	case int64:
		return string(rune(int(val)))
	case float64:
		return string(rune(int(val)))
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// InjectTraceContextToHeader injects trace context into response headers
func InjectTraceContextToHeader(ctx context.Context, w http.ResponseWriter) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		w.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
		w.Header().Set("X-Span-ID", span.SpanContext().SpanID().String())
	}
}

// CreateSpanFromRequest creates a span from an HTTP request with correlation ID
func CreateSpanFromRequest(ctx context.Context, r *http.Request, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer("nexusai-gateway")

	// Extract any incoming trace context
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

	attrs := []attribute.KeyValue{
		attribute.String(AttrMethod, r.Method),
		attribute.String(AttrPath, r.URL.Path),
		attribute.String(AttrRoute, r.URL.Path),
	}

	// Add correlation ID if present
	if corrID := r.Header.Get("X-Correlation-ID"); corrID != "" {
		attrs = append(attrs, attribute.String("correlation.id", corrID))
	}

	ctx, span := tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

// AddRequestAttributes adds common request attributes to a span
func AddRequestAttributes(span trace.Span, requestID, userID, model, tier string, latencyMS int64) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrRequestID, requestID),
		attribute.String(AttrModel, model),
		attribute.String(AttrTier, tier),
		attribute.Int64(AttrLatencyMS, latencyMS),
	}

	if userID != "" {
		attrs = append(attrs, attribute.String(AttrUserID, userID))
	}

	span.SetAttributes(attrs...)
}

// LogSpanError logs an error from a span
func LogSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetAttributes(attribute.Bool("error", true))
		slog.ErrorContext(ctx, "Span recorded error",
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.Any("error", err),
		)
	}
}
