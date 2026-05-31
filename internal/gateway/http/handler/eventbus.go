package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/eventbus"
)

// EventHandler handles event bus API requests
type EventHandler struct {
	bus eventbus.Bus
}

// NewEventHandler creates a new event handler
func NewEventHandler(bus eventbus.Bus) *EventHandler {
	return &EventHandler{
		bus: bus,
	}
}

// PublishRequest represents an event publish request
type PublishRequest struct {
	Type         eventbus.EventType     `json:"type"`
	SourceAgent  string                 `json:"source_agent"`
	TargetAgent  string                 `json:"target_agent,omitempty"`
	Payload      json.RawMessage        `json:"payload"`
	Priority     eventbus.EventPriority `json:"priority,omitempty"`
	CorrelationID string                `json:"correlation_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	TTL          string                 `json:"ttl,omitempty"`
}

// PublishResponse represents an event publish response
type PublishResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Publish handles POST /v1/events - publish an event
func (h *EventHandler) Publish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body")
		return
	}

	var req PublishRequest
	if err := json.Unmarshal(body, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload")
		return
	}

	// Validate required fields
	if req.Type == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "event type is required")
		return
	}
	if req.SourceAgent == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "source_agent is required")
		return
	}
	if req.Payload == nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "payload is required")
		return
	}

	// Create event
	event, err := eventbus.NewEvent(req.Type, req.SourceAgent, req.Payload, req.Priority)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("Failed to create event: %v", err))
		return
	}

	// Apply optional fields
	if req.TargetAgent != "" {
		event.WithTarget(req.TargetAgent)
	}
	if req.CorrelationID != "" {
		event.WithCorrelationID(req.CorrelationID)
	}
	for k, v := range req.Metadata {
		event.WithMetadata(k, v)
	}

	// Publish event
	if err := h.bus.Publish(r.Context(), event); err != nil {
		WriteError(w, http.StatusInternalServerError, "PUBLISH_ERROR", fmt.Sprintf("Failed to publish event: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(PublishResponse{
		Success: true,
		EventID: event.ID,
	})
}

// SubscribeRequest represents an event subscription request
type SubscribeRequest struct {
	ClientID   string                  `json:"client_id"`
	EventTypes []eventbus.EventType    `json:"event_types"`
	TargetAgent string                 `json:"target_agent,omitempty"`
	Filter     *eventbus.Filter       `json:"filter,omitempty"`
}

// SubscribeResponse represents an event subscription response
type SubscribeResponse struct {
	Success        bool                     `json:"success"`
	SubscriptionID string                   `json:"subscription_id,omitempty"`
	Error          string                   `json:"error,omitempty"`
}

// Subscribe handles POST /v1/events/subscribe - create a subscription
func (h *EventHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body")
		return
	}

	var req SubscribeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON payload")
		return
	}

	// Validate required fields
	if req.ClientID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_id is required")
		return
	}
	if len(req.EventTypes) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "at least one event_type is required")
		return
	}

	// Create subscription
	sub := &eventbus.Subscription{
		ClientID:    req.ClientID,
		EventTypes:  req.EventTypes,
		TargetAgent: req.TargetAgent,
		Filter:      req.Filter,
	}

	if err := h.bus.Subscribe(r.Context(), sub); err != nil {
		WriteError(w, http.StatusInternalServerError, "SUBSCRIBE_ERROR", fmt.Sprintf("Failed to subscribe: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SubscribeResponse{
		Success:        true,
		SubscriptionID: sub.ID,
	})
}

// Unsubscribe handles DELETE /v1/events/subscribe/{id} - remove a subscription
func (h *EventHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subscriptionID := extractPathParam(r.URL.Path, "/v1/events/subscribe/")
	if subscriptionID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "subscription_id is required")
		return
	}

	if err := h.bus.Unsubscribe(r.Context(), subscriptionID); err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Subscription not found: %s", subscriptionID))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// GetSubscriptions handles GET /v1/events/subscriptions - list subscriptions
func (h *EventHandler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "client_id query parameter is required")
		return
	}

	subs, err := h.bus.GetSubscriptions(r.Context(), clientID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("Failed to get subscriptions: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"subscriptions": subs,
	})
}

// GetDLQRequest represents a DLQ query request
type GetDLQRequest struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// GetDLQEntries handles GET /v1/events/dlq - list DLQ entries
func (h *EventHandler) GetDLQEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	var dlqStatus eventbus.DLQStatus
	if status != "" {
		dlqStatus = eventbus.DLQStatus(status)
	}

	entries, err := h.bus.GetDLQEntries(r.Context(), dlqStatus, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("Failed to get DLQ entries: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entries": entries,
		"count":   len(entries),
	})
}

// RetryDLQEntry handles POST /v1/events/dlq/{id}/retry - retry a DLQ entry
func (h *EventHandler) RetryDLQEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entryID := extractPathParam(r.URL.Path, "/v1/events/dlq/")
	if entryID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "entry_id is required")
		return
	}

	if err := h.bus.RetryDLQEntry(r.Context(), entryID); err != nil {
		WriteError(w, http.StatusInternalServerError, "RETRY_ERROR", fmt.Sprintf("Failed to retry DLQ entry: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "DLQ entry retry scheduled",
	})
}

// PurgeDLQEntry handles DELETE /v1/events/dlq/{id} - purge a DLQ entry
func (h *EventHandler) PurgeDLQEntry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entryID := extractPathParam(r.URL.Path, "/v1/events/dlq/")
	if entryID == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "entry_id is required")
		return
	}

	if err := h.bus.PurgeDLQEntry(r.Context(), entryID); err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("DLQ entry not found: %s", entryID))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HealthCheck handles GET /v1/events/health - event bus health check
func (h *EventHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.bus.HealthCheck(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "DOWN",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "UP",
	})
}

// extractPathParam extracts a path parameter from a URL path
func extractPathParam(path, prefix string) string {
	if len(path) > len(prefix) {
		return path[len(prefix):]
	}
	return ""
}
