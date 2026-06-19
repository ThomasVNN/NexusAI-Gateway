package format

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// OpenAIMessage represents an OpenAI chat message
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// OpenAIRequest represents an OpenAI chat completion request
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stop        interface{}     `json:"stop,omitempty"`
}

// ClaudeMessage represents a Claude API message
type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or array of content blocks
}

// ClaudeRequest represents a Claude API request
type ClaudeRequest struct {
	Model       string          `json:"model"`
	Messages    []ClaudeMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
}

// ClaudeContentBlock represents Claude message content blocks
type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ConvertOpenAIToClaude converts an OpenAI request to Claude format
func ConvertOpenAIToClaude(openAIReq *OpenAIRequest) (*ClaudeRequest, error) {
	claudeReq := &ClaudeRequest{
		Model:       MapModel(openAIReq.Model, "openai", "anthropic"),
		Stream:      openAIReq.Stream,
		MaxTokens:   openAIReq.MaxTokens,
		Temperature: openAIReq.Temperature,
		TopP:        openAIReq.TopP,
	}

	// Convert messages
	for i, msg := range openAIReq.Messages {
		claudeMsg := ClaudeMessage{
			Role:    MapRole(msg.Role),
			Content: msg.Content,
		}

		// Extract system messages
		if msg.Role == "system" {
			if claudeReq.System != "" {
				claudeReq.System += "\n\n"
			}
			claudeReq.System += msg.Content
		} else {
			claudeReq.Messages = append(claudeReq.Messages, claudeMsg)
		}

		// Log warning for first non-system message
		if i == len(openAIReq.Messages)-len(claudeReq.Messages) && msg.Role != "system" && len(claudeReq.Messages) == 0 {
			// This is the first non-system message
		}
	}

	// Default max_tokens if not set
	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = 4096
	}

	return claudeReq, nil
}

// ConvertClaudeToOpenAI converts a Claude response to OpenAI format
func ConvertClaudeToOpenAI(claudeResp interface{}) (map[string]interface{}, error) {
	openAIResp := map[string]interface{}{
		"id":      "chatcmpl-" + generateID(),
		"object":  "chat.completion",
		"created": generateTimestamp(),
		"model":   "", // Will be set by caller
		"choices": []map[string]interface{}{},
	}

	// Parse Claude response and convert
	content := extractClaudeContent(claudeResp)

	choice := map[string]interface{}{
		"index": 0,
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": content,
		},
		"finish_reason": "stop",
	}

	openAIResp["choices"] = []map[string]interface{}{choice}
	return openAIResp, nil
}

// MapModel maps model names between providers
func MapModel(model, from, to string) string {
	// OpenAI to Anthropic mappings
	openaiToAnthropic := map[string]string{
		"gpt-4":         "claude-3-opus-20240229",
		"gpt-4-turbo":   "claude-3-5-sonnet-20241022",
		"gpt-3.5-turbo": "claude-3-haiku-20240307",
	}

	// OpenAI to Google mappings
	openaiToGoogle := map[string]string{
		"gpt-4":         "gemini-1.5-pro",
		"gpt-4-turbo":   "gemini-1.5-pro",
		"gpt-3.5-turbo": "gemini-1.5-flash",
	}

	// Anthropic to OpenAI mappings
	anthropicToOpenai := map[string]string{
		"claude-3-opus-20240229":     "gpt-4",
		"claude-3-5-sonnet-20241022": "gpt-4-turbo",
		"claude-3-haiku-20240307":    "gpt-3.5-turbo",
	}

	// Anthropic to Google mappings
	anthropicToGoogle := map[string]string{
		"claude-3-opus-20240229":     "gemini-1.5-pro",
		"claude-3-5-sonnet-20241022": "gemini-1.5-pro",
		"claude-3-haiku-20240307":    "gemini-1.5-flash",
	}

	switch from {
	case "openai":
		switch to {
		case "anthropic":
			if mapped, ok := openaiToAnthropic[model]; ok {
				return mapped
			}
		case "google":
			if mapped, ok := openaiToGoogle[model]; ok {
				return mapped
			}
		}
	case "anthropic":
		switch to {
		case "openai":
			if mapped, ok := anthropicToOpenai[model]; ok {
				return mapped
			}
		case "google":
			if mapped, ok := anthropicToGoogle[model]; ok {
				return mapped
			}
		}
	}

	return model
}

// MapRole maps role names between providers
func MapRole(role string) string {
	switch strings.ToLower(role) {
	case "system":
		return "user" // Claude doesn't have system role, use user with system prompt
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	case "function", "tool":
		return "user" // Claude handles tools differently
	default:
		return "user"
	}
}

// extractClaudeContent extracts text content from Claude response
func extractClaudeContent(resp interface{}) string {
	// Try to parse as map
	respMap, ok := resp.(map[string]interface{})
	if !ok {
		// Try JSON string
		if str, ok := resp.(string); ok {
			return str
		}
		return ""
	}

	// Check for content field
	content, ok := respMap["content"]
	if !ok {
		return ""
	}

	// Handle array of content blocks
	if contentArr, ok := content.([]interface{}); ok {
		var text strings.Builder
		for _, block := range contentArr {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if blockType, ok := blockMap["type"].(string); ok && blockType == "text" {
					if textContent, ok := blockMap["text"].(string); ok {
						text.WriteString(textContent)
					}
				}
			}
		}
		return text.String()
	}

	// Handle string content
	if str, ok := content.(string); ok {
		return str
	}

	return ""
}

// ExtractThinkingContent extracts thinking from Claude extended thinking responses
func ExtractThinkingContent(response map[string]interface{}) (thinking string, contentText string) {
	// Look for thinking block in content
	if contentField, ok := response["content"].([]interface{}); ok {
		for _, block := range contentField {
			if blockMap, ok := block.(map[string]interface{}); ok {
				switch blockMap["type"] {
				case "thinking":
					if t, ok := blockMap["thinking"].(string); ok {
						thinking = t
					}
				case "text":
					if t, ok := blockMap["text"].(string); ok {
						contentText = t
					}
				}
			}
		}
	}

	// If no content found, try "output" field
	if contentText == "" {
		if output, ok := response["output"].(string); ok {
			contentText = output
		}
	}

	return thinking, contentText
}

// GenerateRequestHash generates a hash for request deduplication
func GenerateRequestHash(req *OpenAIRequest) string {
	data, _ := json.Marshal(req)
	return hashString(string(data))
}

// Helper functions
func generateID() string {
	// Simple ID generation - in production use UUID
	return randomString(24)
}

func generateTimestamp() int64 {
	return int64(0) // Would use time.Now().Unix() in production
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

func hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}
