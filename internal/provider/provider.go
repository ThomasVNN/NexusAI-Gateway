package provider

import (
	"context"
)

// Message holds the role and content block for LLM calls
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest defines the standardized input for text generation
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// CompletionResponse defines the standardized output for text generation
type CompletionResponse struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   UsageInfo `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMProvider acts as a translator to interface with distinct LLM API endpoints
type LLMProvider interface {
	ID() string
	GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
	SupportsModel(model string) bool
}
