package log

import (
	"time"
)

// RequestLog represents a single API request log entry
type RequestLog struct {
	ID           int64     `json:"id"`
	ChannelID    *int64    `json:"channel_id,omitempty"`
	ChannelName  string    `json:"channel_name,omitempty"`
	Model        string    `json:"model"`
	TokenGroupID *int64    `json:"token_group_id,omitempty"`
	UserID       *int64    `json:"user_id,omitempty"`
	APIKeyID     string    `json:"api_key_id,omitempty"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	LatencyMS    int64     `json:"latency_ms"`
	Status       string    `json:"status"` // "success", "error", "timeout"
	ErrorMessage string    `json:"error_message,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	RequestID    string    `json:"request_id,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	ModelRaw     string    `json:"model_raw,omitempty"` // Original model from request
	CreatedAt    time.Time `json:"created_at"`
}

// RequestLogFilter provides filtering options for log queries
type RequestLogFilter struct {
	ChannelID    *int64
	TokenGroupID *int64
	UserID       *int64
	Model        string
	Status       string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// RequestLogStats provides aggregated statistics
type RequestLogStats struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	TotalInputTokens   int64   `json:"total_input_tokens"`
	TotalOutputTokens  int64   `json:"total_output_tokens"`
	TotalTokens        int64   `json:"total_tokens"`
	AvgLatencyMS       float64 `json:"avg_latency_ms"`
	SuccessRate        float64 `json:"success_rate"`
}

// ModelUsageStats represents usage statistics per model
type ModelUsageStats struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
	CostUSD      float64 `json:"cost_usd"`
}

// ChannelUsageStats represents usage statistics per channel
type ChannelUsageStats struct {
	ChannelID    int64   `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	RequestCount int64   `json:"request_count"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMS float64 `json:"avg_latency_ms"`
}

// DailyUsageStats represents daily usage aggregation
type DailyUsageStats struct {
	Date         string `json:"date"`
	RequestCount int64  `json:"request_count"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
	UniqueUsers  int64  `json:"unique_users"`
	UniqueModels int64  `json:"unique_models"`
}

// AnalyticsOverview provides dashboard overview data
type AnalyticsOverview struct {
	TotalRequests  int64               `json:"total_requests"`
	TodayRequests  int64               `json:"today_requests"`
	TotalTokens    int64               `json:"total_tokens"`
	TodayTokens    int64               `json:"today_tokens"`
	TotalCost      float64             `json:"total_cost"`
	TodayCost      float64             `json:"today_cost"`
	AvgLatencyMS   float64             `json:"avg_latency_ms"`
	SuccessRate    float64             `json:"success_rate"`
	ActiveChannels int                 `json:"active_channels"`
	ActiveModels   int                 `json:"active_models"`
	TopModels      []ModelUsageStats   `json:"top_models"`
	TopChannels    []ChannelUsageStats `json:"top_channels"`
	DailyTrend     []DailyUsageStats   `json:"daily_trend"`
}

// CostBreakdown provides detailed cost analysis
type CostBreakdown struct {
	TotalCost         float64            `json:"total_cost"`
	CostByModel       map[string]float64 `json:"cost_by_model"`
	CostByChannel     map[string]float64 `json:"cost_by_channel"`
	CostByDay         map[string]float64 `json:"cost_by_day"`
	AvgCostPerRequest float64            `json:"avg_cost_per_request"`
}

// Validate checks if the request log is valid
func (r *RequestLog) Validate() error {
	if r.Model == "" {
		return ErrModelRequired
	}
	return nil
}

// Custom errors
type LogError struct {
	Message string
}

func (e *LogError) Error() string {
	return e.Message
}

var (
	ErrModelRequired = &LogError{Message: "model is required for request log"}
)
