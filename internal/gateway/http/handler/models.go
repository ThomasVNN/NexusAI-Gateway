package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/channel"
)

// AIModel represents a supported AI model
type AIModel struct {
	ID                string  `json:"id"`
	Object            string  `json:"object"`
	OwnedBy           string  `json:"owned_by"`
	Provider          string  `json:"provider,omitempty"`
	FullModel         string  `json:"full_model,omitempty"`
	Alias             string  `json:"alias,omitempty"`
	Available         bool    `json:"available"`
	ContextWindow     int     `json:"context_window,omitempty"`
	SupportsVision    bool    `json:"supports_vision,omitempty"`
	SupportsStreaming bool    `json:"supports_streaming,omitempty"`
	InputPrice        float64 `json:"input_price,omitempty"`
	OutputPrice       float64 `json:"output_price,omitempty"`
}

// ModelsResponse represents the OpenAI-compatible models list response
type ModelsResponse struct {
	Object string    `json:"object"`
	Data   []AIModel `json:"data"`
}

// DefaultModels returns the list of default supported models
var DefaultModels = []AIModel{
	// OpenAI Models
	{ID: "gpt-4-turbo", Object: "model", OwnedBy: "openai", Provider: "openai", ContextWindow: 128000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.01, OutputPrice: 0.03},
	{ID: "gpt-4", Object: "model", OwnedBy: "openai", Provider: "openai", ContextWindow: 8192, SupportsStreaming: true, InputPrice: 0.03, OutputPrice: 0.06},
	{ID: "gpt-4o", Object: "model", OwnedBy: "openai", Provider: "openai", ContextWindow: 128000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.005, OutputPrice: 0.015},
	{ID: "gpt-4o-mini", Object: "model", OwnedBy: "openai", Provider: "openai", ContextWindow: 128000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.00015, OutputPrice: 0.0006},
	{ID: "gpt-3.5-turbo", Object: "model", OwnedBy: "openai", Provider: "openai", ContextWindow: 16385, SupportsStreaming: true, InputPrice: 0.0005, OutputPrice: 0.0015},

	// Anthropic Models
	{ID: "claude-3-5-sonnet-latest", Object: "model", OwnedBy: "anthropic", Provider: "anthropic", ContextWindow: 200000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.003, OutputPrice: 0.015},
	{ID: "claude-3-opus-latest", Object: "model", OwnedBy: "anthropic", Provider: "anthropic", ContextWindow: 200000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.015, OutputPrice: 0.075},
	{ID: "claude-3-sonnet-latest", Object: "model", OwnedBy: "anthropic", Provider: "anthropic", ContextWindow: 200000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.003, OutputPrice: 0.015},
	{ID: "claude-3-haiku-latest", Object: "model", OwnedBy: "anthropic", Provider: "anthropic", ContextWindow: 200000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.00025, OutputPrice: 0.00125},

	// Google Models
	{ID: "gemini-1.5-pro", Object: "model", OwnedBy: "google", Provider: "google", ContextWindow: 1000000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.00125, OutputPrice: 0.005},
	{ID: "gemini-1.5-flash", Object: "model", OwnedBy: "google", Provider: "google", ContextWindow: 1000000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.000075, OutputPrice: 0.0003},
	{ID: "gemini-2.0-flash-exp", Object: "model", OwnedBy: "google", Provider: "google", ContextWindow: 1000000, SupportsVision: true, SupportsStreaming: true, InputPrice: 0.000075, OutputPrice: 0.0003},

	// Mistral Models
	{ID: "mistral-large-latest", Object: "model", OwnedBy: "mistral", Provider: "mistral", ContextWindow: 128000, SupportsStreaming: true, InputPrice: 0.002, OutputPrice: 0.006},
	{ID: "mistral-small-latest", Object: "model", OwnedBy: "mistral", Provider: "mistral", ContextWindow: 128000, SupportsStreaming: true, InputPrice: 0.0002, OutputPrice: 0.0006},

	// Cohere Models
	{ID: "command-r-plus", Object: "model", OwnedBy: "cohere", Provider: "cohere", ContextWindow: 128000, SupportsStreaming: true, InputPrice: 0.003, OutputPrice: 0.015},
	{ID: "command-r", Object: "model", OwnedBy: "cohere", Provider: "cohere", ContextWindow: 128000, SupportsStreaming: true, InputPrice: 0.0005, OutputPrice: 0.0015},
}

// ModelHandler handles model-related API endpoints
type ModelHandler struct {
	channelService *channel.Service
}

// NewModelHandler creates a new model handler
func NewModelHandler(channelService *channel.Service) *ModelHandler {
	return &ModelHandler{channelService: channelService}
}

// ServeHTTP handles GET /v1/models
func (h *ModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var availableProviders map[string]bool
	if h.channelService != nil {
		channels, err := h.channelService.List(r.Context(), nil)
		if err == nil {
			availableProviders = make(map[string]bool)
			for _, ch := range channels {
				if ch.IsActive {
					availableProviders[string(ch.Type)] = true
				}
			}
		}
	}

	models := make([]AIModel, 0, len(DefaultModels))
	for _, m := range DefaultModels {
		model := m
		model.FullModel = m.Provider + "/" + m.ID
		if availableProviders != nil {
			model.Available = availableProviders[model.Provider]
		} else {
			model.Available = true
		}
		models = append(models, model)
	}

	resp := ModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleModelByID handles GET /v1/models/{id}
func (h *ModelHandler) HandleModelByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modelID := extractModelID(r.URL.Path)

	for _, m := range DefaultModels {
		if m.ID == modelID || m.FullModel == modelID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(m)
			return
		}
	}

	http.Error(w, "Model not found", http.StatusNotFound)
}

func extractModelID(path string) string {
	parts := splitPath(path)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
