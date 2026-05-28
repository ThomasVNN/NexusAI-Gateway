package router

import (
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/mcp"
)

// New constructs the primary routing multiplexer for http traffic
func New(db *postgres.DB, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// Handler registrations
	chatHandler := handler.NewChatHandler(db)
	modelHandler := handler.NewModelHandler(db)
	mcpHandler := mcp.NewHandler()

	// OpenAI endpoints
	mux.HandleFunc("/v1/chat/completions", chatHandler.ServeHTTP)
	mux.HandleFunc("/v1/models", modelHandler.ServeHTTP)

	// Model Context Protocol (MCP) Stream Endpoint
	mux.HandleFunc("/api/mcp/stream", mcpHandler.ServeHTTP)

	// Single Page Application static server placeholder
	RegisterStaticRoutes(mux)

	return mux
}
