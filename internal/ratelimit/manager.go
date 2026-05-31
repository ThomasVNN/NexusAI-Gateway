package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Redis key patterns for rate limiting:
// rate:<scope>:<tenant_id>:<identifier>:<window>
// Examples:
//   - rate:tenant:tenant123:min -> per-tenant per-minute
//   - rate:user:user456:min -> per-user per-minute
//   - rate:skill:skill789:min -> per-skill per-minute
//   - rate:apikey:key123:min -> per-api-key per-minute

const (
	keyPrefixRate      = "rate"
	keyPrefixQuota     = "quota"
	keyPrefixBurst     = "burst"
	keyPrefixConcurrent = "concurrent"
)

// RateKeyBuilder builds Redis keys for rate limiting
type RateKeyBuilder struct{}

// NewRateKeyBuilder creates a new RateKeyBuilder
func NewRateKeyBuilder() *RateKeyBuilder {
	return &RateKeyBuilder{}
}

// BuildKey creates a rate limit key
func (b *RateKeyBuilder) BuildKey(scope RateLimitScope, tenantID, identifier, window string) string {
	parts := []string{keyPrefixRate, string(scope), tenantID, identifier, window}
	return strings.Join(parts, ":")
}

// BuildTenantKey creates a tenant-scoped key
func (b *RateKeyBuilder) BuildTenantKey(tenantID, window string) string {
	return b.BuildKey(ScopeTenant, tenantID, "global", window)
}

// BuildUserKey creates a user-scoped key
func (b *RateKeyBuilder) BuildUserKey(tenantID, userID, window string) string {
	return b.BuildKey(ScopeUser, tenantID, userID, window)
}

// BuildSkillKey creates a skill-scoped key
func (b *RateKeyBuilder) BuildSkillKey(tenantID, skillID, window string) string {
	return b.BuildKey(ScopeSkill, tenantID, skillID, window)
}

// BuildAPIKeyKey creates an API key-scoped key
func (b *RateKeyBuilder) BuildAPIKeyKey(apiKeyHash, window string) string {
	return b.BuildKey(ScopeAPIKey, "global", apiKeyHash, window)
}

// BuildGlobalKey creates a global key
func (b *RateKeyBuilder) BuildGlobalKey(window string) string {
	return b.BuildKey(ScopeGlobal, "global", "global", window)
}

// WindowConstants defines time window constants
type WindowConstants struct {
	Min  string
	Hour string
	Day  string
}

// Windows contains window string constants
var Windows = WindowConstants{
	Min:  "min",
	Hour: "hour",
	Day:  "day",
}

// QuotaStorage defines the interface for quota storage backends
type QuotaStorage interface {
	// IncrementCount increments the count for a key and returns the new count
	IncrementCount(ctx context.Context, key string, windowDuration time.Duration) (int64, error)

	// GetCount gets the current count for a key
	GetCount(ctx context.Context, key string) (int64, error)

	// GetCountWithWindow gets the count within a specific time window
	GetCountWithWindow(ctx context.Context, key string, windowStart, windowEnd time.Time) (int64, error)

	// SetCount sets the count for a key (used for quota management)
	SetCount(ctx context.Context, key string, count int64, expiration time.Duration) error

	// GetRemaining gets the remaining quota for a key
	GetRemaining(ctx context.Context, key string, limit int64) (int64, error)

	// AcquireBurst attempts to acquire a burst slot
	AcquireBurst(ctx context.Context, key string, maxBurst int, ttl time.Duration) (bool, error)

	// ReleaseBurst releases a burst slot
	ReleaseBurst(ctx context.Context, key string) error

	// IncrementConcurrent increments concurrent request count
	IncrementConcurrent(ctx context.Context, key string) (int64, error)

	// DecrementConcurrent decrements concurrent request count
	DecrementConcurrent(ctx context.Context, key string) error

	// GetConcurrentCount gets the current concurrent request count
	GetConcurrentCount(ctx context.Context, key string) (int64, error)

	// Ping checks the connection health
	Ping(ctx context.Context) error

	// Close closes the storage connection
	Close() error
}

// SlidingWindowRateLimiter implements a sliding window rate limiter
type SlidingWindowRateLimiter struct {
	storage QuotaStorage
	keys    *RateKeyBuilder
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(storage QuotaStorage) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		storage: storage,
		keys:    NewRateKeyBuilder(),
	}
}

// Check checks if a request is allowed within the rate limit
func (r *SlidingWindowRateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	windowEnd := now

	// Get current count in the window
	count, err := r.storage.GetCountWithWindow(ctx, key, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	remaining := int64(limit) - count - 1
	if remaining < 0 {
		remaining = 0
	}

	if count >= int64(limit) {
		// Calculate when the oldest request will expire
		oldestKey := key + ":oldest"
		oldestStr, _ := r.storage.GetCountWithWindow(ctx, oldestKey, windowStart, windowEnd)
		var retryAfter int
		if oldestStr > 0 {
			oldest := windowStart.Add(time.Duration(oldestStr) * time.Nanosecond)
			retryAfter = int(time.Until(oldest.Add(window)).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
		} else {
			retryAfter = int(window.Seconds())
		}

		return &RateLimitResult{
			Allowed:    false,
			Limit:      limit,
			Remaining:  0,
			ResetAt:    now.Add(time.Duration(retryAfter) * time.Second).Unix(),
			RetryAfter: retryAfter,
			Used:       int(count),
		}, nil
	}

	// Increment the counter
	newCount, err := r.storage.IncrementCount(ctx, key, window)
	if err != nil {
		return nil, fmt.Errorf("failed to increment count: %w", err)
	}

	resetAt := now.Add(window).Unix()

	return &RateLimitResult{
		Allowed:        true,
		Limit:          limit,
		Remaining:      int(int64(limit) - newCount),
		ResetAt:        resetAt,
		ResetInSeconds: int(window.Seconds()),
		Used:           int(newCount),
	}, nil
}

// GetStatus returns the current rate limit status without consuming a token
func (r *SlidingWindowRateLimiter) GetStatus(ctx context.Context, key string, limit int, window time.Duration) (*RateLimitResult, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	windowEnd := now

	count, err := r.storage.GetCountWithWindow(ctx, key, windowStart, windowEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	remaining := int64(limit) - count
	if remaining < 0 {
		remaining = 0
	}

	resetAt := now.Add(window).Unix()

	return &RateLimitResult{
		Allowed:        true,
		Limit:          limit,
		Remaining:      int(remaining),
		ResetAt:        resetAt,
		ResetInSeconds: int(window.Seconds()),
		Used:           int(count),
	}, nil
}

// TokenBucketRateLimiter implements a token bucket rate limiter
type TokenBucketRateLimiter struct {
	storage     QuotaStorage
	keys        *RateKeyBuilder
	refillRate  float64 // tokens per second
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(storage QuotaStorage, refillRate float64) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		storage:    storage,
		keys:       NewRateKeyBuilder(),
		refillRate: refillRate,
	}
}

// Acquire attempts to acquire tokens from the bucket
func (t *TokenBucketRateLimiter) Acquire(ctx context.Context, key string, bucketSize int, tokensRequested int) (*RateLimitResult, error) {
	now := time.Now()

	// Get current bucket state
	bucketKey := key + ":tokens"
	countKey := key + ":last_refill"

	lastRefillStr, _ := t.storage.GetCount(ctx, countKey)
	lastRefill := time.Unix(0, lastRefillStr)

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(lastRefill)
	tokensToAdd := (elapsed.Seconds() * t.refillRate)

	// Get current tokens
	currentTokensStr, _ := t.storage.GetCount(ctx, bucketKey)
	currentTokens := float64(currentTokensStr) + tokensToAdd

	// Cap at bucket size
	if currentTokens > float64(bucketSize) {
		currentTokens = float64(bucketSize)
	}

	// Try to acquire tokens
	if currentTokens < float64(tokensRequested) {
		retryAfter := int((float64(tokensRequested) - currentTokens) / t.refillRate)
		if retryAfter < 1 {
			retryAfter = 1
		}

		return &RateLimitResult{
			Allowed:    false,
			Limit:      bucketSize,
			Remaining:  int(currentTokens),
			ResetAt:    now.Add(time.Duration(retryAfter) * time.Second).Unix(),
			RetryAfter: retryAfter,
			Used:       bucketSize - int(currentTokens),
		}, nil
	}

	// Consume tokens
	newTokens := currentTokens - float64(tokensRequested)

	// Update bucket
	t.storage.SetCount(ctx, bucketKey, int64(newTokens), 0)
	t.storage.SetCount(ctx, countKey, now.UnixNano(), 0)

	return &RateLimitResult{
		Allowed:    true,
		Limit:      bucketSize,
		Remaining:  int(newTokens),
		ResetAt:    now.Add(time.Duration(float64(bucketSize)/t.refillRate) * time.Second).Unix(),
		Used:       tokensRequested,
	}, nil
}

// QuotaManager manages rate limits across all scopes
type QuotaManager struct {
	storage   QuotaStorage
	keys      *RateKeyBuilder
	limiter   *SlidingWindowRateLimiter
	config    *RateLimitConfig
	tiers     map[RateLimitTier]TierConfig
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(storage QuotaStorage, config *RateLimitConfig) *QuotaManager {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	tiers := make(map[RateLimitTier]TierConfig)
	for tier, cfg := range DefaultTierConfigs {
		tiers[tier] = cfg
	}

	return &QuotaManager{
		storage: storage,
		keys:    NewRateKeyBuilder(),
		limiter: NewSlidingWindowRateLimiter(storage),
		config:  config,
		tiers:   tiers,
	}
}

// CheckTenantLimit checks the rate limit for a tenant
func (q *QuotaManager) CheckTenantLimit(ctx context.Context, tenantID string, tier RateLimitTier) (*RateLimitResult, error) {
	tierConfig := q.tiers[tier]
	if !tierConfig.Enabled {
		return &RateLimitResult{Allowed: true, Limit: 0, Remaining: 0}, nil
	}

	key := q.keys.BuildTenantKey(tenantID, Windows.Min)
	return q.limiter.Check(ctx, key, tierConfig.RequestsPerMin, time.Minute)
}

// CheckUserLimit checks the rate limit for a user within a tenant
func (q *QuotaManager) CheckUserLimit(ctx context.Context, tenantID, userID string, tier RateLimitTier) (*RateLimitResult, error) {
	tierConfig := q.tiers[tier]
	if !tierConfig.Enabled {
		return &RateLimitResult{Allowed: true, Limit: 0, Remaining: 0}, nil
	}

	key := q.keys.BuildUserKey(tenantID, userID, Windows.Min)
	return q.limiter.Check(ctx, key, tierConfig.RequestsPerMin, time.Minute)
}

// CheckSkillLimit checks the rate limit for a skill within a tenant
func (q *QuotaManager) CheckSkillLimit(ctx context.Context, tenantID, skillID string, tier RateLimitTier) (*RateLimitResult, error) {
	tierConfig := q.tiers[tier]
	if !tierConfig.Enabled {
		return &RateLimitResult{Allowed: true, Limit: 0, Remaining: 0}, nil
	}

	key := q.keys.BuildSkillKey(tenantID, skillID, Windows.Min)
	return q.limiter.Check(ctx, key, tierConfig.RequestsPerMin, time.Minute)
}

// CheckAPIKeyLimit checks the rate limit for an API key
func (q *QuotaManager) CheckAPIKeyLimit(ctx context.Context, apiKeyHash string, tier RateLimitTier) (*RateLimitResult, error) {
	tierConfig := q.tiers[tier]
	if !tierConfig.Enabled {
		return &RateLimitResult{Allowed: true, Limit: 0, Remaining: 0}, nil
	}

	key := q.keys.BuildAPIKeyKey(apiKeyHash, Windows.Min)
	return q.limiter.Check(ctx, key, tierConfig.RequestsPerMin, time.Minute)
}

// CheckAllLimits checks all applicable rate limits and returns the most restrictive result
func (q *QuotaManager) CheckAllLimits(ctx context.Context, tenantID, userID, skillID, apiKeyHash string, tier RateLimitTier) (*RateLimitResult, error) {
	if q.config.BypassLimits {
		return &RateLimitResult{
			Allowed:   true,
			Limit:     0,
			Remaining: 0,
			Tier:      tier,
		}, nil
	}

	// Check tenant limit
	tenantResult, err := q.CheckTenantLimit(ctx, tenantID, tier)
	if err != nil {
		return nil, fmt.Errorf("tenant limit check failed: %w", err)
	}
	if !tenantResult.Allowed {
		tenantResult.Scope = ScopeTenant
		tenantResult.Tier = tier
		return tenantResult, nil
	}

	// Check user limit
	if userID != "" {
		userResult, err := q.CheckUserLimit(ctx, tenantID, userID, tier)
		if err != nil {
			return nil, fmt.Errorf("user limit check failed: %w", err)
		}
		if !userResult.Allowed {
			userResult.Scope = ScopeUser
			userResult.Tier = tier
			return userResult, nil
		}
		// Use the most restrictive remaining count
		if userResult.Remaining < tenantResult.Remaining {
			tenantResult.Remaining = userResult.Remaining
		}
	}

	// Check skill limit
	if skillID != "" {
		skillResult, err := q.CheckSkillLimit(ctx, tenantID, skillID, tier)
		if err != nil {
			return nil, fmt.Errorf("skill limit check failed: %w", err)
		}
		if !skillResult.Allowed {
			skillResult.Scope = ScopeSkill
			skillResult.Tier = tier
			return skillResult, nil
		}
		// Use the most restrictive remaining count
		if skillResult.Remaining < tenantResult.Remaining {
			tenantResult.Remaining = skillResult.Remaining
		}
	}

	// Check API key limit
	if apiKeyHash != "" {
		apiKeyResult, err := q.CheckAPIKeyLimit(ctx, apiKeyHash, tier)
		if err != nil {
			return nil, fmt.Errorf("api key limit check failed: %w", err)
		}
		if !apiKeyResult.Allowed {
			apiKeyResult.Scope = ScopeAPIKey
			apiKeyResult.Tier = tier
			return apiKeyResult, nil
		}
		// Use the most restrictive remaining count
		if apiKeyResult.Remaining < tenantResult.Remaining {
			tenantResult.Remaining = apiKeyResult.Remaining
		}
	}

	tenantResult.Scope = ScopeGlobal
	tenantResult.Tier = tier
	return tenantResult, nil
}

// GetTierStatus returns the current status for all limits of a tier
func (q *QuotaManager) GetTierStatus(ctx context.Context, tenantID, userID string, tier RateLimitTier) (*RateLimitInfo, error) {
	tierConfig := q.tiers[tier]

	// Get tenant status
	tenantKey := q.keys.BuildTenantKey(tenantID, Windows.Min)
	tenantStatus, err := q.limiter.GetStatus(ctx, tenantKey, tierConfig.RequestsPerMin, time.Minute)
	if err != nil {
		return nil, err
	}

	// Get user status if applicable
	var userStatus *RateLimitResult
	if userID != "" {
		userKey := q.keys.BuildUserKey(tenantID, userID, Windows.Min)
		userStatus, err = q.limiter.GetStatus(ctx, userKey, tierConfig.RequestsPerMin, time.Minute)
		if err != nil {
			return nil, err
		}
	}

	// Use the most restrictive values
	remaining := tenantStatus.Remaining
	if userStatus != nil && userStatus.Remaining < remaining {
		remaining = userStatus.Remaining
	}

	return &RateLimitInfo{
		Limit:              tenantStatus.Limit,
		Remaining:          remaining,
		Reset:              tenantStatus.ResetAt,
		ResetInSeconds:     tenantStatus.ResetInSeconds,
		Tier:               string(tier),
		Scope:              string(ScopeTenant),
		RequestsPerMin:     tierConfig.RequestsPerMin,
		RequestsPerHour:    tierConfig.RequestsPerHour,
		RequestsPerDay:     tierConfig.RequestsPerDay,
	}, nil
}

// SetTierLimit updates the rate limit configuration for a tier
func (q *QuotaManager) SetTierLimit(tier RateLimitTier, config TierConfig) error {
	q.tiers[tier] = config
	return nil
}

// GetStorage returns the underlying storage
func (q *QuotaManager) GetStorage() QuotaStorage {
	return q.storage
}

// HealthCheck checks if the quota storage is healthy
func (q *QuotaManager) HealthCheck(ctx context.Context) error {
	return q.storage.Ping(ctx)
}

// ParseTierFromString parses a tier from string
func ParseTierFromString(s string) (RateLimitTier, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "free":
		return TierFree, nil
	case "pro":
		return TierPro, nil
	case "enterprise":
		return TierEnterprise, nil
	case "unlimited":
		return TierUnlimited, nil
	default:
		return TierFree, fmt.Errorf("unknown tier: %s", s)
	}
}

// TierToInt converts a tier to an integer for storage
func TierToInt(tier RateLimitTier) int {
	switch tier {
	case TierFree:
		return 0
	case TierPro:
		return 1
	case TierEnterprise:
		return 2
	case TierUnlimited:
		return 3
	default:
		return 0
	}
}

// IntToTier converts an integer to a tier
func IntToTier(i int) RateLimitTier {
	switch i {
	case 0:
		return TierFree
	case 1:
		return TierPro
	case 2:
		return TierEnterprise
	case 3:
		return TierUnlimited
	default:
		return TierFree
	}
}

// FormatRetryAfter formats retry-after value for headers
func FormatRetryAfter(seconds int) string {
	return strconv.Itoa(seconds)
}

// FormatResetHeader formats reset time for headers (Unix timestamp)
func FormatResetHeader(resetAt int64) string {
	return strconv.FormatInt(resetAt, 10)
}
