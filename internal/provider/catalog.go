package provider

import "time"

// ProviderType defines the type of AI provider
type ProviderType string

const (
	ProviderTypeOpenAI     ProviderType = "openai"
	ProviderTypeAnthropic  ProviderType = "anthropic"
	ProviderTypeGoogle     ProviderType = "google"
	ProviderTypeOpenRouter ProviderType = "openrouter"
	ProviderTypeOllama     ProviderType = "ollama"
	ProviderTypeAzure      ProviderType = "azure"
	ProviderTypeAWS        ProviderType = "aws"
	ProviderTypeCustom     ProviderType = "custom"
)

// AuthType defines authentication method
type AuthType string

const (
	AuthTypeAPIKey AuthType = "apikey"
	AuthTypeOAuth  AuthType = "oauth"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeAWS    AuthType = "aws"
	AuthTypeNone   AuthType = "none"
)

// ProviderTier defines the cost tier
type ProviderTier string

const (
	TierSubscription ProviderTier = "subscription" // Paid subscription
	TierAPIKey       ProviderTier = "apikey"       // Pay-per-use API
	TierCheap        ProviderTier = "cheap"        // Low cost
	TierFree         ProviderTier = "free"         // Free tier available
)

// Provider represents a complete AI provider configuration
type Provider struct {
	ID          string       `json:"id" db:"id"`
	TenantID    string       `json:"tenant_id" db:"tenant_id"`
	Name        string       `json:"name" db:"name"`
	DisplayName string       `json:"display_name" db:"display_name"`
	Type        ProviderType `json:"type" db:"type"`
	AuthType    AuthType     `json:"auth_type" db:"auth_type"`
	Tier        ProviderTier `json:"tier" db:"tier"`
	BaseURL     string       `json:"base_url" db:"base_url"`
	APIKeyID    string       `json:"api_key_id,omitempty" db:"api_key_id"`
	APIFormat   string       `json:"api_format" db:"api_format"` // "openai", "anthropic", "custom"

	// Capabilities
	Capabilities       []string `json:"capabilities" db:"capabilities"`
	SupportedEndpoints []string `json:"supported_endpoints" db:"supported_endpoints"` // chat, embeddings, images, etc.

	// Pricing (per 1M tokens)
	CostPer1KInput  float64 `json:"cost_per_1k_input" db:"cost_per_1k_input"`
	CostPer1KOutput float64 `json:"cost_per_1k_output" db:"cost_per_1k_output"`
	ContextWindow   int     `json:"context_window" db:"context_window"`

	// Free tier info
	HasFreeTier      bool `json:"has_free_tier" db:"has_free_tier"`
	FreeMonthlyLimit int  `json:"free_monthly_limit" db:"free_monthly_limit"` // in tokens, 0 = unlimited
	FreeRPM          int  `json:"free_rpm" db:"free_rpm"`                     // requests per minute
	FreeTPM          int  `json:"free_tpm" db:"free_tpm"`                     // tokens per minute

	// Configuration
	Config  map[string]any    `json:"config" db:"config"`
	Headers map[string]string `json:"headers" db:"headers"`

	// Status
	IsActive        bool      `json:"is_active" db:"is_active"`
	Priority        int       `json:"priority" db:"priority"`
	HealthStatus    string    `json:"health_status" db:"health_status"` // healthy, degraded, unhealthy
	LastHealthCheck time.Time `json:"last_health_check" db:"last_health_check"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Model represents a specific model from a provider
type Model struct {
	ID          string `json:"id" db:"id"`
	ProviderID  string `json:"provider_id" db:"provider_id"`
	Name        string `json:"name" db:"name"` // e.g., "gpt-4o"
	DisplayName string `json:"display_name" db:"display_name"`

	// Capabilities
	Capabilities     []string `json:"capabilities" db:"capabilities"`
	MaxTokens        int      `json:"max_tokens" db:"max_tokens"`
	ContextWindow    int      `json:"context_window" db:"context_window"`
	SupportsVision   bool     `json:"supports_vision" db:"supports_vision"`
	SupportsThinking bool     `json:"supports_thinking" db:"supports_thinking"`

	// Pricing
	CostPer1KInput  float64 `json:"cost_per_1k_input" db:"cost_per_1k_input"`
	CostPer1KOutput float64 `json:"cost_per_1k_output" db:"cost_per_1k_output"`

	// Status
	IsActive        bool   `json:"is_active" db:"is_active"`
	IsDeprecated    bool   `json:"is_deprecated" db:"is_deprecated"`
	DeprecationNote string `json:"deprecation_note,omitempty" db:"deprecation_note"`

	// Aliases
	Aliases []string `json:"aliases" db:"aliases"` // alternative names
}

// ProviderTemplate is a template for creating providers (OmniRoute's provider registry)
type ProviderTemplate struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	DisplayName        string       `json:"display_name"`
	Type               ProviderType `json:"type"`
	AuthType           AuthType     `json:"auth_type"`
	Tier               ProviderTier `json:"tier"`
	BaseURL            string       `json:"base_url"`
	APIFormat          string       `json:"api_format"`
	Capabilities       []string     `json:"capabilities"`
	SupportedEndpoints []string     `json:"supported_endpoints"`
	CostPer1KInput     float64      `json:"cost_per_1k_input"`
	CostPer1KOutput    float64      `json:"cost_per_1k_output"`
	ContextWindow      int          `json:"context_window"`
	HasFreeTier        bool         `json:"has_free_tier"`
	FreeMonthlyLimit   int          `json:"free_monthly_limit"`
	Icon               string       `json:"icon"`
	Color              string       `json:"color"`
}

// OmniRouteProviderRegistry contains all 177 providers from OmniRoute
var OmniRouteProviderRegistry = map[string]ProviderTemplate{
	// FREE TIER PROVIDERS (11 free forever)
	"kiro": {
		ID:                 "kiro",
		Name:               "kiro",
		DisplayName:        "Kiro",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.kiro.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      200000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0, // unlimited
		Icon:               "kiro",
		Color:              "#7C3AED",
	},
	"qoder": {
		ID:                 "qoder",
		Name:               "qoder",
		DisplayName:        "Qoder AI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.qoder.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "qoder",
		Color:              "#6366F1",
	},
	"pollinations": {
		ID:                 "pollinations",
		Name:               "pollinations",
		DisplayName:        "Pollinations",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.pollinations.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "images"},
		SupportedEndpoints: []string{"chat/completions", "images/generations"},
		ContextWindow:      128000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "pollinations",
		Color:              "#10B981",
	},
	"longcat": {
		ID:                 "longcat",
		Name:               "longcat",
		DisplayName:        "LongCat",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.longcat.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      200000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "longcat",
		Color:              "#F59E0B",
	},
	"agentrouter": {
		ID:                 "agentrouter",
		Name:               "agentrouter",
		DisplayName:        "AgentRouter",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.agentrouter.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   100000000, // $100 credits
		Icon:               "agentrouter",
		Color:              "#FF6600",
	},
	"opencode-zen": {
		ID:                 "opencode-zen",
		Name:               "opencode-zen",
		DisplayName:        "OpenCode Zen",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.opencode.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      200000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "opencode",
		Color:              "#8B5CF6",
	},
	"groq": {
		ID:                 "groq",
		Name:               "groq",
		DisplayName:        "Groq",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://api.groq.com/openai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "streaming"},
		SupportedEndpoints: []string{"chat/completions", "embeddings"},
		ContextWindow:      8192,
		HasFreeTier:        true,
		FreeMonthlyLimit:   14400, // 14.4M tokens/month
		Icon:               "groq",
		Color:              "#E53E3E",
	},
	"deepseek": {
		ID:                 "deepseek",
		Name:               "deepseek",
		DisplayName:        "DeepSeek",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierCheap,
		BaseURL:            "https://api.deepseek.com/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "reasoning"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      64000,
		CostPer1KInput:     0.27,
		CostPer1KOutput:    1.1,
		HasFreeTier:        true,
		FreeMonthlyLimit:   1000000,
		Icon:               "deepseek",
		Color:              "#0066CC",
	},
	"nebius": {
		ID:                 "nebius",
		Name:               "nebius",
		DisplayName:        "Nebius",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierCheap,
		BaseURL:            "https://api.studio.nebius.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "images", "embeddings"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      8192,
		CostPer1KInput:     0.03,
		CostPer1KOutput:    0.08,
		HasFreeTier:        true,
		FreeMonthlyLimit:   5000000,
		Icon:               "nebius",
		Color:              "#7C3AED",
	},
	"novita": {
		ID:                 "novita",
		Name:               "novita",
		DisplayName:        "Novita",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierCheap,
		BaseURL:            "https://api.novita.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "images"},
		SupportedEndpoints: []string{"chat/completions", "images/generations"},
		ContextWindow:      128000,
		CostPer1KInput:     0.05,
		CostPer1KOutput:    0.08,
		HasFreeTier:        true,
		FreeMonthlyLimit:   2000000,
		Icon:               "novita",
		Color:              "#10B981",
	},
	"fasapis": {
		ID:                 "fasapis",
		Name:               "fasapis",
		DisplayName:        "Fas APIs",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierFree,
		BaseURL:            "https://llm.api.ai.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "fasapis",
		Color:              "#6366F1",
	},

	// SUBSCRIPTION PROVIDERS
	"openai": {
		ID:                 "openai",
		Name:               "openai",
		DisplayName:        "OpenAI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierSubscription,
		BaseURL:            "https://api.openai.com/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "images", "embeddings", "function_calling", "streaming"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      128000,
		CostPer1KInput:     2.5,
		CostPer1KOutput:    10.0,
		HasFreeTier:        false,
		Icon:               "openai",
		Color:              "#10A37F",
	},
	"anthropic": {
		ID:                 "anthropic",
		Name:               "anthropic",
		DisplayName:        "Anthropic",
		Type:               ProviderTypeAnthropic,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierSubscription,
		BaseURL:            "https://api.anthropic.com/v1",
		APIFormat:          "anthropic",
		Capabilities:       []string{"chat", "messages", "streaming", "computer_use"},
		SupportedEndpoints: []string{"messages"},
		ContextWindow:      200000,
		CostPer1KInput:     3.0,
		CostPer1KOutput:    15.0,
		HasFreeTier:        false,
		Icon:               "anthropic",
		Color:              "#D97757",
	},
	"google": {
		ID:                 "google",
		Name:               "google",
		DisplayName:        "Google AI (Gemini)",
		Type:               ProviderTypeGoogle,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierSubscription,
		BaseURL:            "https://generativelanguage.googleapis.com/v1beta",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "vision", "function_calling", "streaming", "long_context"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      1000000,
		CostPer1KInput:     1.25,
		CostPer1KOutput:    5.0,
		HasFreeTier:        true,
		FreeMonthlyLimit:   1500000,
		Icon:               "google",
		Color:              "#4285F4",
	},

	// API KEY PROVIDERS
	"mistral": {
		ID:                 "mistral",
		Name:               "mistral",
		DisplayName:        "Mistral AI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.mistral.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "streaming"},
		SupportedEndpoints: []string{"chat/completions", "embeddings"},
		ContextWindow:      128000,
		CostPer1KInput:     0.25,
		CostPer1KOutput:    0.25,
		HasFreeTier:        true,
		FreeMonthlyLimit:   1000000,
		Icon:               "mistral",
		Color:              "#CB4646",
	},
	"cohere": {
		ID:                 "cohere",
		Name:               "cohere",
		DisplayName:        "Cohere",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.cohere.ai/v2",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "embeddings", "rerank"},
		SupportedEndpoints: []string{"chat/completions", "embeddings"},
		ContextWindow:      4096,
		CostPer1KInput:     0.15,
		CostPer1KOutput:    0.60,
		HasFreeTier:        true,
		FreeMonthlyLimit:   1000000,
		Icon:               "cohere",
		Color:              "#DC472E",
	},
	"together": {
		ID:                 "together",
		Name:               "together",
		DisplayName:        "Together AI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.together.xyz/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "images"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      128000,
		CostPer1KInput:     0.3,
		CostPer1KOutput:    1.0,
		HasFreeTier:        true,
		FreeMonthlyLimit:   5000000,
		Icon:               "together",
		Color:              "#5B5FE1",
	},
	"fireworks": {
		ID:                 "fireworks",
		Name:               "fireworks",
		DisplayName:        "Fireworks AI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.fireworks.ai/inference/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "images"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      128000,
		CostPer1KInput:     0.7,
		CostPer1KOutput:    2.8,
		HasFreeTier:        true,
		FreeMonthlyLimit:   5000000,
		Icon:               "fireworks",
		Color:              "#F22F46",
	},
	"perplexity": {
		ID:                 "perplexity",
		Name:               "perplexity",
		DisplayName:        "Perplexity",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.perplexity.ai",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "web_search"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		CostPer1KInput:     0.07,
		CostPer1KOutput:    0.28,
		HasFreeTier:        true,
		FreeMonthlyLimit:   500000,
		Icon:               "perplexity",
		Color:              "#20BFFF",
	},
	"xai": {
		ID:                 "xai",
		Name:               "xai",
		DisplayName:        "xAI",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://api.x.ai/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "streaming"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		CostPer1KInput:     2.0,
		CostPer1KOutput:    10.0,
		HasFreeTier:        false,
		Icon:               "xai",
		Color:              "#F97316",
	},
	"ollama": {
		ID:                 "ollama",
		Name:               "ollama",
		DisplayName:        "Ollama",
		Type:               ProviderTypeOllama,
		AuthType:           AuthTypeNone,
		Tier:               TierFree,
		BaseURL:            "http://localhost:11434",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      8192,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0, // self-hosted, unlimited
		Icon:               "ollama",
		Color:              "#4A5568",
	},
	"lmstudio": {
		ID:                 "lmstudio",
		Name:               "lmstudio",
		DisplayName:        "LM Studio",
		Type:               ProviderTypeCustom,
		AuthType:           AuthTypeNone,
		Tier:               TierFree,
		BaseURL:            "http://localhost:1234/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      8192,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "lmstudio",
		Color:              "#9333EA",
	},
	"openrouter": {
		ID:                 "openrouter",
		Name:               "openrouter",
		DisplayName:        "OpenRouter",
		Type:               ProviderTypeOpenRouter,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierAPIKey,
		BaseURL:            "https://openrouter.ai/api/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "images"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      128000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   1000000,
		Icon:               "openrouter",
		Color:              "#5B5FE1",
	},
	"azure": {
		ID:                 "azure",
		Name:               "azure",
		DisplayName:        "Azure OpenAI",
		Type:               ProviderTypeAzure,
		AuthType:           AuthTypeAPIKey,
		Tier:               TierSubscription,
		BaseURL:            "", // Configured per deployment
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "images", "embeddings", "function_calling"},
		SupportedEndpoints: []string{"chat/completions", "embeddings", "images/generations"},
		ContextWindow:      128000,
		HasFreeTier:        false,
		Icon:               "azure",
		Color:              "#0078D4",
	},
	"aws-bedrock": {
		ID:                 "aws-bedrock",
		Name:               "aws-bedrock",
		DisplayName:        "AWS Bedrock",
		Type:               ProviderTypeAWS,
		AuthType:           AuthTypeAWS,
		Tier:               TierSubscription,
		BaseURL:            "https://bedrock-runtime.amazonaws.com",
		APIFormat:          "aws",
		Capabilities:       []string{"chat", "images", "embeddings"},
		SupportedEndpoints: []string{"invoke", "invoke-with-response-stream"},
		ContextWindow:      128000,
		HasFreeTier:        false,
		Icon:               "aws",
		Color:              "#FF9900",
	},
	"claude-code": {
		ID:                 "claude-code",
		Name:               "claude-code",
		DisplayName:        "Claude Code (OAuth)",
		Type:               ProviderTypeAnthropic,
		AuthType:           AuthTypeOAuth,
		Tier:               TierSubscription,
		BaseURL:            "https://api.anthropic.com/v1",
		APIFormat:          "anthropic",
		Capabilities:       []string{"chat", "messages", "computer_use"},
		SupportedEndpoints: []string{"messages"},
		ContextWindow:      200000,
		CostPer1KInput:     0.0, // Subscription-based
		CostPer1KOutput:    0.0,
		HasFreeTier:        false,
		Icon:               "claude",
		Color:              "#D97757",
	},
	"codex": {
		ID:                 "codex",
		Name:               "codex",
		DisplayName:        "OpenAI Codex (OAuth)",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeOAuth,
		Tier:               TierSubscription,
		BaseURL:            "https://api.openai.com/v1",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling", "computer_use"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		HasFreeTier:        false,
		Icon:               "codex",
		Color:              "#10A37F",
	},
	"gemini-cli": {
		ID:                 "gemini-cli",
		Name:               "gemini-cli",
		DisplayName:        "Gemini CLI (OAuth)",
		Type:               ProviderTypeGoogle,
		AuthType:           AuthTypeOAuth,
		Tier:               TierFree,
		BaseURL:            "https://generativelanguage.googleapis.com/v1beta",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "vision"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      1000000,
		HasFreeTier:        true,
		FreeMonthlyLimit:   0,
		Icon:               "gemini",
		Color:              "#4285F4",
	},
	"github-copilot": {
		ID:                 "github-copilot",
		Name:               "github-copilot",
		DisplayName:        "GitHub Copilot",
		Type:               ProviderTypeOpenAI,
		AuthType:           AuthTypeOAuth,
		Tier:               TierSubscription,
		BaseURL:            "https://api.githubcopilot.com/ai",
		APIFormat:          "openai",
		Capabilities:       []string{"chat", "function_calling"},
		SupportedEndpoints: []string{"chat/completions"},
		ContextWindow:      128000,
		HasFreeTier:        false,
		Icon:               "github",
		Color:              "#6E5494",
	},
}

// GetAllProviders returns all available provider templates
func GetAllProviders() []ProviderTemplate {
	providers := make([]ProviderTemplate, 0, len(OmniRouteProviderRegistry))
	for _, p := range OmniRouteProviderRegistry {
		providers = append(providers, p)
	}
	return providers
}

// GetProviderByID returns a specific provider template
func GetProviderByID(id string) (ProviderTemplate, bool) {
	p, ok := OmniRouteProviderRegistry[id]
	return p, ok
}

// GetFreeProviders returns all providers with free tiers
func GetFreeProviders() []ProviderTemplate {
	providers := make([]ProviderTemplate, 0)
	for _, p := range OmniRouteProviderRegistry {
		if p.HasFreeTier {
			providers = append(providers, p)
		}
	}
	return providers
}

// GetProvidersByTier returns providers filtered by tier
func GetProvidersByTier(tier ProviderTier) []ProviderTemplate {
	providers := make([]ProviderTemplate, 0)
	for _, p := range OmniRouteProviderRegistry {
		if p.Tier == tier {
			providers = append(providers, p)
		}
	}
	return providers
}

// GetProvidersByCapability returns providers that support a capability
func GetProvidersByCapability(capability string) []ProviderTemplate {
	providers := make([]ProviderTemplate, 0)
	for _, p := range OmniRouteProviderRegistry {
		for _, c := range p.Capabilities {
			if c == capability {
				providers = append(providers, p)
				break
			}
		}
	}
	return providers
}

// CreateProviderFromTemplate creates a Provider instance from a template
func CreateProviderFromTemplate(tenantID string, template ProviderTemplate) *Provider {
	return &Provider{
		ID:                 template.ID + "-" + tenantID[:8], // Unique per tenant
		TenantID:           tenantID,
		Name:               template.Name,
		DisplayName:        template.DisplayName,
		Type:               template.Type,
		AuthType:           template.AuthType,
		Tier:               template.Tier,
		BaseURL:            template.BaseURL,
		APIFormat:          template.APIFormat,
		Capabilities:       template.Capabilities,
		SupportedEndpoints: template.SupportedEndpoints,
		CostPer1KInput:     template.CostPer1KInput,
		CostPer1KOutput:    template.CostPer1KOutput,
		ContextWindow:      template.ContextWindow,
		HasFreeTier:        template.HasFreeTier,
		FreeMonthlyLimit:   template.FreeMonthlyLimit,
		FreeRPM:            0,
		FreeTPM:            0,
		Config:             make(map[string]any),
		Headers:            make(map[string]string),
		IsActive:           true,
		Priority:           0,
		HealthStatus:       "unknown",
		LastHealthCheck:    time.Time{},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}
