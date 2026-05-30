package quota

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

// QuotaPlan defines rate limit tiers for different tenant plans.
type QuotaPlan struct {
	Name              string
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	TokensPerDay      int64
	BurstLimit       int
}

// DefaultQuotaPlans provides standard quota configurations.
var DefaultQuotaPlans = map[string]QuotaPlan{
	"free": {
		RequestsPerMinute: 10,
		RequestsPerHour:   100,
		RequestsPerDay:    500,
		TokensPerDay:      100_000,
		BurstLimit:       5,
	},
	"standard": {
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		TokensPerDay:      1_000_000,
		BurstLimit:       20,
	},
	"enterprise": {
		RequestsPerMinute: 300,
		RequestsPerHour:   10000,
		RequestsPerDay:    100000,
		TokensPerDay:      10_000_000,
		BurstLimit:       50,
	},
}

// QuotaUsage tracks current usage against limits.
type QuotaUsage struct {
	TenantID          string
	Plan              string
	CurrentMinute     int
	CurrentHour       int
	CurrentDay        int
	CurrentTokens     int64
	RemainingMinute   int
	RemainingHour     int
	RemainingDay      int
	RemainingTokens   int64
	ResetAt           time.Time
	LimitExceeded     bool
	ExceededLimitType string
}

// QuotaRepository manages tenant quota tracking in the database.
type QuotaRepository struct {
	db *postgres.DB
}

// NewQuotaRepository creates a new QuotaRepository.
func NewQuotaRepository(db *postgres.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// GetQuotaUsage retrieves current quota usage for a tenant.
func (r *QuotaRepository) GetQuotaUsage(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	if r.db == nil {
		return nil, errors.New("database unavailable")
	}

	query := `
		SELECT 
			t.slug as plan,
			COALESCE(SUM(CASE WHEN created_at >= $2 THEN 1 ELSE 0 END), 0) as minute_count,
			COALESCE(SUM(CASE WHEN created_at >= $3 THEN 1 ELSE 0 END), 0) as hour_count,
			COALESCE(SUM(CASE WHEN created_at >= $4 THEN 1 ELSE 0 END), 0) as day_count,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as day_tokens
		FROM usage_records ur
		JOIN api_keys ak ON ur.key_id = ak.id
		JOIN tenants t ON ak.tenant_id = t.id
		WHERE t.id = $1
		GROUP BY t.slug`

	now := time.Now()
	minuteStart := now.Add(-1 * time.Minute)
	hourStart := now.Add(-1 * time.Hour)
	dayStart := now.Truncate(24 * time.Hour)

	var plan string
	var minuteCount, hourCount, dayCount int
	var dayTokens int64

	err := r.db.QueryRowContext(ctx, query, tenantID, minuteStart, hourStart, dayStart).Scan(
		&plan, &minuteCount, &hourCount, &dayCount, &dayTokens,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get quota usage: %w", err)
	}

	// Get plan limits
	planLimits := DefaultQuotaPlans["standard"]
	if p, ok := DefaultQuotaPlans[plan]; ok {
		planLimits = p
	}

	usage := &QuotaUsage{
		TenantID:        tenantID,
		Plan:            plan,
		CurrentMinute:   minuteCount,
		CurrentHour:     hourCount,
		CurrentDay:      dayCount,
		CurrentTokens:   dayTokens,
		RemainingMinute: planLimits.RequestsPerMinute - minuteCount,
		RemainingHour:   planLimits.RequestsPerHour - hourCount,
		RemainingDay:    planLimits.RequestsPerDay - dayCount,
		RemainingTokens: planLimits.TokensPerDay - dayTokens,
		ResetAt:        now.Add(24 * time.Hour).Truncate(24 * time.Hour),
	}

	// Check limits
	if minuteCount >= planLimits.RequestsPerMinute {
		usage.LimitExceeded = true
		usage.ExceededLimitType = "minute"
	} else if hourCount >= planLimits.RequestsPerHour {
		usage.LimitExceeded = true
		usage.ExceededLimitType = "hour"
	} else if dayCount >= planLimits.RequestsPerDay {
		usage.LimitExceeded = true
		usage.ExceededLimitType = "day"
	} else if dayTokens >= planLimits.TokensPerDay {
		usage.LimitExceeded = true
		usage.ExceededLimitType = "tokens"
	}

	return usage, nil
}

// TenantRateLimiter implements per-tenant rate limiting with sliding window.
type TenantRateLimiter struct {
	mu         sync.RWMutex
	window     time.Duration
	limits     map[string]*tenantLimits
	planLimits map[string]QuotaPlan
}

// tenantLimits tracks rate limit state per tenant.
type tenantLimits struct {
	requests []time.Time
	tokens   int64
}

var defaultTenantRateLimiter = &TenantRateLimiter{
	window:     1 * time.Minute,
	limits:     make(map[string]*tenantLimits),
	planLimits: DefaultQuotaPlans,
}

// NewTenantRateLimiter creates a new tenant rate limiter.
func NewTenantRateLimiter() *TenantRateLimiter {
	return defaultTenantRateLimiter
}

// Allow checks if a request is allowed under the rate limit.
func (l *TenantRateLimiter) Allow(tenantID, plan string, tokens int64) (bool, *QuotaUsage) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.window)

	// Get or create tenant limits
	limits, exists := l.limits[tenantID]
	if !exists {
		limits = &tenantLimits{requests: make([]time.Time, 0)}
		l.limits[tenantID] = limits
	}

	// Clean old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, t := range limits.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	limits.requests = validRequests

	// Get plan limits
	planLimits := l.planLimits["standard"]
	if p, ok := l.planLimits[plan]; ok {
		planLimits = p
	}

	// Check rate limit
	currentCount := len(validRequests)
	remaining := planLimits.RequestsPerMinute - currentCount

	allowed := currentCount < planLimits.RequestsPerMinute
	if allowed {
		validRequests = append(validRequests, now)
		limits.requests = validRequests
		limits.tokens += tokens
	}

	usage := &QuotaUsage{
		TenantID:        tenantID,
		Plan:            plan,
		CurrentMinute:   currentCount,
		RemainingMinute: remaining,
		CurrentTokens:   limits.tokens,
		RemainingTokens: planLimits.TokensPerDay - limits.tokens,
		LimitExceeded:   !allowed,
	}

	if !allowed {
		usage.ExceededLimitType = "minute"
	}

	return allowed, usage
}

// Cleanup removes stale entries to prevent memory leaks.
func (l *TenantRateLimiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for tenantID, limits := range l.limits {
		hasRecent := false
		for _, t := range limits.requests {
			if t.After(cutoff) {
				hasRecent = true
				break
			}
		}
		if !hasRecent {
			delete(l.limits, tenantID)
		}
	}
}
