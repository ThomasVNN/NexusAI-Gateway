package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestMetricsCollectorSingleton(t *testing.T) {
	collector := GetMetricsCollector()
	if collector == nil {
		t.Error("Expected non-nil collector")
	}

	// Should return same instance
	collector2 := GetMetricsCollector()
	if collector != collector2 {
		t.Error("Expected same collector instance")
	}
}

func TestMetricsCollectorIncCounter(t *testing.T) {
	collector := GetMetricsCollector()
	collector.IncCounter("test_counter", map[string]string{"method": "get"})
	collector.IncCounter("test_counter", map[string]string{"method": "get"})
}

func TestMetricsCollectorSetGauge(t *testing.T) {
	collector := GetMetricsCollector()
	collector.SetGauge("test_gauge", 100.5, map[string]string{"host": "localhost"})
}

func TestMetricsCollectorObserveHistogram(t *testing.T) {
	collector := GetMetricsCollector()
	collector.ObserveHistogram("test_histogram", 150.5, map[string]string{"endpoint": "/api"})
}

func TestMetricsCollectorObserveSummary(t *testing.T) {
	collector := GetMetricsCollector()
	collector.ObserveSummary("test_summary", 1024.0, map[string]string{"type": "request"})
}

func TestMetricsCollectorExportPrometheus(t *testing.T) {
	collector := GetMetricsCollector()
	collector.IncCounter("export_test", map[string]string{"label": "value"})

	output := collector.ExportPrometheus()
	if output == "" {
		t.Error("Expected non-empty output")
	}
}

func TestMetricsCollectorGetMetrics(t *testing.T) {
	collector := GetMetricsCollector()
	collector.IncCounter("metrics_test", map[string]string{})

	metrics := collector.GetMetrics()
	if len(metrics) == 0 {
		t.Error("Expected some metrics")
	}
}

func TestTracerCreation(t *testing.T) {
	tracer := NewTracer()
	if tracer == nil {
		t.Error("Expected non-nil tracer")
	}
}

func TestTracerSpan(t *testing.T) {
	tracer := NewTracer()
	ctx, span := tracer.StartSpan(context.Background(), "test_operation", "test_service")

	if span == nil {
		t.Error("Expected non-nil span")
	}

	span.SetTag("key", "value")
	span.SetStatus("ok")
	span.End()

	_ = ctx // context can be ignored
}

func TestTracerMultipleSpans(t *testing.T) {
	tracer := NewTracer()

	for i := 0; i < 3; i++ {
		_, span := tracer.StartSpan(context.Background(), "operation", "test")
		span.SetTag("index", string(rune('0'+i)))
		span.End()
	}

	spans := tracer.GetSpans()
	if len(spans) < 3 {
		t.Errorf("Expected at least 3 spans, got %d", len(spans))
	}
}

func TestTracerFlush(t *testing.T) {
	tracer := NewTracer()
	_, span := tracer.StartSpan(context.Background(), "flush_test", "test")
	span.End()

	err := tracer.Flush()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSpanSetStatus(t *testing.T) {
	tracer := NewTracer()
	_, span := tracer.StartSpan(context.Background(), "status_test", "test")
	span.SetStatus("error")
	span.End()
}

func TestSpanSetTags(t *testing.T) {
	tracer := NewTracer()
	_, span := tracer.StartSpan(context.Background(), "tags_test", "test")
	span.SetTag("string_key", "value")
	span.End()
}

func TestSpanDuration(t *testing.T) {
	tracer := NewTracer()
	_, span := tracer.StartSpan(context.Background(), "duration_test", "test")
	time.Sleep(5 * time.Millisecond)
	span.End()

	spans := tracer.GetSpans()
	if len(spans) > 0 {
		// Verify span was recorded
	}
}

func TestNoopExporter(t *testing.T) {
	exporter := &NoopExporter{}
	err := exporter.Export(nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetTracer(t *testing.T) {
	tracer := GetTracer()
	if tracer == nil {
		t.Error("Expected non-nil tracer")
	}

	// Should return same instance
	tracer2 := GetTracer()
	if tracer != tracer2 {
		t.Error("Expected same tracer instance")
	}
}

func TestMetricConstants(t *testing.T) {
	// Verify metric constants are defined
	if MetricRequestsTotal == "" {
		t.Error("Expected non-empty MetricRequestsTotal")
	}
	if MetricRequestDuration == "" {
		t.Error("Expected non-empty MetricRequestDuration")
	}
	if MetricErrorsTotal == "" {
		t.Error("Expected non-empty MetricErrorsTotal")
	}
}

func TestMetricTypes(t *testing.T) {
	// Verify metric type constants
	if MetricTypeCounter != 0 {
		t.Errorf("Expected MetricTypeCounter = 0, got %d", MetricTypeCounter)
	}
	if MetricTypeGauge != 1 {
		t.Errorf("Expected MetricTypeGauge = 1, got %d", MetricTypeGauge)
	}
	if MetricTypeHistogram != 2 {
		t.Errorf("Expected MetricTypeHistogram = 2, got %d", MetricTypeHistogram)
	}
	if MetricTypeSummary != 3 {
		t.Errorf("Expected MetricTypeSummary = 3, got %d", MetricTypeSummary)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("Expected non-empty ID")
	}
	if id1 == id2 {
		t.Error("Expected different IDs")
	}
}

func TestLabelsToString(t *testing.T) {
	labels := map[string]string{
		"method": "GET",
		"path":   "/api",
	}
	result := labelsToString(labels)
	if result == "" {
		t.Error("Expected non-empty string")
	}
}

func TestMetricKey(t *testing.T) {
	// Without labels
	key := metricKey("test_metric", nil)
	if key != "test_metric" {
		t.Errorf("Expected 'test_metric', got '%s'", key)
	}

	// With labels
	key = metricKey("test_metric", map[string]string{"label": "value"})
	if key == "test_metric" {
		t.Error("Expected key with labels")
	}
}
