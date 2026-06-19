package token

import (
	"encoding/json"
	"time"
)

// TokenGroup represents a grouping of tokens with quota limits
type TokenGroup struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	AllowedModels []string  `json:"allowed_models"` // Empty means all models
	DailyQuota    int64     `json:"daily_quota"`
	HourlyQuota   int64     `json:"hourly_quota"`
	MonthlyQuota  int64     `json:"monthly_quota"`
	UsedToday     int64     `json:"used_today"`
	UsedThisHour  int64     `json:"used_this_hour"`
	UsedThisMonth int64     `json:"used_this_month"`
	IsActive      bool      `json:"is_active"`
	Priority      int       `json:"priority"` // Higher priority groups get access first
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TokenUsage represents token usage within a quota period
type TokenUsage struct {
	ID           int64     `json:"id"`
	TokenGroupID int64     `json:"token_group_id"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	RequestCount int64     `json:"request_count"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenGroupStats provides aggregated statistics for a token group
type TokenGroupStats struct {
	GroupID           int64   `json:"group_id"`
	Name              string  `json:"name"`
	DailyQuota        int64   `json:"daily_quota"`
	DailyUsed         int64   `json:"daily_used"`
	DailyRemaining    int64   `json:"daily_remaining"`
	DailyUsagePercent float64 `json:"daily_usage_percent"`
	HourlyQuota       int64   `json:"hourly_quota"`
	HourlyUsed        int64   `json:"hourly_used"`
	HourlyRemaining   int64   `json:"hourly_remaining"`
	MonthlyQuota      int64   `json:"monthly_quota"`
	MonthlyUsed       int64   `json:"monthly_used"`
	MonthlyRemaining  int64   `json:"monthly_remaining"`
	TotalRequests     int64   `json:"total_requests"`
	AvgTokensPerReq   float64 `json:"avg_tokens_per_request"`
}

// IsModelAllowed checks if a model is allowed in this token group
func (tg *TokenGroup) IsModelAllowed(model string) bool {
	if len(tg.AllowedModels) == 0 {
		return true // Empty means all models allowed
	}
	for _, m := range tg.AllowedModels {
		if m == model || m == "*" {
			return true
		}
	}
	return false
}

// CanConsume checks if the token group has remaining quota
func (tg *TokenGroup) CanConsume(tokens int64) bool {
	if !tg.IsActive {
		return false
	}

	if tg.HourlyQuota > 0 && tg.UsedThisHour+tokens > tg.HourlyQuota {
		return false
	}

	if tg.DailyQuota > 0 && tg.UsedToday+tokens > tg.DailyQuota {
		return false
	}

	if tg.MonthlyQuota > 0 && tg.UsedThisMonth+tokens > tg.MonthlyQuota {
		return false
	}

	return true
}

// RemainingQuota returns the remaining quota for the most restrictive limit
func (tg *TokenGroup) RemainingQuota() int64 {
	var remaining int64 = -1 // -1 means unlimited

	if tg.HourlyQuota > 0 {
		hourlyRemaining := tg.HourlyQuota - tg.UsedThisHour
		if remaining == -1 || hourlyRemaining < remaining {
			remaining = hourlyRemaining
		}
	}

	if tg.DailyQuota > 0 {
		dailyRemaining := tg.DailyQuota - tg.UsedToday
		if remaining == -1 || dailyRemaining < remaining {
			remaining = dailyRemaining
		}
	}

	if tg.MonthlyQuota > 0 {
		monthlyRemaining := tg.MonthlyQuota - tg.UsedThisMonth
		if remaining == -1 || monthlyRemaining < remaining {
			remaining = monthlyRemaining
		}
	}

	return remaining
}

// ToJSON converts token group to JSON string
func (tg *TokenGroup) ToJSON() (string, error) {
	data, err := json.Marshal(tg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TokenGroupFromJSON parses token group from JSON string
func TokenGroupFromJSON(data string) (*TokenGroup, error) {
	var tg TokenGroup
	if err := json.Unmarshal([]byte(data), &tg); err != nil {
		return nil, err
	}
	return &tg, nil
}

// Validate checks if the token group configuration is valid
func (tg *TokenGroup) Validate() error {
	if tg.Name == "" {
		return ErrTokenGroupNameRequired
	}
	if tg.HourlyQuota < 0 || tg.DailyQuota < 0 || tg.MonthlyQuota < 0 {
		return ErrInvalidQuota
	}
	return nil
}

// Custom errors
type TokenError struct {
	Message string
}

func (e *TokenError) Error() string {
	return e.Message
}

var (
	ErrTokenGroupNameRequired = &TokenError{Message: "token group name is required"}
	ErrInvalidQuota           = &TokenError{Message: "quota values must be non-negative"}
	ErrTokenGroupNotFound     = &TokenError{Message: "token group not found"}
	ErrQuotaExceeded          = &TokenError{Message: "quota exceeded"}
)
