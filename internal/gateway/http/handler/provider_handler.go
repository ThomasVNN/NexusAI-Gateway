package handler

import (
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/provider"
)

// ProviderHandler handles provider API requests
type ProviderHandler struct {
	engine *provider.ProviderEngine
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler() *ProviderHandler {
	return &ProviderHandler{
		engine: provider.GetProviderEngine(),
	}
}

// ListProviders handles GET /api/v1/providers
func (h *ProviderHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.engine.ListProviders()
	respondJSON(w, providers, 200)
}

// GetProvider handles GET /api/v1/providers/{id}
func (h *ProviderHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	prov := h.engine.GetProvider(id)
	if prov == nil {
		respondJSON(w, map[string]string{"error": "Provider not found"}, 404)
		return
	}
	respondJSON(w, prov, 200)
}

// ListModels handles GET /api/v1/models
func (h *ProviderHandler) ListModels(w http.ResponseWriter, r *http.Request) {
	models := h.engine.ListModels()
	respondJSON(w, map[string]interface{}{
		"models": models,
		"total":  len(models),
	}, 200)
}

// GetModel handles GET /api/v1/models/{id}
func (h *ProviderHandler) GetModel(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	model := h.engine.GetModel(id)
	if model == nil {
		respondJSON(w, map[string]string{"error": "Model not found"}, 404)
		return
	}
	respondJSON(w, model, 200)
}

// EnableProvider handles POST /api/v1/providers/{id}/enable
func (h *ProviderHandler) EnableProvider(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	err := h.engine.EnableProvider(id)
	if err != nil {
		respondJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	respondJSON(w, map[string]string{"status": "enabled"}, 200)
}

// DisableProvider handles POST /api/v1/providers/{id}/disable
func (h *ProviderHandler) DisableProvider(w http.ResponseWriter, r *http.Request) {
	id := extractProviderID(r.URL.Path)
	err := h.engine.DisableProvider(id)
	if err != nil {
		respondJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	respondJSON(w, map[string]string{"status": "disabled"}, 200)
}

// GetProvidersByCategory handles GET /api/v1/providers/category/{category}
func (h *ProviderHandler) GetProvidersByCategory(w http.ResponseWriter, r *http.Request) {
	category := extractProviderID(r.URL.Path)
	providers := h.engine.GetProvidersByCategory(category)
	respondJSON(w, providers, 200)
}

// GetFreeProviders handles GET /api/v1/providers/free
func (h *ProviderHandler) GetFreeProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.engine.GetFreeProviders()
	respondJSON(w, providers, 200)
}

// GetStats handles GET /api/v1/providers/stats
func (h *ProviderHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.engine.GetStats()
	respondJSON(w, stats, 200)
}

// extractProviderID extracts provider/model ID from path
func extractProviderID(path string) string {
	// Remove /api/v1/providers/ or /api/v1/models/ prefix
	path = strings.TrimPrefix(path, "/api/v1/providers/")
	path = strings.TrimPrefix(path, "/api/v1/models/")
	// Remove any trailing /enable, /disable, etc.
	path = strings.Split(path, "/")[0]
	return path
}
