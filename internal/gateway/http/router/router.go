package router

import (
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/mcp"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	storage "github.com/ThomasVNN/NexusAI-Gateway/internal/storage/postgres"
)

// New constructs the primary routing multiplexer for http traffic
func New(db *postgres.DB, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// Initialize repositories
	keyRepo := storage.NewKeyRepository(db)
	usageRepo := storage.NewUsageRepository(db)
	piiEngine := privacy.NewEngine()

	// Handler registrations
	chatHandler := handler.NewChatHandler(keyRepo, usageRepo, piiEngine)
	modelHandler := handler.NewModelHandler(db)
	mcpHandler := mcp.NewHandler(piiEngine)
	adminHandler := handler.NewAdminHandler(keyRepo, usageRepo)

	// OpenAI endpoints
	mux.HandleFunc("/v1/chat/completions", chatHandler.ServeHTTP)
	mux.HandleFunc("/v1/models", modelHandler.ServeHTTP)

	// Model Context Protocol (MCP) Stream & Message Endpoints
	mux.HandleFunc("/api/mcp/stream", mcpHandler.ServeHTTP)
	mux.HandleFunc("/api/mcp/message", mcpHandler.ServeHTTP)

	// Admin API Endpoints
	mux.HandleFunc("/api/admin/keys", adminHandler.HandleKeys)
	mux.HandleFunc("/api/admin/usage", adminHandler.HandleUsage)

	// Single Page Application static server placeholder
	RegisterStaticRoutes(mux)

	return mux
}
