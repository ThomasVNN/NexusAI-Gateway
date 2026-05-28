package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/mcp"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	storage "github.com/ThomasVNN/NexusAI-Gateway/internal/storage/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/storage/memory"
)

var startTime = time.Now()

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
		chatHandler = handler.NewChatHandler(&keyRepo, &usageRepo, piiEngine, cfg.EnableSandboxFallback)
	} else {
		// If DB is down, chat completions dynamically fall back to in-memory quota tracking
		chatHandler = handler.NewChatHandler(memStore, memStore, piiEngine, cfg.EnableSandboxFallback)
	}

	modelHandler := handler.NewModelHandler(db)
	mcpHandler := mcp.NewHandler(piiEngine)

	// Admin and system diagnostics mapping
	var adminHandler *handler.AdminHandler
	if isDbHealthy {
		adminHandler = handler.NewAdminHandler(&keyRepo, &usageRepo, memStore, true, cfg.InitialPassword)
	} else {
		adminHandler = handler.NewAdminHandler(nil, nil, memStore, false, cfg.InitialPassword)
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

	// Auth and login compatibility
	mux.HandleFunc("/api/auth/login", adminHandler.HandleLogin)
	mux.HandleFunc("/api/settings/require-login", adminHandler.HandleRequireLogin)

	// Diagnostics & Observability endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"UP"}`))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "disconnected"
		
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err == nil {
				dbStatus = "connected"
			} else {
				dbStatus = "degraded"
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"status":"UP","database":"%s"}`, dbStatus)))
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		dbConnected := 0
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err == nil {
				dbConnected = 1
			}
		}

		fmt.Fprintf(w, "# HELP nexusai_gateway_database_connected Database connection status (1 = connected, 0 = disconnected).\n")
		fmt.Fprintf(w, "# TYPE nexusai_gateway_database_connected gauge\n")
		fmt.Fprintf(w, "nexusai_gateway_database_connected %d\n", dbConnected)
		
		fmt.Fprintf(w, "# HELP nexusai_gateway_uptime_seconds Uptime of the gateway in seconds.\n")
		fmt.Fprintf(w, "# TYPE nexusai_gateway_uptime_seconds gauge\n")
		fmt.Fprintf(w, "nexusai_gateway_uptime_seconds %.0f\n", time.Since(startTime).Seconds())
	})

	// Single Page Application static server
	RegisterStaticRoutes(mux)

	// Wrap routing stack in our production-grade middleware layers
	return WithRecovery(
		WithCorrelationID(
			WithStructuredLogging(
				WithRateLimiting(mux),
			),
		),
	)
}
