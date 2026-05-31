package billing

import (
	"time"
)

// ModelPricing represents pricing configuration for a model
type ModelPricing struct {
	ID                   int64     `json:"id"`
	ModelName            string    `json:"model_name"`
	InputPricePer1K      float64   `json:"input_price_per_1k"`       // Price per 1K input tokens in USD
	OutputPricePer1K     float64   `json:"output_price_per_1k"`      // Price per 1K output tokens in USD
	BatchInputPricePer1K float64   `json:"batch_input_price_per_1k"` // For batch models
	IsActive             bool      `json:"is_active"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// CalculateCost calculates the cost for a given token count
func (p *ModelPricing) CalculateCost(inputTokens, outputTokens int64) float64 {
	inputCost := float64(inputTokens) / 1000 * p.InputPricePer1K
	outputCost := float64(outputTokens) / 1000 * p.OutputPricePer1K
	return inputCost + outputCost
}

// CalculateCostWithBatch calculates cost considering batch input pricing
func (p *ModelPricing) CalculateCostWithBatch(inputTokens, outputTokens, batchInputTokens int64) float64 {
	standardInputCost := float64(inputTokens-batchInputTokens) / 1000 * p.InputPricePer1K
	batchInputCost := float64(batchInputTokens) / 1000 * p.BatchInputPricePer1K
	outputCost := float64(outputTokens) / 1000 * p.OutputPricePer1K
	return standardInputCost + batchInputCost + outputCost
}

// BillingRecord represents a billing transaction
type BillingRecord struct {
	ID             int64     `json:"id"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	UserID         *int64    `json:"user_id,omitempty"`
	APIKeyID       string    `json:"api_key_id,omitempty"`
	Model          string    `json:"model"`
	InputTokens    int64     `json:"input_tokens"`
	OutputTokens   int64     `json:"output_tokens"`
	InputCost      float64   `json:"input_cost"`
	OutputCost     float64   `json:"output_cost"`
	TotalCost      float64   `json:"total_cost"`
	Currency       string    `json:"currency"`
	ChannelID      *int64    `json:"channel_id,omitempty"`
	TokenGroupID   *int64    `json:"token_group_id,omitempty"`
	RequestID      string    `json:"request_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// BillingSummary provides aggregated billing data
type BillingSummary struct {
	OrganizationID    *int64             `json:"organization_id,omitempty"`
	UserID            *int64             `json:"user_id,omitempty"`
	Period            string             `json:"period"` // "daily", "monthly", "yearly"
	TotalRequests     int64              `json:"total_requests"`
	TotalInputTokens  int64              `json:"total_input_tokens"`
	TotalOutputTokens int64              `json:"total_output_tokens"`
	TotalCost         float64            `json:"total_cost"`
	CostByModel       map[string]float64 `json:"cost_by_model"`
	CostByDay         map[string]float64 `json:"cost_by_day"`
	Currency          string             `json:"currency"`
	StartDate         time.Time          `json:"start_date"`
	EndDate           time.Time          `json:"end_date"`
}

// UsageRecord represents usage for quota tracking
type UsageRecord struct {
	ID             int64     `json:"id"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	UserID         *int64    `json:"user_id,omitempty"`
	APIKeyID       string    `json:"api_key_id,omitempty"`
	Model          string    `json:"model"`
	InputTokens    int64     `json:"input_tokens"`
	OutputTokens   int64     `json:"output_tokens"`
	Cost           float64   `json:"cost"`
	Period         string    `json:"period"` // "hourly", "daily", "monthly"
	CreatedAt      time.Time `json:"created_at"`
}

// Balance represents an organization's or user's balance
type Balance struct {
	ID             int64     `json:"id"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	UserID         *int64    `json:"user_id,omitempty"`
	Balance        float64   `json:"balance"`
	Currency       string    `json:"currency"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Transaction represents a balance transaction
type Transaction struct {
	ID             int64     `json:"id"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	UserID         *int64    `json:"user_id,omitempty"`
	Amount         float64   `json:"amount"` // Positive for credit, negative for debit
	BalanceBefore  float64   `json:"balance_before"`
	BalanceAfter   float64   `json:"balance_after"`
	Type           string    `json:"type"` // "credit", "debit", "refund"
	Description    string    `json:"description"`
	ReferenceID    string    `json:"reference_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// PredefinedModelPrices contains default pricing for common models
var PredefinedModelPrices = map[string]struct {
	Input  float64
	Output float64
}{
	// OpenAI models
	"gpt-4o":               {0.005, 0.015},
	"gpt-4o-mini":          {0.00015, 0.0006},
	"gpt-4-turbo":          {0.01, 0.03},
	"gpt-4":                {0.03, 0.06},
	"gpt-3.5-turbo":        {0.0005, 0.0015},
	"gpt-4o-audio-preview": {0.005, 0.015},

	// Anthropic models
	"claude-3-5-sonnet-20241022": {0.003, 0.015},
	"claude-3-5-sonnet-latest":   {0.003, 0.015},
	"claude-3-opus-20240229":     {0.015, 0.075},
	"claude-3-sonnet-20240229":   {0.003, 0.015},
	"claude-3-haiku-20240307":    {0.00025, 0.00125},

	// Google models
	"gemini-1.5-pro":          {0.00125, 0.005},
	"gemini-1.5-flash":        {0.000075, 0.0003},
	"gemini-1.5-pro-latest":   {0.00125, 0.005},
	"gemini-1.5-flash-latest": {0.000075, 0.0003},

	// Mistral models
	"mistral-large-latest": {0.002, 0.006},
	"mistral-small-latest": {0.0002, 0.0006},
}

// GetPredefinedPrice returns the predefined price for a model
func GetPredefinedPrice(modelName string) (input, output float64, found bool) {
	if price, ok := PredefinedModelPrices[modelName]; ok {
		return price.Input, price.Output, true
	}
	return 0, 0, false
}
