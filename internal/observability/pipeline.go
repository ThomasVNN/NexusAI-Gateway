package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics represents system metrics
type Metrics struct {
	mu sync.RWMutex
	// Request metrics
	TotalRequests      int64
	SuccessRequests    int64
	ErrorRequests       int64
	PendingRequests     int64
	
	// Latency metrics (in milliseconds)
	MinLatency         int64
	MaxLatency         int64
	TotalLatency       int64
	LatencyCount       int64
	
	// Model metrics
	ModelRequests      map[string]int64
	ModelTokens        map[string]int64
	ModelErrors        map[string]int64
	
	// Tenant metrics
	TenantRequests     map[string]int64
	
	// System metrics
	StartTime          time.Time
	Uptime             time.Duration
}

// GlobalMetrics is the global metrics instance
var GlobalMetrics = &Metrics{
	ModelRequests:  make(map[string]int64),
	ModelTokens:    make(map[string]int64),
	ModelErrors:    make(map[string]int64),
	TenantRequests: make(map[string]int64),
	StartTime:      time.Now(),
}

// RecordRequest records a request
func (m *Metrics) RecordRequest(tenantID, modelID string, success bool, latencyMs int64) {
	atomic.AddInt64(&m.TotalRequests, 1)
	
	if success {
		atomic.AddInt64(&m.SuccessRequests, 1)
	} else {
		atomic.AddInt64(&m.ErrorRequests, 1)
	}
	
	// Update latency stats
	atomic.AddInt64(&m.LatencyCount, 1)
	atomic.AddInt64(&m.TotalLatency, latencyMs)
	
	for {
		min := atomic.LoadInt64(&m.MinLatency)
		if latencyMs < min || min == 0 {
			atomic.CompareAndSwapInt64(&m.MinLatency, min, latencyMs)
		}
		break
	}
	
	for {
		max := atomic.LoadInt64(&m.MaxLatency)
		if latencyMs > max {
			atomic.CompareAndSwapInt64(&m.MaxLatency, max, latencyMs)
		}
		break
	}
	
	// Update model metrics
	if modelID != "" {
		m.mu.Lock()
		m.ModelRequests[modelID]++
		if success {
			m.ModelTokens[modelID]++
		} else {
			m.ModelErrors[modelID]++
		}
		m.mu.Unlock()
	}
	
	// Update tenant metrics
	if tenantID != "" {
		m.mu.Lock()
		m.TenantRequests[tenantID]++
		m.mu.Unlock()
	}
}

// GetMetrics returns current metrics as a map
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	total := atomic.LoadInt64(&m.TotalRequests)
	success := atomic.LoadInt64(&m.SuccessRequests)
	errors := atomic.LoadInt64(&m.ErrorRequests)
	count := atomic.LoadInt64(&m.LatencyCount)
	totalLatency := atomic.LoadInt64(&m.TotalLatency)
	
	var avgLatency float64
	if count > 0 {
		avgLatency = float64(totalLatency) / float64(count)
	}
	
	// Calculate uptime
	uptime := time.Since(m.StartTime)
	
	return map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"uptime_seconds": uptime.Seconds(),
		"requests": map[string]interface{}{
			"total":     total,
			"success":   success,
			"errors":    errors,
			"pending":   atomic.LoadInt64(&m.PendingRequests),
			"success_rate": func() float64 {
				if total == 0 { return 0 }
				return float64(success) / float64(total) * 100
			}(),
		},
		"latency_ms": map[string]interface{}{
			"min":   atomic.LoadInt64(&m.MinLatency),
			"max":   atomic.LoadInt64(&m.MaxLatency),
			"avg":   avgLatency,
			"count": count,
		},
		"models": m.ModelRequests,
		"tenants": m.TenantRequests,
	}
}

// Health represents system health status
type Health struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// HealthChecker interface for custom health checks
type HealthChecker interface {
	Name() string
	Check() (bool, string)
}

// GlobalHealthChecks holds all registered health checks
var GlobalHealthChecks []HealthChecker

// RegisterHealthCheck registers a health check
func RegisterHealthCheck(check HealthChecker) {
	GlobalHealthChecks = append(GlobalHealthChecks, check)
}

// PerformHealthCheck runs all health checks
func PerformHealthCheck() Health {
	checks := make(map[string]HealthCheck)
	allHealthy := true
	
	for _, check := range GlobalHealthChecks {
		start := time.Now()
		healthy, msg := check.Check()
		latency := time.Since(start)
		
		status := "healthy"
		if !healthy {
			status = "unhealthy"
			allHealthy = false
		}
		
		checks[check.Name()] = HealthCheck{
			Status:  status,
			Message: msg,
			Latency: latency.String(),
		}
	}
	
	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}
	
	return Health{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
	}
}

// TracingContext holds trace context
type TracingContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	StartTime  time.Time
	EndTime    time.Time
	Attributes map[string]string
}

// TraceCollector collects traces
type TraceCollector struct {
	mu     sync.RWMutex
	traces []TracingContext
	max    int // Maximum traces to keep
}

// NewTraceCollector creates a new trace collector
func NewTraceCollector(maxTraces int) *TraceCollector {
	if maxTraces <= 0 {
		maxTraces = 1000
	}
	return &TraceCollector{
		traces: make([]TracingContext, 0, maxTraces),
		max:    maxTraces,
	}
}

// AddTrace adds a trace to the collector
func (c *TraceCollector) AddTrace(trace TracingContext) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.traces = append(c.traces, trace)
	
	// Trim if over capacity
	if len(c.traces) > c.max {
		c.traces = c.traces[len(c.traces)-c.max:]
	}
}

// GetTraces returns recent traces
func (c *TraceCollector) GetTraces(limit int) []TracingContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if limit <= 0 || limit > len(c.traces) {
		limit = len(c.traces)
	}
	
	result := make([]TracingContext, limit)
	copy(result, c.traces[len(c.traces)-limit:])
	return result
}

// GlobalTraceCollector is the global trace collector
var GlobalTraceCollector = NewTraceCollector(1000)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Level       string            `json:"level"`
	Message     string            `json:"message"`
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	TenantID    string            `json:"tenant_id,omitempty"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Duration    string            `json:"duration,omitempty"`
	StatusCode  int               `json:"status_code,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// LogCollector collects structured logs
type LogCollector struct {
	mu     sync.RWMutex
	logs   []LogEntry
	max    int
}

// NewLogCollector creates a new log collector
func NewLogCollector(maxLogs int) *LogCollector {
	if maxLogs <= 0 {
		maxLogs = 10000
	}
	return &LogCollector{
		logs: make([]LogEntry, 0, maxLogs),
		max:  maxLogs,
	}
}

// AddLog adds a log entry
func (c *LogCollector) AddLog(entry LogEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	entry.Timestamp = time.Now()
	c.logs = append(c.logs, entry)
	
	// Trim if over capacity
	if len(c.logs) > c.max {
		c.logs = c.logs[len(c.logs)-c.max:]
	}
}

// GetLogs returns recent logs
func (c *LogCollector) GetLogs(limit int, level string) []LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var result []LogEntry
	for i := len(c.logs) - 1; i >= 0; i-- {
		if level == "" || c.logs[i].Level == level {
			result = append(result, c.logs[i])
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result
}

// GlobalLogCollector is the global log collector
var GlobalLogCollector = NewLogCollector(10000)

// ObservabilityHandler handles observability endpoints
type ObservabilityHandler struct{}

// NewObservabilityHandler creates a new observability handler
func NewObservabilityHandler() *ObservabilityHandler {
	return &ObservabilityHandler{}
}

// HandleMetrics returns metrics endpoint
func (h *ObservabilityHandler) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	metrics := GlobalMetrics.GetMetrics()
	_ = json.NewEncoder(w).Encode(metrics)
}

// HandleHealth returns health check endpoint
func (h *ObservabilityHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := PerformHealthCheck()
	
	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(health)
}

// HandleTraces returns trace endpoint
func (h *ObservabilityHandler) HandleTraces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	limit := 100
	traces := GlobalTraceCollector.GetTraces(limit)
	
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"traces": traces,
		"count":  len(traces),
	})
}

// HandleLogs returns log endpoint
func (h *ObservabilityHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	level := r.URL.Query().Get("level")
	limit := 100
	
	logs := GlobalLogCollector.GetLogs(limit, level)
	
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
	})
}

// HandleStatus returns combined status
func (h *ObservabilityHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := PerformHealthCheck()
	metrics := GlobalMetrics.GetMetrics()
	
	status := map[string]interface{}{
		"service": "nexusai-gateway",
		"version": "1.0.0",
		"health":  health,
		"metrics": metrics,
	}
	
	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(status)
}

// Logf logs a formatted message
func Logf(level, format string, args ...interface{}) {
	entry := LogEntry{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
		Service: "nexusai-gateway",
		Version: "1.0.0",
	}
	GlobalLogCollector.AddLog(entry)
}
