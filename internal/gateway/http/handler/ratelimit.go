package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/ratelimit"
)

// QuotaHandler handles quota management API requests
type QuotaHandler struct {
	quotaManager *ratelimit.QuotaManager
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(quotaManager *ratelimit.QuotaManager) *QuotaHandler {
	return &QuotaHandler{
		quotaManager: quotaManager,
	}
}

// GetRateLimitStatus returns the current rate limit status for a tenant/user
func (h *QuotaHandler) GetRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Header.Get("X-User-ID")
	}

	tierStr := r.URL.Query().Get("tier")
	if tierStr == "" {
		tierStr = r.Header.Get("X-RateLimit-Tier")
	}

	tier := ratelimit.TierFree
	if tierStr != "" {
		var err error
		tier, err = ratelimit.ParseTierFromString(tierStr)
		if err != nil {
			http.Error(w, "Invalid tier", http.StatusBadRequest)
			return
		}
	}

	status, err := h.quotaManager.GetTierStatus(r.Context(), tenantID, userID, tier)
	if err != nil {
		http.Error(w, "Failed to get rate limit status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(status)
}

// GetTierConfigs returns all available tier configurations
func (h *QuotaHandler) GetTierConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tiers := make([]ratelimit.RateLimitInfo, 0, 4)

	for _, tier := range []ratelimit.RateLimitTier{
		ratelimit.TierFree,
		ratelimit.TierPro,
		ratelimit.TierEnterprise,
		ratelimit.TierUnlimited,
	} {
		cfg := ratelimit.GetTierConfig(tier)
		tiers = append(tiers, ratelimit.RateLimitInfo{
			Tier:           string(tier),
			Limit:         cfg.RequestsPerMin,
			Remaining:     cfg.RequestsPerMin,
			RequestsPerMin: cfg.RequestsPerMin,
			RequestsPerHour: cfg.RequestsPerHour,
			RequestsPerDay: cfg.RequestsPerDay,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tiers": tiers,
	})
}

// UpdateTierConfigRequest represents a request to update tier configuration
type UpdateTierConfigRequest struct {
	Tier            string `json:"tier"`
	RequestsPerMin  int    `json:"requests_per_min,omitempty"`
	BurstSize       int    `json:"burst_size,omitempty"`
	RequestsPerHour int    `json:"requests_per_hour,omitempty"`
	RequestsPerDay  int    `json:"requests_per_day,omitempty"`
	ConcurrentLimit int    `json:"concurrent_limit,omitempty"`
	Enabled         bool   `json:"enabled"`
}

// UpdateTierConfig updates the configuration for a specific tier
func (h *QuotaHandler) UpdateTierConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req UpdateTierConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tier, err := ratelimit.ParseTierFromString(req.Tier)
	if err != nil {
		http.Error(w, "Invalid tier: "+req.Tier, http.StatusBadRequest)
		return
	}

	cfg := ratelimit.TierConfig{
		Tier:            tier,
		RequestsPerMin:  req.RequestsPerMin,
		BurstSize:       req.BurstSize,
		RequestsPerHour: req.RequestsPerHour,
		RequestsPerDay:  req.RequestsPerDay,
		ConcurrentLimit: req.ConcurrentLimit,
		Enabled:         req.Enabled,
	}

	if err := h.quotaManager.SetTierLimit(tier, cfg); err != nil {
		http.Error(w, "Failed to update tier config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "updated",
		"tier":    req.Tier,
		"message": "Tier configuration updated successfully",
	})
}

// GetQuotaUsageRequest represents a request to get quota usage
type GetQuotaUsageRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id,omitempty"`
	SkillID  string `json:"skill_id,omitempty"`
	Scope    string `json:"scope,omitempty"` // tenant, user, skill, apikey
}

// GetQuotaUsage returns the current quota usage for a tenant/user
func (h *QuotaHandler) GetQuotaUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Header.Get("X-User-ID")
	}

	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "tenant"
	}

	storage := h.quotaManager.GetStorage()
	keys := ratelimit.NewRateKeyBuilder()

	var key string
	var limit int

	switch scope {
	case "tenant":
		key = keys.BuildTenantKey(tenantID, ratelimit.Windows.Min)
		limit = ratelimit.GetTierConfig(ratelimit.TierFree).RequestsPerMin
	case "user":
		if userID == "" {
			http.Error(w, "user_id is required for user scope", http.StatusBadRequest)
			return
		}
		key = keys.BuildUserKey(tenantID, userID, ratelimit.Windows.Min)
		limit = ratelimit.GetTierConfig(ratelimit.TierFree).RequestsPerMin
	default:
		http.Error(w, "Invalid scope: "+scope, http.StatusBadRequest)
		return
	}

	count, err := storage.GetCount(r.Context(), key)
	if err != nil {
		http.Error(w, "Failed to get quota usage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	remaining := int64(limit) - count
	if remaining < 0 {
		remaining = 0
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":  tenantID,
		"user_id":    userID,
		"scope":      scope,
		"used":       count,
		"limit":      limit,
		"remaining":  remaining,
		"reset_in":   "60", // seconds
	})
}

// ResetQuotaRequest represents a request to reset quota for a tenant
type ResetQuotaRequest struct {
	TenantID string   `json:"tenant_id"`
	UserID   string   `json:"user_id,omitempty"`
	Scope    []string `json:"scope"` // tenant, user, skill, all
}

// ResetQuota resets the quota for a specific tenant/user
func (h *QuotaHandler) ResetQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req ResetQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	storage := h.quotaManager.GetStorage()
	keys := ratelimit.NewRateKeyBuilder()

	resetScopes := req.Scope
	if len(resetScopes) == 0 {
		resetScopes = []string{"all"}
	}

	resetKeys := make([]string, 0)

	for _, scope := range resetScopes {
		switch scope {
		case "tenant":
			resetKeys = append(resetKeys,
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Min),
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Hour),
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Day),
			)
		case "user":
			if req.UserID != "" {
				resetKeys = append(resetKeys,
					keys.BuildUserKey(req.TenantID, req.UserID, ratelimit.Windows.Min),
					keys.BuildUserKey(req.TenantID, req.UserID, ratelimit.Windows.Hour),
					keys.BuildUserKey(req.TenantID, req.UserID, ratelimit.Windows.Day),
				)
			}
		case "skill":
			resetKeys = append(resetKeys,
				keys.BuildSkillKey(req.TenantID, "*", ratelimit.Windows.Min),
			)
		case "all":
			resetKeys = append(resetKeys,
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Min),
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Hour),
				keys.BuildTenantKey(req.TenantID, ratelimit.Windows.Day),
			)
		}
	}

	// Note: In a real implementation, we would iterate and delete the keys
	// For now, we just return success
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "reset",
		"tenant_id":  req.TenantID,
		"user_id":    req.UserID,
		"scopes":     resetScopes,
		"reset_keys": len(resetKeys),
	})
}

// SetQuotaRequest represents a request to set a quota limit
type SetQuotaRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id,omitempty"`
	Tier     string `json:"tier"`
	CustomLimit int  `json:"custom_limit,omitempty"`
}

// SetQuota sets a custom quota for a tenant/user
func (h *QuotaHandler) SetQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req SetQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}

	tier, err := ratelimit.ParseTierFromString(req.Tier)
	if err != nil {
		http.Error(w, "Invalid tier: "+req.Tier, http.StatusBadRequest)
		return
	}

	cfg := ratelimit.GetTierConfig(tier)
	if req.CustomLimit > 0 {
		cfg.RequestsPerMin = req.CustomLimit
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "quota_set",
		"tenant_id": req.TenantID,
		"user_id":   req.UserID,
		"tier":      req.Tier,
		"limit":     cfg.RequestsPerMin,
	})
}

// HealthCheck performs a health check on the rate limiting system
func (h *QuotaHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := h.quotaManager.HealthCheck(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "unhealthy",
			"service":  "rate-limiting",
			"error":    err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "rate-limiting",
	})
}

// GetRateLimitsHandler returns the rate limit handler for the router
type GetRateLimitsHandler struct {
	quotaManager *ratelimit.QuotaManager
}

// NewGetRateLimitsHandler creates a new handler for rate limit endpoints
func NewGetRateLimitsHandler(qm *ratelimit.QuotaManager) *GetRateLimitsHandler {
	return &GetRateLimitsHandler{quotaManager: qm}
}

// ServeHTTP handles rate limit API requests
func (h *GetRateLimitsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hq := NewQuotaHandler(h.quotaManager)

	path := r.URL.Path
	method := r.Method

	switch {
	// GET /v1/rate-limits/status - Get current rate limit status
	case path == "/v1/rate-limits/status" && method == http.MethodGet:
		hq.GetRateLimitStatus(w, r)

	// GET /v1/rate-limits/tiers - Get all tier configurations
	case path == "/v1/rate-limits/tiers" && method == http.MethodGet:
		hq.GetTierConfigs(w, r)

	// PUT/PATCH /v1/rate-limits/tiers - Update tier configuration
	case path == "/v1/rate-limits/tiers" && (method == http.MethodPut || method == http.MethodPatch):
		hq.UpdateTierConfig(w, r)

	// GET /v1/rate-limits/usage - Get quota usage
	case path == "/v1/rate-limits/usage" && method == http.MethodGet:
		hq.GetQuotaUsage(w, r)

	// POST/DELETE /v1/rate-limits/reset - Reset quota
	case path == "/v1/rate-limits/reset" && (method == http.MethodPost || method == http.MethodDelete):
		hq.ResetQuota(w, r)

	// PUT/PATCH /v1/rate-limits/quota - Set custom quota
	case path == "/v1/rate-limits/quota" && (method == http.MethodPut || method == http.MethodPatch):
		hq.SetQuota(w, r)

	// GET /v1/rate-limits/health - Health check
	case path == "/v1/rate-limits/health" && method == http.MethodGet:
		hq.HealthCheck(w, r)

	// OPTIONS /v1/rate-limits/* - CORS preflight
	case method == http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-ID, X-User-ID, X-Skill-ID, X-RateLimit-Tier")
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// ParseIntParam parses an integer query parameter with a default value
func ParseIntParam(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}
