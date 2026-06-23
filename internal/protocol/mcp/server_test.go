package mcp

import (
	"encoding/json"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	// Test registering a tool
	tool := &Tool{
		Name:        "test_tool",
		Description: "Test tool",
		InputSchema: emptySchema,
		Handler: func(ctx interface{}, arguments json.RawMessage) (interface{}, error) {
			return "test_result", nil
		},
	}

	if err := registry.Register(tool); err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Test getting a tool
	gotTool, exists := registry.Get("test_tool")
	if !exists {
		t.Fatal("Tool not found after registration")
	}
	if gotTool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", gotTool.Name)
	}

	// Test listing tools
	tools := registry.List()
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	// Test calling a tool
	result, err := registry.Call("test_tool", nil)
	if err != nil {
		t.Fatalf("Failed to call tool: %v", err)
	}
	if result != "test_result" {
		t.Errorf("Expected 'test_result', got '%v'", result)
	}

	// Test duplicate registration
	if err := registry.Register(tool); err == nil {
		t.Error("Expected error for duplicate registration")
	}

	// Test getting non-existent tool
	_, exists = registry.Get("non_existent")
	if exists {
		t.Error("Expected tool to not exist")
	}

	// Test calling non-existent tool
	_, err = registry.Call("non_existent", nil)
	if err == nil {
		t.Error("Expected error for calling non-existent tool")
	}

	// Test count
	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}
}

func TestScopes(t *testing.T) {
	scopes := DefaultScopes()

	// Test that all expected scopes exist
	expectedScopes := []string{
		ScopeMemoryRead,
		ScopeMemoryWrite,
		ScopeMemoryDelete,
		ScopeRoutingRead,
		ScopeRoutingWrite,
		ScopeBudgetRead,
		ScopeBudgetWrite,
		ScopeSkillRead,
		ScopeSkillWrite,
		ScopeSkillExecute,
		ScopeAdminRead,
		ScopeAdminWrite,
	}

	for _, scopeName := range expectedScopes {
		scope, exists := scopes[scopeName]
		if !exists {
			t.Errorf("Scope %s not found", scopeName)
			continue
		}
		if scope.Name != scopeName {
			t.Errorf("Expected scope name '%s', got '%s'", scopeName, scope.Name)
		}
	}

	// Test scope levels
	for _, scope := range scopes {
		if scope.Level < 1 || scope.Level > 3 {
			t.Errorf("Invalid scope level %d for scope %s", scope.Level, scope.Name)
		}
	}
}

func TestJSONSchema(t *testing.T) {
	schema := memorySearchSchema

	if schema.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", schema.Type)
	}

	if len(schema.Required) != 1 || schema.Required[0] != "query" {
		t.Errorf("Expected required field 'query', got %v", schema.Required)
	}

	props := schema.Properties
	if _, ok := props["query"]; !ok {
		t.Error("Expected 'query' property")
	}
	if _, ok := props["memory_type"]; !ok {
		t.Error("Expected 'memory_type' property")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("Expected 'limit' property")
	}
}

func TestServerCreation(t *testing.T) {
	server := NewServer()

	// Test that tools are registered
	tools := server.ListTools()
	if len(tools) == 0 {
		t.Error("Expected tools to be registered")
	}

	// Test tool count
	count := server.ToolCount()
	if count == 0 {
		t.Error("Expected non-zero tool count")
	}
}

func TestServerProcessRequest(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name           string
		method         string
		params         interface{}
		expectedResult bool
	}{
		{
			name:           "tools/list",
			method:         "tools/list",
			params:         nil,
			expectedResult: true,
		},
		{
			name:           "initialize",
			method:         "initialize",
			params:         nil,
			expectedResult: true,
		},
		{
			name:           "ping",
			method:         "ping",
			params:         nil,
			expectedResult: true,
		},
		{
			name:           "unknown method",
			method:         "unknown",
			params:         nil,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params json.RawMessage
			if tt.params != nil {
				params, _ = json.Marshal(tt.params)
			}

			req := &Request{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tt.method,
				Params:  params,
			}

			resp := server.processRequest(nil, req)

			if tt.expectedResult {
				if resp.Error != nil {
					t.Errorf("Expected no error, got %v", resp.Error)
				}
			} else {
				if resp.Error == nil {
					t.Error("Expected error for unknown method")
				}
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	server := NewServer()

	resp := server.errorResponse(123, -32601, "Method not found")

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC '2.0', got '%s'", resp.JSONRPC)
	}
	if resp.ID != 123 {
		t.Errorf("Expected ID 123, got %v", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be set")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("Expected message 'Method not found', got '%s'", resp.Error.Message)
	}
}
