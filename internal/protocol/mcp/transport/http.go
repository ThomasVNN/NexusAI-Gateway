package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/protocol/mcp"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
	addr     string
	handler  func(req *mcp.Request) *mcp.Response
	mux      *http.ServeMux
	srv      *http.Server
	mu       sync.RWMutex
	running  bool
	logger   *slog.Logger
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(addr string, handler func(req *mcp.Request) *mcp.Response) *HTTPTransport {
	t := &HTTPTransport{
		addr:    addr,
		handler: handler,
		mux:     http.NewServeMux(),
		logger:  slog.Default(),
	}

	t.setupRoutes()
	return t
}

func (t *HTTPTransport) setupRoutes() {
	t.mux.HandleFunc("/mcp", t.handleMCP)
	t.mux.HandleFunc("/mcp/message", t.handleMessage)
	t.mux.HandleFunc("/mcp/stream", t.handleStream)
	t.mux.HandleFunc("/mcp/sse", t.handleSSE)
}

// Start starts the HTTP transport
func (t *HTTPTransport) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("transport already running")
	}
	t.running = true
	t.mu.Unlock()

	t.srv = &http.Server{
		Addr:         t.addr,
		Handler:      t.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		t.logger.Info("Starting MCP HTTP transport", slog.String("addr", t.addr))
		if err := t.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.logger.Error("HTTP transport error", slog.Any("error", err))
		}
	}()

	return nil
}

// Stop stops the HTTP transport
func (t *HTTPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	if t.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.srv.Shutdown(ctx)
	}

	return nil
}

func (t *HTTPTransport) handleMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		t.handleSSE(w, r)
	case http.MethodPost:
		t.handleMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (t *HTTPTransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.sendError(w, nil, -32700, "Parse error")
		return
	}

	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		t.sendError(w, nil, -32700, "Parse error")
		return
	}

	mcpReq := t.mapToRequest(req)
	resp := t.handler(mcpReq)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (t *HTTPTransport) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "event: endpoint\ndata: /mcp/message\n\n")
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

func (t *HTTPTransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	fmt.Fprintf(w, "event: info\ndata: {\"transport\":\"http\",\"version\":\"1.0\"}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

func (t *HTTPTransport) mapToRequest(msg map[string]interface{}) *mcp.Request {
	req := &mcp.Request{
		JSONRPC: "2.0",
	}

	if v, ok := msg["jsonrpc"].(string); ok {
		req.JSONRPC = v
	}
	if v, ok := msg["id"].(interface{}); ok {
		req.ID = v
	}
	if v, ok := msg["method"].(string); ok {
		req.Method = v
	}
	if v, ok := msg["params"].(interface{}); ok {
		if paramsJSON, err := json.Marshal(v); err == nil {
			req.Params = paramsJSON
		}
	}

	return req
}

func (t *HTTPTransport) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := &mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
