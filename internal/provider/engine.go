package provider

import (
	"sync"
	"time"
)

// ProviderEngine manages all provider configurations
type ProviderEngine struct {
	mu        sync.RWMutex
	providers map[string]*Provider
}

var (
	providerEngine *ProviderEngine
	engineOnce     sync.Once
)

// GetProviderEngine returns the singleton provider engine
func GetProviderEngine() *ProviderEngine {
	engineOnce.Do(func() {
		providerEngine = &ProviderEngine{
			providers: make(map[string]*Provider),
		}
		providerEngine.initProviders()
	})
	return providerEngine
}

func (p *ProviderEngine) initProviders() {
	providers := []*Provider{
		// OpenAI (10 models)
		{
			ID: "provider-openai", Name: "OpenAI", Category: "openai", APIType: "openai",
			BaseURL: "https://api.openai.com/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 500,
			Capabilities: []string{"streaming", "vision", "function_calling", "json_mode"},
			Models: []Model{
				{ID: "gpt-5.5", Name: "GPT-5.5", Provider: "openai", InputCostPer1K: 0.015, OutputCostPer1K: 0.06, ContextWindow: 256000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 65, QualityScore: 99, Available: true},
				{ID: "gpt-5.5-instant", Name: "GPT-5.5 Instant", Provider: "openai", InputCostPer1K: 0.003, OutputCostPer1K: 0.012, ContextWindow: 256000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 80, QualityScore: 97, Available: true},
				{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", InputCostPer1K: 0.005, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 95, Available: true},
				{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", InputCostPer1K: 0.00015, OutputCostPer1K: 0.0006, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 95, QualityScore: 85, Available: true},
				{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Provider: "openai", InputCostPer1K: 0.01, OutputCostPer1K: 0.03, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 80, QualityScore: 95, Available: true},
				{ID: "gpt-4", Name: "GPT-4", Provider: "openai", InputCostPer1K: 0.03, OutputCostPer1K: 0.06, ContextWindow: 8192, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 75, QualityScore: 95, Available: true},
				{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai", InputCostPer1K: 0.0005, OutputCostPer1K: 0.0015, ContextWindow: 16385, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 95, QualityScore: 80, Available: true},
				{ID: "o1-preview", Name: "o1 Preview", Provider: "openai", InputCostPer1K: 0.015, OutputCostPer1K: 0.06, ContextWindow: 128000, SupportsStreaming: false, SupportsFunctions: false, SupportsJSON: false, LatencyScore: 60, QualityScore: 98, Available: true},
				{ID: "o1-mini", Name: "o1 Mini", Provider: "openai", InputCostPer1K: 0.003, OutputCostPer1K: 0.012, ContextWindow: 65536, SupportsStreaming: false, SupportsFunctions: false, SupportsJSON: false, LatencyScore: 75, QualityScore: 90, Available: true},
				{ID: "o3-mini", Name: "o3 Mini", Provider: "openai", InputCostPer1K: 0.0011, OutputCostPer1K: 0.0044, ContextWindow: 65536, SupportsStreaming: false, SupportsFunctions: false, SupportsJSON: false, LatencyScore: 80, QualityScore: 92, Available: true},
				{ID: "chatgpt-4o-latest", Name: "GPT-4o (Latest)", Provider: "openai", InputCostPer1K: 0.005, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 96, Available: true},
			},
		},
		// Anthropic (7 models)
		{
			ID: "provider-anthropic", Name: "Anthropic", Category: "anthropic", APIType: "anthropic",
			BaseURL: "https://api.anthropic.com/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 300,
			Capabilities: []string{"streaming", "vision", "function_calling", "json_mode", "extended_thinking"},
			Models: []Model{
				{ID: "claude-opus-4", Name: "Claude Opus 4", Provider: "anthropic", InputCostPer1K: 0.015, OutputCostPer1K: 0.075, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 70, QualityScore: 98, Available: true},
				{ID: "claude-sonnet-4.8", Name: "Claude Sonnet 4.8", Provider: "anthropic", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 97, Available: true},
				{ID: "claude-sonnet-4", Name: "Claude Sonnet 4", Provider: "anthropic", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 96, Available: true},
				{ID: "claude-haiku-3-5", Name: "Claude Haiku 3.5", Provider: "anthropic", InputCostPer1K: 0.0008, OutputCostPer1K: 0.004, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 95, QualityScore: 88, Available: true},
				{ID: "claude-3-5-sonnet", Name: "Claude 3.5 Sonnet", Provider: "anthropic", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 88, QualityScore: 96, Available: true},
				{ID: "claude-3-opus", Name: "Claude 3 Opus", Provider: "anthropic", InputCostPer1K: 0.015, OutputCostPer1K: 0.075, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 75, QualityScore: 97, Available: true},
				{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", Provider: "anthropic", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 88, QualityScore: 94, Available: true},
				{ID: "claude-3-haiku", Name: "Claude 3 Haiku", Provider: "anthropic", InputCostPer1K: 0.00025, OutputCostPer1K: 0.00125, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 98, QualityScore: 86, Available: true},
			},
		},
		// Google Gemini (7 models)
		{
			ID: "provider-google", Name: "Google Gemini", Category: "google", APIType: "rest",
			BaseURL: "https://generativelanguage.googleapis.com/v1beta", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 400,
			Capabilities: []string{"streaming", "vision", "function_calling"},
			Models: []Model{
				{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash", Provider: "google", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 1000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 98, QualityScore: 90, Available: true},
				{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Provider: "google", InputCostPer1K: 0.00125, OutputCostPer1K: 0.005, ContextWindow: 2000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 88, QualityScore: 92, Available: true},
				{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Provider: "google", InputCostPer1K: 0.000075, OutputCostPer1K: 0.0003, ContextWindow: 1000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 95, QualityScore: 85, Available: true},
				{ID: "gemini-1.5-flash-8b", Name: "Gemini 1.5 Flash 8B", Provider: "google", InputCostPer1K: 0.0000375, OutputCostPer1K: 0.00015, ContextWindow: 1000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 98, QualityScore: 82, Available: true},
				{ID: "gemini-pro", Name: "Gemini Pro", Provider: "google", InputCostPer1K: 0.00125, OutputCostPer1K: 0.005, ContextWindow: 32768, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 88, Available: true},
				{ID: "gemini-pro-vision", Name: "Gemini Pro Vision", Provider: "google", InputCostPer1K: 0.00125, OutputCostPer1K: 0.005, ContextWindow: 12288, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: false, LatencyScore: 82, QualityScore: 90, Available: true},
				{ID: "gemini-2.5-pro-preview", Name: "Gemini 2.5 Pro Preview", Provider: "google", InputCostPer1K: 0.00125, OutputCostPer1K: 0.01, ContextWindow: 2000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 95, Available: true},
			},
		},
		// Azure OpenAI (4 models)
		{
			ID: "provider-azure", Name: "Azure OpenAI", Category: "azure", APIType: "azure",
			BaseURL: "${AZURE_OPENAI_ENDPOINT}", Enabled: false, Healthy: false,
			Tier: "dedicated", RateLimit: 1000,
			Capabilities: []string{"streaming", "vision", "function_calling", "json_mode"},
			Models: []Model{
				{ID: "gpt-4o-azure", Name: "GPT-4o (Azure)", Provider: "azure", InputCostPer1K: 0.006, OutputCostPer1K: 0.018, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 82, QualityScore: 95, Available: true},
				{ID: "gpt-4-turbo-azure", Name: "GPT-4 Turbo (Azure)", Provider: "azure", InputCostPer1K: 0.012, OutputCostPer1K: 0.036, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 78, QualityScore: 95, Available: true},
				{ID: "gpt-35-turbo-azure", Name: "GPT-3.5 Turbo (Azure)", Provider: "azure", InputCostPer1K: 0.0005, OutputCostPer1K: 0.0015, ContextWindow: 16385, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 92, QualityScore: 80, Available: true},
				{ID: "o1-azure", Name: "o1 (Azure)", Provider: "azure", InputCostPer1K: 0.015, OutputCostPer1K: 0.06, ContextWindow: 128000, SupportsStreaming: false, SupportsFunctions: false, SupportsJSON: false, LatencyScore: 60, QualityScore: 98, Available: true},
			},
		},
		// AWS Bedrock (7 models)
		{
			ID: "provider-aws", Name: "AWS Bedrock", Category: "aws", APIType: "bedrock",
			BaseURL: "https://bedrock-runtime.us-east-1.amazonaws.com", Enabled: false, Healthy: false,
			Tier: "dedicated", RateLimit: 500,
			Region:       "us-east-1",
			Capabilities: []string{"streaming", "vision", "function_calling"},
			Models: []Model{
				{ID: "anthropic-claude-opus-3", Name: "Claude 3 Opus (Bedrock)", Provider: "aws", InputCostPer1K: 0.015, OutputCostPer1K: 0.075, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 75, QualityScore: 98, Available: true},
				{ID: "anthropic-claude-sonnet-3", Name: "Claude 3 Sonnet (Bedrock)", Provider: "aws", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 88, QualityScore: 95, Available: true},
				{ID: "anthropic-claude-haiku-3", Name: "Claude 3 Haiku (Bedrock)", Provider: "aws", InputCostPer1K: 0.00025, OutputCostPer1K: 0.00125, ContextWindow: 200000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 95, QualityScore: 88, Available: true},
				{ID: "meta-llama-3-1-70b", Name: "Llama 3.1 70B (Bedrock)", Provider: "aws", InputCostPer1K: 0.00265, OutputCostPer1K: 0.0035, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 85, QualityScore: 88, Available: true},
				{ID: "meta-llama-3-1-8b", Name: "Llama 3.1 8B (Bedrock)", Provider: "aws", InputCostPer1K: 0.00022, OutputCostPer1K: 0.00024, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 92, QualityScore: 78, Available: true},
				{ID: "mistral-large", Name: "Mistral Large (Bedrock)", Provider: "aws", InputCostPer1K: 0.002, OutputCostPer1K: 0.006, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 92, Available: true},
				{ID: "ai21-jamba-1-5-large", Name: "Jamba 1.5 Large (Bedrock)", Provider: "aws", InputCostPer1K: 0.002, OutputCostPer1K: 0.008, ContextWindow: 256000, SupportsStreaming: true, LatencyScore: 80, QualityScore: 85, Available: true},
			},
		},
		// Cohere (5 models)
		{
			ID: "provider-cohere", Name: "Cohere", Category: "cohere", APIType: "rest",
			BaseURL: "https://api.cohere.ai/v2", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 200,
			Capabilities: []string{"streaming", "function_calling", "json_mode", "rerank"},
			Models: []Model{
				{ID: "command-r-plus-08", Name: "Command R+ 08-2024", Provider: "cohere", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 82, QualityScore: 92, Available: true},
				{ID: "command-r-plus", Name: "Command R+", Provider: "cohere", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 82, QualityScore: 90, Available: true},
				{ID: "command-r7b-12-2024", Name: "Command R7B (12-2024)", Provider: "cohere", InputCostPer1K: 0.0005, OutputCostPer1K: 0.002, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 92, QualityScore: 85, Available: true},
				{ID: "command-r", Name: "Command R", Provider: "cohere", InputCostPer1K: 0.0005, OutputCostPer1K: 0.0015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 88, QualityScore: 85, Available: true},
				{ID: "command", Name: "Command", Provider: "cohere", InputCostPer1K: 0.00015, OutputCostPer1K: 0.0006, ContextWindow: 4096, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 92, QualityScore: 78, Available: true},
			},
		},
		// DeepSeek (5 models)
		{
			ID: "provider-deepseek", Name: "DeepSeek", Category: "deepseek", APIType: "openai",
			BaseURL: "https://api.deepseek.com/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 500,
			Capabilities: []string{"streaming", "function_calling"},
			Models: []Model{
				{ID: "deepseek-v4", Name: "DeepSeek V4", Provider: "deepseek", InputCostPer1K: 0.00014, OutputCostPer1K: 0.00048, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 88, QualityScore: 94, Available: true},
				{ID: "deepseek-chat", Name: "DeepSeek V3 Chat", Provider: "deepseek", InputCostPer1K: 0.0001, OutputCostPer1K: 0.00027, ContextWindow: 64000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 92, QualityScore: 88, Available: true},
				{ID: "deepseek-coder", Name: "DeepSeek Coder", Provider: "deepseek", InputCostPer1K: 0.00014, OutputCostPer1K: 0.00028, ContextWindow: 16000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 90, QualityScore: 90, Available: true},
				{ID: "deepseek-reasoner", Name: "DeepSeek R1", Provider: "deepseek", InputCostPer1K: 0.00055, OutputCostPer1K: 0.00219, ContextWindow: 64000, SupportsStreaming: true, SupportsFunctions: false, LatencyScore: 70, QualityScore: 95, Available: true},
				{ID: "deepseek-chat-latest", Name: "DeepSeek V3 (Latest)", Provider: "deepseek", InputCostPer1K: 0.00007, OutputCostPer1K: 0.00024, ContextWindow: 64000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 95, QualityScore: 90, Available: true},
			},
		},
		// Groq (6 models)
		{
			ID: "provider-groq", Name: "Groq", Category: "groq", APIType: "openai",
			BaseURL: "https://api.groq.com/openai/v1", Enabled: true, Healthy: true,
			Tier: "free", RateLimit: 60,
			Capabilities: []string{"streaming"},
			Models: []Model{
				{ID: "llama-3.1-70b-versatile", Name: "Llama 3.1 70B Versatile", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00079, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 98, QualityScore: 88, Available: true},
				{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00019, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 98, QualityScore: 78, Available: true},
				{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00024, ContextWindow: 32768, SupportsStreaming: true, LatencyScore: 95, QualityScore: 82, Available: true},
				{ID: "gemma2-9b-it", Name: "Gemma 2 9B", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00019, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 98, QualityScore: 75, Available: true},
				{ID: "llama-3-groq-70b-tool-use", Name: "Llama 3 Groq 70B (Tool)", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00079, ContextWindow: 8192, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 95, QualityScore: 90, Available: true},
				{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Provider: "groq", InputCostPer1K: 0, OutputCostPer1K: 0.00089, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 95, QualityScore: 92, Available: true},
			},
		},
		// Ollama Local (10 models)
		{
			ID: "provider-ollama", Name: "Ollama (Local)", Category: "ollama", APIType: "openai",
			BaseURL: "http://localhost:11434/v1", Enabled: true, Healthy: true,
			Tier: "free", RateLimit: 10000,
			Region:       "local",
			Capabilities: []string{"streaming"},
			Models: []Model{
				{ID: "llama3.3", Name: "Llama 3.3", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 88, QualityScore: 90, Available: true},
				{ID: "llama3.1", Name: "Llama 3.1", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 90, QualityScore: 85, Available: true},
				{ID: "llama3", Name: "Llama 3", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 92, QualityScore: 82, Available: true},
				{ID: "mistral", Name: "Mistral", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 92, QualityScore: 80, Available: true},
				{ID: "codellama", Name: "Code Llama", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 16384, SupportsStreaming: true, LatencyScore: 90, QualityScore: 85, Available: true},
				{ID: "phi3", Name: "Phi-3", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 4096, SupportsStreaming: true, LatencyScore: 95, QualityScore: 72, Available: true},
				{ID: "phi3.5-mini", Name: "Phi-3.5 Mini", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 4096, SupportsStreaming: true, LatencyScore: 95, QualityScore: 75, Available: true},
				{ID: "qwen2.5", Name: "Qwen 2.5", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 131072, SupportsStreaming: true, LatencyScore: 88, QualityScore: 88, Available: true},
				{ID: "nomic-embed-text", Name: "Nomic Embed Text", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: false, LatencyScore: 95, QualityScore: 78, Available: true},
				{ID: "granite3.3", Name: "Granite 3.3", Provider: "ollama", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 90, QualityScore: 85, Available: true},
			},
		},
		// Perplexity (4 models)
		{
			ID: "provider-perplexity", Name: "Perplexity", Category: "perplexity", APIType: "openai",
			BaseURL: "https://api.perplexity.ai", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 100,
			Capabilities: []string{"streaming", "function_calling"},
			Models: []Model{
				{ID: "sonar-pro", Name: "Sonar Pro", Provider: "perplexity", InputCostPer1K: 0.003, OutputCostPer1K: 0.003, ContextWindow: 127072, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 92, Available: true},
				{ID: "sonar", Name: "Sonar", Provider: "perplexity", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 127072, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 92, QualityScore: 85, Available: true},
				{ID: "sonar-pro-online", Name: "Sonar Pro (Online)", Provider: "perplexity", InputCostPer1K: 0.005, OutputCostPer1K: 0.005, ContextWindow: 127072, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 80, QualityScore: 95, Available: true},
				{ID: "sonar-reasoning-pro", Name: "Sonar Reasoning Pro", Provider: "perplexity", InputCostPer1K: 0.005, OutputCostPer1K: 0.02, ContextWindow: 127072, SupportsStreaming: true, SupportsFunctions: false, LatencyScore: 75, QualityScore: 95, Available: true},
			},
		},
		// Mistral AI (6 models)
		{
			ID: "provider-mistral", Name: "Mistral AI", Category: "mistral", APIType: "openai",
			BaseURL: "https://api.mistral.ai/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 200,
			Capabilities: []string{"streaming", "function_calling", "json_mode"},
			Models: []Model{
				{ID: "mistral-large", Name: "Mistral Large", Provider: "mistral", InputCostPer1K: 0.002, OutputCostPer1K: 0.006, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 92, Available: true},
				{ID: "mistral-large-latest", Name: "Mistral Large (Latest)", Provider: "mistral", InputCostPer1K: 0.002, OutputCostPer1K: 0.006, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 85, QualityScore: 94, Available: true},
				{ID: "mistral-small", Name: "Mistral Small", Provider: "mistral", InputCostPer1K: 0.0002, OutputCostPer1K: 0.0006, ContextWindow: 32000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 92, QualityScore: 82, Available: true},
				{ID: "mistral-nemo", Name: "Mistral Nemo", Provider: "mistral", InputCostPer1K: 0.00015, OutputCostPer1K: 0.00015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, SupportsJSON: true, LatencyScore: 95, QualityScore: 78, Available: true},
				{ID: "codestral", Name: "Codestral", Provider: "mistral", InputCostPer1K: 0.0002, OutputCostPer1K: 0.0006, ContextWindow: 32000, SupportsStreaming: true, LatencyScore: 90, QualityScore: 88, Available: true},
				{ID: "codestral-mamba", Name: "Codestral Mamba", Provider: "mistral", InputCostPer1K: 0.00015, OutputCostPer1K: 0.00015, ContextWindow: 256000, SupportsStreaming: true, LatencyScore: 92, QualityScore: 85, Available: true},
			},
		},
		// HuggingFace (5 models)
		{
			ID: "provider-huggingface", Name: "HuggingFace", Category: "huggingface", APIType: "rest",
			BaseURL: "https://api-inference.huggingface.co/v1", Enabled: true, Healthy: true,
			Tier: "free", RateLimit: 30,
			Capabilities: []string{"streaming", "vision"},
			Models: []Model{
				{ID: "meta-llama-3.1-70b", Name: "Llama 3.1 70B (HF)", Provider: "huggingface", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 60, QualityScore: 90, Available: true},
				{ID: "meta-llama-3.1-8b", Name: "Llama 3.1 8B (HF)", Provider: "huggingface", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 75, QualityScore: 80, Available: true},
				{ID: "mistralai-mistral-7b", Name: "Mistral 7B (HF)", Provider: "huggingface", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 70, QualityScore: 75, Available: true},
				{ID: "microsoft-phi-3-mini", Name: "Phi-3 Mini (HF)", Provider: "huggingface", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 4096, SupportsStreaming: true, LatencyScore: 80, QualityScore: 72, Available: true},
				{ID: "google-gemma-2-9b", Name: "Gemma 2 9B (HF)", Provider: "huggingface", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 75, QualityScore: 78, Available: true},
			},
		},
		// Replicate (4 models)
		{
			ID: "provider-replicate", Name: "Replicate", Category: "replicate", APIType: "rest",
			BaseURL: "https://api.replicate.com/v1", Enabled: true, Healthy: true,
			Tier: "pay-per-use", RateLimit: 100,
			Capabilities: []string{"streaming"},
			Models: []Model{
				{ID: "meta-llama-3-70b", Name: "Llama 3 70B (Replicate)", Provider: "replicate", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 55, QualityScore: 90, Available: true},
				{ID: "meta-llama-3-8b", Name: "Llama 3 8B (Replicate)", Provider: "replicate", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 8192, SupportsStreaming: true, LatencyScore: 70, QualityScore: 80, Available: true},
				{ID: "mistralai-mixtral-8x22b", Name: "Mixtral 8x22B (Replicate)", Provider: "replicate", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 65536, SupportsStreaming: true, LatencyScore: 50, QualityScore: 88, Available: true},
				{ID: "anthropic-claude-3.5-sonnet", Name: "Claude 3.5 Sonnet (Replicate)", Provider: "replicate", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 200000, SupportsStreaming: true, LatencyScore: 60, QualityScore: 96, Available: true},
			},
		},
		// Together AI (4 models)
		{
			ID: "provider-together", Name: "Together AI", Category: "together", APIType: "openai",
			BaseURL: "https://api.together.xyz/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 100,
			Capabilities: []string{"streaming", "function_calling"},
			Models: []Model{
				{ID: "meta-llama-3.3-70b", Name: "Llama 3.3 70B (Together)", Provider: "together", InputCostPer1K: 0.00024, OutputCostPer1K: 0.00024, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 75, QualityScore: 92, Available: true},
				{ID: "mistralai-mixtral-8x22b", Name: "Mixtral 8x22B (Together)", Provider: "together", InputCostPer1K: 0.00024, OutputCostPer1K: 0.00024, ContextWindow: 65536, SupportsStreaming: true, LatencyScore: 70, QualityScore: 88, Available: true},
				{ID: "qwen-2.5-72b", Name: "Qwen 2.5 72B (Together)", Provider: "together", InputCostPer1K: 0.00036, OutputCostPer1K: 0.00036, ContextWindow: 32768, SupportsStreaming: true, LatencyScore: 65, QualityScore: 88, Available: true},
				{ID: "deepseek-v3", Name: "DeepSeek V3 (Together)", Provider: "together", InputCostPer1K: 0.00008, OutputCostPer1K: 0.00024, ContextWindow: 64000, SupportsStreaming: true, LatencyScore: 80, QualityScore: 90, Available: true},
			},
		},
		// Meta AI (8 models)
		{
			ID: "provider-meta", Name: "Meta AI", Category: "meta", APIType: "openai",
			BaseURL: "https://api.meta.ai/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 200,
			Capabilities: []string{"streaming", "vision", "function_calling"},
			Models: []Model{
				{ID: "llama-4-405b-instruct", Name: "Llama 4 405B Instruct", Provider: "meta", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 50, QualityScore: 96, Available: true},
				{ID: "llama-4-70b-instruct", Name: "Llama 4 70B Instruct", Provider: "meta", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 70, QualityScore: 92, Available: true},
				{ID: "llama-4-17b-instruct", Name: "Llama 4 17B Instruct", Provider: "meta", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 80, QualityScore: 88, Available: true},
				{ID: "llama-3.3-70b-instruct", Name: "Llama 3.3 70B Instruct", Provider: "meta", InputCostPer1K: 0.0002, OutputCostPer1K: 0.0002, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 75, QualityScore: 92, Available: true},
				{ID: "llama-3.2-90b-vision-instruct", Name: "Llama 3.2 90B Vision", Provider: "meta", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 65, QualityScore: 90, Available: true},
				{ID: "llama-3.2-11b-vision-instruct", Name: "Llama 3.2 11B Vision", Provider: "meta", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 78, QualityScore: 86, Available: true},
				{ID: "llama-3.1-405b-instruct", Name: "Llama 3.1 405B Instruct", Provider: "meta", InputCostPer1K: 0.00035, OutputCostPer1K: 0.00035, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 55, QualityScore: 94, Available: true},
				{ID: "llama-3.1-8b-instruct", Name: "Llama 3.1 8B Instruct", Provider: "meta", InputCostPer1K: 0.0002, OutputCostPer1K: 0.0002, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 85, Available: true},
			},
		},
		// Fireworks AI (4 models)
		{
			ID: "provider-fireworks", Name: "Fireworks AI", Category: "fireworks", APIType: "openai",
			BaseURL: "https://api.fireworks.ai/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 100,
			Capabilities: []string{"streaming", "function_calling"},
			Models: []Model{
				{ID: "llama-3.1-405b", Name: "Llama 3.1 405B (Fireworks)", Provider: "fireworks", InputCostPer1K: 0.0016, OutputCostPer1K: 0.0016, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 55, QualityScore: 95, Available: true},
				{ID: "mixtral-8x22b", Name: "Mixtral 8x22B (Fireworks)", Provider: "fireworks", InputCostPer1K: 0.00024, OutputCostPer1K: 0.00024, ContextWindow: 65536, SupportsStreaming: true, LatencyScore: 75, QualityScore: 88, Available: true},
				{ID: "qwen2-72b", Name: "Qwen2 72B (Fireworks)", Provider: "fireworks", InputCostPer1K: 0.00036, OutputCostPer1K: 0.00036, ContextWindow: 32768, SupportsStreaming: true, LatencyScore: 70, QualityScore: 88, Available: true},
				{ID: "llama-3.1-8b", Name: "Llama 3.1 8B (Fireworks)", Provider: "fireworks", InputCostPer1K: 0.0002, OutputCostPer1K: 0.0002, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 90, QualityScore: 80, Available: true},
			},
		},
		// Vertex AI (2 models)
		{
			ID: "provider-vertex", Name: "Vertex AI", Category: "vertex", APIType: "vertex",
			BaseURL: "${VERTEX_ENDPOINT}", Enabled: false, Healthy: false,
			Tier: "dedicated", RateLimit: 500,
			Region:       "us-central1",
			Capabilities: []string{"streaming", "vision", "function_calling"},
			Models: []Model{
				{ID: "gemini-2.0-flash-exp-vertex", Name: "Gemini 2.0 Flash (Vertex)", Provider: "vertex", InputCostPer1K: 0, OutputCostPer1K: 0, ContextWindow: 1000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 95, QualityScore: 90, Available: true},
				{ID: "gemini-1.5-pro-002-vertex", Name: "Gemini 1.5 Pro (Vertex)", Provider: "vertex", InputCostPer1K: 0.00125, OutputCostPer1K: 0.005, ContextWindow: 2000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 88, QualityScore: 92, Available: true},
				{ID: "gemini-2.5-pro-preview-vertex", Name: "Gemini 2.5 Pro (Vertex)", Provider: "vertex", InputCostPer1K: 0.00125, OutputCostPer1K: 0.01, ContextWindow: 2000000, SupportsStreaming: true, SupportsVision: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 95, Available: true},
			},
		},
		// AI21 (4 models)
		{
			ID: "provider-ai21", Name: "AI21", Category: "ai21", APIType: "openai",
			BaseURL: "https://api.ai21.com/v1", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 100,
			Capabilities: []string{"streaming", "function_calling"},
			Models: []Model{
				{ID: "jamba-1.5-large", Name: "Jamba 1.5 Large", Provider: "ai21", InputCostPer1K: 0.002, OutputCostPer1K: 0.008, ContextWindow: 256000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 70, QualityScore: 90, Available: true},
				{ID: "jamba-1.5-mini", Name: "Jamba 1.5 Mini", Provider: "ai21", InputCostPer1K: 0.00015, OutputCostPer1K: 0.0002, ContextWindow: 256000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 85, Available: true},
				{ID: "jamba-1-8b", Name: "Jamba 1 8B Instruct", Provider: "ai21", InputCostPer1K: 0.00013, OutputCostPer1K: 0.00013, ContextWindow: 256000, SupportsStreaming: true, LatencyScore: 90, QualityScore: 80, Available: true},
				{ID: "jamba-1-405b", Name: "Jamba 1 405B Instruct", Provider: "ai21", InputCostPer1K: 0.001, OutputCostPer1K: 0.001, ContextWindow: 256000, SupportsStreaming: true, LatencyScore: 55, QualityScore: 92, Available: true},
			},
		},
		// Cohere (5 models)
		{
			ID: "provider-cohere", Name: "Cohere", Category: "cohere", APIType: "rest",
			BaseURL: "https://api.cohere.ai/v2", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 200,
			Capabilities: []string{"streaming", "function_calling", "reranking"},
			Models: []Model{
				{ID: "command-r-plus-08-2024", Name: "Command R+ (Aug 2024)", Provider: "cohere", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 60, QualityScore: 94, Available: true},
				{ID: "command-r7b-12-2024", Name: "Command R7B (Dec 2024)", Provider: "cohere", InputCostPer1K: 0.0005, OutputCostPer1K: 0.0015, ContextWindow: 128000, SupportsStreaming: true, SupportsFunctions: true, LatencyScore: 85, QualityScore: 88, Available: true},
				{ID: "command", Name: "Command", Provider: "cohere", InputCostPer1K: 0.001, OutputCostPer1K: 0.002, ContextWindow: 4096, SupportsStreaming: true, LatencyScore: 92, QualityScore: 80, Available: true},
				{ID: "command-light", Name: "Command Light", Provider: "cohere", InputCostPer1K: 0.0003, OutputCostPer1K: 0.0006, ContextWindow: 4096, SupportsStreaming: true, LatencyScore: 95, QualityScore: 75, Available: true},
				{ID: "embed-v4", Name: "Embed V4", Provider: "cohere", InputCostPer1K: 0.0001, OutputCostPer1K: 0, ContextWindow: 4096, LatencyScore: 98, QualityScore: 90, Available: true},
			},
		},
		// Perplexity (4 models)
		{
			ID: "provider-perplexity", Name: "Perplexity", Category: "perplexity", APIType: "openai",
			BaseURL: "https://api.perplexity.ai", Enabled: true, Healthy: true,
			Tier: "shared", RateLimit: 200,
			Capabilities: []string{"streaming", "vision"},
			Models: []Model{
				{ID: "sonar-pro", Name: "Sonar Pro", Provider: "perplexity", InputCostPer1K: 0.003, OutputCostPer1K: 0.015, ContextWindow: 128000, SupportsStreaming: true, SupportsVision: true, LatencyScore: 65, QualityScore: 92, Available: true},
				{ID: "sonar", Name: "Sonar", Provider: "perplexity", InputCostPer1K: 0.001, OutputCostPer1K: 0.001, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 80, QualityScore: 85, Available: true},
				{ID: "sonar-reasoning-pro", Name: "Sonar Reasoning Pro", Provider: "perplexity", InputCostPer1K: 0.005, OutputCostPer1K: 0.02, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 55, QualityScore: 95, Available: true},
				{ID: "sonar-reasoning", Name: "Sonar Reasoning", Provider: "perplexity", InputCostPer1K: 0.001, OutputCostPer1K: 0.005, ContextWindow: 128000, SupportsStreaming: true, LatencyScore: 75, QualityScore: 90, Available: true},
			},
		},
	}

	for _, prov := range providers {
		p.providers[prov.ID] = prov
	}
}

// Provider represents an AI provider configuration
type Provider struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	APIType      string    `json:"api_type"`
	BaseURL      string    `json:"base_url"`
	Enabled      bool      `json:"enabled"`
	Healthy      bool      `json:"healthy"`
	Models       []Model   `json:"models"`
	Capabilities []string  `json:"capabilities"`
	Region       string    `json:"region,omitempty"`
	Tier         string    `json:"tier"`
	RateLimit    int       `json:"rate_limit"`
	LastHealth   time.Time `json:"last_health,omitempty"`
}

// Model represents a model configuration
type Model struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Provider          string  `json:"provider"`
	InputCostPer1K    float64 `json:"input_cost_per_1k"`
	OutputCostPer1K   float64 `json:"output_cost_per_1k"`
	ContextWindow     int     `json:"context_window"`
	MaxOutputTokens   int     `json:"max_output_tokens,omitempty"`
	SupportsStreaming bool    `json:"supports_streaming"`
	SupportsVision    bool    `json:"supports_vision"`
	SupportsFunctions bool    `json:"supports_functions"`
	SupportsJSON      bool    `json:"supports_json"`
	LatencyScore      float64 `json:"latency_score"`
	QualityScore      float64 `json:"quality_score"`
	Available         bool    `json:"available"`
}

// ListProviders returns all providers
func (p *ProviderEngine) ListProviders() []*Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Provider, 0, len(p.providers))
	for _, prov := range p.providers {
		result = append(result, prov)
	}
	return result
}

// GetProvider returns a specific provider
func (p *ProviderEngine) GetProvider(id string) *Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.providers[id]
}

// ListModels returns all models from all enabled providers
func (p *ProviderEngine) ListModels() []Model {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var models []Model
	for _, prov := range p.providers {
		if prov.Enabled && prov.Healthy {
			for i := range prov.Models {
				if prov.Models[i].Available {
					models = append(models, prov.Models[i])
				}
			}
		}
	}
	return models
}

// GetModel returns a specific model by ID
func (p *ProviderEngine) GetModel(modelID string) *Model {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, prov := range p.providers {
		for i := range prov.Models {
			if prov.Models[i].ID == modelID {
				return &prov.Models[i]
			}
		}
	}
	return nil
}

// EnableProvider enables a provider
func (p *ProviderEngine) EnableProvider(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	prov, exists := p.providers[id]
	if !exists {
		return nil
	}
	prov.Enabled = true
	return nil
}

// DisableProvider disables a provider
func (p *ProviderEngine) DisableProvider(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	prov, exists := p.providers[id]
	if !exists {
		return nil
	}
	prov.Enabled = false
	return nil
}

// GetProvidersByCategory returns providers filtered by category
func (p *ProviderEngine) GetProvidersByCategory(category string) []*Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Provider
	for _, prov := range p.providers {
		if prov.Category == category {
			result = append(result, prov)
		}
	}
	return result
}

// GetFreeProviders returns providers with free tier
func (p *ProviderEngine) GetFreeProviders() []*Provider {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Provider
	for _, prov := range p.providers {
		if prov.Enabled && prov.Tier == "free" {
			result = append(result, prov)
		}
	}
	return result
}

// GetStats returns provider statistics
func (p *ProviderEngine) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := len(p.providers)
	enabled := 0
	healthy := 0
	totalModels := 0

	for _, prov := range p.providers {
		if prov.Enabled {
			enabled++
		}
		if prov.Healthy {
			healthy++
		}
		totalModels += len(prov.Models)
	}

	return map[string]interface{}{
		"total_providers":   total,
		"enabled_providers": enabled,
		"healthy_providers": healthy,
		"total_models":      totalModels,
	}
}
