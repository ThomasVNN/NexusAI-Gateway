package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

// JSONSchema represents the input schema for a tool
type JSONSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

// ToolHandler is the function signature for tool handlers
type ToolHandler func(ctx interface{}, arguments json.RawMessage) (interface{}, error)

// Tool represents an MCP tool definition
type Tool struct {
	Name           string
	Description   string
	InputSchema   JSONSchema
	Handler       ToolHandler
	RequiredScope string
}

// ToolRegistry manages MCP tools
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// Register registers a tool
func (r *ToolRegistry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *ToolRegistry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// Call executes a tool by name
func (r *ToolRegistry) Call(name string, arguments json.RawMessage) (interface{}, error) {
	tool, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return tool.Handler(nil, arguments)
}

// Count returns the number of registered tools
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}
