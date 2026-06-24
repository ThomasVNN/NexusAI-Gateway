package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	MaxConnections    int
	PingInterval      time.Duration
	PongTimeout       time.Duration
	WriteTimeout      time.Duration
	ReadBufferSize    int
	WriteBufferSize   int
	ReadTimeout       time.Duration
	EnableCompression bool
}

// DefaultWebSocketConfig returns default WebSocket configuration
func DefaultWebSocketConfig() WebSocketConfig {
	return WebSocketConfig{
		MaxConnections:    1000,
		PingInterval:      30 * time.Second,
		PongTimeout:       60 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		ReadTimeout:       60 * time.Second,
		EnableCompression: false,
	}
}

// StreamMessage represents a WebSocket streaming message
type StreamMessage struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Delta     string          `json:"delta,omitempty"`
	Completed bool            `json:"completed,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// WebSocketHandler handles WebSocket streaming connections
type WebSocketHandler struct {
	config   WebSocketConfig
	registry *ConnectionRegistry
	upgrader websocket.Upgrader
}

// ConnectionRegistry manages active WebSocket connections
type ConnectionRegistry struct {
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex
	maxConns    int
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID        string
	Conn      *websocket.Conn
	CreatedAt time.Time
	LastPing  time.Time
	UserID    string
	Metadata  map[string]string
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(config WebSocketConfig) *WebSocketHandler {
	return &WebSocketHandler{
		config:   config,
		registry: NewConnectionRegistry(config.MaxConnections),
		upgrader: upgrader,
	}
}

// NewConnectionRegistry creates a new connection registry
func NewConnectionRegistry(maxConnections int) *ConnectionRegistry {
	return &ConnectionRegistry{
		connections: make(map[string]*WebSocketConnection),
		maxConns:    maxConnections,
	}
}

// ServeHTTP handles WebSocket upgrade requests
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check connection limit
	h.registry.mu.RLock()
	if len(h.registry.connections) >= h.registry.maxConns {
		h.registry.mu.RUnlock()
		http.Error(w, "Connection limit reached", http.StatusServiceUnavailable)
		return
	}
	h.registry.mu.RUnlock()

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", slog.Any("error", err))
		return
	}

	// Create connection ID
	connID := generateConnectionID()
	wsc := &WebSocketConnection{
		ID:        connID,
		Conn:      conn,
		CreatedAt: time.Now(),
		LastPing:  time.Now(),
		Metadata:  make(map[string]string),
	}

	// Register connection
	h.registry.Register(wsc)

	// Start connection handler
	go h.handleConnection(wsc)
}

// handleConnection manages a single WebSocket connection
func (h *WebSocketHandler) handleConnection(wsc *WebSocketConnection) {
	conn := wsc.Conn
	connID := wsc.ID

	defer func() {
		h.registry.Unregister(connID)
		conn.Close()
		slog.Info("WebSocket connection closed", slog.String("conn_id", connID))
	}()

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
	conn.SetPongHandler(func(appData string) error {
		wsc.LastPing = time.Now()
		conn.SetReadDeadline(time.Now().Add(h.config.ReadTimeout))
		return nil
	})

	// Start ping goroutine
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(h.config.PingInterval)
		defer ticker.Stop()
		defer close(pingDone)

		for {
			select {
			case <-pingDone:
				return
			case <-time.After(h.config.PingInterval):
				conn.SetWriteDeadline(time.Now().Add(h.config.WriteTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					slog.Warn("WebSocket ping failed", slog.String("conn_id", connID), slog.Any("error", err))
					return
				}
			}
		}
	}()

	slog.Info("WebSocket connection established", slog.String("conn_id", connID))

	// Read loop
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket read error", slog.String("conn_id", connID), slog.Any("error", err))
			}
			break
		}

		// Handle message
		if err := h.handleMessage(wsc, messageType, message); err != nil {
			slog.Error("WebSocket message handling failed", slog.String("conn_id", connID), slog.Any("error", err))
			h.sendError(wsc, err.Error())
		}
	}

	// Wait for ping goroutine to finish
	<-pingDone
}

// handleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(wsc *WebSocketConnection, messageType int, message []byte) error {
	var msg StreamMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		return err
	}

	switch msg.Type {
	case "chat.completion":
		return h.handleChatCompletion(wsc, msg)
	case "ping":
		return h.handlePing(wsc)
	default:
		return h.handleUnknownMessage(wsc, msg)
	}
}

// handleChatCompletion handles chat completion streaming requests
func (h *WebSocketHandler) handleChatCompletion(wsc *WebSocketConnection, msg StreamMessage) error {
	// Process streaming response
	go func() {
		// Simulate streaming response (replace with actual implementation)
		tokens := []string{"Hello", ", ", "how ", "can ", "I ", "help ", "you?"}
		for _, token := range tokens {
			response := StreamMessage{
				Type:  "content.delta",
				ID:    msg.ID,
				Delta: token,
			}

			if err := wsc.Conn.WriteJSON(response); err != nil {
				slog.Error("Failed to send stream message", slog.Any("error", err))
				return
			}

			// Small delay between tokens for demo
			time.Sleep(50 * time.Millisecond)
		}

		// Send completion message
		completion := StreamMessage{
			Type:      "completion",
			ID:        msg.ID,
			Completed: true,
		}
		if err := wsc.Conn.WriteJSON(completion); err != nil {
			slog.Error("Failed to send completion message", slog.Any("error", err))
		}
	}()

	return nil
}

// handlePing handles ping messages
func (h *WebSocketHandler) handlePing(wsc *WebSocketConnection) error {
	response := StreamMessage{
		Type: "pong",
	}
	return wsc.Conn.WriteJSON(response)
}

// handleUnknownMessage handles unknown message types
func (h *WebSocketHandler) handleUnknownMessage(wsc *WebSocketConnection, msg StreamMessage) error {
	return h.sendError(wsc, "Unknown message type: "+msg.Type)
}

// sendError sends an error message to the client
func (h *WebSocketHandler) sendError(wsc *WebSocketConnection, errMsg string) error {
	response := StreamMessage{
		Type:  "error",
		Error: errMsg,
	}
	return wsc.Conn.WriteJSON(response)
}

// Register adds a connection to the registry
func (r *ConnectionRegistry) Register(conn *WebSocketConnection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[conn.ID] = conn
	slog.Info("WebSocket connection registered",
		slog.String("conn_id", conn.ID),
		slog.Int("total_connections", len(r.connections)),
	)
}

// Unregister removes a connection from the registry
func (r *ConnectionRegistry) Unregister(connID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.connections, connID)
	slog.Info("WebSocket connection unregistered",
		slog.String("conn_id", connID),
		slog.Int("total_connections", len(r.connections)),
	)
}

// GetConnection returns a connection by ID
func (r *ConnectionRegistry) GetConnection(connID string) (*WebSocketConnection, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, ok := r.connections[connID]
	return conn, ok
}

// ConnectionCount returns the current number of connections
func (r *ConnectionRegistry) ConnectionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connections)
}

// Broadcast sends a message to all connections
func (r *ConnectionRegistry) Broadcast(msg StreamMessage) []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errors []error
	for _, conn := range r.connections {
		if err := conn.Conn.WriteJSON(msg); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return time.Now().Format("20060102150405.000000000")
}
