package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/quota"
)

// QuotaHandler handles quota management endpoints
type QuotaHandler struct {
	quotaRepo quota.QuotaRepository
	limiter   *quota.TenantRateLimiter
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(qr quota.QuotaRepository, limiter *quota.TenantRateLimiter) *QuotaHandler {
	return &QuotaHandler{
		quotaRepo: qr,
		limiter:   limiter,
	}
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

	plan, err := h.quotaRepo.GetPlan(ctx, usage.Plan)
	if err != nil {
		// Use default free plan
		plans := quota.DefaultQuotaPlans()
		plan = plans["free"]
	}

	remaining := h.limiter.GetRemaining(tenantID)
	resetTime := h.limiter.GetResetTime(tenantID)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":    usage.TenantID,
		"plan":         usage.Plan,
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
			"reset_at":      resetTime,
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
		"status":    "reset",
		"tenant_id": tenantID,
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
		Name             string `json:"name"`
		RequestsPerMin   int    `json:"requests_per_minute"`
		RequestsPerHour  int    `json:"requests_per_hour"`
		RequestsPerDay   int    `json:"requests_per_day"`
		MaxTokensPerDay  int    `json:"max_tokens_per_day"`
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

	ctx := context.Background()
	if err := h.quotaRepo.UpdatePlan(ctx, plan); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update plan: %v", err), http.StatusInternalServerError)
		return
	}

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
		"allowed":    allowed,
		"tenant_id":  tenantID,
		"remaining":  remaining,
		"reset_at":   resetTime,
		"limit":      60, // requests per minute
	})
}
