package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/apierror"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// RateLimitMiddleware provides multi-layered rate limiting
type RateLimitMiddleware struct {
	quotaManager *QuotaManager
	tenantResolver tenancy.TenantResolver
	config       *RateLimitConfig
	logger       *slog.Logger
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(quotaManager *QuotaManager, tenantResolver tenancy.TenantResolver, config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return &RateLimitMiddleware{
		quotaManager:  quotaManager,
		tenantResolver: tenantResolver,
		config:        config,
		logger:        slog.Default(),
	}
}

// Middleware returns an HTTP middleware function for rate limiting
func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		// Extract identifiers from request
		identifiers := m.extractIdentifiers(r)

		// Get tier from tenant or use default
		tier := m.getTier(ctx, identifiers.TenantID)

		// Check all rate limits
		result, err := m.quotaManager.CheckAllLimits(
			ctx,
			identifiers.TenantID,
			identifiers.UserID,
			identifiers.SkillID,
			identifiers.APIKeyHash,
			tier,
		)

		if err != nil {
			m.logger.ErrorContext(ctx, "Rate limit check failed",
				slog.String("error", err.Error()),
				slog.Any("identifiers", identifiers),
			)
			// Fail open with warning - allow request but log
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		m.setRateLimitHeaders(w, result)

		if !result.Allowed {
			m.logger.WarnContext(ctx, "Rate limit exceeded",
				slog.String("scope", string(result.Scope)),
				slog.String("tier", string(result.Tier)),
				slog.Int("limit", result.Limit),
				slog.Int("retry_after", result.RetryAfter),
				slog.Any("identifiers", identifiers),
			)

			// Return 429 Too Many Requests
			apierror.WriteError(
				w,
				http.StatusTooManyRequests,
				"RATE_LIMIT_EXCEEDED",
				fmt.Sprintf("Rate limit exceeded. Retry after %d seconds.", result.RetryAfter),
			)
			return
		}

		// Store rate limit info in context for downstream use
		ctx = context.WithValue(ctx, RateLimitContextKey{}, result)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIdentifiers contains all identifiers for rate limiting
type RequestIdentifiers struct {
	TenantID   string
	UserID     string
	SkillID    string
	APIKeyHash string
	IPAddress  string
}

// extractIdentifiers extracts all relevant identifiers from the request
func (m *RateLimitMiddleware) extractIdentifiers(r *http.Request) *RequestIdentifiers {
	identifiers := &RequestIdentifiers{
		TenantID: "default-tenant",
	}

	// Extract tenant from header or context
	tenantHeader := r.Header.Get("X-Tenant-ID")
	if tenantHeader != "" {
		identifiers.TenantID = tenantHeader
	}

	// Extract user from header or auth context
	userHeader := r.Header.Get("X-User-ID")
	if userHeader != "" {
		identifiers.UserID = userHeader
	}

	// Extract skill from request body or header
	skillHeader := r.Header.Get("X-Skill-ID")
	if skillHeader != "" {
		identifiers.SkillID = skillHeader
	}

	// Extract API key hash
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			identifiers.APIKeyHash = hashAPIKey(apiKey)
		} else if strings.HasPrefix(authHeader, "ApiKey ") {
			apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
			identifiers.APIKeyHash = hashAPIKey(apiKey)
		} else {
			identifiers.APIKeyHash = hashAPIKey(authHeader)
		}
	}

	// Extract IP address
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	// Check for X-Forwarded-For header (proxy scenarios)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}
	identifiers.IPAddress = ip

	return identifiers
}

// getTier determines the rate limit tier for a tenant
func (m *RateLimitMiddleware) getTier(ctx context.Context, tenantID string) RateLimitTier {
	// Try to get tenant from context
	if tenant, err := tenancy.GetTenant(ctx); err == nil && tenant != nil {
		switch strings.ToLower(tenant.Plan) {
		case "free":
			return TierFree
		case "pro":
			return TierPro
		case "enterprise":
			return TierEnterprise
		case "unlimited":
			return TierUnlimited
		}
	}

	// Try to resolve tenant from tenant resolver
	if m.tenantResolver != nil {
		if tenant, err := m.tenantResolver.Resolve(ctx, tenantID); err == nil && tenant != nil {
			switch strings.ToLower(tenant.Plan) {
			case "free":
				return TierFree
			case "pro":
				return TierPro
			case "enterprise":
				return TierEnterprise
			case "unlimited":
				return TierUnlimited
			}
		}
	}

	// Check for tier override header
	tierHeader := ctx.Value("X-RateLimit-Tier")
	if tierHeader != nil {
		if tier, ok := tierHeader.(string); ok {
			switch strings.ToLower(tier) {
			case "free":
				return TierFree
			case "pro":
				return TierPro
			case "enterprise":
				return TierEnterprise
			case "unlimited":
				return TierUnlimited
			}
		}
	}

	return m.config.DefaultTier
}

// setRateLimitHeaders sets the standard rate limit headers on the response
func (m *RateLimitMiddleware) setRateLimitHeaders(w http.ResponseWriter, result *RateLimitResult) {
	// X-RateLimit-Limit: The maximum number of requests allowed in the current window
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))

	// X-RateLimit-Remaining: The number of requests remaining in the current window
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))

	// X-RateLimit-Reset: Unix timestamp when the rate limit resets
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt))

	// X-RateLimit-Used: Number of requests used in the current window
	w.Header().Set("X-RateLimit-Used", fmt.Sprintf("%d", result.Used))

	// X-RateLimit-Scope: The scope of the rate limit (global, tenant, user, skill, apikey)
	w.Header().Set("X-RateLimit-Scope", string(result.Scope))

	// X-RateLimit-Tier: The tier being applied
	w.Header().Set("X-RateLimit-Tier", string(result.Tier))

	// Retry-After: Seconds until retry is allowed (only when rate limited)
	if !result.Allowed && result.RetryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfter))
	}

	// X-RateLimit-Window: The time window in seconds
	w.Header().Set("X-RateLimit-Window", "60")
}

// hashAPIKey creates a SHA256 hash of an API key for storage
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// RateLimitContextKey is the context key for rate limit info
type RateLimitContextKey struct{}

// GetRateLimitResult retrieves the rate limit result from context
func GetRateLimitResult(ctx context.Context) *RateLimitResult {
	if val := ctx.Value(RateLimitContextKey{}); val != nil {
		if result, ok := val.(*RateLimitResult); ok {
			return result
		}
	}
	return nil
}

// TierConfigOverride allows dynamic tier configuration per tenant
type TierConfigOverride struct {
	TenantID string
	Tier     RateLimitTier
	CustomLimits *TierConfig
}

// UpdateTierConfig updates the tier configuration for a specific tenant
func (m *RateLimitMiddleware) UpdateTierConfig(tenantID string, tier RateLimitTier, customLimits *TierConfig) error {
	if customLimits != nil {
		return m.quotaManager.SetTierLimit(tier, *customLimits)
	}
	return nil
}

// GetCurrentStatus returns the current rate limit status for a request
func (m *RateLimitMiddleware) GetCurrentStatus(r *http.Request) (*RateLimitInfo, error) {
	ctx := r.Context()
	identifiers := m.extractIdentifiers(r)
	tier := m.getTier(ctx, identifiers.TenantID)

	return m.quotaManager.GetTierStatus(ctx, identifiers.TenantID, identifiers.UserID, tier)
}

// GracefulDegradation returns a middleware that handles rate limit failures gracefully
func (m *RateLimitMiddleware) GracefulDegradation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to execute the rate limited handler
		// The handler will set headers even when rate limited
		originalHandler := m.Middleware(next)

		// Check if quota manager is healthy
		ctx := r.Context()
		if err := m.quotaManager.HealthCheck(ctx); err != nil {
			// Storage unhealthy - log and continue with degraded service
			m.logger.WarnContext(ctx, "Rate limit storage unhealthy, operating in degraded mode",
				slog.String("error", err.Error()),
			)

			// Set degraded mode headers
			w.Header().Set("X-RateLimit-Degraded", "true")
			w.Header().Set("X-RateLimit-Mode", "best-effort")

			// Continue without rate limiting
			next.ServeHTTP(w, r)
			return
		}

		originalHandler.ServeHTTP(w, r)
	})
}

// WithBurstProtection returns a middleware that adds burst protection
func (m *RateLimitMiddleware) WithBurstProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifiers := m.extractIdentifiers(r)
		ctx := r.Context()
		tier := m.getTier(ctx, identifiers.TenantID)
		tierConfig := GetTierConfig(tier)

		// Check burst limit
		burstKey := fmt.Sprintf("burst:%s:%s", identifiers.TenantID, identifiers.IPAddress)
		storage := m.quotaManager.GetStorage()

		acquired, err := storage.AcquireBurst(ctx, burstKey, tierConfig.BurstSize, time.Minute)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to check burst limit",
				slog.String("error", err.Error()),
			)
			// Fail open
			next.ServeHTTP(w, r)
			return
		}

		if !acquired {
			m.logger.WarnContext(ctx, "Burst limit exceeded",
				slog.String("ip", identifiers.IPAddress),
				slog.String("tenant", identifiers.TenantID),
			)

			apierror.WriteError(
				w,
				http.StatusTooManyRequests,
				"BURST_LIMIT_EXCEEDED",
				"Burst limit exceeded. Please slow down your requests.",
			)
			return
		}

		// Release burst slot when request completes
		defer func() {
			_ = storage.ReleaseBurst(ctx, burstKey)
		}()

		next.ServeHTTP(w, r)
	})
}

// WithConcurrentLimit returns a middleware that limits concurrent requests
func (m *RateLimitMiddleware) WithConcurrentLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifiers := m.extractIdentifiers(r)
		ctx := r.Context()
		tier := m.getTier(ctx, identifiers.TenantID)
		tierConfig := GetTierConfig(tier)

		// Check concurrent limit
		concurrentKey := fmt.Sprintf("concurrent:%s:%s", identifiers.TenantID, identifiers.IPAddress)
		storage := m.quotaManager.GetStorage()

		count, err := storage.IncrementConcurrent(ctx, concurrentKey)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to increment concurrent count",
				slog.String("error", err.Error()),
			)
			// Fail open
			next.ServeHTTP(w, r)
			return
		}

		// Check if limit exceeded
		if int(count) > tierConfig.ConcurrentLimit {
			// Decrement since we're not proceeding
			_ = storage.DecrementConcurrent(ctx, concurrentKey)

			apierror.WriteError(
				w,
				http.StatusTooManyRequests,
				"CONCURRENT_LIMIT_EXCEEDED",
				"Too many concurrent requests. Please wait for some requests to complete.",
			)
			return
		}

		// Decrement when request completes
		defer func() {
			_ = storage.DecrementConcurrent(ctx, concurrentKey)
		}()

		next.ServeHTTP(w, r)
	})
}
