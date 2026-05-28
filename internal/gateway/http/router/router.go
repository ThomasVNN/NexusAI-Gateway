package router

import (
	"net/http"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/mcp"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	storage "github.com/ThomasVNN/NexusAI-Gateway/internal/storage/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/storage/memory"
)

// New constructs the primary routing multiplexer for http traffic
func New(db *postgres.DB, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// 1. Initialize repositories
	isDbHealthy := db != nil
	var keyRepo storage.KeyRepository
	var usageRepo storage.UsageRepository
	
	if isDbHealthy {
		keyRepo = *storage.NewKeyRepository(db)
		usageRepo = *storage.NewUsageRepository(db)
	}

	// 2. Initialize in-memory fail-safe Store
	memStore := memory.NewStore()
	piiEngine := privacy.NewEngine()

	// 3. Handler registrations
	var chatHandler *handler.ChatHandler
	if isDbHealthy {
		chatHandler = handler.NewChatHandler(&keyRepo, &usageRepo, piiEngine)
	} else {
		// If DB is down, chat completions dynamically fall back to in-memory quota tracking
		chatHandler = handler.NewChatHandler(memStore, memStore, piiEngine)
	}

	modelHandler := handler.NewModelHandler(db)
	mcpHandler := mcp.NewHandler(piiEngine)

	// Admin and system diagnostics mapping
	var adminHandler *handler.AdminHandler
	if isDbHealthy {
		adminHandler = handler.NewAdminHandler(&keyRepo, &usageRepo, memStore, true)
	} else {
		adminHandler = handler.NewAdminHandler(nil, nil, memStore, false)
	}

	// OpenAI endpoints
	mux.HandleFunc("/v1/chat/completions", chatHandler.ServeHTTP)
	mux.HandleFunc("/v1/models", modelHandler.ServeHTTP)

	// Model Context Protocol (MCP) Stream & Message Endpoints
	mux.HandleFunc("/api/mcp/stream", mcpHandler.ServeHTTP)
	mux.HandleFunc("/api/mcp/message", mcpHandler.ServeHTTP)

	// Admin API Endpoints - mapped exactly to OmniRoute features
	mux.HandleFunc("/api/admin/keys", adminHandler.HandleKeys)
	mux.HandleFunc("/api/admin/usage", adminHandler.HandleUsage)
	mux.HandleFunc("/api/admin/logs", adminHandler.HandleLogs)
	
	// OmniRoute UI Compatibility API Endpoints
	mux.HandleFunc("/api/providers", adminHandler.HandleProviders)
	mux.HandleFunc("/api/models", adminHandler.HandleModels)
	mux.HandleFunc("/api/provider-metrics", adminHandler.HandleUsage)
	mux.HandleFunc("/api/system/version", adminHandler.HandleSystemVersion)

	// Single Page Application static server
	RegisterStaticRoutes(mux)

	return mux
}
