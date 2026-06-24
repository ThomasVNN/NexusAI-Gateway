package billing

import (
	"context"
	"sync"
	"time"
)

// BudgetPeriod defines the budget tracking period
type BudgetPeriod string

const (
	BudgetPeriodDaily   BudgetPeriod = "daily"
	BudgetPeriodWeekly  BudgetPeriod = "weekly"
	BudgetPeriodMonthly BudgetPeriod = "monthly"
)

// BudgetLimit defines spending limits
type BudgetLimit struct {
	Period       BudgetPeriod `json:"period"`
	AmountUSD    float64      `json:"amount_usd"`
	TokenLimit   int64        `json:"token_limit"`
	AlertThreshold float64    `json:"alert_threshold"` // 0.0-1.0, send alert when reached
}

// OrganizationBudget tracks budget for an organization
type OrganizationBudget struct {
	mu           sync.RWMutex
	OrgID        string           `json:"org_id"`
	Limits       map[BudgetPeriod]*BudgetLimit `json:"limits"`
	Usage        map[BudgetPeriod]*BudgetUsage `json:"usage"`
	AlertsSent   map[BudgetPeriod]bool        `json:"alerts_sent"`
	LastReset    map[BudgetPeriod]time.Time    `json:"last_reset"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// BudgetUsage tracks current usage within a period
type BudgetUsage struct {
	mu              sync.RWMutex
	Period         BudgetPeriod    `json:"period"`
	SpentUSD       float64         `json:"spent_usd"`
	InputTokens    int64           `json:"input_tokens"`
	OutputTokens   int64           `json:"output_tokens"`
	TotalTokens    int64           `json:"total_tokens"`
	RequestCount   int64           `json:"request_count"`
	ModelUsage     map[string]*ModelUsage `json:"model_usage"` // model_name -> usage
	ProviderUsage  map[string]*ProviderUsage `json:"provider_usage"` // provider -> usage
	WindowStart    time.Time       `json:"window_start"`
	WindowEnd      time.Time       `json:"window_end"`
}

// ModelUsage tracks usage per model
type ModelUsage struct {
	ModelName     string  `json:"model_name"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	TotalTokens   int64  `json:"total_tokens"`
	RequestCount  int64  `json:"request_count"`
	CostUSD       float64 `json:"cost_usd"`
}

// ProviderUsage tracks usage per provider
type ProviderUsage struct {
	ProviderID    string  `json:"provider_id"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	TotalTokens   int64  `json:"total_tokens"`
	RequestCount  int64  `json:"request_count"`
	CostUSD       float64 `json:"cost_usd"`
}

// NewOrganizationBudget creates a new budget tracker
func NewOrganizationBudget(orgID string) *OrganizationBudget {
	now := time.Now()
	return &OrganizationBudget{
		OrgID:      orgID,
		Limits:     make(map[BudgetPeriod]*BudgetLimit),
		Usage:      make(map[BudgetPeriod]*BudgetUsage),
		AlertsSent: make(map[BudgetPeriod]bool),
		LastReset:  make(map[BudgetPeriod]time.Time),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// SetLimit sets a budget limit for a period
func (b *OrganizationBudget) SetLimit(period BudgetPeriod, limit *BudgetLimit) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Limits[period] = limit
	b.UpdatedAt = time.Now()

	// Initialize usage tracking
	if _, exists := b.Usage[period]; !exists {
		b.Usage[period] = &BudgetUsage{
			Period:      period,
			ModelUsage:  make(map[string]*ModelUsage),
			ProviderUsage: make(map[string]*ProviderUsage),
			WindowStart: b.getWindowStart(period),
			WindowEnd:   b.getWindowEnd(period),
		}
	}
}

// RecordUsage records token usage for cost attribution
func (b *OrganizationBudget) RecordUsage(ctx context.Context, record *UsageRecord) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	// Record in all applicable periods
	for _, period := range []BudgetPeriod{BudgetPeriodDaily, BudgetPeriodWeekly, BudgetPeriodMonthly} {
		usage, exists := b.Usage[period]
		if !exists {
			usage = &BudgetUsage{
				Period:        period,
				ModelUsage:    make(map[string]*ModelUsage),
				ProviderUsage: make(map[string]*ProviderUsage),
				WindowStart:   b.getWindowStart(period),
				WindowEnd:     b.getWindowEnd(period),
			}
			b.Usage[period] = usage
		}

		// Check if we need to reset
		if now.After(usage.WindowEnd) {
			b.resetUsage(period)
			usage = b.Usage[period]
		}

		usage.mu.Lock()

		// Update totals
		usage.SpentUSD += record.Cost
		usage.InputTokens += record.InputTokens
		usage.OutputTokens += record.OutputTokens
		usage.TotalTokens += record.InputTokens + record.OutputTokens
		usage.RequestCount++

		// Update model usage
		modelName := record.Model
		if modelUsage, exists := usage.ModelUsage[modelName]; exists {
			modelUsage.InputTokens += record.InputTokens
			modelUsage.OutputTokens += record.OutputTokens
			modelUsage.TotalTokens += record.InputTokens + record.OutputTokens
			modelUsage.RequestCount++
			modelUsage.CostUSD += record.Cost
		} else {
			usage.ModelUsage[modelName] = &ModelUsage{
				ModelName:    modelName,
				InputTokens:  record.InputTokens,
				OutputTokens: record.OutputTokens,
				TotalTokens:  record.InputTokens + record.OutputTokens,
				RequestCount: 1,
				CostUSD:      record.Cost,
			}
		}

		// Update provider usage (use APIKeyID as provider identifier)
		providerID := record.APIKeyID
		if providerUsage, exists := usage.ProviderUsage[providerID]; exists {
			providerUsage.InputTokens += record.InputTokens
			providerUsage.OutputTokens += record.OutputTokens
			providerUsage.TotalTokens += record.InputTokens + record.OutputTokens
			providerUsage.RequestCount++
			providerUsage.CostUSD += record.Cost
		} else {
			usage.ProviderUsage[providerID] = &ProviderUsage{
				ProviderID:   providerID,
				InputTokens:  record.InputTokens,
				OutputTokens: record.OutputTokens,
				TotalTokens:  record.InputTokens + record.OutputTokens,
				RequestCount: 1,
				CostUSD:      record.Cost,
			}
		}

		usage.mu.Unlock()

		// Check budget alerts
		b.checkAlerts(period)
	}

	b.UpdatedAt = now
	return nil
}


// CheckBudget checks if a request would exceed budget limits
func (b *OrganizationBudget) CheckBudget(ctx context.Context, providerID, modelName string, estimatedTokens int64) *BudgetCheckResult {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := &BudgetCheckResult{
		Allowed:          true,
		EstimatedCostUSD: 0,
		RemainingUSD:    0,
		RemainingTokens: 0,
	}

	// Calculate estimated cost (rough estimate)
	result.EstimatedCostUSD = float64(estimatedTokens) * 0.00001 // $0.01 per 1K tokens

	// Check all periods
	for period, limit := range b.Limits {
		if usage, exists := b.Usage[period]; exists {
			// Check USD limit
			if limit.AmountUSD > 0 {
				remaining := limit.AmountUSD - usage.SpentUSD
				if remaining <= 0 {
					result.Allowed = false
					result.Reason = "budget_exceeded"
					result.LimitType = "usd"
					result.LimitPeriod = period
					return result
				}
				if remaining < result.RemainingUSD || result.RemainingUSD == 0 {
					result.RemainingUSD = remaining
				}
			}

			// Check token limit
			if limit.TokenLimit > 0 {
				remaining := limit.TokenLimit - usage.TotalTokens
				if remaining <= 0 {
					result.Allowed = false
					result.Reason = "token_limit_exceeded"
					result.LimitType = "tokens"
					result.LimitPeriod = period
					return result
				}
				if remaining < result.RemainingTokens || result.RemainingTokens == 0 {
					result.RemainingTokens = remaining
				}
			}
		}
	}

	return result
}

// BudgetCheckResult is the result of a budget check
type BudgetCheckResult struct {
	Allowed           bool         `json:"allowed"`
	Reason            string       `json:"reason,omitempty"`
	LimitType         string       `json:"limit_type,omitempty"`
	LimitPeriod       BudgetPeriod `json:"limit_period,omitempty"`
	EstimatedCostUSD  float64      `json:"estimated_cost_usd"`
	RemainingUSD      float64      `json:"remaining_usd"`
	RemainingTokens   int64        `json:"remaining_tokens"`
}

// GetUsage returns current usage for a period
func (b *OrganizationBudget) GetUsage(period BudgetPeriod) *BudgetUsage {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Usage[period]
}

// GetAllUsage returns all usage data
func (b *OrganizationBudget) GetAllUsage() map[BudgetPeriod]*BudgetUsage {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make(map[BudgetPeriod]*BudgetUsage)
	for k, v := range b.Usage {
		result[k] = v
	}
	return result
}

// GetLimits returns all limits
func (b *OrganizationBudget) GetLimits() map[BudgetPeriod]*BudgetLimit {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make(map[BudgetPeriod]*BudgetLimit)
	for k, v := range b.Limits {
		result[k] = v
	}
	return result
}

// GetUsagePercentages returns usage as percentage of limits
func (b *OrganizationBudget) GetUsagePercentages() map[BudgetPeriod]*UsagePercentage {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[BudgetPeriod]*UsagePercentage)
	now := time.Now()

	for period := range b.Limits {
		pct := &UsagePercentage{Period: period}

		if usage, exists := b.Usage[period]; exists {
			if limit, exists := b.Limits[period]; exists {
				if limit.AmountUSD > 0 {
					pct.USD = (usage.SpentUSD / limit.AmountUSD) * 100
				}
				if limit.TokenLimit > 0 {
					pct.Tokens = (float64(usage.TotalTokens) / float64(limit.TokenLimit)) * 100
				}
			}
			pct.RequestCount = usage.RequestCount
			pct.WindowStart = usage.WindowStart
			pct.WindowEnd = usage.WindowEnd
			pct.WindowRemaining = usage.WindowEnd.Sub(now)
		}
	}

	return result
}

// UsagePercentage holds usage percentages
type UsagePercentage struct {
	Period          BudgetPeriod `json:"period"`
	USD             float64      `json:"usd_pct"`
	Tokens          float64      `json:"tokens_pct"`
	RequestCount    int64        `json:"request_count"`
	WindowStart     time.Time    `json:"window_start"`
	WindowEnd       time.Time    `json:"window_end"`
	WindowRemaining time.Duration `json:"window_remaining"`
}

// ResetUsage resets usage for a period
func (b *OrganizationBudget) ResetUsage(period BudgetPeriod) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resetUsage(period)
}

// resetUsage internal reset without locking
func (b *OrganizationBudget) resetUsage(period BudgetPeriod) {
	now := time.Now()
	b.Usage[period] = &BudgetUsage{
		Period:        period,
		ModelUsage:    make(map[string]*ModelUsage),
		ProviderUsage: make(map[string]*ProviderUsage),
		WindowStart:   b.getWindowStart(period),
		WindowEnd:     b.getWindowEnd(period),
	}
	b.AlertsSent[period] = false
	b.LastReset[period] = now
}

// getWindowStart calculates the start of the current window
func (b *OrganizationBudget) getWindowStart(period BudgetPeriod) time.Time {
	now := time.Now()
	switch period {
	case BudgetPeriodDaily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case BudgetPeriodWeekly:
		// Start of week (Sunday)
		weekday := int(now.Weekday())
		startOfWeek := now.AddDate(0, 0, -weekday)
		return time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, now.Location())
	case BudgetPeriodMonthly:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	default:
		return now
	}
}

// getWindowEnd calculates the end of the current window
func (b *OrganizationBudget) getWindowEnd(period BudgetPeriod) time.Time {
	start := b.getWindowStart(period)
	switch period {
	case BudgetPeriodDaily:
		return start.Add(24 * time.Hour)
	case BudgetPeriodWeekly:
		return start.Add(7 * 24 * time.Hour)
	case BudgetPeriodMonthly:
		return start.AddDate(0, 1, 0)
	default:
		return start.Add(24 * time.Hour)
	}
}

// checkAlerts checks if budget alerts should be sent
func (b *OrganizationBudget) checkAlerts(period BudgetPeriod) {
	limit, hasLimit := b.Limits[period]
	usage, hasUsage := b.Usage[period]

	if !hasLimit || !hasUsage {
		return
	}

	// Check if alert already sent
	if b.AlertsSent[period] {
		return
	}

	var percentage float64
	if limit.AmountUSD > 0 {
		percentage = (usage.SpentUSD / limit.AmountUSD) * 100
	} else if limit.TokenLimit > 0 {
		percentage = (float64(usage.TotalTokens) / float64(limit.TokenLimit)) * 100
	}

	// Send alert if threshold reached
	if percentage >= limit.AlertThreshold*100 {
		b.AlertsSent[period] = true
		// In production, this would trigger a notification
		// EventBus.Publish("budget_alert", BudgetAlertEvent{...})
	}
}

// GetAlertStatus returns whether alerts have been sent
func (b *OrganizationBudget) GetAlertStatus() map[BudgetPeriod]bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make(map[BudgetPeriod]bool)
	for k, v := range b.AlertsSent {
		result[k] = v
	}
	return result
}

// NewUsageRecord creates a new usage record
func NewUsageRecord(orgID, providerID, modelName string, inputTokens, outputTokens int64, costUSD float64) *UsageRecord {
	return &UsageRecord{
		ID:             0, // Will be auto-generated by DB
		OrganizationID: nil,
		UserID:         nil,
		APIKeyID:       "",
		Model:          modelName,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		Cost:           costUSD,
		Period:         "hourly",
		CreatedAt:      time.Now(),
	}
}

// Ensure types are used
var _ = context.Background
