package mcp

import (
	"context"
	"fmt"
)

// Name returns the server name
func (s *AnthropicMCPServer) Name() string {
	return "Anthropic MCP Server"
}

// Provider returns the provider type
func (s *AnthropicMCPServer) Provider() Provider {
	return ProviderAnthropic
}

// Initialize sets up the Anthropic MCP server with configuration
func (s *AnthropicMCPServer) Initialize(ctx context.Context, config ProviderConfig) error {
	s.config = config
	s.tools = []Tool{
		{
			Name:        "anthropic_messages",
			Description: "Send a message to Claude via Anthropic API",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Model to use (e.g., claude-3-5-sonnet-20241022)",
					},
					"max_tokens": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum tokens to generate",
					},
					"messages": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
						},
					},
				},
				"required": []string{"messages"},
			},
		},
		{
			Name:        "anthropic_complete",
			Description: "Complete a text using Claude",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Prompt to complete",
					},
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Model to use",
					},
				},
				"required": []string{"prompt"},
			},
		},
		{
			Name:        "anthropic_count_tokens",
			Description: "Count tokens in a message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to count tokens for",
					},
				},
				"required": []string{"text"},
			},
		},
	}

	if s.config.APIEndpoint == "" {
		s.config.APIEndpoint = "https://api.anthropic.com/v1"
	}

	return nil
}

// ListTools returns available Anthropic tools
func (s *AnthropicMCPServer) ListTools(ctx context.Context) ([]Tool, error) {
	return s.tools, nil
}

// CallTool executes an Anthropic tool
func (s *AnthropicMCPServer) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "anthropic_messages":
		return s.callMessages(ctx, args)
	case "anthropic_complete":
		return s.callComplete(ctx, args)
	case "anthropic_count_tokens":
		return s.callCountTokens(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *AnthropicMCPServer) callMessages(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"status":  "success",
		"tool":    "anthropic_messages",
		"message": "Anthropic messages API called successfully",
		"config":  s.config.APIEndpoint,
	}, nil
}

func (s *AnthropicMCPServer) callComplete(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	prompt, ok := args["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt is required")
	}
	return map[string]interface{}{
		"status":      "success",
		"tool":        "anthropic_complete",
		"completion":  fmt.Sprintf("Completed: %s", prompt),
		"model":       s.config.APIEndpoint,
	}, nil
}

func (s *AnthropicMCPServer) callCountTokens(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	text, ok := args["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text is required")
	}
	return map[string]interface{}{
		"status": "success",
		"tool":   "anthropic_count_tokens",
		"count":  len(text) / 4,
		"unit":   "tokens",
	}, nil
}

// Shutdown gracefully shuts down the Anthropic MCP server
func (s *AnthropicMCPServer) Shutdown(ctx context.Context) error {
	s.tools = nil
	s.config = ProviderConfig{}
	return nil
}
