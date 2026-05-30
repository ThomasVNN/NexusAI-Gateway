package quota

import (
	"context"
	"errors"
	"sync"
	"time"
)

// QuotaPlan represents a tier-based quota configuration
type QuotaPlan struct {
	Name         string            // e.g., "free", "standard", "enterprise"
	RequestsPerMin int             `json:"requests_per_minute"`
	RequestsPerHour int            `json:"requests_per_hour"`
	RequestsPerDay int             `json:"requests_per_day"`
	MaxTokensPerDay int            `json:"max_tokens_per_day"`
	Features      map[string]bool  // e.g., {"streaming": true, "analytics": false}
}

// DefaultQuotaPlans returns predefined quota plans
func DefaultQuotaPlans() map[string]*QuotaPlan {
	return map[string]*QuotaPlan{
		"free": {
			Name:            "free",
			RequestsPerMin:  10,
			RequestsPerHour: 100,
			RequestsPerDay:  1000,
			MaxTokensPerDay: 100000,
			Features: map[string]bool{
				"streaming":      false,
				"analytics":      false,
				"priority_queue": false,
				"custom_models":  false,
			},
		},
		"standard": {
			Name:            "standard",
			RequestsPerMin:  60,
			RequestsPerHour: 1000,
			RequestsPerDay:  10000,
			MaxTokensPerDay: 1000000,
			Features: map[string]bool{
				"streaming":      true,
				"analytics":      true,
				"priority_queue": false,
				"custom_models":  false,
			},
		},
		"enterprise": {
			Name:            "enterprise",
			RequestsPerMin:  300,
			RequestsPerHour: 10000,
			RequestsPerDay:  100000,
			MaxTokensPerDay: 10000000,
			Features: map[string]bool{
				"streaming":      true,
				"analytics":      true,
				"priority_queue": true,
				"custom_models":  true,
			},
		},
	}
}

// QuotaUsage tracks current quota consumption
type QuotaUsage struct {
	TenantID         string    `json:"tenant_id"`
	Plan             string    `json:"plan"`
	MinuteRequests   int       `json:"requests_this_minute"`
	HourRequests     int       `json:"requests_this_hour"`
	DayRequests      int       `json:"requests_today"`
	TokensToday      int       `json:"tokens_today"`
	LastMinuteReset  time.Time `json:"last_minute_reset"`
	LastHourReset    time.Time `json:"last_hour_reset"`
	LastDayReset     time.Time `json:"last_day_reset"`
}

// QuotaRepository defines the interface for quota persistence
type QuotaRepository interface {
	GetQuotaUsage(ctx context.Context, tenantID string) (*QuotaUsage, error)
	SaveQuotaUsage(ctx context.Context, usage *QuotaUsage) error
	ResetMinute(ctx context.Context, tenantID string) error
	ResetHour(ctx context.Context, tenantID string) error
	ResetDay(ctx context.Context, tenantID string) error
}

// InMemoryQuotaRepository implements QuotaRepository in-memory
type InMemoryQuotaRepository struct {
	mu     sync.RWMutex
	usage  map[string]*QuotaUsage
	plans  map[string]*QuotaPlan
}

// NewInMemoryQuotaRepository creates a new in-memory quota repository
func NewInMemoryQuotaRepository() *InMemoryQuotaRepository {
	return &InMemoryQuotaRepository{
		usage: make(map[string]*QuotaUsage),
		plans: DefaultQuotaPlans(),
	}
}

// GetQuotaUsage returns quota usage for a tenant
func (r *InMemoryQuotaRepository) GetQuotaUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if usage, ok := r.usage[tenantID]; ok {
		return usage, nil
	}

	return &QuotaUsage{
		TenantID:        tenantID,
		Plan:            "free",
		LastMinuteReset: time.Now(),
		LastHourReset:   time.Now(),
		LastDayReset:    time.Now(),
	}, nil
}

// SaveQuotaUsage saves quota usage for a tenant
func (r *InMemoryQuotaRepository) SaveQuotaUsage(ctx context.Context, usage *QuotaUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.usage[usage.TenantID] = usage
	return nil
}

// ResetMinute resets minute counters
func (r *InMemoryQuotaRepository) ResetMinute(ctx context.Context, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if usage, ok := r.usage[tenantID]; ok {
		usage.MinuteRequests = 0
		usage.LastMinuteReset = time.Now()
	}
	return nil
}

// ResetHour resets hour counters
func (r *InMemoryQuotaRepository) ResetHour(ctx context.Context, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if usage, ok := r.usage[tenantID]; ok {
		usage.HourRequests = 0
		usage.MinuteRequests = 0
		usage.LastHourReset = time.Now()
	}
	return nil
}

// ResetDay resets day counters
func (r *InMemoryQuotaRepository) ResetDay(ctx context.Context, tenantID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if usage, ok := r.usage[tenantID]; ok {
		usage.DayRequests = 0
		usage.TokensToday = 0
		usage.HourRequests = 0
		usage.MinuteRequests = 0
		usage.LastDayReset = time.Now()
	}
	return nil
}

// GetPlan returns the quota plan for a tenant
func (r *InMemoryQuotaRepository) GetPlan(ctx context.Context, planName string) (*QuotaPlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if plan, ok := r.plans[planName]; ok {
		return plan, nil
	}
	return nil, errors.New("plan not found")
}

// UpdatePlan updates or adds a quota plan
func (r *InMemoryQuotaRepository) UpdatePlan(ctx context.Context, plan *QuotaPlan) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plans[plan.Name] = plan
	return nil
}

// TenantRateLimiter manages rate limiting per tenant
type TenantRateLimiter struct {
	mu        sync.RWMutex
	requests  map[string][]time.Time // tenantID -> request timestamps
	plan      *QuotaPlan
	planName  string
}

// NewTenantRateLimiter creates a new tenant rate limiter
func NewTenantRateLimiter(planName string) *TenantRateLimiter {
	plans := DefaultQuotaPlans()
	plan, ok := plans[planName]
	if !ok {
		plan = plans["free"]
	}

	return &TenantRateLimiter{
		requests: make(map[string][]time.Time),
		plan:     plan,
		planName: planName,
	}
}

// Allow checks if a request is allowed under rate limits
func (r *TenantRateLimiter) Allow(tenantID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.cleanupExpired(tenantID, now)

	requests := r.requests[tenantID]
	if requests == nil {
		requests = []time.Time{}
	}

	// Check minute limit
	if len(requests) >= r.plan.RequestsPerMin {
		return false
	}

	// Add this request
	requests = append(requests, now)
	r.requests[tenantID] = requests
	return true
}

// GetRemaining returns remaining requests in current window
func (r *TenantRateLimiter) GetRemaining(tenantID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	requests := r.requests[tenantID]
	if requests == nil {
		return r.plan.RequestsPerMin
	}

	// Count non-expired requests
	valid := 0
	for _, t := range requests {
		if now.Sub(t) < time.Minute {
			valid++
		}
	}

	remaining := r.plan.RequestsPerMin - valid
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetResetTime returns when the rate limit will reset
func (r *TenantRateLimiter) GetResetTime(tenantID string) time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	requests := r.requests[tenantID]
	if requests == nil || len(requests) == 0 {
		return time.Now()
	}

	oldest := requests[0]
	return oldest.Add(time.Minute)
}

// cleanupExpired removes expired request timestamps
func (r *TenantRateLimiter) cleanupExpired(tenantID string, now time.Time) {
	requests := r.requests[tenantID]
	if requests == nil {
		return
	}

	// Keep only requests within the last minute
	valid := make([]time.Time, 0)
	for _, t := range requests {
		if now.Sub(t) < time.Minute {
			valid = append(valid, t)
		}
	}
	r.requests[tenantID] = valid
}

// Reset clears all rate limit data for a tenant
func (r *TenantRateLimiter) Reset(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.requests, tenantID)
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	PlanName    string
	LimitType   string // "minute", "hour", "day", "tokens"
	CurrentUsage int
	Limit       int
	ResetTime   time.Time
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded: " + e.LimitType + " limit reached (" + 
		string(rune(e.CurrentUsage)) + "/" + string(rune(e.Limit)) + ")"
}

// CheckQuota verifies if a tenant can make a request
func CheckQuota(usage *QuotaUsage, plan *QuotaPlan, tokens int) error {
	now := time.Now()

	// Check and reset minute counter
	if now.Sub(usage.LastMinuteReset) >= time.Minute {
		usage.MinuteRequests = 0
		usage.LastMinuteReset = now
	}

	// Check and reset hour counter
	if now.Sub(usage.LastHourReset) >= time.Hour {
		usage.HourRequests = 0
		usage.MinuteRequests = 0
		usage.LastHourReset = now
	}

	// Check and reset day counter
	if now.Sub(usage.LastDayReset) >= 24*time.Hour {
		usage.DayRequests = 0
		usage.TokensToday = 0
		usage.HourRequests = 0
		usage.MinuteRequests = 0
		usage.LastDayReset = now
	}

	// Check minute limit
	if usage.MinuteRequests >= plan.RequestsPerMin {
		return &QuotaExceededError{
			PlanName:    plan.Name,
			LimitType:   "minute",
			CurrentUsage: usage.MinuteRequests,
			Limit:       plan.RequestsPerMin,
			ResetTime:   usage.LastMinuteReset.Add(time.Minute),
		}
	}

	// Check hour limit
	if usage.HourRequests >= plan.RequestsPerHour {
		return &QuotaExceededError{
			PlanName:    plan.Name,
			LimitType:   "hour",
			CurrentUsage: usage.HourRequests,
			Limit:       plan.RequestsPerHour,
			ResetTime:   usage.LastHourReset.Add(time.Hour),
		}
	}

	// Check day limit
	if usage.DayRequests >= plan.RequestsPerDay {
		return &QuotaExceededError{
			PlanName:    plan.Name,
			LimitType:   "day",
			CurrentUsage: usage.DayRequests,
			Limit:       plan.RequestsPerDay,
			ResetTime:   usage.LastDayReset.Add(24 * time.Hour),
		}
	}

	// Check token limit
	if usage.TokensToday+tokens > plan.MaxTokensPerDay {
		return &QuotaExceededError{
			PlanName:    plan.Name,
			LimitType:   "tokens",
			CurrentUsage: usage.TokensToday,
			Limit:       plan.MaxTokensPerDay,
			ResetTime:   usage.LastDayReset.Add(24 * time.Hour),
		}
	}

	return nil
}
