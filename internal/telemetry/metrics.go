package telemetry

import (
	"fmt"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType int

const (
	MetricTypeCounter MetricType = iota
	MetricTypeGauge
	MetricTypeHistogram
	MetricTypeSummary
)

// Metric represents a Prometheus metric
type Metric struct {
	Name   string
	Type   MetricType
	Help   string
	Labels []string
}

// MetricValue represents a metric value
type MetricValue struct {
	Name      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
}

// MetricsCollector collects and exposes metrics
type MetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]*CounterMetric
	gauges     map[string]*GaugeMetric
	histograms map[string]*HistogramMetric
	summaries  map[string]*SummaryMetric
}

// CounterMetric represents a counter metric
type CounterMetric struct {
	Value float64
}

// GaugeMetric represents a gauge metric
type GaugeMetric struct {
	Value float64
}

// HistogramMetric represents a histogram metric
type HistogramMetric struct {
	Values  []float64
	Count   int
	Sum     float64
	Buckets map[float64]int
}

// SummaryMetric represents a summary metric
type SummaryMetric struct {
	Values []float64
	Count  int
	Sum    float64
}

var (
	metricsCollector *MetricsCollector
	metricsOnce      sync.Once
)

// GetMetricsCollector returns the global metrics collector
func GetMetricsCollector() *MetricsCollector {
	metricsOnce.Do(func() {
		metricsCollector = &MetricsCollector{
			counters:   make(map[string]*CounterMetric),
			gauges:     make(map[string]*GaugeMetric),
			histograms: make(map[string]*HistogramMetric),
			summaries:  make(map[string]*SummaryMetric),
		}
	})
	return metricsCollector
}

// IncCounter increments a counter
func (m *MetricsCollector) IncCounter(name string, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey(name, labels)
	if _, ok := m.counters[key]; !ok {
		m.counters[key] = &CounterMetric{}
	}
	m.counters[key].Value++
}

// SetGauge sets a gauge value
func (m *MetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey(name, labels)
	m.gauges[key] = &GaugeMetric{Value: value}
}

// ObserveHistogram observes a value for histogram
func (m *MetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey(name, labels)
	if _, ok := m.histograms[key]; !ok {
		m.histograms[key] = &HistogramMetric{
			Buckets: make(map[float64]int),
		}
	}
	h := m.histograms[key]
	h.Values = append(h.Values, value)
	h.Count++
	h.Sum += value
}

// ObserveSummary observes a value for summary
func (m *MetricsCollector) ObserveSummary(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := metricKey(name, labels)
	if _, ok := m.summaries[key]; !ok {
		m.summaries[key] = &SummaryMetric{}
	}
	s := m.summaries[key]
	s.Values = append(s.Values, value)
	s.Count++
	s.Sum += value
}

// GetMetrics returns all current metrics
func (m *MetricsCollector) GetMetrics() []*MetricValue {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var metrics []*MetricValue
	now := time.Now()

	// Export counters
	for key, c := range m.counters {
		name := key
		labels := parseLabels(key)
		metrics = append(metrics, &MetricValue{
			Name:      name,
			Value:     c.Value,
			Labels:    labels,
			Timestamp: now,
		})
	}

	// Export gauges
	for key, g := range m.gauges {
		name := key
		labels := parseLabels(key)
		metrics = append(metrics, &MetricValue{
			Name:      name,
			Value:     g.Value,
			Labels:    labels,
			Timestamp: now,
		})
	}

	return metrics
}

// ExportPrometheus exports metrics in Prometheus format
func (m *MetricsCollector) ExportPrometheus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var output string

	// Export counters
	for key, c := range m.counters {
		name := key
		output += fmt.Sprintf("# TYPE %s counter\n", name)
		output += fmt.Sprintf("%s %f\n", name, c.Value)
	}

	// Export gauges
	for key, g := range m.gauges {
		name := key
		output += fmt.Sprintf("# TYPE %s gauge\n", name)
		output += fmt.Sprintf("%s %f\n", name, g.Value)
	}

	return output
}

// metricKey creates a unique key for a metric with labels
func metricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	return fmt.Sprintf("%s{%s}", name, labelsToString(labels))
}

// labelsToString converts labels to string format
func labelsToString(labels map[string]string) string {
	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return joinStrings(parts, ",")
}

// parseLabels parses labels from metric key
func parseLabels(key string) map[string]string {
	labels := make(map[string]string)
	return labels
}

// joinStrings joins strings with separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// Common metric names
const (
	MetricRequestsTotal     = "nexusai_requests_total"
	MetricRequestDuration   = "nexusai_request_duration_seconds"
	MetricErrorsTotal       = "nexusai_errors_total"
	MetricActiveConnections = "nexusai_active_connections"
	MetricTokensUsed        = "nexusai_tokens_used_total"
	MetricCostTotal         = "nexusai_cost_total"
)
