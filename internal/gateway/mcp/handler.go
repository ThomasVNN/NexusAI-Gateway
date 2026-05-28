package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/pkg/mcp"
)

type Handler struct {
	registry *Registry
}

func NewHandler() *Handler {
	return &Handler{
		registry: NewRegistry(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Negotiate Server-Sent Events for Model Context Protocol communication
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Notify connection established and send endpoint details
	initMsg := mcp.JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "mcp/initialized",
		Params:  map[string]string{"session": fmt.Sprintf("sess_%d", time.Now().Unix())},
	}

	data, _ := json.Marshal(initMsg)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", string(data))
	flusher.Flush()
}
