package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/routing"
)

// RoutingHandler handles routing strategy and combo API endpoints
type RoutingHandler struct {
	selector    *routing.StrategySelector
	comboEngine *routing.ComboEngine
	mu          sync.RWMutex
	configs     map[string]*routing.StrategyConfig
	combos      map[string]*ComboPreset
}

type ComboPreset struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Strategies  []string       `json:"strategies"` // Ordered list of strategies to try
	Weights     map[string]int `json:"weights,omitempty"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
}

// NewRoutingHandler creates a new routing handler
func NewRoutingHandler() *RoutingHandler {
	return &RoutingHandler{
		selector:    routing.NewStrategySelector(),
		comboEngine: routing.NewComboEngine(),
		configs:     make(map[string]*routing.StrategyConfig),
		combos:      make(map[string]*ComboPreset),
	}
}

// GetStrategies returns all available routing strategies
func (h *RoutingHandler) GetStrategies(w http.ResponseWriter, r *http.Request) {
	strategies := []map[string]interface{}{
		{"id": "latency", "name": "Latency", "description": "Route to fastest responding provider"},
		{"id": "cost", "name": "Cost", "description": "Route to cheapest provider"},
		{"id": "quality", "name": "Quality", "description": "Route to highest quality model"},
		{"id": "fallback", "name": "Fallback", "description": "Primary + fallback chain"},
		{"id": "round_robin", "name": "Round Robin", "description": "Distribute evenly across providers"},
		{"id": "weighted", "name": "Weighted", "description": "Weighted distribution based on config"},
		{"id": "affinity", "name": "Affinity", "description": "Sticky session to one provider"},
		{"id": "burst", "name": "Burst", "description": "Handle sudden traffic spikes"},
		{"id": "degraded", "name": "Degraded", "description": "Use lower-quality providers when budget low"},
		{"id": "guardian", "name": "Guardian", "description": "Safety-first routing"},
		{"id": "parallel", "name": "Parallel", "description": "Fan out to multiple providers"},
		{"id": "cascade", "name": "Cascade", "description": "Try primary, cascade through fallbacks"},
		{"id": "geo", "name": "Geo", "description": "Geographic routing"},
		{"id": "free_tier", "name": "Free Tier", "description": "Use free tier first"},
		{"id": "composite", "name": "Composite", "description": "Combine multiple strategies"},
	}

	writeJSON(w, map[string]interface{}{
		"strategies": strategies,
		"count":      len(strategies),
	})
}

// GetRoutes returns all configured routes
func (h *RoutingHandler) GetRoutes(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	routes := make([]*routing.StrategyConfig, 0, len(h.configs))
	for _, cfg := range h.configs {
		routes = append(routes, cfg)
	}

	writeJSON(w, map[string]interface{}{
		"routes": routes,
		"count":  len(routes),
	})
}

// CreateRoute creates a new routing strategy configuration
func (h *RoutingHandler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config routing.StrategyConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Generate ID if not provided
	if config.ID == "" {
		config.ID = generateRouteID()
	}

	h.configs[config.ID] = &config

	writeJSONStatus(w, map[string]interface{}{
		"route":  config,
		"status": "created",
	}, http.StatusCreated)
}

// UpdateRoute updates an existing route configuration
func (h *RoutingHandler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config routing.StrategyConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.configs[config.ID]; !exists {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	h.configs[config.ID] = &config

	writeJSON(w, map[string]interface{}{
		"route":  config,
		"status": "updated",
	})
}

// DeleteRoute removes a route configuration
func (h *RoutingHandler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.configs[id]; !exists {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	delete(h.configs, id)

	writeJSON(w, map[string]interface{}{
		"status": "deleted",
		"id":     id,
	})
}

// GetCombos returns all configured combo presets
func (h *RoutingHandler) GetCombos(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	combos := make([]*ComboPreset, 0, len(h.combos))
	for _, combo := range h.combos {
		combos = append(combos, combo)
	}

	writeJSON(w, map[string]interface{}{
		"combos": combos,
		"count":  len(combos),
	})
}

// CreateCombo creates a new combo preset
func (h *RoutingHandler) CreateCombo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var combo ComboPreset
	if err := json.NewDecoder(r.Body).Decode(&combo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if combo.ID == "" {
		combo.ID = generateRouteID()
	}

	h.combos[combo.ID] = &combo

	writeJSONStatus(w, map[string]interface{}{
		"combo":  combo,
		"status": "created",
	}, http.StatusCreated)
}

// UpdateCombo updates an existing combo preset
func (h *RoutingHandler) UpdateCombo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var combo ComboPreset
	if err := json.NewDecoder(r.Body).Decode(&combo); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.combos[combo.ID]; !exists {
		http.Error(w, "Combo not found", http.StatusNotFound)
		return
	}

	h.combos[combo.ID] = &combo

	writeJSON(w, map[string]interface{}{
		"combo":  combo,
		"status": "updated",
	})
}

// DeleteCombo removes a combo preset
func (h *RoutingHandler) DeleteCombo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Combo ID required", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.combos[id]; !exists {
		http.Error(w, "Combo not found", http.StatusNotFound)
		return
	}

	delete(h.combos, id)

	writeJSON(w, map[string]interface{}{
		"status": "deleted",
		"id":     id,
	})
}

// ScoreModels calculates combo scores for all available models
func (h *RoutingHandler) ScoreModels(w http.ResponseWriter, r *http.Request) {
	router := routing.NewModelRouter()
	candidates := router.ListModels()

	req := &routing.RoutingRequest{}
	scores := h.comboEngine.ScoreModels(r.Context(), candidates, req)

	writeJSON(w, map[string]interface{}{
		"scores": scores,
		"count":  len(scores),
	})
}

// GetFallbackTiers returns the 4-tier fallback structure
func (h *RoutingHandler) GetFallbackTiers(w http.ResponseWriter, r *http.Request) {
	router := routing.NewModelRouter()
	candidates := router.ListModels()

	req := &routing.RoutingRequest{}
	scores := h.comboEngine.ScoreModels(r.Context(), candidates, req)
	tiers := h.comboEngine.GetFallbackTiers(scores)

	writeJSON(w, map[string]interface{}{
		"tiers": []map[string]interface{}{
			{"tier": 1, "name": "Best Match", "models": tiers[0]},
			{"tier": 2, "name": "Fallbacks", "models": tiers[1]},
			{"tier": 3, "name": "Free Tier", "models": tiers[2]},
			{"tier": 4, "name": "Graceful Degradation", "models": tiers[3]},
		},
	})
}

// GetComboConfig returns the current combo configuration
func (h *RoutingHandler) GetComboConfig(w http.ResponseWriter, r *http.Request) {
	config := h.comboEngine.Config

	writeJSON(w, map[string]interface{}{
		"config": config,
	})
}

// SetComboConfig updates the combo configuration weights
func (h *RoutingHandler) SetComboConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var config routing.ComboConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.comboEngine.Config = &config

	writeJSON(w, map[string]interface{}{
		"config": config,
		"status": "updated",
	})
}

// writeJSON is a helper to write JSON responses with default 200 status
func writeJSON(w http.ResponseWriter, data interface{}) {
	writeJSONStatus(w, data, http.StatusOK)
}

// writeJSONStatus writes JSON with custom status code
func writeJSONStatus(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// generateRouteID creates a simple unique ID
func generateRouteID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return "route_" + string(b)
}
