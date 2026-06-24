package mcp

import (
	"context"
	"testing"
)

func TestProviderConstants(t *testing.T) {
	if ProviderAnthropic != "anthropic" {
		t.Errorf("ProviderAnthropic = %q, want %q", ProviderAnthropic, "anthropic")
	}
	if ProviderGoogle != "google" {
		t.Errorf("ProviderGoogle = %q, want %q", ProviderGoogle, "google")
	}
	if ProviderMeta != "meta" {
		t.Errorf("ProviderMeta = %q, want %q", ProviderMeta, "meta")
	}
}

func TestNewMultiProviderRegistry(t *testing.T) {
	registry := NewMultiProviderRegistry()
	if registry == nil {
		t.Fatal("NewMultiProviderRegistry returned nil")
	}
	if len(registry.servers) != 0 {
		t.Errorf("Expected empty servers map, got %d servers", len(registry.servers))
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	registry := NewMultiProviderRegistry()

	anthropic := &AnthropicMCPServer{}
	google := &GoogleMCPServer{}
	meta := &MetaMCPServer{}

	registry.Register(anthropic)
	registry.Register(google)
	registry.Register(meta)

	if len(registry.servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(registry.servers))
	}

	server, ok := registry.Get(ProviderAnthropic)
	if !ok {
		t.Error("Expected to get Anthropic server")
	}
	if server.Name() != "Anthropic MCP Server" {
		t.Errorf("Got wrong name: %s", server.Name())
	}

	server, ok = registry.Get(ProviderGoogle)
	if !ok {
		t.Error("Expected to get Google server")
	}
	if server.Name() != "Google MCP Server" {
		t.Errorf("Got wrong name: %s", server.Name())
	}

	server, ok = registry.Get(ProviderMeta)
	if !ok {
		t.Error("Expected to get Meta server")
	}
	if server.Name() != "Meta MCP Server" {
		t.Errorf("Got wrong name: %s", server.Name())
	}

	_, ok = registry.Get("unknown")
	if ok {
		t.Error("Expected not to get unknown provider")
	}
}

func TestListProviders(t *testing.T) {
	registry := NewMultiProviderRegistry()

	registry.Register(&AnthropicMCPServer{})
	registry.Register(&GoogleMCPServer{})

	providers := registry.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	foundAnthropic := false
	foundGoogle := false
	for _, p := range providers {
		if p == ProviderAnthropic {
			foundAnthropic = true
		}
		if p == ProviderGoogle {
			foundGoogle = true
		}
	}
	if !foundAnthropic {
		t.Error("Anthropic provider not found")
	}
	if !foundGoogle {
		t.Error("Google provider not found")
	}
}

func TestAnthropicInitialize(t *testing.T) {
	server := &AnthropicMCPServer{}
	ctx := context.Background()
	config := ProviderConfig{
		Name:        "test-anthropic",
		APIEndpoint: "https://api.anthropic.com",
		AuthHeader:  "Bearer test-token",
	}

	err := server.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if server.Name() != "Anthropic MCP Server" {
		t.Errorf("Name = %q, want %q", server.Name(), "Anthropic MCP Server")
	}

	if server.Provider() != ProviderAnthropic {
		t.Errorf("Provider = %q, want %q", server.Provider(), ProviderAnthropic)
	}

	tools, err := server.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}
}

func TestAnthropicCallTool(t *testing.T) {
	server := &AnthropicMCPServer{}
	ctx := context.Background()

	err := server.Initialize(ctx, ProviderConfig{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	result, err := server.CallTool(ctx, "anthropic_messages", map[string]interface{}{
		"messages": []interface{}{},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if resultMap["status"] != "success" {
		t.Errorf("Expected success status, got %v", resultMap["status"])
	}

	_, err = server.CallTool(ctx, "unknown_tool", nil)
	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

func TestAnthropicShutdown(t *testing.T) {
	server := &AnthropicMCPServer{}
	ctx := context.Background()

	err := server.Initialize(ctx, ProviderConfig{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	err = server.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestGoogleInitialize(t *testing.T) {
	server := &GoogleMCPServer{}
	ctx := context.Background()
	config := ProviderConfig{
		Name:        "test-google",
		APIEndpoint: "https://generativelanguage.googleapis.com",
	}

	err := server.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if server.Name() != "Google MCP Server" {
		t.Errorf("Name = %q, want %q", server.Name(), "Google MCP Server")
	}

	if server.Provider() != ProviderGoogle {
		t.Errorf("Provider = %q, want %q", server.Provider(), ProviderGoogle)
	}

	tools, err := server.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(tools))
	}
}

func TestGoogleCallTool(t *testing.T) {
	server := &GoogleMCPServer{}
	ctx := context.Background()

	err := server.Initialize(ctx, ProviderConfig{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	result, err := server.CallTool(ctx, "google_gemini_generate", map[string]interface{}{
		"prompt": "test prompt",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if resultMap["status"] != "success" {
		t.Errorf("Expected success status, got %v", resultMap["status"])
	}
}

func TestMetaInitialize(t *testing.T) {
	server := &MetaMCPServer{}
	ctx := context.Background()
	config := ProviderConfig{
		Name:        "test-meta",
		APIEndpoint: "https://api.meta.ai",
	}

	err := server.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if server.Name() != "Meta MCP Server" {
		t.Errorf("Name = %q, want %q", server.Name(), "Meta MCP Server")
	}

	if server.Provider() != ProviderMeta {
		t.Errorf("Provider = %q, want %q", server.Provider(), ProviderMeta)
	}

	tools, err := server.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(tools))
	}
}

func TestMetaCallTool(t *testing.T) {
	server := &MetaMCPServer{}
	ctx := context.Background()

	err := server.Initialize(ctx, ProviderConfig{})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	result, err := server.CallTool(ctx, "meta_llama_generate", map[string]interface{}{
		"prompt": "test prompt",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}
	if resultMap["status"] != "success" {
		t.Errorf("Expected success status, got %v", resultMap["status"])
	}
}

func TestToolStruct(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "Test tool description",
		InputSchema: map[string]interface{}{
			"type": "object",
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("Name = %q, want %q", tool.Name, "test_tool")
	}
	if tool.Description != "Test tool description" {
		t.Errorf("Description = %q, want %q", tool.Description, "Test tool description")
	}
	if tool.InputSchema["type"] != "object" {
		t.Errorf("InputSchema type = %v, want object", tool.InputSchema["type"])
	}
}

func TestProviderConfigStruct(t *testing.T) {
	config := ProviderConfig{
		Name:        "test-provider",
		APIEndpoint: "https://api.test.com",
		AuthHeader:  "Bearer token",
		Tools:       []Tool{},
	}

	if config.Name != "test-provider" {
		t.Errorf("Name = %q, want %q", config.Name, "test-provider")
	}
	if config.APIEndpoint != "https://api.test.com" {
		t.Errorf("APIEndpoint = %q, want %q", config.APIEndpoint, "https://api.test.com")
	}
}
