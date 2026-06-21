package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/provider"
)

// ProviderHandler handles provider health and rotation endpoints
type ProviderHandler struct {
	healthChecker    *provider.HealthChecker
	providerSelector *provider.ProviderSelector
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(healthChecker *provider.HealthChecker, selector *provider.ProviderSelector) *ProviderHandler {
	return &ProviderHandler{
		healthChecker:    healthChecker,
		providerSelector: selector,
	}
}

// HandleProviders handles /v1/providers endpoints
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

// HandleProvider handles /v1/providers/:id endpoints
func (h *ProviderHandler) HandleProvider(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
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

// HandleProviderHealth handles /v1/providers/:id/health endpoints
func (h *ProviderHandler) HandleProviderHealth(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
		return
	}

	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.getProviderHealth(w, r, id)
}

// HandleProviderSelect handles /v1/providers/select endpoint
func (h *ProviderHandler) HandleProviderSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.selectProvider(w, r)
}

// HandleProviderMetrics handles /v1/providers/:id/metrics endpoints
func (h *ProviderHandler) HandleProviderMetrics(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid provider ID", http.StatusBadRequest)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.getProviderMetrics(w, r, id)
}

// HandleAllProviderHealth handles /v1/providers/health endpoint
func (h *ProviderHandler) HandleAllProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.getAllProviderHealth(w, r)
}

func (h *ProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	if h.providerSelector == nil {
		respondError(w, fmt.Errorf("provider service not initialized"), http.StatusNotImplemented)
		return
	}

	statuses := h.providerSelector.GetProviderStatus()
	respondJSON(w, map[string]interface{}{
		"providers": statuses,
		"count":     len(statuses),
	})
}

func (h *ProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id string) {
	if h.providerSelector == nil {
		respondError(w, fmt.Errorf("provider service not initialized"), http.StatusNotImplemented)
		return
	}

	statuses := h.providerSelector.GetProviderStatus()
	for _, s := range statuses {
		if s.Provider != nil && s.Provider.ID == id {
			respondJSON(w, s)
			return
		}
	}

	http.Error(w, "Provider not found", http.StatusNotFound)
}

func (h *ProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Endpoint string `json:"endpoint"`
		APIKey   string `json:"api_key"`
		Priority int    `json:"priority"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	// TODO: Call provider repository to create provider
	respondError(w, fmt.Errorf("provider creation not implemented - depends on NX-204"), http.StatusNotImplemented)
}

func (h *ProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Endpoint string `json:"endpoint"`
		APIKey   string `json:"api_key"`
		Priority int    `json:"priority"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, err, http.StatusBadRequest)
		return
	}

	// TODO: Call provider repository to update provider
	respondError(w, fmt.Errorf("provider update not implemented - depends on NX-204"), http.StatusNotImplemented)
}

func (h *ProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id string) {
	// TODO: Call provider repository to delete provider
	respondError(w, fmt.Errorf("provider deletion not implemented - depends on NX-204"), http.StatusNotImplemented)
}

func (h *ProviderHandler) getProviderHealth(w http.ResponseWriter, r *http.Request, id string) {
	if h.healthChecker == nil {
		respondError(w, fmt.Errorf("health checker not initialized"), http.StatusNotImplemented)
		return
	}

	status, exists := h.healthChecker.GetHealthStatus(id)
	if !exists {
		http.Error(w, "Provider health status not found", http.StatusNotFound)
		return
	}

	respondJSON(w, status)
}

func (h *ProviderHandler) getProviderMetrics(w http.ResponseWriter, r *http.Request, id string) {
	if h.healthChecker == nil {
		respondError(w, fmt.Errorf("health checker not initialized"), http.StatusNotImplemented)
		return
	}

	metrics, exists := h.healthChecker.GetMetrics(id)
	if !exists {
		http.Error(w, "Provider metrics not found", http.StatusNotFound)
		return
	}

	respondJSON(w, metrics)
}

func (h *ProviderHandler) selectProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerSelector == nil {
		respondError(w, fmt.Errorf("provider selector not initialized"), http.StatusNotImplemented)
		return
	}

	// Parse optional strategy from request body
	var req struct {
		Strategy string `json:"strategy"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	// Apply strategy if specified
	if req.Strategy != "" {
		slog.Debug("Provider select requested with strategy", slog.String("strategy", req.Strategy))
	}

	selected, err := h.providerSelector.SelectProvider(r.Context())
	if err != nil {
		if strings.Contains(err.Error(), "no healthy providers") {
			http.Error(w, "No healthy providers available", http.StatusServiceUnavailable)
			return
		}
		respondError(w, err, http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]interface{}{
		"selected_provider": selected,
	})
}

func (h *ProviderHandler) getAllProviderHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthChecker == nil {
		respondError(w, fmt.Errorf("health checker not initialized"), http.StatusNotImplemented)
		return
	}

	statuses := h.healthChecker.GetAllHealthStatus()
	respondJSON(w, map[string]interface{}{
		"health_status": statuses,
		"count":         len(statuses),
	})
}

// extractProviderID extracts the provider ID from the URL path
func extractProviderID(path string) string {
	// Expected format: /v1/providers/{id} or /v1/providers/{id}/health
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}
