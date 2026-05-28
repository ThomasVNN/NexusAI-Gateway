package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
)

type Handler struct {
	registry *Registry
}

func NewHandler(pe *privacy.Engine) *Handler {
	return &Handler{
		registry: NewRegistry(pe),
	}
}

type JSONRPCRequest struct {
	JSONRPC   string          `json:"jsonrpc"`
	ID        interface{}     `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.handleStream(w, r)
		return
	}
	if r.Method == http.MethodPost {
		h.handleMessage(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send endpoint event notifying client of standard JSON-RPC HTTP POST ingress
	_, _ = fmt.Fprint(w, "event: endpoint\ndata: /api/mcp/message\n\n")
	flusher.Flush()

	// Keep stream alive with heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprint(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) handleMessage(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.sendErrorResponse(w, nil, -32700, "Parse error")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.Method {
	case "tools/list":
		tools := h.registry.ListTools()
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": tools,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)

	case "tools/call":
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			h.sendErrorResponse(w, req.ID, -32602, "Invalid params")
			return
		}

		res, err := h.registry.CallTool(r.Context(), callParams.Name, callParams.Arguments)
		if err != nil {
			h.sendErrorResponse(w, req.ID, -32603, fmt.Sprintf("Tool execution failed: %v", err))
			return
		}

		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("%v", res),
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)

	default:
		h.sendErrorResponse(w, req.ID, -32601, "Method not found")
	}
}

func (h *Handler) sendErrorResponse(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}
