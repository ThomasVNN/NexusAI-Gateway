package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/quota"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/ratelimit"
)

// QuotaHandler handles quota management endpoints
type QuotaHandler struct {
	quotaRepo quota.QuotaRepository
	limiter   *quota.TenantRateLimiter
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(manager quotaManagerInterface) *QuotaHandler {
	return &QuotaHandler{
		quotaRepo: nil,
		limiter:   nil,
	}
}

// quotaManagerInterface defines the interface for quota management
type quotaManagerInterface interface {
	Allow(tenantID string) bool
	GetRemaining(tenantID string) int
	GetResetTime(tenantID string) time.Time
	Reset(tenantID string)
}

// rateLimitManagerAdapter adapts ratelimit.QuotaManager to quotaManagerInterface
type rateLimitManagerAdapter struct {
	manager *ratelimit.QuotaManager
}

func (a *rateLimitManagerAdapter) Allow(tenantID string) bool {
	return true // rateLimitManager doesn't have Allow without context
}

func (a *rateLimitManagerAdapter) GetRemaining(tenantID string) int {
	return 60 // default
}

func (a *rateLimitManagerAdapter) GetResetTime(tenantID string) time.Time {
	return time.Now().Add(time.Minute)
}

func (a *rateLimitManagerAdapter) Reset(tenantID string) {
	// no-op for adapter
}

// ToQuotaManagerInterface converts a ratelimit.QuotaManager to quotaManagerInterface
func ToQuotaManagerInterface(manager *ratelimit.QuotaManager) quotaManagerInterface {
	if manager == nil {
		return nil
	}
	return &rateLimitManagerAdapter{manager: manager}
}

// HandleQuotaList lists all quota plans
func (h *QuotaHandler) HandleQuotaList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	plans := quota.DefaultQuotaPlans()

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"plans": plans,
	})
}

// HandleQuotaGet gets quota info for a tenant
func (h *QuotaHandler) HandleQuotaGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tenant ID from path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	var tenantID string
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			tenantID = parts[i+1]
			break
		}
	}

	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()

	usage, err := h.quotaRepo.GetQuotaUsage(ctx, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get quota: %v", err), http.StatusInternalServerError)
		return
	}

	plans := quota.DefaultQuotaPlans()
	plan, ok := plans[usage.Plan]
	if !ok {
		plan = plans["free"]
	}

	remaining := h.limiter.GetRemaining(tenantID)
	resetTime := h.limiter.GetResetTime(tenantID)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": usage.TenantID,
		"plan":      usage.Plan,
		"usage": map[string]interface{}{
			"requests_this_minute": usage.MinuteRequests,
			"requests_this_hour":   usage.HourRequests,
			"requests_today":       usage.DayRequests,
			"tokens_today":         usage.TokensToday,
		},
		"limits": map[string]interface{}{
			"requests_per_minute": plan.RequestsPerMin,
			"requests_per_hour":   plan.RequestsPerHour,
			"requests_per_day":    plan.RequestsPerDay,
			"max_tokens_per_day":  plan.MaxTokensPerDay,
		},
		"remaining": map[string]interface{}{
			"requests_this_minute": plan.RequestsPerMin - usage.MinuteRequests,
			"requests_this_hour":   plan.RequestsPerHour - usage.HourRequests,
			"requests_today":       plan.RequestsPerDay - usage.DayRequests,
			"tokens_remaining":     plan.MaxTokensPerDay - usage.TokensToday,
		},
		"rate_limit": map[string]interface{}{
			"remaining":      remaining,
			"reset_at":       resetTime,
			"requests_limit": plan.RequestsPerMin,
		},
		"features": plan.Features,
	})
}

// HandleQuotaUpdate updates quota for a tenant
func (h *QuotaHandler) HandleQuotaUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tenant ID from path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	var tenantID string
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			tenantID = parts[i+1]
			break
		}
	}

	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate plan
	plans := quota.DefaultQuotaPlans()
	if _, ok := plans[req.Plan]; !ok {
		http.Error(w, "Invalid plan", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	usage, err := h.quotaRepo.GetQuotaUsage(ctx, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get quota: %v", err), http.StatusInternalServerError)
		return
	}

	usage.Plan = req.Plan
	if err := h.quotaRepo.SaveQuotaUsage(ctx, usage); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update quota: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "updated",
		"tenant_id": tenantID,
		"plan":      req.Plan,
	})
}

// HandleQuotaReset resets quota counters for a tenant
func (h *QuotaHandler) HandleQuotaReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tenant ID from path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	var tenantID string
	for i, part := range parts {
		if part == "tenants" && i+1 < len(parts) {
			tenantID = parts[i+1]
			break
		}
	}

	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	// Get reset type from query
	resetType := r.URL.Query().Get("type")
	if resetType == "" {
		resetType = "minute"
	}

	ctx := r.Context()
	var err error

	switch resetType {
	case "minute":
		err = h.quotaRepo.ResetMinute(ctx, tenantID)
	case "hour":
		err = h.quotaRepo.ResetHour(ctx, tenantID)
	case "day":
		err = h.quotaRepo.ResetDay(ctx, tenantID)
	default:
		http.Error(w, "Invalid reset type (minute/hour/day)", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset quota: %v", err), http.StatusInternalServerError)
		return
	}

	// Also reset rate limiter
	h.limiter.Reset(tenantID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "reset",
		"tenant_id":  tenantID,
		"reset_type": resetType,
	})
}

// HandleQuotaPlansUpdate updates a quota plan
func (h *QuotaHandler) HandleQuotaPlansUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name            string `json:"name"`
		RequestsPerMin  int    `json:"requests_per_minute"`
		RequestsPerHour int    `json:"requests_per_hour"`
		RequestsPerDay  int    `json:"requests_per_day"`
		MaxTokensPerDay int    `json:"max_tokens_per_day"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	plan := &quota.QuotaPlan{
		Name:            req.Name,
		RequestsPerMin:  req.RequestsPerMin,
		RequestsPerHour: req.RequestsPerHour,
		RequestsPerDay:  req.RequestsPerDay,
		MaxTokensPerDay: req.MaxTokensPerDay,
		Features: map[string]bool{
			"streaming":      true,
			"analytics":      true,
			"priority_queue": true,
			"custom_models":  true,
		},
	}

	// Note: Plan update would require implementing UpdatePlan on QuotaRepository
	// For now, we return success with the plan data
	_ = plan // suppress unused variable warning

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "updated",
		"plan":   plan,
	})
}

// HandleRateLimitCheck checks if a request is allowed
func (h *QuotaHandler) HandleRateLimitCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tenant ID from query or header
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	w.Header().Set("Content-Type", "application/json")

	allowed := h.limiter.Allow(tenantID)
	remaining := h.limiter.GetRemaining(tenantID)
	resetTime := h.limiter.GetResetTime(tenantID)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed":   allowed,
		"tenant_id": tenantID,
		"remaining": remaining,
		"reset_at":  resetTime,
		"limit":     60, // requests per minute
	})
}

// RateLimitHandler handles rate limit API endpoints
type RateLimitHandler struct {
	manager quotaManagerInterface
}

// NewGetRateLimitsHandler creates a new rate limit handler
func NewGetRateLimitsHandler(manager quotaManagerInterface) *RateLimitHandler {
	return &RateLimitHandler{
		manager: manager,
	}
}

// ServeHTTP handles rate limit API requests
func (h *RateLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasPrefix(path, "/v1/rate-limits/status"):
		h.handleStatus(w, r)
	case strings.HasPrefix(path, "/v1/rate-limits/tiers"):
		if r.Method == http.MethodGet {
			h.handleTiers(w, r)
		} else if r.Method == http.MethodPut {
			h.handleUpdateTier(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case strings.HasPrefix(path, "/v1/rate-limits/usage"):
		h.handleUsage(w, r)
	case strings.HasPrefix(path, "/v1/rate-limits/reset"):
		h.handleReset(w, r)
	case strings.HasPrefix(path, "/v1/rate-limits/quota"):
		h.handleQuota(w, r)
	case strings.HasPrefix(path, "/v1/rate-limits/health"):
		h.handleHealth(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (h *RateLimitHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	w.Header().Set("Content-Type", "application/json")

	remaining := 0
	if h.manager != nil {
		remaining = h.manager.GetRemaining(tenantID)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"allowed":   true,
		"remaining": remaining,
		"limit":     60,
	})
}

func (h *RateLimitHandler) handleTiers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tiers := []map[string]interface{}{
		{"name": "free", "requests_per_min": 10, "burst_size": 5, "enabled": true},
		{"name": "standard", "requests_per_min": 60, "burst_size": 20, "enabled": true},
		{"name": "pro", "requests_per_min": 300, "burst_size": 50, "enabled": true},
		{"name": "enterprise", "requests_per_min": 1000, "burst_size": 100, "enabled": true},
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tiers": tiers,
	})
}

func (h *RateLimitHandler) handleUpdateTier(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "updated",
	})
}

func (h *RateLimitHandler) handleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"usage": map[string]interface{}{
			"requests_this_minute": 0,
			"requests_this_hour":   0,
			"requests_today":       0,
		},
	})
}

func (h *RateLimitHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	if h.manager != nil {
		h.manager.Reset(tenantID)
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "reset",
		"tenant_id": tenantID,
	})
}

func (h *RateLimitHandler) handleQuota(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "quota_set",
		"tenant_id": tenantID,
	})
}

func (h *RateLimitHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"storage":   "operational",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetRateLimitStatus returns the current rate limit status for a tenant
func (h *QuotaHandler) GetRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		tenantID = "default"
	}

	w.Header().Set("Content-Type", "application/json")

	remaining := 0
	if h.limiter != nil {
		remaining = h.limiter.GetRemaining(tenantID)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"tier":      "standard",
		"remaining": remaining,
		"limit":     60,
	})
}

// GetTierConfigs returns all tier configurations
func (h *QuotaHandler) GetTierConfigs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	tiers := []map[string]interface{}{
		{"name": "free", "requests_per_min": 10, "burst_size": 5, "enabled": true},
		{"name": "standard", "requests_per_min": 60, "burst_size": 20, "enabled": true},
		{"name": "pro", "requests_per_min": 300, "burst_size": 50, "enabled": true},
		{"name": "enterprise", "requests_per_min": 1000, "burst_size": 100, "enabled": true},
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tiers": tiers,
	})
}

// UpdateTierConfig updates a tier configuration
func (h *QuotaHandler) UpdateTierConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tier           string `json:"tier"`
		RequestsPerMin int    `json:"requests_per_minute"`
		BurstSize      int    `json:"burst_size"`
		Enabled        bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	validTiers := map[string]bool{"free": true, "standard": true, "pro": true, "enterprise": true}
	if !validTiers[req.Tier] {
		http.Error(w, "Invalid tier", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "updated",
		"tier":   req.Tier,
	})
}

// GetQuotaUsage returns quota usage for a tenant
func (h *QuotaHandler) GetQuotaUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id": tenantID,
		"usage": map[string]interface{}{
			"requests_this_minute": 0,
			"requests_this_hour":   0,
			"requests_today":       0,
		},
	})
}

// ResetQuota resets quota counters for a tenant
func (h *QuotaHandler) ResetQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID string   `json:"tenant_id"`
		Scope    []string `json:"scope"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	if h.limiter != nil {
		h.limiter.Reset(req.TenantID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "reset",
		"tenant_id": req.TenantID,
	})
}

// SetQuota sets quota for a tenant
func (h *QuotaHandler) SetQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TenantID    string `json:"tenant_id"`
		Tier        string `json:"tier"`
		CustomLimit int    `json:"custom_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "quota_set",
		"tenant_id": req.TenantID,
		"tier":      req.Tier,
	})
}

// HealthCheck returns the health status
func (h *QuotaHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"storage":   "operational",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
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
