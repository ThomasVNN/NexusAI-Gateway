package channel

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ChannelType defines the type of upstream AI provider
type ChannelType string

const (
	ChannelTypeOpenAI    ChannelType = "openai"
	ChannelTypeAnthropic ChannelType = "anthropic"
	ChannelTypeGoogle    ChannelType = "google"
	ChannelTypeAzure     ChannelType = "azure"
	ChannelTypeCohere    ChannelType = "cohere"
	ChannelTypeMistral   ChannelType = "mistral"
	ChannelTypeOllama    ChannelType = "ollama"
	ChannelTypeCustom    ChannelType = "custom"
)

// Channel represents an upstream AI provider connection
type Channel struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	Type            ChannelType `json:"type"`
	BaseURL         string      `json:"base_url"`
	APIKeyEncrypted string      `json:"-"` // Never expose in JSON
	Models          []string    `json:"models"`
	Priority        int         `json:"priority"`
	Ratio           int         `json:"ratio"` // Weight for load balancing
	IsActive        bool        `json:"is_active"`
	Balance         float64     `json:"balance"`      // Account balance
	BalanceType     string      `json:"balance_type"` // "prepay" or "postpay"
	GroupName       string      `json:"group_name"`   // For grouping channels
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// ChannelGroup represents a group of channels for load balancing
type ChannelGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Priority  int       `json:"priority"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
}

// ChannelTestResult represents the result of testing a channel connection
type ChannelTestResult struct {
	ChannelID int64     `json:"channel_id"`
	Success   bool      `json:"success"`
	LatencyMS int64     `json:"latency_ms"`
	ErrorMsg  string    `json:"error_message,omitempty"`
	TestedAt  time.Time `json:"tested_at"`
}

// ChannelStatus represents the current health status of a channel
type ChannelStatus struct {
	ChannelID     int64        `json:"channel_id"`
	State         string       `json:"state"` // "healthy", "degraded", "circuit_open"
	FailureCount  int          `json:"failure_count"`
	LastFailureAt sql.NullTime `json:"last_failure_at,omitempty"`
	LastSuccessAt sql.NullTime `json:"last_success_at,omitempty"`
}

// ToJSON converts channel to JSON string for storage
func (c *Channel) ToJSON() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ChannelFromJSON parses channel from JSON string
func ChannelFromJSON(data string) (*Channel, error) {
	var ch Channel
	if err := json.Unmarshal([]byte(data), &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// IsModelSupported checks if the channel supports a given model
func (c *Channel) IsModelSupported(model string) bool {
	if len(c.Models) == 0 {
		return true // Empty list means all models supported
	}
	for _, m := range c.Models {
		if m == model || m == "*" {
			return true
		}
	}
	return false
}

// GetEffectiveURL returns the base URL for API calls
func (c *Channel) GetEffectiveURL() string {
	if c.BaseURL == "" {
		return getDefaultURL(c.Type)
	}
	return c.BaseURL
}

// getDefaultURL returns the default API URL for a channel type
func getDefaultURL(chType ChannelType) string {
	switch chType {
	case ChannelTypeOpenAI:
		return "https://api.openai.com/v1"
	case ChannelTypeAnthropic:
		return "https://api.anthropic.com/v1"
	case ChannelTypeGoogle:
		return "https://generativelanguage.googleapis.com/v1"
	case ChannelTypeAzure:
		return "" // Azure uses custom endpoints
	case ChannelTypeCohere:
		return "https://api.cohere.ai/v1"
	case ChannelTypeMistral:
		return "https://api.mistral.ai/v1"
	case ChannelTypeOllama:
		return "http://localhost:11434/v1"
	default:
		return ""
	}
}

// Validate checks if the channel configuration is valid
func (c *Channel) Validate() error {
	if c.Name == "" {
		return ErrChannelNameRequired
	}
	if c.Type == "" {
		return ErrChannelTypeRequired
	}
	if c.APIKeyEncrypted == "" && c.Type != ChannelTypeOllama {
		return ErrChannelAPIKeyRequired
	}
	return nil
}

// Custom errors
type ChannelError struct {
	Message string
}

func (e *ChannelError) Error() string {
	return e.Message
}

var (
	ErrChannelNameRequired   = &ChannelError{Message: "channel name is required"}
	ErrChannelTypeRequired   = &ChannelError{Message: "channel type is required"}
	ErrChannelAPIKeyRequired = &ChannelError{Message: "API key is required for this channel type"}
	ErrChannelNotFound       = &ChannelError{Message: "channel not found"}
	ErrChannelInactive       = &ChannelError{Message: "channel is inactive"}
)
