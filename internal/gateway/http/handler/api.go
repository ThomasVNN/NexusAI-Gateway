package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/billing"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/channel"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/log"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/token"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/user"
)

// NewAPIHandler creates a handler for new-api admin endpoints
func NewAPIHandler(
	chService *channel.Service,
	tgService *token.Service,
	uService *user.Service,
	logService *log.Service,
	billingService *billing.Service,
) *APIHandler {
	return &APIHandler{
		chService:      chService,
		tgService:      tgService,
		uService:       uService,
		logService:     logService,
		billingService: billingService,
	}
}

// APIHandler handles new-api administrative endpoints
type APIHandler struct {
	chService      *channel.Service
	tgService      *token.Service
	uService       *user.Service
	logService     *log.Service
	billingService *billing.Service
}

// Channel Handlers

// HandleChannels handles /api/channels
func (h *APIHandler) HandleChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listChannels(w, r)
	case "POST":
		h.createChannel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleChannel handles /api/channels/:id
func (h *APIHandler) HandleChannel(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path)
	if id == 0 {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getChannel(w, r, id)
	case "PUT":
		h.updateChannel(w, r, id)
	case "DELETE":
		h.deleteChannel(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleChannelTest handles /api/channels/:id/test
func (h *APIHandler) HandleChannelTest(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path)
	if id == 0 {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := h.chService.TestChannel(r.Context(), id, "")
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

func (h *APIHandler) listChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := h.chService.List(r.Context(), nil)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, channels)
}

func (h *APIHandler) createChannel(w http.ResponseWriter, r *http.Request) {
	var ch channel.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.chService.Create(r.Context(), &ch); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, ch, http.StatusCreated)
}

func (h *APIHandler) getChannel(w http.ResponseWriter, r *http.Request, id int64) {
	ch, err := h.chService.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusNotFound)
		return
	}

	respondJSON(w, ch)
}

func (h *APIHandler) updateChannel(w http.ResponseWriter, r *http.Request, id int64) {
	var ch channel.Channel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	ch.ID = id
	if err := h.chService.Update(r.Context(), &ch); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, ch)
}

func (h *APIHandler) deleteChannel(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.chService.Delete(r.Context(), id); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]bool{"deleted": true})
}

// Token Group Handlers

// HandleTokenGroups handles /api/token-groups
func (h *APIHandler) HandleTokenGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listTokenGroups(w, r)
	case "POST":
		h.createTokenGroup(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTokenGroup handles /api/token-groups/:id
func (h *APIHandler) HandleTokenGroup(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path)
	if id == 0 {
		http.Error(w, "Invalid token group ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getTokenGroup(w, r, id)
	case "PUT":
		h.updateTokenGroup(w, r, id)
	case "DELETE":
		h.deleteTokenGroup(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) listTokenGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.tgService.List(r.Context())
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, groups)
}

func (h *APIHandler) createTokenGroup(w http.ResponseWriter, r *http.Request) {
	var tg token.TokenGroup
	if err := json.NewDecoder(r.Body).Decode(&tg); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.tgService.Create(r.Context(), &tg); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, tg, http.StatusCreated)
}

func (h *APIHandler) getTokenGroup(w http.ResponseWriter, r *http.Request, id int64) {
	tg, err := h.tgService.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusNotFound)
		return
	}

	respondJSON(w, tg)
}

func (h *APIHandler) updateTokenGroup(w http.ResponseWriter, r *http.Request, id int64) {
	var tg token.TokenGroup
	if err := json.NewDecoder(r.Body).Decode(&tg); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	tg.ID = id
	if err := h.tgService.Update(r.Context(), &tg); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, tg)
}

func (h *APIHandler) deleteTokenGroup(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.tgService.Delete(r.Context(), id); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]bool{"deleted": true})
}

// User Handlers

// HandleUsers handles /api/users
func (h *APIHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listUsers(w, r)
	case "POST":
		h.createUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleUser handles /api/users/:id
func (h *APIHandler) HandleUser(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path)
	if id == 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getUser(w, r, id)
	case "PUT":
		h.updateUser(w, r, id)
	case "DELETE":
		h.deleteUser(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.uService.List(r.Context())
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, users)
}

func (h *APIHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var u user.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.uService.Create(r.Context(), &u); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, u, http.StatusCreated)
}

func (h *APIHandler) getUser(w http.ResponseWriter, r *http.Request, id int64) {
	u, err := h.uService.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusNotFound)
		return
	}

	respondJSON(w, u)
}

func (h *APIHandler) updateUser(w http.ResponseWriter, r *http.Request, id int64) {
	var u user.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	u.ID = id
	if err := h.uService.Update(r.Context(), &u); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, u)
}

func (h *APIHandler) deleteUser(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.uService.Delete(r.Context(), id); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]bool{"deleted": true})
}

// Analytics Handlers

// HandleAnalyticsOverview handles /api/analytics/overview
func (h *APIHandler) HandleAnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	overview, err := h.logService.GetOverview(r.Context())
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, overview)
}

// HandleAnalyticsModels handles /api/analytics/models
func (h *APIHandler) HandleAnalyticsModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := h.logService.GetModelUsage(r.Context(), 30)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, models)
}

// HandleAnalyticsChannels handles /api/analytics/channels
func (h *APIHandler) HandleAnalyticsChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	channels, err := h.logService.GetChannelUsage(r.Context(), 30)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, channels)
}

// Request Log Handlers

// HandleLogs handles /api/logs
func (h *APIHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs, err := h.logService.List(r.Context(), nil)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, logs)
}

// HandleLog handles /api/logs/:id
func (h *APIHandler) HandleLog(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path)
	if id == 0 {
		http.Error(w, "Invalid log ID", http.StatusBadRequest)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	l, err := h.logService.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusNotFound)
		return
	}

	respondJSON(w, l)
}

// Billing Handlers

// HandleBilling handles /api/billing
func (h *APIHandler) HandleBilling(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary, err := h.billingService.GetBillingSummary(r.Context(), nil, nil, "monthly")
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, summary)
}

// HandleBillingPricing handles /api/billing/pricing
func (h *APIHandler) HandleBillingPricing(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		pricing, err := h.billingService.ListModelPricing(r.Context())
		if err != nil {
			respondError(w, err, http.StatusInternalServerError)
			return
		}
		respondJSON(w, pricing)
	case "PUT":
		var p billing.ModelPricing
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			respondError(w, err, http.StatusBadRequest)
			return
		}
		if err := h.billingService.UpdateModelPricing(r.Context(), &p); err != nil {
			respondError(w, err, http.StatusInternalServerError)
			return
		}
		respondJSON(w, p)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper functions

func extractID(path string) int64 {
	parts := splitPath(path)
	if len(parts) < 2 {
		return 0
	}
	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func splitPath(path string) []string {
	var parts []string
	var current string
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func respondJSON(w http.ResponseWriter, data interface{}, statusCode ...int) {
	w.Header().Set("Content-Type", "application/json")
	if len(statusCode) > 0 {
		w.WriteHeader(statusCode[0])
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", slog.Any("error", err))
	}
}

func respondError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
