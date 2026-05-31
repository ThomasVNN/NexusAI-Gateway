package observability

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics defines all Prometheus metrics for the gateway
type Metrics struct {
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	// Connection metrics
	ActiveConnections prometheus.Gauge

	// Model metrics
	ModelCallsTotal      *prometheus.CounterVec
	ModelCallDuration    *prometheus.HistogramVec
	ModelTokensTotal     *prometheus.CounterVec
	ModelErrorsTotal     *prometheus.CounterVec

	// Pipeline metrics
	PipelineExecutionsTotal *prometheus.CounterVec
	PipelineDuration       *prometheus.HistogramVec

	// System metrics
	UptimeSeconds    prometheus.Gauge
	DatabaseConnected prometheus.Gauge

	// Token usage metrics
	PromptTokensTotal     prometheus.Counter
	CompletionTokensTotal prometheus.Counter
	TotalTokensTotal      prometheus.Counter

	mu sync.RWMutex
}

var (
	globalMetrics *Metrics
	metricsOnce   sync.Once
)

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status_code"},
		),

		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency in seconds",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),

		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),

		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active connections",
			},
		),

		ModelCallsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "model_calls_total",
				Help:      "Total number of model API calls",
			},
			[]string{"model", "status"},
		),

		ModelCallDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "model_call_duration_seconds",
				Help:      "Model API call duration in seconds",
				Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
			},
			[]string{"model", "provider"},
		),

		ModelTokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "token_usage_total",
				Help:      "Total token consumption",
			},
			[]string{"model", "token_type"},
		),

		ModelErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "model_errors_total",
				Help:      "Total number of model API errors",
			},
			[]string{"model", "error_type"},
		),

		PipelineExecutionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "pipeline_executions_total",
				Help:      "Total number of pipeline executions",
			},
			[]string{"pipeline", "status"},
		),

		PipelineDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "pipeline_duration_seconds",
				Help:      "Pipeline execution duration in seconds",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
			},
			[]string{"pipeline"},
		),

		UptimeSeconds: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "uptime_seconds",
				Help:      "Gateway uptime in seconds",
			},
		),

		DatabaseConnected: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "database_connected",
				Help:      "Database connection status (1 = connected, 0 = disconnected)",
			},
		),

		PromptTokensTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "prompt_tokens_total",
				Help:      "Total number of prompt tokens consumed",
			},
		),

		CompletionTokensTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "completion_tokens_total",
				Help:      "Total number of completion tokens generated",
			},
		),

		TotalTokensTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tokens_total",
				Help:      "Total number of tokens (prompt + completion)",
			},
		),
	}

	return m
}

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *Metrics {
	metricsOnce.Do(func() {
		globalMetrics = NewMetrics("nexusai_gateway")
	})
	return globalMetrics
}

// ObserveRequest records HTTP request metrics
func (m *Metrics) ObserveRequest(method, path string, statusCode int, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, path, statusCodeToString(statusCode)).Inc()
	m.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// IncrementRequestsInFlight increments the in-flight request counter
func (m *Metrics) IncrementRequestsInFlight() {
	m.RequestsInFlight.Inc()
}

// DecrementRequestsInFlight decrements the in-flight request counter
func (m *Metrics) DecrementRequestsInFlight() {
	m.RequestsInFlight.Dec()
}

// IncrementActiveConnections increments the active connections gauge
func (m *Metrics) IncrementActiveConnections() {
	m.ActiveConnections.Inc()
}

// DecrementActiveConnections decrements the active connections gauge
func (m *Metrics) DecrementActiveConnections() {
	m.ActiveConnections.Dec()
}

// RecordModelCall records model API call metrics
func (m *Metrics) RecordModelCall(model, status string, duration time.Duration) {
	m.ModelCallsTotal.WithLabelValues(model, status).Inc()
	m.ModelCallDuration.WithLabelValues(model, "default").Observe(duration.Seconds())
}

// RecordModelTokens records token usage metrics
func (m *Metrics) RecordModelTokens(model string, promptTokens, completionTokens int64) {
	m.ModelTokensTotal.WithLabelValues(model, "prompt").Add(float64(promptTokens))
	m.ModelTokensTotal.WithLabelValues(model, "completion").Add(float64(completionTokens))
	m.PromptTokensTotal.Add(float64(promptTokens))
	m.CompletionTokensTotal.Add(float64(completionTokens))
	m.TotalTokensTotal.Add(float64(promptTokens + completionTokens))
}

// RecordModelError records model API error metrics
func (m *Metrics) RecordModelError(model, errorType string) {
	m.ModelErrorsTotal.WithLabelValues(model, errorType).Inc()
}

// RecordPipelineExecution records pipeline execution metrics
func (m *Metrics) RecordPipelineExecution(pipeline, status string, duration time.Duration) {
	m.PipelineExecutionsTotal.WithLabelValues(pipeline, status).Inc()
	m.PipelineDuration.WithLabelValues(pipeline).Observe(duration.Seconds())
}

// SetUptime updates the uptime gauge
func (m *Metrics) SetUptime(seconds float64) {
	m.UptimeSeconds.Set(seconds)
}

// SetDatabaseConnected updates the database connection status
func (m *Metrics) SetDatabaseConnected(connected bool) {
	if connected {
		m.DatabaseConnected.Set(1)
	} else {
		m.DatabaseConnected.Set(0)
	}
}

// StatusCodeToString converts HTTP status code to string label
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// MetricsCollector implementation for Prometheus
type PrometheusMetricsCollector struct {
	metrics *Metrics
}

// NewPrometheusMetricsCollector creates a new Prometheus-based metrics collector
func NewPrometheusMetricsCollector() *PrometheusMetricsCollector {
	return &PrometheusMetricsCollector{
		metrics: GetGlobalMetrics(),
	}
}

// IncrementCounter implements MetricsCollector interface
func (c *PrometheusMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	// Map generic counter names to Prometheus metrics
	switch name {
	case "requests_total":
		c.metrics.RequestsTotal.WithLabelValues(
			tags["method"], tags["path"], tags["status"],
		).Inc()
	case "model_calls_total":
		c.metrics.ModelCallsTotal.WithLabelValues(
			tags["model"], tags["status"],
		).Inc()
	}
}

// ObserveHistogram implements MetricsCollector interface
func (c *PrometheusMetricsCollector) ObserveHistogram(name string, value float64, tags map[string]string) {
	switch name {
	case "request_duration_seconds":
		c.metrics.RequestDuration.WithLabelValues(
			tags["method"], tags["path"],
		).Observe(value)
	case "model_call_duration_seconds":
		c.metrics.ModelCallDuration.WithLabelValues(
			tags["model"], tags["provider"],
		).Observe(value)
	}
}

// GlobalMetricsInstance holds the initialized metrics instance
var GlobalMetricsInstance *Metrics

// InitGlobalMetricsCollector initializes the global metrics collector
func InitGlobalMetricsCollector() {
	GlobalMetricsInstance = GetGlobalMetrics()
}

// PrometheusHandler returns an HTTP handler for Prometheus metrics
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsUpdater starts a background goroutine to update system metrics
func StartMetricsUpdater(ctx context.Context, startTime time.Time, dbChecker func() bool) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m := GetGlobalMetrics()
				if m != nil {
					m.SetUptime(time.Since(startTime).Seconds())
					m.SetDatabaseConnected(dbChecker())
				}
			}
		}
	}()
}
