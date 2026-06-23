package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/protocol/mcp"
)

// StdioTransport implements MCP over stdio
type StdioTransport struct {
	addr     string
	handler  func(req *mcp.Request) *mcp.Response
	reader   *bufio.Reader
	writer   *bufio.Writer
	mu       sync.Mutex
	running  bool
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(addr string, handler func(req *mcp.Request) *mcp.Response) *StdioTransport {
	return &StdioTransport{
		addr:    addr,
		handler: handler,
		reader: bufio.NewReader(os.Stdin),
		writer: bufio.NewWriter(os.Stdout),
	}
}

// Start starts the stdio transport
func (t *StdioTransport) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("transport already running")
	}
	t.running = true
	t.mu.Unlock()

	handshake := map[string]interface{}{
		"jsonrpc": "2.0",
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "nexusai-gateway",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"prompts":   map[string]interface{}{},
				"resources":  map[string]interface{}{},
			},
		},
	}
	if err := t.send(handshake); err != nil {
		return fmt.Errorf("failed to send handshake: %w", err)
	}

	for t.running {
		msg, err := t.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		go t.handleMessage(msg)
	}

	return nil
}

// Stop stops the stdio transport
func (t *StdioTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false
	return nil
}

func (t *StdioTransport) readMessage() (map[string]interface{}, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var msg map[string]interface{}
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return msg, nil
}

func (t *StdioTransport) handleMessage(msg map[string]interface{}) {
	method, _ := msg["method"].(string)
	id := msg["id"]

	req := &mcp.Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}

	if params, ok := msg["params"].(map[string]interface{}); ok {
		paramsJSON, _ := json.Marshal(params)
		req.Params = paramsJSON
	}

	resp := t.handler(req)
	t.sendResponse(resp)
}

func (t *StdioTransport) send(msg interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if _, err := t.writer.Write(data); err != nil {
		return err
	}
	if _, err := t.writer.WriteString("\n"); err != nil {
		return err
	}

	return t.writer.Flush()
}

func (t *StdioTransport) sendResponse(resp *mcp.Response) {
	t.send(resp)
}
