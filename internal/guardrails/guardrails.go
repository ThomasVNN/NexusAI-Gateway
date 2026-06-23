package guardrails

import (
	"context"
	"encoding/json"
	"time"
)

// GuardrailStage represents when a guardrail runs in the request lifecycle
type GuardrailStage string

const (
	StagePreCall  GuardrailStage = "preCall"
	StagePostCall GuardrailStage = "postCall"
)

// GuardrailAction defines what action to take when a guardrail triggers
type GuardrailAction int

const (
	ActionAllow GuardrailAction = iota
	ActionBlock
	ActionWarn
	ActionLog
)

// GuardrailMode defines the overall guardrail enforcement mode
type GuardrailMode string

const (
	ModeBlock GuardrailMode = "block"
	ModeWarn  GuardrailMode = "warn"
	ModeLog   GuardrailMode = "log"
)

// Guardrail is the interface that all guardrails must implement
type Guardrail interface {
	Name() string
	Priority() int
	Stage() GuardrailStage
	Check(ctx context.Context, gc *GuardrailContext) (*GuardrailResult, error)
}

// GuardrailContext contains all context needed for guardrail evaluation
type GuardrailContext struct {
	Request        *AIRequest
	Response       *AIResponse
	UserID         string
	OrganizationID string
	Headers        map[string]string
	Metadata       map[string]interface{}
	Timestamp      time.Time
	TraceID        string
	SpanID         string
}

// AIRequest represents an AI API request
type AIRequest struct {
	Model       string                   `json:"model"`
	Messages    []Message                `json:"messages"`
	Prompt      string                   `json:"prompt,omitempty"`
	SystemPrompt string                  `json:"system_prompt,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Provider    string                   `json:"provider,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	Images      []ImageData              `json:"images,omitempty"`
	Tools       []ToolDefinition         `json:"tools,omitempty"`
}

// AIResponse represents an AI API response
type AIResponse struct {
	Content    string       `json:"content,omitempty"`
	Message    *Message    `json:"message,omitempty"`
	Model      string      `json:"model,omitempty"`
	Provider   string      `json:"provider,omitempty"`
	FinishReason string    `json:"finish_reason,omitempty"`
	Usage      *TokenUsage `json:"usage,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ImageData represents an image in a request
type ImageData struct {
	Type      string `json:"type"` // "url" or "base64"
	URL       string `json:"url,omitempty"`
	Data      string `json:"data,omitempty"` // base64 encoded
	MimeType  string `json:"mime_type,omitempty"`
}

// ToolDefinition represents a tool/function definition
type ToolDefinition struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// GuardrailResult contains the outcome of a guardrail check
type GuardrailResult struct {
	Passed     bool              `json:"passed"`
	Action     GuardrailAction   `json:"action"`
	Message    string            `json:"message"`
	Redacted   *RedactedContent  `json:"redacted,omitempty"`
	Detections []*Detection       `json:"detections,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Duration   time.Duration    `json:"duration,omitempty"`
}

// RedactedContent represents content that has been redacted
type RedactedContent struct {
	Original string            `json:"original"`
	Result   string            `json:"result"`
	Type     string            `json:"type"`
	Count    int              `json:"count"`
}

// Detection represents a single threat/PII detection
type Detection struct {
	Type      string  `json:"type"`
	Value     string  `json:"value,omitempty"`
	Start     int     `json:"start,omitempty"`
	End       int     `json:"end,omitempty"`
	Severity  Severity `json:"severity,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Severity represents the severity level of a detection
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for GuardrailResult
func (r *GuardrailResult) MarshalJSON() ([]byte, error) {
	type Alias GuardrailResult
	return json.Marshal(&struct {
		*Alias
		ActionStr string `json:"action"`
	}{
		Alias:     (*Alias)(r),
		ActionStr: r.Action.String(),
	})
}

// String returns the string representation of GuardrailAction
func (a GuardrailAction) String() string {
	switch a {
	case ActionAllow:
		return "allow"
	case ActionBlock:
		return "block"
	case ActionWarn:
		return "warn"
	case ActionLog:
		return "log"
	default:
		return "unknown"
	}
}

// GuardrailConfig holds configuration for guardrails
type GuardrailConfig struct {
	Mode        GuardrailMode `json:"mode"`
	Enabled     bool          `json:"enabled"`
	Strict      bool          `json:"strict"`
	ScanBoundKB int           `json:"scan_bound_kb"`
}

// DefaultGuardrailConfig returns the default configuration
func DefaultGuardrailConfig() *GuardrailConfig {
	return &GuardrailConfig{
		Mode:        ModeWarn,
		Enabled:     true,
		Strict:      false,
		ScanBoundKB: 16, // 16KB default scan bound for injection detection
	}
}

// GuardrailStats holds statistics about guardrail execution
type GuardrailStats struct {
	TotalChecks    int64                  `json:"total_checks"`
	Passed         int64                  `json:"passed"`
	Blocked        int64                  `json:"blocked"`
	Warnings       int64                  `json:"warnings"`
	TotalDetections int64                 `json:"total_detections"`
	PIIDetections  int64                 `json:"pii_detections"`
	InjectionDetections int64             `json:"injection_detections"`
	AverageDuration time.Duration         `json:"average_duration"`
}
