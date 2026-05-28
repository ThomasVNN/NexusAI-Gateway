package model

import "time"

// RegisteredKey represents an authorized client API key
type RegisteredKey struct {
	ID          string    `json:"id"`
	KeyHash     string    `json:"key_hash"`
	Name        string    `json:"name"`
	SourceApp   string    `json:"source_app"`
	DailyQuota  int       `json:"daily_quota"`
	HourlyQuota int       `json:"hourly_quota"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UsageRecord represents an individual completion invocation log
type UsageRecord struct {
	ID               int       `json:"id"`
	KeyID            string    `json:"key_id"`
	ModelID          string    `json:"model_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	LatencyMS        int       `json:"latency_ms"`
	SourceApp        string    `json:"source_app"`
	CreatedAt        time.Time `json:"created_at"`
}

// MCPTool represents metadata for a Model Context Protocol tool
type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ProviderConnection represents active AI model provider integration settings
type ProviderConnection struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key"`
	Endpoint  string    `json:"endpoint"`
	IsActive  bool      `json:"is_active"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SystemSettings holds gateway portal metadata
type SystemSettings struct {
	SetupComplete bool   `json:"setup_complete"`
	Theme         string `json:"theme"`
	LoggingLevel  string `json:"logging_level"`
}
