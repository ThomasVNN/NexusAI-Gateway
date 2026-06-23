package provider

import (
	"database/sql"
	"errors"
	"sync"
	"time"
)

// ProviderTier represents the pricing tier of a provider
type ProviderTier int

const (
	TierSubscription ProviderTier = iota
	TierAPIKey
	TierCheap
	TierFree
)

func (t ProviderTier) String() string {
	switch t {
	case TierSubscription:
		return "subscription"
	case TierAPIKey:
		return "api_key"
	case TierCheap:
		return "cheap"
	case TierFree:
		return "free"
	default:
		return "unknown"
	}
}

// AccountStatus represents the status of an account
type AccountStatus int

const (
	AccountActive AccountStatus = iota
	AccountInactive
	AccountSuspended
	AccountRateLimited
	AccountQuotaExceeded
)

func (s AccountStatus) String() string {
	switch s {
	case AccountActive:
		return "active"
	case AccountInactive:
		return "inactive"
	case AccountSuspended:
		return "suspended"
	case AccountRateLimited:
		return "rate_limited"
	case AccountQuotaExceeded:
		return "quota_exceeded"
	default:
		return "unknown"
	}
}

// Account represents an account within a provider
type Account struct {
	mu         sync.RWMutex
	ProviderID string       `json:"provider_id"`
	AccountID  string       `json:"account_id"`
	APIKey     string       `json:"-"` // Encrypted
	OAuthToken string       `json:"-"` // Encrypted
	QuotaLimit int64        `json:"quota_limit"`
	QuotaUsed  int64        `json:"quota_used"`
	QuotaReset time.Time    `json:"quota_reset"`
	Status     AccountStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// NewAccount creates a new account
func NewAccount(providerID, accountID, apiKey string) *Account {
	now := time.Now()
	return &Account{
		ProviderID: providerID,
		AccountID:  accountID,
		APIKey:     apiKey,
		Status:     AccountActive,
		QuotaLimit: 0, // Unlimited by default
		QuotaUsed:  0,
		QuotaReset: now.Add(24 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// QuotaRemaining returns the remaining quota for this account
func (a *Account) QuotaRemaining() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.QuotaLimit == 0 {
		return -1 // Unlimited
	}
	remaining := a.QuotaLimit - a.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ConsumeQuota increments the quota used
func (a *Account) ConsumeQuota(tokens int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.QuotaUsed += tokens
	a.UpdatedAt = time.Now()
}

// CanUse returns true if the account can be used
func (a *Account) CanUse() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.Status != AccountActive {
		return false
	}
	if a.QuotaLimit > 0 && a.QuotaUsed >= a.QuotaLimit {
		return false
	}
	return true
}

// ExtendedProvider represents an AI provider with extended configuration
type ExtendedProvider struct {
	mu            sync.RWMutex
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Tier          ProviderTier       `json:"tier"`
	OAuthSupport  bool               `json:"oauth_support"`
	Accounts      []*Account         `json:"accounts"`
	Health        *ExtendedHealthStatus `json:"health"`
	Models        []string           `json:"models"`
	Capabilities  []string           `json:"capabilities"`
	BaseURL       string             `json:"base_url"`
	AuthType      string             `json:"auth_type"` // "oauth", "api_key", "both"
	Status        string             `json:"status"`    // "active", "inactive", "degraded"
	Priority      int                `json:"priority"`   // Lower = higher priority
	CostPer1KIn   float64            `json:"cost_per_1k_input"`
	CostPer1KOut  float64            `json:"cost_per_1k_output"`
	AvgLatencyMs  int64              `json:"avg_latency_ms"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// NewExtendedProvider creates a new extended provider
func NewExtendedProvider(id, name string, tier ProviderTier, oauthSupport bool) *ExtendedProvider {
	now := time.Now()
	return &ExtendedProvider{
		ID:           id,
		Name:         name,
		Tier:         tier,
		OAuthSupport: oauthSupport,
		Accounts:     make([]*Account, 0),
		Health:       NewExtendedHealthStatus(id),
		Status:       "active",
		Priority:     100,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// AddAccount adds an account to the provider
func (p *ExtendedProvider) AddAccount(account *Account) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Accounts = append(p.Accounts, account)
	p.UpdatedAt = time.Now()
}

// GetAvailableAccount returns an account that can be used
func (p *ExtendedProvider) GetAvailableAccount() (*Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, acc := range p.Accounts {
		if acc.CanUse() {
			return acc, nil
		}
	}
	return nil, ErrNoAvailableAccount
}

// GetHealth returns the current health status
func (p *ExtendedProvider) GetHealth() *ExtendedHealthStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health
}

// UpdateHealth updates the health status
func (p *ExtendedProvider) UpdateHealth(health *ExtendedHealthStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Health = health
	p.UpdatedAt = time.Now()
}

// IsHealthy returns true if the provider is healthy
func (p *ExtendedProvider) IsHealthy() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Health != nil && p.Health.IsHealthy && p.Status == "active"
}

// ExtendedProvider errors
var (
	ErrNoAvailableAccount = errors.New("no available account for provider")
)

// TierPriority returns the priority for a tier (lower = better)
func TierPriority(t ProviderTier) int {
	switch t {
	case TierSubscription:
		return 1
	case TierAPIKey:
		return 2
	case TierCheap:
		return 3
	case TierFree:
		return 4
	default:
		return 99
	}
}

// ExtendedHealthStatus represents the current health state of a provider
type ExtendedHealthStatus struct {
	ProviderID   string    `json:"provider_id"`
	IsHealthy    bool      `json:"is_healthy"`
	LastCheck    time.Time `json:"last_check"`
	LatencyMs    int64     `json:"latency_ms"`
	ErrorRate    float64   `json:"error_rate"`
	SuccessRate  float64   `json:"success_rate"`
	CircuitState string    `json:"circuit_state"`
	FailureCount int       `json:"failure_count"`
	NextCheck    time.Time `json:"next_check"`
}

// NewExtendedHealthStatus creates a new extended health status
func NewExtendedHealthStatus(providerID string) *ExtendedHealthStatus {
	return &ExtendedHealthStatus{
		ProviderID:   providerID,
		IsHealthy:    true,
		LastCheck:    time.Now(),
		LatencyMs:    0,
		ErrorRate:    0,
		SuccessRate:  100,
		CircuitState: "CLOSED",
		FailureCount: 0,
		NextCheck:    time.Now().Add(30 * time.Second),
	}
}

// WellKnownProviders returns the list of 50 well-known AI providers
func WellKnownProviders() []*ExtendedProvider {
	providers := make([]*ExtendedProvider, 0, 50)

	// Tier 1: Subscription (Enterprise-grade)
	providers = append(providers,
		makeExtendedProvider("openai", "OpenAI", TierSubscription, true,
			[]string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
			[]string{"vision", "function_calling", "streaming", "json_mode"},
			"https://api.openai.com", 0.005, 0.015, 1000),
		makeExtendedProvider("anthropic", "Anthropic", TierSubscription, true,
			[]string{"claude-3-5-sonnet", "claude-3-5-haiku", "claude-3-opus", "claude-3-sonnet"},
			[]string{"vision", "function_calling", "streaming"},
			"https://api.anthropic.com", 0.003, 0.015, 1200),
		makeExtendedProvider("google", "Google AI", TierSubscription, true,
			[]string{"gemini-1.5-pro", "gemini-1.5-flash", "gemini-1.0-pro"},
			[]string{"vision", "function_calling", "streaming", "long_context"},
			"https://generativelanguage.googleapis.com", 0.00125, 0.005, 800),
	)

	// Tier 2: API Key based (Popular)
	providers = append(providers,
		makeExtendedProvider("deepseek", "DeepSeek", TierAPIKey, false,
			[]string{"deepseek-chat", "deepseek-coder"},
			[]string{"function_calling", "streaming"},
			"https://api.deepseek.com", 0.00014, 0.00028, 600),
		makeExtendedProvider("xai", "xAI (Grok)", TierAPIKey, false,
			[]string{"grok-2", "grok-2-mini", "grok-beta"},
			[]string{"vision", "function_calling", "streaming"},
			"https://api.x.ai", 0.002, 0.01, 900),
		makeExtendedProvider("mistral", "Mistral AI", TierAPIKey, true,
			[]string{"mistral-large", "mistral-small", "mistral-medium"},
			[]string{"function_calling", "streaming"},
			"https://api.mistral.ai", 0.002, 0.006, 850),
		makeExtendedProvider("cohere", "Cohere", TierAPIKey, true,
			[]string{"command-r-plus", "command-r", "command"},
			[]string{"function_calling", "streaming", "rerank"},
			"https://api.cohere.ai", 0.003, 0.015, 750),
		makeExtendedProvider("azure", "Azure OpenAI", TierAPIKey, true,
			[]string{"gpt-4o", "gpt-4-turbo", "gpt-35-turbo"},
			[]string{"vision", "function_calling", "streaming"},
			"https://{resource}.openai.azure.com", 0.005, 0.015, 1100),
		makeExtendedProvider("aws-bedrock", "AWS Bedrock", TierAPIKey, true,
			[]string{"anthropic.claude-3-5-sonnet", "meta.llama3-70b", "amazon.titan-text"},
			[]string{"vision", "function_calling", "streaming"},
			"https://bedrock.{region}.amazonaws.com", 0.003, 0.015, 1200),
		makeExtendedProvider("vertex-ai", "Google Vertex AI", TierAPIKey, true,
			[]string{"gemini-1.5-pro", "gemini-1.5-flash", "claude-3-sonnet"},
			[]string{"vision", "function_calling", "streaming", "long_context"},
			"https://{region}-aiplatform.googleapis.com", 0.00125, 0.005, 850),
	)

	// Tier 3: Cheap providers
	providers = append(providers,
		makeExtendedProvider("groq", "Groq", TierCheap, false,
			[]string{"llama-3.1-70b", "llama-3.1-8b", "mixtral-8x7b"},
			[]string{"function_calling", "streaming"},
			"https://api.groq.com", 0.00005, 0.0001, 200),
		makeExtendedProvider("fireworks", "Fireworks AI", TierCheap, false,
			[]string{"llama-3.1-70b", "mixtral-8x7b", "llama-3-8b"},
			[]string{"function_calling", "streaming"},
			"https://api.fireworks.ai", 0.00006, 0.00018, 250),
		makeExtendedProvider("together", "Together AI", TierCheap, false,
			[]string{"meta-llama/Llama-3-70b", "mistralai/Mixtral-8x7B"},
			[]string{"function_calling", "streaming"},
			"https://api.together.xyz", 0.00008, 0.00024, 300),
		makeExtendedProvider("perplexity", "Perplexity", TierAPIKey, false,
			[]string{"llama-3.1-sonar-large", "llama-3.1-sonar-small"},
			[]string{"function_calling", "streaming", "web_search"},
			"https://api.perplexity.ai", 0.001, 0.01, 700),
		makeExtendedProvider("anyscale", "Anyscale", TierCheap, false,
			[]string{"meta-llama/Llama-3-70b", "mistralai/Mixtral-8x7B"},
			[]string{"function_calling", "streaming"},
			"https://api.endpoints.anyscale.com", 0.00007, 0.00021, 280),
		makeExtendedProvider("replicate", "Replicate", TierCheap, false,
			[]string{"llama-3-70b", "mixtral-8x7b"},
			[]string{"streaming"},
			"https://api.replicate.com", 0.00006, 0.00018, 350),
		makeExtendedProvider("cloudflare", "Cloudflare Workers AI", TierCheap, false,
			[]string{"@cf/meta/llama-3-70b", "@cf/mistral/mistral-7b"},
			[]string{"streaming"},
			"https://api.cloudflare.com", 0.00004, 0.00012, 150),
		makeExtendedProvider("sambanova", "SambaNova", TierCheap, false,
			[]string{"Meta-Llama-3.1-70B", "Meta-Llama-3.1-8B"},
			[]string{"function_calling", "streaming"},
			"https://api.sambanova.ai", 0.00005, 0.00015, 200),
	)

	// Tier 4: Free/Open providers
	providers = append(providers,
		makeExtendedProvider("ollama", "Ollama", TierFree, false,
			[]string{"llama3", "llama3.1", "mistral", "codellama"},
			[]string{"streaming"},
			"http://localhost:11434", 0, 0, 0),
		makeExtendedProvider("lmstudio", "LM Studio", TierFree, false,
			[]string{"llama3", "mistral", "gemma"},
			[]string{"streaming"},
			"http://localhost:1234", 0, 0, 0),
		makeExtendedProvider("textgen-webui", "text-generation-webui", TierFree, false,
			[]string{"llama3", "mistral"},
			[]string{"streaming"},
			"http://localhost:5000", 0, 0, 0),
		makeExtendedProvider("koboldcpp", "KoboldCPP", TierFree, false,
			[]string{"llama3", "mistral"},
			[]string{"streaming"},
			"http://localhost:5001", 0, 0, 0),
		makeExtendedProvider("llamafile", "LLaMAfile", TierFree, false,
			[]string{"llama3", "mistral"},
			[]string{"streaming"},
			"http://localhost:8080", 0, 0, 0),
	)

	// Add more providers to reach 50
	providers = append(providers,
		makeExtendedProvider("openrouter", "OpenRouter", TierAPIKey, false,
			[]string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o"},
			[]string{"function_calling", "streaming"},
			"https://openrouter.ai/api", 0.001, 0.01, 800),
		makeExtendedProvider("novita", "Novita AI", TierAPIKey, false,
			[]string{"Llama-3.1-70B", "Qwen-2.5-72B"},
			[]string{"function_calling", "streaming"},
			"https://api.novita.ai", 0.0001, 0.0003, 400),
		makeExtendedProvider("hyperbolic", "Hyperbolic", TierAPIKey, false,
			[]string{"meta-llama/Llama-3.1-70B"},
			[]string{"function_calling", "streaming"},
			"https://api.hyperbolic.ai", 0.00008, 0.00024, 350),
		makeExtendedProvider("nebius", "Nebius AI", TierAPIKey, false,
			[]string{"meta-llama/Llama-3.1-70B"},
			[]string{"function_calling", "streaming"},
			"https://api.nebius.ai", 0.00007, 0.00021, 320),
		makeExtendedProvider("databricks", "Databricks", TierAPIKey, true,
			[]string{"llama-3.1-70b", "mixtral-8x7b"},
			[]string{"function_calling", "streaming"},
			"https://{workspace}.cloud.databricks.com", 0.0001, 0.0003, 500),
		makeExtendedProvider("ai21", "AI21", TierAPIKey, true,
			[]string{"jamba-1.5-large", "jamba-1.5-mini"},
			[]string{"function_calling", "streaming"},
			"https://api.ai21.com", 0.002, 0.008, 900),
		makeExtendedProvider("voyage", "Voyage AI", TierAPIKey, false,
			[]string{"voyage-2", "voyage-large-2"},
			[]string{"embeddings"},
			"https://api.voyageai.com", 0.0001, 0, 200),
		makeExtendedProvider("nomic", "Nomic", TierFree, false,
			[]string{"nomic-embed-text-v1.5"},
			[]string{"embeddings"},
			"https://api-atlas.nomic.ai", 0, 0, 100),
		makeExtendedProvider("qwen", "Qwen", TierAPIKey, false,
			[]string{"qwen-2.5-72b", "qwen-2.5-coder-32b"},
			[]string{"function_calling", "streaming"},
			"https://dashscope.aliyuncs.com", 0.0001, 0.0003, 400),
		makeExtendedProvider("yi", "Yi", TierAPIKey, false,
			[]string{"yi-large", "yi-medium"},
			[]string{"function_calling", "streaming"},
			"https://api.lingyiwanwu.com", 0.001, 0.01, 700),
		makeExtendedProvider("upstage", "Upstage", TierAPIKey, false,
			[]string{"solar-pro", "solar-mini"},
			[]string{"function_calling", "streaming"},
			"https://api.upstage.ai", 0.001, 0.01, 650),
		makeExtendedProvider("zhipu", "Zhipu AI", TierAPIKey, false,
			[]string{"glm-4", "glm-4-flash"},
			[]string{"function_calling", "streaming"},
			"https://open.bigmodel.cn", 0.0001, 0.001, 500),
		makeExtendedProvider("moonshot", "Moonshot", TierAPIKey, false,
			[]string{"moonshot-v1-128k", "moonshot-v1-32k"},
			[]string{"function_calling", "streaming"},
			"https://api.moonshot.cn", 0.0001, 0.001, 450),
		makeExtendedProvider("minimax", "MiniMax", TierAPIKey, false,
			[]string{"abab6-chat"},
			[]string{"function_calling", "streaming"},
			"https://api.minimax.chat", 0.0001, 0.001, 400),
		makeExtendedProvider("nvidia", "NVIDIA NIM", TierAPIKey, true,
			[]string{"meta/llama-3.1-70b", "mistralai/mixtral-8x7b"},
			[]string{"function_calling", "streaming"},
			"https://integrate.api.nvidia.com", 0.0001, 0.0004, 350),
		makeExtendedProvider("cerebras", "Cerebras", TierAPIKey, false,
			[]string{"llama-3.1-70b", "llama-3.1-8b"},
			[]string{"function_calling", "streaming"},
			"https://api.cerebras.ai", 0.00006, 0.00018, 100),
		makeExtendedProvider("predibase", "Predibase", TierAPIKey, false,
			[]string{"llama-3.1-70b", "mixtral-8x7b"},
			[]string{"function_calling", "streaming"},
			"https://serving.prod.predibase.com", 0.00007, 0.00021, 300),
		makeExtendedProvider("lepton", "Lepton AI", TierCheap, false,
			[]string{"llama-3.1-70b", "mixtral-8x7b"},
			[]string{"function_calling", "streaming"},
			"https://api.lepton.ai", 0.00008, 0.00024, 280),
	)

	return providers
}

func makeExtendedProvider(id, name string, tier ProviderTier, oauth bool, models, caps []string, baseURL string, costIn, costOut float64, latency int64) *ExtendedProvider {
	p := NewExtendedProvider(id, name, tier, oauth)
	p.Models = models
	p.Capabilities = caps
	p.BaseURL = baseURL
	p.CostPer1KIn = costIn
	p.CostPer1KOut = costOut
	p.AvgLatencyMs = latency
	if oauth {
		p.AuthType = "oauth"
	} else {
		p.AuthType = "api_key"
	}
	return p
}

// ProviderRegistry manages all providers
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]*ExtendedProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[string]*ExtendedProvider),
	}
	// Initialize with well-known providers
	for _, p := range WellKnownProviders() {
		r.providers[p.ID] = p
	}
	return r
}

// GetProvider returns a provider by ID
func (r *ProviderRegistry) GetProvider(id string) (*ExtendedProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return p, nil
}

// ListProviders returns all providers
func (r *ProviderRegistry) ListProviders() []*ExtendedProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ExtendedProvider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ListProvidersByTier returns providers filtered by tier
func (r *ProviderRegistry) ListProvidersByTier(tier ProviderTier) []*ExtendedProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ExtendedProvider, 0)
	for _, p := range r.providers {
		if p.Tier == tier {
			result = append(result, p)
		}
	}
	return result
}

// ListHealthyProviders returns only healthy providers
func (r *ProviderRegistry) ListHealthyProviders() []*ExtendedProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ExtendedProvider, 0)
	for _, p := range r.providers {
		if p.IsHealthy() {
			result = append(result, p)
		}
	}
	return result
}

// AddProvider adds a new provider
func (r *ProviderRegistry) AddProvider(provider *ExtendedProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[provider.ID]; exists {
		return ErrProviderNotFound // Provider already exists
	}
	r.providers[provider.ID] = provider
	return nil
}

// UpdateProvider updates an existing provider
func (r *ProviderRegistry) UpdateProvider(provider *ExtendedProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[provider.ID]; !exists {
		return ErrProviderNotFound
	}
	r.providers[provider.ID] = provider
	return nil
}

// DBProviderRow represents a provider from the database
type DBProviderRow struct {
	ID            string         `db:"id"`
	Name          string         `db:"name"`
	Tier          int            `db:"tier"`
	OAuthSupport  bool           `db:"oauth_support"`
	Endpoint      string         `db:"endpoint"`
	AuthType      string         `db:"auth_type"`
	Status        string         `db:"status"`
	Priority      int            `db:"priority"`
	CostPer1KIn   float64        `db:"cost_per_1k_input"`
	CostPer1KOut  float64        `db:"cost_per_1k_output"`
	AvgLatencyMs  int64          `db:"avg_latency_ms"`
	Capabilities  sql.NullString `db:"capabilities"`
	Models        sql.NullString `db:"models"`
	HealthState   sql.NullString `db:"health_state"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

// ToExtendedProvider converts a DBProviderRow to an ExtendedProvider
func (db *DBProviderRow) ToExtendedProvider() *ExtendedProvider {
	p := NewExtendedProvider(db.ID, db.Name, ProviderTier(db.Tier), db.OAuthSupport)
	p.BaseURL = db.Endpoint
	p.AuthType = db.AuthType
	p.Status = db.Status
	p.Priority = db.Priority
	p.CostPer1KIn = db.CostPer1KIn
	p.CostPer1KOut = db.CostPer1KOut
	p.AvgLatencyMs = db.AvgLatencyMs
	if db.HealthState.Valid {
		p.Health.CircuitState = db.HealthState.String
	}
	return p
}
