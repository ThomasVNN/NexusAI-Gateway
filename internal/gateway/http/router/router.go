package router

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/billing"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/channel"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/handler"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/mcp"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/integration"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/log"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/observability"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/ratelimit"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/runtime"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/storage/memory"
	storage "github.com/ThomasVNN/NexusAI-Gateway/internal/storage/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/token"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/user"
)

var startTime = time.Now()

// New constructs the primary routing multiplexer for http traffic
func New(db *postgres.DB, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// 1. Initialize repositories
	isDbHealthy := db != nil
	var keyRepo *storage.KeyRepository
	var usageRepo *storage.UsageRepository

	if isDbHealthy {
		keyRepo = storage.NewKeyRepository(db)
		usageRepo = storage.NewUsageRepository(db)
	}

	// 2. Initialize in-memory fail-safe Store
	memStore := memory.NewStore()
	piiEngine := privacy.NewEngine()

	// 3. Construct PipelineExecutor and Handler registrations
	var chatHandler *handler.ChatHandler

	tenantResolver := tenancy.NewDefaultTenantResolver()
	knowledgeClient := integration.NewDefaultKnowledgeClient()
	skillsClient := integration.NewDefaultSkillsClient()
	modelPlatform := integration.NewDefaultModelPlatformClient()

	if isDbHealthy {
		authenticator := auth.NewAPIKeyAuthenticator(keyRepo)
		pipelineExecutor := runtime.NewPipelineExecutor(
			authenticator,
			tenantResolver,
			piiEngine,
			knowledgeClient,
			skillsClient,
			modelPlatform,
		)
		chatHandler = handler.NewChatHandler(keyRepo, usageRepo, piiEngine, cfg.EnableSandboxFallback, pipelineExecutor)
	} else {
		// If DB is down, chat completions dynamically fall back to in-memory quota tracking
		authenticator := auth.NewAPIKeyAuthenticator(memStore)
		pipelineExecutor := runtime.NewPipelineExecutor(
			authenticator,
			tenantResolver,
			piiEngine,
			knowledgeClient,
			skillsClient,
			modelPlatform,
		)
		chatHandler = handler.NewChatHandler(memStore, memStore, piiEngine, cfg.EnableSandboxFallback, pipelineExecutor)
	}

	modelHandler := handler.NewModelHandler(db)
	mcpHandler := mcp.NewHandler(piiEngine)

	// Admin and system diagnostics mapping
	var adminHandler *handler.AdminHandler
	if isDbHealthy {
		adminHandler = handler.NewAdminHandler(keyRepo, usageRepo, memStore, true, cfg.InitialPassword)
	} else {
		adminHandler = handler.NewAdminHandler(nil, nil, memStore, false, cfg.InitialPassword)
	}

	// 4. Initialize rate limiting
	rateLimitConfig := ratelimit.DefaultRateLimitConfig()
	rateLimitConfig.RedisURL = cfg.RedisURL

	quotaStorage, err := ratelimit.CreateRedisStorage(cfg.RedisURL)
	if err != nil {
		slog.Warn("Failed to initialize Redis rate limiting storage, using in-memory fallback", slog.Any("error", err))
		quotaStorage = ratelimit.NewInMemoryStorage()
	}

	quotaManager := ratelimit.NewQuotaManager(quotaStorage, rateLimitConfig)
	rateLimitMiddleware := ratelimit.NewRateLimitMiddleware(quotaManager, tenantResolver, rateLimitConfig)
	rateLimitHandler := handler.NewGetRateLimitsHandler(handler.ToQuotaManagerInterface(quotaManager))

	// Initialize new-api services (only if DB is healthy)
	var apiHandler *handler.APIHandler
	var chService *channel.Service
	var tgService *token.Service
	var uService *user.Service
	var logService *log.Service
	var billingService *billing.Service

	if isDbHealthy {
		// Channel management
		chRepo := channel.NewRepository(db.DB)
		chService = channel.NewService(chRepo)

		// Token group management
		tgRepo := token.NewRepository(db.DB)
		tgService = token.NewService(tgRepo)

		// User management
		uRepo := user.NewRepository(db.DB)
		uService = user.NewService(uRepo)

		// Request logging
		logRepo := log.NewRepository(db.DB)
		logService = log.NewService(logRepo)

		// Billing
		billingRepo := billing.NewRepository(db.DB)
		billingService = billing.NewService(billingRepo)

		// Initialize API handler with all services
		apiHandler = handler.NewAPIHandler(chService, tgService, uService, logService, billingService)
	}

	// OpenAI endpoints
	mux.HandleFunc("POST /v1/chat/completions", chatHandler.ServeHTTP)
	mux.HandleFunc("GET /v1/models", modelHandler.ServeHTTP)

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

	// Rate Limiting API Endpoints
	mux.HandleFunc("/v1/rate-limits/status", rateLimitHandler.ServeHTTP)
	mux.HandleFunc("/v1/rate-limits/tiers", rateLimitHandler.ServeHTTP)
	mux.HandleFunc("/v1/rate-limits/usage", rateLimitHandler.ServeHTTP)
	mux.HandleFunc("/v1/rate-limits/reset", rateLimitHandler.ServeHTTP)
	mux.HandleFunc("/v1/rate-limits/quota", rateLimitHandler.ServeHTTP)
	mux.HandleFunc("/v1/rate-limits/health", rateLimitHandler.ServeHTTP)

	// new-api: Channel Management Endpoints
	if apiHandler != nil {
		mux.HandleFunc("/api/channels", apiHandler.HandleChannels)
		mux.HandleFunc("/api/channels/", apiHandler.HandleChannel)
		mux.HandleFunc("/api/channels/{id}/test", apiHandler.HandleChannelTest)

		// Token Group Endpoints
		mux.HandleFunc("/api/token-groups", apiHandler.HandleTokenGroups)
		mux.HandleFunc("/api/token-groups/", apiHandler.HandleTokenGroup)

		// User Management Endpoints
		mux.HandleFunc("/api/users", apiHandler.HandleUsers)
		mux.HandleFunc("/api/users/", apiHandler.HandleUser)

		// Analytics Endpoints
		mux.HandleFunc("/api/analytics/overview", apiHandler.HandleAnalyticsOverview)
		mux.HandleFunc("/api/analytics/models", apiHandler.HandleAnalyticsModels)
		mux.HandleFunc("/api/analytics/channels", apiHandler.HandleAnalyticsChannels)

		// Request Log Endpoints
		mux.HandleFunc("/api/logs", apiHandler.HandleLogs)
		mux.HandleFunc("/api/logs/", apiHandler.HandleLog)

		// Billing Endpoints
		mux.HandleFunc("/api/billing", apiHandler.HandleBilling)
	}

	// Cost Tracking Endpoints (REC-P1-3)
	costHandler := handler.NewCostHandler()
	mux.HandleFunc("/api/billing/summary", costHandler.GetCostSummary)
	mux.HandleFunc("/api/billing/pricing", costHandler.GetModelPricing)
	mux.HandleFunc("/api/billing/free-tier", costHandler.GetFreeTierUsage)

	// Token Compression Endpoints (REC-P2-1)
	mux.HandleFunc("/api/v1/compression/config", costHandler.GetCompressionConfig)
	mux.HandleFunc("/api/v1/compression/config/set", costHandler.SetCompressionConfig)
	mux.HandleFunc("/api/v1/compression/stats", costHandler.GetCompressionStats)
	mux.HandleFunc("/api/v1/compression/methods", costHandler.GetCompressionMethods)

	// Routing Strategies Endpoints (REC-P1-4)
	routingHandler := handler.NewRoutingHandler()
	mux.HandleFunc("/api/v1/strategies", routingHandler.GetStrategies)
	mux.HandleFunc("/api/v1/routes", routingHandler.GetRoutes)
	mux.HandleFunc("/api/v1/routes/create", routingHandler.CreateRoute)
	mux.HandleFunc("/api/v1/routes/update", routingHandler.UpdateRoute)
	mux.HandleFunc("/api/v1/routes/delete", routingHandler.DeleteRoute)

	// Auto-Combo Engine Endpoints (REC-P1-5)
	mux.HandleFunc("/api/v1/combos", routingHandler.GetCombos)
	mux.HandleFunc("/api/v1/combos/create", routingHandler.CreateCombo)
	mux.HandleFunc("/api/v1/combos/update", routingHandler.UpdateCombo)
	mux.HandleFunc("/api/v1/combos/delete", routingHandler.DeleteCombo)
	mux.HandleFunc("/api/v1/combos/score", routingHandler.ScoreModels)
	mux.HandleFunc("/api/v1/combos/fallback-tiers", routingHandler.GetFallbackTiers)
	mux.HandleFunc("/api/v1/combos/config", routingHandler.GetComboConfig)
	mux.HandleFunc("/api/v1/combos/config/set", routingHandler.SetComboConfig)

	// Proxy Management Endpoints (REC-P3-1)
	proxyHandler := handler.NewProxyHandler()
	mux.HandleFunc("/api/v1/proxies", proxyHandler.ListProxies)
	mux.HandleFunc("/api/v1/proxies/create", proxyHandler.CreateProxy)
	mux.HandleFunc("/api/v1/proxies/update", proxyHandler.UpdateProxy)
	mux.HandleFunc("/api/v1/proxies/delete", proxyHandler.DeleteProxy)
	mux.HandleFunc("/api/v1/proxies/enable", proxyHandler.EnableProxy)
	mux.HandleFunc("/api/v1/proxies/disable", proxyHandler.DisableProxy)
	mux.HandleFunc("/api/v1/proxies/test", proxyHandler.TestProxy)
	mux.HandleFunc("/api/v1/proxies/health", proxyHandler.GetPoolHealth)
	mux.HandleFunc("/api/v1/proxies/rotate", proxyHandler.RotateProxy)
	mux.HandleFunc("/api/v1/proxies/tls-stealth", proxyHandler.GetTLSConfig)
	mux.HandleFunc("/api/v1/proxies/tls-stealth/set", proxyHandler.SetTLSConfig)
	mux.HandleFunc("/api/v1/proxies/{id}", proxyHandler.GetProxy)
	mux.HandleFunc("/api/v1/proxy-chains", proxyHandler.ListChains)
	mux.HandleFunc("/api/v1/proxy-chains/create", proxyHandler.CreateChain)
	mux.HandleFunc("/api/v1/proxy-chains/delete", proxyHandler.DeleteChain)
	mux.HandleFunc("/api/v1/proxy-chains/{id}", proxyHandler.GetChain)

	// Provider Management Endpoints (REC-P3-2)
	providerHandler := handler.NewProviderHandler()
	mux.HandleFunc("/api/v1/providers", providerHandler.ListProviders)
	mux.HandleFunc("/api/v1/providers/stats", providerHandler.GetStats)
	mux.HandleFunc("/api/v1/providers/free", providerHandler.GetFreeProviders)
	mux.HandleFunc("/api/v1/providers/{id}", providerHandler.GetProvider)
	mux.HandleFunc("/api/v1/providers/{id}/enable", providerHandler.EnableProvider)
	mux.HandleFunc("/api/v1/providers/{id}/disable", providerHandler.DisableProvider)

	// Model Registry Endpoints
	mux.HandleFunc("/api/v1/models", providerHandler.ListModels)
	mux.HandleFunc("/api/v1/models/{id}", providerHandler.GetModel)

	// Circuit Breaker Endpoints (REC-CB-1)
	cbHandler := handler.NewCircuitBreakerHandler()
	mux.HandleFunc("/api/v1/circuit-breakers", cbHandler.ListBreakers)
	mux.HandleFunc("/api/v1/circuit-breakers/reset-all", cbHandler.ResetAllBreakers)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}", cbHandler.GetBreaker)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/state", cbHandler.GetBreakerState)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/reset", cbHandler.ResetBreaker)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/check", cbHandler.CheckBreaker)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/success", cbHandler.RecordSuccess)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/failure", cbHandler.RecordFailure)
	mux.HandleFunc("/api/v1/circuit-breakers/{provider}/timeout", cbHandler.RecordTimeout)

	// Diagnostics & Observability endpoints
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"UP","service":"nexusai-gateway","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))))
	})

	// Readiness probe — standard k8s naming (/ready)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "disconnected"
		redisStatus := "disconnected"
		isReady := true

		// Check DB
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err == nil {
				dbStatus = "connected"
			} else {
				dbStatus = "degraded"
				if !cfg.EnableSandboxFallback {
					isReady = false
				}
			}
		} else {
			if !cfg.EnableSandboxFallback {
				isReady = false
			}
		}

		// Check Redis via quotaStorage
		if quotaStorage != nil {
			if err := quotaStorage.Ping(context.Background()); err == nil {
				redisStatus = "connected"
			} else {
				redisStatus = "degraded"
			}
		}

		statusStr := "ok"
		statusCode := http.StatusOK
		if !isReady {
			statusStr = "not_ready"
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"status":"%s","service":"nexusai-gateway","database":"%s","redis":"%s","sandbox_fallback_active":%t,"timestamp":"%s"}`,
			statusStr,
			dbStatus,
			redisStatus,
			cfg.EnableSandboxFallback,
			time.Now().UTC().Format(time.RFC3339),
		)))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		dbStatus := "disconnected"
		isReady := true

		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.PingContext(ctx); err == nil {
				dbStatus = "connected"
			} else {
				dbStatus = "degraded"
				if !cfg.EnableSandboxFallback {
					isReady = false
				}
			}
		} else {
			if !cfg.EnableSandboxFallback {
				isReady = false
			}
		}

		statusStr := "UP"
		statusCode := http.StatusOK
		if !isReady {
			statusStr = "DOWN"
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"status":"%s","service":"nexusai-gateway","database":"%s","sandbox_fallback_active":%t,"timestamp":"%s"}`,
			statusStr,
			dbStatus,
			cfg.EnableSandboxFallback,
			time.Now().UTC().Format(time.RFC3339),
		)))
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

		// Export Prometheus metrics using the observability handler
		observability.PrometheusHandler().ServeHTTP(w, r)
	})

	// Single Page Application static server
	RegisterStaticRoutes(mux)

	// Wrap routing stack in our production-grade middleware layers
	return WithRecovery(
		WithCorrelationID(
			WithStructuredLogging(
				WithRateLimiting(
					rateLimitMiddleware.Middleware(mux),
				),
			),
		),
	)
}
