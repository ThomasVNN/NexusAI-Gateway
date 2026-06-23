package observability

import (
	"net/http"
	"strconv"
	"time"
)

// Telemetry header constants for response metadata
const (
	HeaderResponseCost     = "X-NexusAI-Response-Cost"
	HeaderTokensIn         = "X-NexusAI-Tokens-In"
	HeaderTokensOut        = "X-NexusAI-Tokens-Out"
	HeaderModel            = "X-NexusAI-Model"
	HeaderProvider         = "X-NexusAI-Provider"
	HeaderLatencyMs        = "X-NexusAI-Latency-Ms"
	HeaderCacheHit         = "X-NexusAI-Cache-Hit"
	HeaderFallbackAttempts  = "X-NexusAI-Fallback-Attempts"
	HeaderRequestID        = "X-NexusAI-Request-ID"
	HeaderTraceID           = "X-NexusAI-Trace-ID"
	HeaderSpanID           = "X-NexusAI-Span-ID"
)

// TelemetryData holds all telemetry information for a request
type TelemetryData struct {
	ResponseCost      float64
	TokensIn          int
	TokensOut         int
	Model             string
	Provider          string
	LatencyMs         int64
	CacheHit          bool
	FallbackAttempts  int
	RequestID         string
	TraceID           string
	SpanID            string
}

// NewTelemetryData creates a new TelemetryData with defaults
func NewTelemetryData() *TelemetryData {
	return &TelemetryData{
		FallbackAttempts: 0,
		CacheHit:         false,
		TokensIn:         0,
		TokensOut:        0,
		ResponseCost:     0.0,
	}
}

// SetFromLatency calculates latency from start time
func (t *TelemetryData) SetFromLatency(startTime time.Time) {
	t.LatencyMs = time.Since(startTime).Milliseconds()
}

// SetResponseHeaders sets all telemetry headers on the response
func (t *TelemetryData) SetResponseHeaders(w http.ResponseWriter) {
	w.Header().Set(HeaderResponseCost, strconv.FormatFloat(t.ResponseCost, 'f', 6, 64))
	w.Header().Set(HeaderTokensIn, strconv.Itoa(t.TokensIn))
	w.Header().Set(HeaderTokensOut, strconv.Itoa(t.TokensOut))
	w.Header().Set(HeaderModel, t.Model)
	w.Header().Set(HeaderProvider, t.Provider)
	w.Header().Set(HeaderLatencyMs, strconv.FormatInt(t.LatencyMs, 10))
	w.Header().Set(HeaderCacheHit, strconv.FormatBool(t.CacheHit))
	w.Header().Set(HeaderFallbackAttempts, strconv.Itoa(t.FallbackAttempts))

	if t.RequestID != "" {
		w.Header().Set(HeaderRequestID, t.RequestID)
	}
	if t.TraceID != "" {
		w.Header().Set(HeaderTraceID, t.TraceID)
	}
	if t.SpanID != "" {
		w.Header().Set(HeaderSpanID, t.SpanID)
	}
}

// GetResponseHeaders returns headers map for event emission
func (t *TelemetryData) GetResponseHeaders() map[string]string {
	return map[string]string{
		HeaderResponseCost:     strconv.FormatFloat(t.ResponseCost, 'f', 6, 64),
		HeaderTokensIn:         strconv.Itoa(t.TokensIn),
		HeaderTokensOut:        strconv.Itoa(t.TokensOut),
		HeaderModel:            t.Model,
		HeaderProvider:        t.Provider,
		HeaderLatencyMs:       strconv.FormatInt(t.LatencyMs, 10),
		HeaderCacheHit:        strconv.FormatBool(t.CacheHit),
		HeaderFallbackAttempts: strconv.Itoa(t.FallbackAttempts),
	}
}

// ToMap converts TelemetryData to a map for event logging
func (t *TelemetryData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"response_cost":      t.ResponseCost,
		"tokens_in":           t.TokensIn,
		"tokens_out":          t.TokensOut,
		"model":               t.Model,
		"provider":            t.Provider,
		"latency_ms":          t.LatencyMs,
		"cache_hit":          t.CacheHit,
		"fallback_attempts":  t.FallbackAttempts,
		"request_id":         t.RequestID,
		"trace_id":           t.TraceID,
		"span_id":            t.SpanID,
	}
}

// TelemetryMiddleware provides HTTP middleware for telemetry injection
type TelemetryMiddleware struct{}

// NewTelemetryMiddleware creates a new telemetry middleware
func NewTelemetryMiddleware() *TelemetryMiddleware {
	return &TelemetryMiddleware{}
}

// InjectTelemetryHeaders injects telemetry headers into response
// Use this as a wrapper around response writer for automatic header injection
func (m *TelemetryMiddleware) InjectTelemetryHeaders(w http.ResponseWriter, data *TelemetryData) {
	data.SetResponseHeaders(w)
}

// CostCalculator provides cost calculation utilities
type CostCalculator struct {
	Pricing map[string]*ModelPricing
}

// ModelPricing holds pricing information for a model
type ModelPricing struct {
	InputPricePer1K  float64
	OutputPricePer1K float64
	BatchInputPer1K  float64
	Currency         string
}

// NewCostCalculator creates a cost calculator with default pricing
func NewCostCalculator() *CostCalculator {
	return &CostCalculator{
		Pricing: make(map[string]*ModelPricing),
	}
}

// CalculateCost calculates the cost for token usage
func (c *CostCalculator) CalculateCost(model string, tokensIn, tokensOut int) float64 {
	pricing, ok := c.Pricing[model]
	if !ok {
		return 0.0
	}

	inputCost := (float64(tokensIn) / 1000.0) * pricing.InputPricePer1K
	outputCost := (float64(tokensOut) / 1000.0) * pricing.OutputPricePer1K

	return inputCost + outputCost
}

// SetModelPricing sets pricing for a specific model
func (c *CostCalculator) SetModelPricing(model string, inputPrice, outputPrice float64) {
	c.Pricing[model] = &ModelPricing{
		InputPricePer1K:  inputPrice,
		OutputPricePer1K: outputPrice,
		Currency:         "USD",
	}
}

// Global cost calculator instance
var GlobalCostCalculator = NewCostCalculator()
