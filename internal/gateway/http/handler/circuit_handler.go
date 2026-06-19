package handler

import (
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/circuitbreaker"
)

// CircuitBreakerHandler handles circuit breaker API requests
type CircuitBreakerHandler struct {
	manager *circuitbreaker.Manager
}

// NewCircuitBreakerHandler creates a new circuit breaker handler
func NewCircuitBreakerHandler() *CircuitBreakerHandler {
	return &CircuitBreakerHandler{
		manager: circuitbreaker.GetManager(),
	}
}

// ListBreakers handles GET /api/v1/circuit-breakers
func (h *CircuitBreakerHandler) ListBreakers(w http.ResponseWriter, r *http.Request) {
	stats := h.manager.GetAllStats()
	respondJSON(w, stats, 200)
}

// GetBreaker handles GET /api/v1/circuit-breakers/{provider}
func (h *CircuitBreakerHandler) GetBreaker(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	cb := h.manager.Get(provider)
	if cb == nil {
		respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
		return
	}

	respondJSON(w, cb.GetStats(), 200)
}

// GetBreakerState handles GET /api/v1/circuit-breakers/{provider}/state
func (h *CircuitBreakerHandler) GetBreakerState(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	cb := h.manager.Get(provider)
	if cb == nil {
		respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
		return
	}

	respondJSON(w, map[string]interface{}{
		"provider": provider,
		"state":    cb.GetState().String(),
	}, 200)
}

// ResetBreaker handles POST /api/v1/circuit-breakers/{provider}/reset
func (h *CircuitBreakerHandler) ResetBreaker(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	if h.manager.Reset(provider) {
		respondJSON(w, map[string]string{
			"status":   "reset",
			"provider": provider,
		}, 200)
		return
	}

	respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
}

// ResetAllBreakers handles POST /api/v1/circuit-breakers/reset-all
func (h *CircuitBreakerHandler) ResetAllBreakers(w http.ResponseWriter, r *http.Request) {
	h.manager.ResetAll()
	respondJSON(w, map[string]string{"status": "all_circuit_breakers_reset"}, 200)
}

// CheckBreaker handles POST /api/v1/circuit-breakers/{provider}/check
func (h *CircuitBreakerHandler) CheckBreaker(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	// Get or create the circuit breaker
	cb := h.manager.GetOrCreate(provider)

	allowed := cb.Allow()
	state := cb.GetState().String()

	respondJSON(w, map[string]interface{}{
		"provider": provider,
		"allowed":  allowed,
		"state":    state,
	}, 200)
}

// RecordSuccess handles POST /api/v1/circuit-breakers/{provider}/success
func (h *CircuitBreakerHandler) RecordSuccess(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	cb := h.manager.Get(provider)
	if cb == nil {
		respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
		return
	}

	cb.RecordSuccess()
	respondJSON(w, map[string]string{"status": "success_recorded"}, 200)
}

// RecordFailure handles POST /api/v1/circuit-breakers/{provider}/failure
func (h *CircuitBreakerHandler) RecordFailure(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	cb := h.manager.Get(provider)
	if cb == nil {
		respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
		return
	}

	cb.RecordFailure()
	respondJSON(w, map[string]string{"status": "failure_recorded"}, 200)
}

// RecordTimeout handles POST /api/v1/circuit-breakers/{provider}/timeout
func (h *CircuitBreakerHandler) RecordTimeout(w http.ResponseWriter, r *http.Request) {
	provider := extractCircuitProviderID(r.URL.Path)

	cb := h.manager.Get(provider)
	if cb == nil {
		respondJSON(w, map[string]string{"error": "circuit breaker not found"}, 404)
		return
	}

	cb.RecordTimeout()
	respondJSON(w, map[string]string{"status": "timeout_recorded"}, 200)
}

// extractCircuitProviderID extracts provider ID from circuit breaker path
func extractCircuitProviderID(path string) string {
	// Remove /api/v1/circuit-breakers/ prefix
	path = trimPathPrefix(path, "/api/v1/circuit-breakers/")
	// Remove /state, /reset, /check, /success, /failure, /timeout suffixes
	path = trimPathSuffix(path, "/state")
	path = trimPathSuffix(path, "/reset")
	path = trimPathSuffix(path, "/check")
	path = trimPathSuffix(path, "/success")
	path = trimPathSuffix(path, "/failure")
	path = trimPathSuffix(path, "/timeout")
	return path
}

func trimPathPrefix(path, prefix string) string {
	if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
		return path[len(prefix):]
	}
	return path
}

func trimPathSuffix(path, suffix string) string {
	if len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix {
		return path[:len(path)-len(suffix)]
	}
	return path
}
