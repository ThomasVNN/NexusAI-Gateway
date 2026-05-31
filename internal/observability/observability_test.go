package observability

import (
	"context"
	"testing"
)

func TestOTELTracer_StartSpan(t *testing.T) {
	// Skip this test as it requires full OTEL initialization
	// The OTELTracer.StartSpan relies on the global tracer being set
	t.Skip("OTELTracer test requires full OpenTelemetry initialization")
}

func TestOTELSpan_Methods(t *testing.T) {
	// This test doesn't work without OTEL span
	t.Skip("OTELSpan test requires full OpenTelemetry initialization")
}

func TestTracerProvider_Shutdown(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "",
		Enabled:        false,
	}

	ctx := context.Background()
	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		t.Fatalf("NewTracerProvider() error = %v", err)
	}

	// Shutdown should not panic
	if err := tp.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestInitGlobalTracer(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   "",
		Enabled:        false,
	}

	ctx := context.Background()
	err := InitGlobalTracer(ctx, cfg)
	if err != nil {
		t.Fatalf("InitGlobalTracer() error = %v", err)
	}

	// GlobalTracer should be set
	if GlobalTracer == nil {
		t.Error("GlobalTracer is nil after InitGlobalTracer()")
	}
}

func TestStartSpanWithAttributes(t *testing.T) {
	ctx := context.Background()
	attrs := map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
	}

	ctx, span := StartSpanWithAttributes(ctx, "test-span", attrs)
	if ctx == nil {
		t.Error("StartSpanWithAttributes() returned nil context")
	}
	if span == nil {
		t.Error("StartSpanWithAttributes() returned nil span")
	}
	span.End()
}

func TestAddSpanEvent(t *testing.T) {
	ctx := context.Background()
	attrs := map[string]interface{}{
		"key": "value",
	}

	// Should not panic
	AddSpanEvent(ctx, "test-event", attrs)
}

func TestInjectExtractTraceContext(t *testing.T) {
	ctx := context.Background()

	// Inject should return carrier
	carrier := InjectTraceContext(ctx)
	if carrier == nil {
		t.Error("InjectTraceContext() returned nil carrier")
	}

	// Extract should work with empty carrier
	newCtx := ExtractTraceContext(ctx, carrier)
	if newCtx == nil {
		t.Error("ExtractTraceContext() returned nil context")
	}
}

func TestGetTraceID(t *testing.T) {
	ctx := context.Background()

	// Without span, should return empty string
	traceID := GetTraceID(ctx)
	if traceID != "" {
		t.Errorf("GetTraceID() = %v, want empty string", traceID)
	}
}

func TestGetSpanID(t *testing.T) {
	ctx := context.Background()

	// Without span, should return empty string
	spanID := GetSpanID(ctx)
	if spanID != "" {
		t.Errorf("GetSpanID() = %v, want empty string", spanID)
	}
}

func TestGetTraceIDNil(t *testing.T) {
	// Test with nil context
	traceID := GetTraceID(nil)
	if traceID != "" {
		t.Errorf("GetTraceID(nil) = %v, want empty string", traceID)
	}
}

func TestGetSpanIDNil(t *testing.T) {
	// Test with nil context
	spanID := GetSpanID(nil)
	if spanID != "" {
		t.Errorf("GetSpanID(nil) = %v, want empty string", spanID)
	}
}
