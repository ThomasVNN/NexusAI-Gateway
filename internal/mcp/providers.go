package mcp

import (
	"context"
)

// Provider represents an MCP provider
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderGoogle    Provider = "google"
	ProviderMeta      Provider = "meta"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// ProviderConfig contains provider-specific configuration
type ProviderConfig struct {
	Name        string
	APIEndpoint string
	AuthHeader  string
	Tools       []Tool
}

// MCPServer interface for MCP server operations
type MCPServer interface {
	Name() string
	Provider() Provider
	Initialize(ctx context.Context, config ProviderConfig) error
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
	Shutdown(ctx context.Context) error
}

// AnthropicMCPServer implements MCP for Anthropic
type AnthropicMCPServer struct {
	config ProviderConfig
	tools  []Tool
}

// GoogleMCPServer implements MCP for Google
type GoogleMCPServer struct {
	config ProviderConfig
	tools  []Tool
}

// MetaMCPServer implements MCP for Meta
type MetaMCPServer struct {
	config ProviderConfig
	tools  []Tool
}

// MultiProviderRegistry manages multiple MCP servers
type MultiProviderRegistry struct {
	servers map[Provider]MCPServer
}

// NewMultiProviderRegistry creates a new registry
func NewMultiProviderRegistry() *MultiProviderRegistry {
	return &MultiProviderRegistry{
		servers: make(map[Provider]MCPServer),
	}
}

// Register adds a server to the registry
func (r *MultiProviderRegistry) Register(server MCPServer) {
	r.servers[server.Provider()] = server
}

// Get returns a server by provider
func (r *MultiProviderRegistry) Get(p Provider) (MCPServer, bool) {
	server, ok := r.servers[p]
	return server, ok
}

// ListProviders returns all registered providers
func (r *MultiProviderRegistry) ListProviders() []Provider {
	providers := make([]Provider, 0, len(r.servers))
	for p := range r.servers {
		providers = append(providers, p)
	}
	return providers
}

// InitializeAll initializes all registered servers with their configs
func (r *MultiProviderRegistry) InitializeAll(ctx context.Context, configs map[Provider]ProviderConfig) error {
	for p, server := range r.servers {
		config, ok := configs[p]
		if !ok {
			config = ProviderConfig{Name: string(p)}
		}
		if err := server.Initialize(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

// ShutdownAll shuts down all servers
func (r *MultiProviderRegistry) ShutdownAll(ctx context.Context) error {
	for _, server := range r.servers {
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
