package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/proxy"
)

type ProxyHandler struct {
	engine *proxy.ProxyEngine
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		engine: proxy.GetProxyEngine(),
	}
}

func (h *ProxyHandler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ProxyHandler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}

// ListProxies handles GET /api/v1/proxies
func (h *ProxyHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	proxies := h.engine.ListProxies()
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxies": proxies,
		"count":   len(proxies),
	})
}

// CreateProxy handles POST /api/v1/proxies
func (h *ProxyHandler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var proxyConfig proxy.ProxyConfig
	if err := json.Unmarshal(body, &proxyConfig); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.engine.CreateProxy(&proxyConfig); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"proxy":  proxyConfig,
		"status": "created",
	})
}

// GetProxy handles GET /api/v1/proxies/{id}
func (h *ProxyHandler) GetProxy(w http.ResponseWriter, r *http.Request) {
	id := extractProxyIDFromPath(r.URL.Path)

	proxyConfig := h.engine.GetProxy(id)
	if proxyConfig == nil {
		h.errorResponse(w, http.StatusNotFound, "Proxy not found")
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxy": proxyConfig,
	})
}

// UpdateProxy handles POST /api/v1/proxies/update
func (h *ProxyHandler) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var updates proxy.ProxyConfig
	if err := json.Unmarshal(body, &updates); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.engine.UpdateProxy(id, &updates); err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	proxyConfig := h.engine.GetProxy(id)
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxy":  proxyConfig,
		"status": "updated",
	})
}

// DeleteProxy handles DELETE /api/v1/proxies/delete
func (h *ProxyHandler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if err := h.engine.DeleteProxy(id); err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
		"id":     id,
	})
}

// EnableProxy handles POST /api/v1/proxies/enable
func (h *ProxyHandler) EnableProxy(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = extractProxyIDFromPath(r.URL.Path)
	}

	if err := h.engine.EnableProxy(id); err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	proxyConfig := h.engine.GetProxy(id)
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxy":  proxyConfig,
		"status": "enabled",
	})
}

// DisableProxy handles POST /api/v1/proxies/disable
func (h *ProxyHandler) DisableProxy(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = extractProxyIDFromPath(r.URL.Path)
	}

	if err := h.engine.DisableProxy(id); err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	proxyConfig := h.engine.GetProxy(id)
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxy":  proxyConfig,
		"status": "disabled",
	})
}

// TestProxy handles POST /api/v1/proxies/test
func (h *ProxyHandler) TestProxy(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		h.errorResponse(w, http.StatusBadRequest, "Proxy ID required (id query param)")
		return
	}

	proxyConfig, err := h.engine.TestProxy(id)
	if err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"proxy":  proxyConfig,
		"status": "tested",
	})
}

// GetPoolHealth handles GET /api/v1/proxies/health
func (h *ProxyHandler) GetPoolHealth(w http.ResponseWriter, r *http.Request) {
	health := h.engine.GetPoolHealth()
	h.jsonResponse(w, http.StatusOK, health)
}

// RotateProxy handles POST /api/v1/proxies/rotate
func (h *ProxyHandler) RotateProxy(w http.ResponseWriter, r *http.Request) {
	if err := h.engine.RotateProxy(); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "rotated",
		"timestamp": "now",
	})
}

// ListChains handles GET /api/v1/proxy-chains
func (h *ProxyHandler) ListChains(w http.ResponseWriter, r *http.Request) {
	chains := h.engine.ListChains()
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"chains": chains,
		"count":  len(chains),
	})
}

// CreateChain handles POST /api/v1/proxy-chains
func (h *ProxyHandler) CreateChain(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var chain proxy.ProxyChain
	if err := json.Unmarshal(body, &chain); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.engine.CreateChain(&chain); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"chain":  chain,
		"status": "created",
	})
}

// DeleteChain handles DELETE /api/v1/proxy-chains/:id
func (h *ProxyHandler) DeleteChain(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/proxy-chains/delete/")

	if err := h.engine.DeleteChain(id); err != nil {
		h.errorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
		"id":     id,
	})
}

// GetChain handles GET /api/v1/proxy-chains/:id
func (h *ProxyHandler) GetChain(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/proxy-chains/")

	chains := h.engine.ListChains()
	for _, chain := range chains {
		if chain.ID == id {
			h.jsonResponse(w, http.StatusOK, map[string]interface{}{
				"chain": chain,
			})
			return
		}
	}

	h.errorResponse(w, http.StatusNotFound, "Chain not found")
}

// GetTLSConfig handles GET /api/v1/proxies/tls-stealth
func (h *ProxyHandler) GetTLSConfig(w http.ResponseWriter, r *http.Request) {
	config := h.engine.GetTLSConfig()
	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"config": config,
	})
}

// SetTLSConfig handles POST /api/v1/proxies/tls-stealth
func (h *ProxyHandler) SetTLSConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var config proxy.TLSStealthConfig
	if err := json.Unmarshal(body, &config); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.engine.SetTLSConfig(&config); err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"config": config,
		"status": "updated",
	})
}

func extractProxyID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/proxies/"), "/")
	if len(parts) > 0 && parts[0] != "" && !strings.Contains(parts[0], "?") {
		return strings.Split(parts[0], "?")[0]
	}
	return ""
}

func extractProxyIDFromPath(path string) string {
	// Handle /api/v1/proxies/{id}
	if strings.HasPrefix(path, "/api/v1/proxies/") {
		id := strings.TrimPrefix(path, "/api/v1/proxies/")
		if !strings.Contains(id, "/") {
			return id
		}
	}
	// Handle /api/v1/proxies/{id}/action
	if strings.HasPrefix(path, "/api/v1/proxies/") {
		rest := strings.TrimPrefix(path, "/api/v1/proxies/")
		parts := strings.Split(rest, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

func extractProxyIDFromQuery(query string) string {
	parts := strings.Split(query, "=")
	if len(parts) == 2 && parts[0] == "id" {
		return parts[1]
	}
	return ""
}
