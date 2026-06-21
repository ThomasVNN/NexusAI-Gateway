package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/provider"
)

// NewProviderHandler creates a handler for provider endpoints
func NewProviderHandler(svc *provider.Service, healthChecker *provider.HealthChecker, selector *provider.ProviderSelector) *ProviderHandler {
	return &ProviderHandler{
		svc:              svc,
		healthChecker:    healthChecker,
		providerSelector: selector,
	}
}

// ProviderHandler handles provider CRUD endpoints
type ProviderHandler struct {
	svc              *provider.Service
	healthChecker    *provider.HealthChecker
	providerSelector *provider.ProviderSelector
}

// HandleProviders handles /v1/providers
func (h *ProviderHandler) HandleProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listProviders(w, r)
	case "POST":
		h.createProvider(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleProvider handles /v1/providers/{id}
func (h *ProviderHandler) HandleProvider(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	if id == "" {
		http.Error(w, `{"error": "provider id is required"}`, http.StatusBadRequest)
		return
	}

	// Handle /v1/providers/health specially
	if id == "health" {
		h.HandleAllProviderHealth(w, r)
		return
	}

	switch r.Method {
	case "GET":
		h.getProvider(w, r, id)
	case "PUT":
		h.updateProvider(w, r, id)
	case "DELETE":
		h.deleteProvider(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAllProviderHealth handles /v1/providers/health
func (h *ProviderHandler) HandleAllProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.healthChecker == nil {
		http.Error(w, `{"error": "health checker not available"}`, http.StatusServiceUnavailable)
		return
	}

	status := h.healthChecker.GetAllHealthStatus()
	respondJSON(w, status)
}

// HandleProviderHealth handles /v1/providers/{id}/health
func (h *ProviderHandler) HandleProviderHealth(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	if id == "" {
		http.Error(w, `{"error": "provider id is required"}`, http.StatusBadRequest)
		return
	}

	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == "GET" {
		h.getProviderHealth(w, r, id)
	} else {
		h.checkProviderHealth(w, r, id)
	}
}

// HandleProviderSelect handles /v1/providers/select
func (h *ProviderHandler) HandleProviderSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.providerSelector == nil || h.svc == nil {
		http.Error(w, `{"error": "selector not available"}`, http.StatusServiceUnavailable)
		return
	}

	providerType := r.URL.Query().Get("type")
	p, err := h.svc.SelectProvider(r.Context(), provider.ProviderType(providerType))
	if err != nil {
		respondError(w, err, http.StatusServiceUnavailable)
		return
	}

	respondJSON(w, p.ToResponse())
}

func (h *ProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	filter := &provider.ProviderFilter{}

	if providerType := r.URL.Query().Get("type"); providerType != "" {
		filter.Type = provider.ProviderType(providerType)
	}
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		enabledBool := enabled == "true"
		filter.Enabled = &enabledBool
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	includeHealth := r.URL.Query().Get("include_health") == "true"

	var result interface{}

	if includeHealth {
		providers, listErr := h.svc.ListWithHealth(r.Context(), filter)
		if listErr != nil {
			respondError(w, listErr, http.StatusInternalServerError)
			return
		}
		result = providers
	} else {
		providers, listErr := h.svc.List(r.Context(), filter)
		if listErr != nil {
			respondError(w, listErr, http.StatusInternalServerError)
			return
		}
		response := make([]*provider.ProviderResponse, len(providers))
		for i, p := range providers {
			response[i] = p.ToResponse()
		}
		result = response
	}

	respondJSON(w, result)
}

func (h *ProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var p provider.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.svc.Create(r.Context(), &p); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	slog.Info("Provider created", slog.String("id", p.ID), slog.String("name", p.Name))
	respondJSON(w, p.ToResponse(), http.StatusCreated)
}

func (h *ProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id string) {
	includeHealth := r.URL.Query().Get("include_health") == "true"

	if includeHealth {
		pwh, err := h.svc.GetWithHealth(r.Context(), id)
		if err != nil {
			respondError(w, err, http.StatusNotFound)
			return
		}
		respondJSON(w, pwh)
	} else {
		p, err := h.svc.GetByID(r.Context(), id)
		if err != nil {
			respondError(w, err, http.StatusNotFound)
			return
		}
		respondJSON(w, p.ToResponse())
	}
}

func (h *ProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id string) {
	var p provider.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	p.ID = id
	if err := h.svc.Update(r.Context(), &p); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	slog.Info("Provider updated", slog.String("id", p.ID), slog.String("name", p.Name))
	respondJSON(w, p.ToResponse())
}

func (h *ProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	slog.Info("Provider deleted", slog.String("id", id))
	respondJSON(w, map[string]bool{"deleted": true})
}

func (h *ProviderHandler) getProviderHealth(w http.ResponseWriter, r *http.Request, id string) {
	health, err := h.svc.GetWithHealth(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusNotFound)
		return
	}

	respondJSON(w, health.Health)
}

func (h *ProviderHandler) checkProviderHealth(w http.ResponseWriter, r *http.Request, id string) {
	health, err := h.svc.CheckHealth(r.Context(), id)
	if err != nil {
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, health)
}

// extractProviderID extracts the provider ID from the URL path
func extractProviderID(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "v1" && parts[1] == "providers" {
		return parts[2]
	}
	return ""
}
