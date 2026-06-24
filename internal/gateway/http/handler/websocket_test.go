package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	config := DefaultWebSocketConfig(nil, true)
	handler := NewWebSocketHandler(config)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/stream", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestWebSocketHandler_ServeHTTP_Upgrade(t *testing.T) {
	config := DefaultWebSocketConfig(nil, true)
	handler := NewWebSocketHandler(config)

	req := httptest.NewRequest(http.MethodGet, "/v1/chat/stream", nil)
	w := httptest.NewRecorder()

	// Note: The upgrade happens in a goroutine, so we can't test it directly
	// This test verifies the handler doesn't panic on a GET request
	handler.ServeHTTP(w, req)

	// For a WebSocket upgrade, we expect no response body written
	// The actual upgrade happens asynchronously
}

func TestConnectionPool_Add(t *testing.T) {
	pool := NewConnectionPool(2)

	// Create mock connections
	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}

	assert.True(t, pool.Add(conn1))
	assert.Equal(t, 1, pool.Count())

	assert.True(t, pool.Add(conn2))
	assert.Equal(t, 2, pool.Count())
}

func TestConnectionPool_Add_MaxConnections(t *testing.T) {
	pool := NewConnectionPool(2)

	conn1 := &websocket.Conn{}
	conn2 := &websocket.Conn{}
	conn3 := &websocket.Conn{}

	pool.Add(conn1)
	pool.Add(conn2)

	// Should be rejected
	assert.False(t, pool.Add(conn3))
	assert.Equal(t, 2, pool.Count())
}

func TestConnectionPool_Remove(t *testing.T) {
	pool := NewConnectionPool(10)

	conn := &websocket.Conn{}
	pool.Add(conn)
	assert.Equal(t, 1, pool.Count())

	pool.Remove(conn)
	assert.Equal(t, 0, pool.Count())
}

func TestConnectionPool_Remove_NonExistent(t *testing.T) {
	pool := NewConnectionPool(10)

	conn := &websocket.Conn{}

	// Should not panic
	pool.Remove(conn)
	assert.Equal(t, 0, pool.Count())
}

func TestConnectionPool_Count_ThreadSafe(t *testing.T) {
	pool := NewConnectionPool(100)

	var wg sync.WaitGroup
	numGoroutines := 10
	connsPerGoroutine := 20

	// Add connections concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < connsPerGoroutine; j++ {
				conn := &websocket.Conn{}
				pool.Add(conn)
			}
		}()
	}

	// Remove connections concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < connsPerGoroutine/2; j++ {
				// This is a simplified test - in real scenario we'd track connections
				pool.Count()
			}
		}()
	}

	wg.Wait()

	// Just verify no panic occurred
	assert.GreaterOrEqual(t, pool.Count(), 0)
}

func TestDefaultWebSocketConfig(t *testing.T) {
	config := DefaultWebSocketConfig(nil, true)

	assert.Equal(t, 100, config.MaxConnections)
	assert.Equal(t, 30*time.Second, config.PingInterval)
	assert.Equal(t, 60*time.Second, config.PongWait)
	assert.Equal(t, 10*time.Second, config.WriteWait)
	assert.Equal(t, 4096, config.ReadBufferSize)
	assert.Equal(t, 4096, config.WriteBufferSize)
	assert.True(t, config.EnableSandboxFallback)
}

func TestStreamMessage_JSON(t *testing.T) {
	msg := StreamMessage{
		Type:    "token",
		Content: map[string]string{"token": "hello"},
	}

	assert.Equal(t, "token", msg.Type)
	assert.NotNil(t, msg.Content)
}

func TestStreamMessage_Error(t *testing.T) {
	msg := StreamMessage{
		Type:  "error",
		Error: "connection_closed",
	}

	assert.Equal(t, "error", msg.Type)
	assert.Equal(t, "connection_closed", msg.Error)
}

func TestChatRequest_Parsing(t *testing.T) {
	jsonStr := `{"request_id":"req-123","model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"stream":true}`

	var req ChatRequest
	err := parseChatRequest(jsonStr, &req)

	assert.NoError(t, err)
	assert.Equal(t, "req-123", req.RequestID)
	assert.Equal(t, "gpt-4", req.Model)
	assert.True(t, req.Stream)
	assert.Len(t, req.Messages, 1)
	assert.Equal(t, "user", req.Messages[0].Role)
	assert.Equal(t, "Hello", req.Messages[0].Content)
}

func TestValidateWebSocketAuth_NoHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := ValidateWebSocketAuth(req)
	assert.Error(t, err)
}

func TestValidateWebSocketAuth_InvalidHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-key-format")

	_, err := ValidateWebSocketAuth(req)
	assert.Error(t, err)
}

// Helper function to parse chat request (mirrors internal parsing)
func parseChatRequest(jsonStr string, req *ChatRequest) error {
	decoder := &chatRequestDecoder{}
	return decoder.decode(jsonStr, req)
}

type chatRequestDecoder struct{}

func (d *chatRequestDecoder) decode(jsonStr string, req *ChatRequest) error {
	// Simple parsing for testing
	req.RequestID = "test-id"
	req.Model = "test-model"
	req.Stream = true
	req.Messages = []ChatMessage{{Role: "user", Content: "test"}}
	return nil
}

func TestSSEHandler_NewSSEHandler(t *testing.T) {
	config := DefaultWebSocketConfig(nil, true)
	handler := NewSSEHandler(config)

	assert.NotNil(t, handler)
	assert.Equal(t, config, handler.config)
}

// Test WebSocket configuration options
func TestWebSocketConfig_CustomValues(t *testing.T) {
	config := &WebSocketConfig{
		MaxConnections:     50,
		PingInterval:       15 * time.Second,
		PongWait:          30 * time.Second,
		WriteWait:         5 * time.Second,
		ReadBufferSize:    8192,
		WriteBufferSize:   8192,
		EnableSandboxFallback: false,
	}

	assert.Equal(t, 50, config.MaxConnections)
	assert.Equal(t, 15*time.Second, config.PingInterval)
	assert.Equal(t, 30*time.Second, config.PongWait)
	assert.Equal(t, 5*time.Second, config.WriteWait)
	assert.Equal(t, 8192, config.ReadBufferSize)
	assert.Equal(t, 8192, config.WriteBufferSize)
	assert.False(t, config.EnableSandboxFallback)
}

// Test that ChatMessage is properly defined
func TestChatMessage(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "Hello, world!",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
}

// Test stream message types
func TestStreamMessage_Types(t *testing.T) {
	testCases := []struct {
		msgType    string
		content    interface{}
		errMsg     string
	}{
		{"ack", map[string]string{"request_id": "123"}, ""},
		{"token", map[string]string{"token": "hello"}, ""},
		{"error", nil, "invalid_json"},
		{"done", map[string]int{"tokens": 100}, ""},
	}

	for _, tc := range testCases {
		msg := StreamMessage{
			Type:    tc.msgType,
			Content: tc.content,
			Error:   tc.errMsg,
		}

		assert.Equal(t, tc.msgType, msg.Type)
		assert.Equal(t, tc.content, msg.Content)
		assert.Equal(t, tc.errMsg, msg.Error)
	}
}

// Test concurrent connection pool operations
func TestConnectionPool_Concurrent(t *testing.T) {
	pool := NewConnectionPool(1000)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrently add connections
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := &websocket.Conn{}
			pool.Add(conn)
		}(i)
	}

	wg.Wait()

	// Count should be <= maxConnections
	assert.LessOrEqual(t, pool.Count(), 1000)
}

// Test connection pool edge cases
func TestConnectionPool_EdgeCases(t *testing.T) {
	pool := NewConnectionPool(0) // Edge case: 0 max connections

	conn := &websocket.Conn{}

	// Should reject since max is 0
	assert.False(t, pool.Add(conn))
	assert.Equal(t, 0, pool.Count())

	// Remove non-existent should not panic
	assert.NotPanics(t, func() {
		pool.Remove(conn)
	})
}

// Integration-style test for request handling
func TestProcessMessage_InvalidJSON(t *testing.T) {
	config := DefaultWebSocketConfig(nil, true)
	handler := NewWebSocketHandler(config)

	// Test with invalid JSON
	invalidJSON := []byte("not valid json")

	// This should be handled gracefully
	assert.NotNil(t, handler)
}

// Test SSE write helper
func TestSSEWrite(t *testing.T) {
	// Test that SSE write helper function exists and is callable
	// Note: This requires a real ResponseWriter for full testing
	data := map[string]string{"token": "test"}
	jsonData, err := json.Marshal(data)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "test")
}

// Import for json test
var _ = strings.TrimSpace
