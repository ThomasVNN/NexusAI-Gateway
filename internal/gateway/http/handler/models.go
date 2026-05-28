package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
)

type ModelHandler struct {
	db *postgres.DB
}

func NewModelHandler(db *postgres.DB) *ModelHandler {
	return &ModelHandler{db: db}
}

type ModelItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelItem `json:"data"`
}

func (h *ModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := ModelsResponse{
		Object: "list",
		Data: []ModelItem{
			{ID: "gpt-4", Object: "model", OwnedBy: "openai"},
			{ID: "claude-3-5-sonnet", Object: "model", OwnedBy: "anthropic"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
