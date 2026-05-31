package ratelimit

// RateLimitTier defines the rate limiting tier for a tenant/user
type RateLimitTier string

const (
	TierFree       RateLimitTier = "free"
	TierPro        RateLimitTier = "pro"
	TierEnterprise RateLimitTier = "enterprise"
	TierUnlimited  RateLimitTier = "unlimited"
)

// TierConfig holds rate limit configuration for a specific tier
type TierConfig struct {
	Tier             RateLimitTier
	RequestsPerMin   int
	BurstSize        int
	RequestsPerHour  int
	RequestsPerDay   int
	ConcurrentLimit  int
	Enabled          bool
}

// DefaultTierConfigs provides default rate limit configurations
var DefaultTierConfigs = map[RateLimitTier]TierConfig{
	TierFree: {
		Tier:            TierFree,
		RequestsPerMin:  10,
		BurstSize:       5,
		RequestsPerHour: 100,
		RequestsPerDay:  500,
		ConcurrentLimit: 3,
		Enabled:         true,
	},
	TierPro: {
		Tier:            TierPro,
		RequestsPerMin:  100,
		BurstSize:       50,
		RequestsPerHour: 1000,
		RequestsPerDay:  10000,
		ConcurrentLimit: 10,
		Enabled:         true,
	},
	TierEnterprise: {
		Tier:            TierEnterprise,
		RequestsPerMin:  1000,
		BurstSize:       500,
		RequestsPerHour: 20000,
		RequestsPerDay:  100000,
		ConcurrentLimit: 50,
		Enabled:         true,
	},
	TierUnlimited: {
		Tier:            TierUnlimited,
		RequestsPerMin:  10000,
		BurstSize:       5000,
		RequestsPerHour: 100000,
		RequestsPerDay:  1000000,
		ConcurrentLimit: 200,
		Enabled:         true,
	},
}

// GetTierConfig returns the configuration for a given tier
func GetTierConfig(tier RateLimitTier) TierConfig {
	if cfg, ok := DefaultTierConfigs[tier]; ok {
		return cfg
	}
	return DefaultTierConfigs[TierFree]
}

// RateLimitScope defines the scope of rate limiting
type RateLimitScope string

const (
	ScopeGlobal  RateLimitScope = "global"
	ScopeTenant  RateLimitScope = "tenant"
	ScopeUser    RateLimitScope = "user"
	ScopeSkill   RateLimitScope = "skill"
	ScopeAPIKey  RateLimitScope = "apikey"
)

// RateLimitType defines the type of rate limit
type RateLimitType string

const (
	TypeSlidingWindow RateLimitType = "sliding_window"
	TypeTokenBucket   RateLimitType = "token_bucket"
	TypeFixedWindow   RateLimitType = "fixed_window"
)

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed        bool
	Limit          int
	Remaining      int
	ResetAt        int64 // Unix timestamp when the limit resets
	RetryAfter     int   // Seconds until retry is allowed (only set when not allowed)
	Scope          RateLimitScope
	Tier           RateLimitTier
	Used           int
	ResetInSeconds int
}

// RateLimitInfo contains detailed rate limit information for response headers
type RateLimitInfo struct {
	Limit              int   `json:"limit"`
	Remaining          int   `json:"remaining"`
	Reset              int64 `json:"reset"`
	ResetInSeconds     int   `json:"reset_in_seconds"`
	RetryAfter         int   `json:"retry_after,omitempty"`
	Tier               string `json:"tier"`
	Scope              string `json:"scope"`
	RequestsPerMin     int    `json:"requests_per_min"`
	RequestsPerHour    int    `json:"requests_per_hour"`
	RequestsPerDay     int    `json:"requests_per_day"`
}

// RateLimitConfig holds the full rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool
	DefaultTier     RateLimitTier
	RedisURL        string
	WindowSizeSecs  int
	BypassLimits    bool // For testing/admin purposes
}

// DefaultRateLimitConfig returns a sensible default configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:        true,
		DefaultTier:     TierFree,
		WindowSizeSecs: 60,
		BypassLimits:   false,
	}
}
