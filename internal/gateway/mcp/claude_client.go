package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ClaudeMCPConfig holds Claude MCP configuration
type ClaudeMCPConfig struct {
	Endpoint       string
	APIKey        string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// ClaudeMCPClient implements an MCP client that connects to Claude
type ClaudeMCPClient struct {
	config   ClaudeMCPConfig
	serverURL string
	tools    []Tool
	mu       sync.RWMutex
	conn     *MCPConnection
}

// MCPConnection represents a connection to an MCP server
type MCPConnection struct {
	ID        string
	ServerURL string
	Status    ConnectionStatus
	CreatedAt time.Time
	LastSeen  time.Time
}

// ConnectionStatus represents the status of an MCP connection
type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusConnecting   ConnectionStatus = "connecting"
	StatusError        ConnectionStatus = "error"
)

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// NewClaudeMCPClient creates a new Claude MCP client
func NewClaudeMCPClient(config ClaudeMCPConfig) *ClaudeMCPClient {
	return &ClaudeMCPClient{
		config:    config,
		serverURL: config.Endpoint,
		tools:     make([]Tool, 0),
	}
}

// Connect establishes a connection to the Claude MCP server
func (c *ClaudeMCPClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	slog.Info("Connecting to Claude MCP server", slog.String("endpoint", c.serverURL))

	// Simulate connection establishment
	c.conn = &MCPConnection{
		ID:        generateConnectionID(),
		ServerURL: c.serverURL,
		Status:    StatusConnected,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	// Initialize tools from server
	if err := c.fetchTools(ctx); err != nil {
		c.conn.Status = StatusError
		return fmt.Errorf("failed to fetch tools: %w", err)
	}

	slog.Info("Connected to Claude MCP server", slog.String("conn_id", c.conn.ID))
	return nil
}

// Disconnect closes the connection to the Claude MCP server
func (c *ClaudeMCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Status = StatusDisconnected
		c.conn = nil
	}

	slog.Info("Disconnected from Claude MCP server")
	return nil
}

// fetchTools fetches available tools from the MCP server
func (c *ClaudeMCPClient) fetchTools(ctx context.Context) error {
	// In a real implementation, this would call the MCP server's tools/list endpoint
	// For now, we define standard tools that NexusAI exposes to Claude

	c.tools = []Tool{
		{
			Name:        "nexusai_chat",
			Description: "Send a chat message to NexusAI and get a response",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message to send",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "The model to use (default: claude-3-5-sonnet)",
					},
					"stream": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to stream the response",
					},
				},
				"required": []string{"message"},
			},
		},
		{
			Name:        "nexusai_route",
			Description: "Route a request to the best AI provider",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The prompt to route",
					},
					"constraints": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"max_cost":      map[string]interface{}{"type": "number"},
							"max_latency":   map[string]interface{}{"type": "number"},
							"preferred_model": map[string]interface{}{"type": "string"},
						},
					},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Name:        "nexusai_search",
			Description: "Search the knowledge base",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "nexusai_execute_skill",
			Description: "Execute a NexusAI skill",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"skill_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the skill to execute",
					},
					"params": map[string]interface{}{
						"type":        "object",
						"description": "Parameters for the skill",
					},
				},
				"required": []string{"skill_name"},
			},
		},
		{
			Name:        "nexusai_get_models",
			Description: "Get available AI models",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"provider": map[string]interface{}{
						"type":        "string",
						"description": "Filter by provider",
					},
				},
			},
		},
		{
			Name:        "nexusai_check_quota",
			Description: "Check API quota and usage",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return nil
}

// ListTools returns all available tools
func (c *ClaudeMCPClient) ListTools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tools
}

// CallTool calls a tool on the MCP server
func (c *ClaudeMCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Update last seen
	if c.conn != nil {
		c.conn.LastSeen = time.Now()
	}

	switch toolName {
	case "nexusai_chat":
		return c.handleChat(ctx, args)
	case "nexusai_route":
		return c.handleRoute(ctx, args)
	case "nexusai_search":
		return c.handleSearch(ctx, args)
	case "nexusai_execute_skill":
		return c.handleExecuteSkill(ctx, args)
	case "nexusai_get_models":
		return c.handleGetModels(ctx, args)
	case "nexusai_check_quota":
		return c.handleCheckQuota(ctx)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// handleChat handles chat requests
func (c *ClaudeMCPClient) handleChat(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required")
	}

	model, _ := args["model"].(string)
	if model == "" {
		model = "claude-3-5-sonnet"
	}

	// In a real implementation, this would call the chat completion API
	result := map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("Processed: %s", message),
		"model":    model,
		"tokens":   len(message) / 4, // Rough estimate
	}

	return result, nil
}

// handleRoute handles routing requests
func (c *ClaudeMCPClient) handleRoute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	prompt, ok := args["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt is required")
	}

	// Simple routing logic
	result := map[string]interface{}{
		"provider": "anthropic",
		"model":     "claude-3-5-sonnet-20241022",
		"confidence": 0.95,
		"reason":    fmt.Sprintf("Optimal for: %s", prompt[:min(50, len(prompt))]),
	}

	return result, nil
}

// handleSearch handles knowledge base searches
func (c *ClaudeMCPClient) handleSearch(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	limit, _ := args["limit"].(int)
	if limit == 0 {
		limit = 10
	}

	// Return mock results
	results := []map[string]interface{}{
		{
			"title":   "Result 1",
			"content": fmt.Sprintf("Content related to: %s", query),
			"score":   0.95,
		},
		{
			"title":   "Result 2",
			"content": fmt.Sprintf("Another result for: %s", query),
			"score":   0.85,
		},
	}

	return map[string]interface{}{
		"query":   query,
		"results": results[:min(limit, len(results))],
		"total":   len(results),
	}, nil
}

// handleExecuteSkill handles skill execution
func (c *ClaudeMCPClient) handleExecuteSkill(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	skillName, ok := args["skill_name"].(string)
	if !ok {
		return nil, fmt.Errorf("skill_name is required")
	}

	params, _ := args["params"].(map[string]interface{})

	return map[string]interface{}{
		"skill":    skillName,
		"status":   "executed",
		"params":   params,
		"output":   fmt.Sprintf("Skill %s executed successfully", skillName),
	}, nil
}

// handleGetModels handles model listing
func (c *ClaudeMCPClient) handleGetModels(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	models := []map[string]interface{}{
		{
			"id":            "claude-3-5-sonnet-20241022",
			"provider":     "anthropic",
			"name":          "Claude 3.5 Sonnet",
			"context_window": 200000,
		},
		{
			"id":            "gpt-4o",
			"provider":     "openai",
			"name":          "GPT-4o",
			"context_window": 128000,
		},
	}

	return map[string]interface{}{
		"models": models,
	}, nil
}

// handleCheckQuota handles quota checking
func (c *ClaudeMCPClient) handleCheckQuota(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{
		"total_tokens":   1000000,
		"used_tokens":    250000,
		"remaining_tokens": 750000,
		"reset_at":       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}, nil
}

// Status returns the current connection status
func (c *ClaudeMCPClient) Status() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil {
		return StatusDisconnected
	}
	return c.conn.Status
}

// GetConnection returns the current connection
func (c *ClaudeMCPClient) GetConnection() *MCPConnection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	return fmt.Sprintf("mcp-%d-%d", time.Now().UnixNano(), time.Now().UnixMicro())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MarshalJSON implements json.Marshaler for ClaudeMCPClient
func (c *ClaudeMCPClient) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	type Alias ClaudeMCPClient
	return json.Marshal(&struct {
		*Alias
		Status ConnectionStatus `json:"status"`
	}{
		Alias:  (*Alias)(c),
		Status: c.Status(),
	})
}
