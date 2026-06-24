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
	"github.com/ThomasVNN/NexusAI-Gateway/internal/provider"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/ratelimit"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/runtime"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/storage/memory"
	storage "github.com/ThomasVNN/NexusAI-Gateway/internal/storage/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/token"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/user"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/webhooks"
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
	var providerService *provider.Service
	var providerHandler *handler.ProviderHandler

	// Provider health and rotation (NX-24)
	var healthChecker *provider.HealthChecker
	var providerSelector *provider.ProviderSelector

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

		// Provider management (NX-204 + NX-24)
		providerRepo := provider.NewRepository(db.DB)
		providerService = provider.NewService(providerRepo)

		// Provider health checker and selector (NX-24)
		healthConfig := provider.DefaultHealthCheckConfig()
		healthChecker = provider.NewHealthChecker(healthConfig)
		providerSelector = provider.NewProviderSelector(healthChecker, provider.PriorityBased, nil)

		// Provider handler with CRUD + health (NX-204 + NX-24)
		providerHandler = handler.NewProviderHandler(providerService, healthChecker, providerSelector)

		// Initialize API handler with all services
		apiHandler = handler.NewAPIHandler(chService, tgService, uService, logService, billingService)
	}

	// 5. Initialize Webhook Receiver
	webhookReceiver := webhooks.New(webhooks.DefaultConfig())
	webhookReceiver.StartDeliveryWorker(context.Background())

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

	// new-api: Provider Registry Endpoints (NX-204)
	if providerHandler != nil {
		mux.HandleFunc("/v1/providers", providerHandler.HandleProviders)
		mux.HandleFunc("/v1/providers/", providerHandler.HandleProvider)
		mux.HandleFunc("/v1/providers/select", providerHandler.HandleProviderSelect)
	}

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
		mux.HandleFunc("/api/billing/pricing", apiHandler.HandleBillingPricing)
	}

	// Provider Health & Rotation Endpoints (NX-24)
	// DEPENDENCY: Requires NX-204 (Provider Registry) to be merged first
	if providerHandler != nil {
		mux.HandleFunc("/v1/providers", providerHandler.HandleProviders)
		mux.HandleFunc("/v1/providers/", providerHandler.HandleProvider)
		mux.HandleFunc("/v1/providers/select", providerHandler.HandleProviderSelect)
		mux.HandleFunc("/v1/providers/health", providerHandler.HandleAllProviderHealth)
	}

	// Webhook Endpoints
	mux.HandleFunc("POST /api/webhooks/receive", webhookReceiver.HandleWebhook)

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

	// Comprehensive health check endpoint - checks all services
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Gateway version - can be updated via build ldflags
		const gatewayVersion = "1.0.0"

		// Calculate uptime in seconds
		uptime := time.Since(startTime).Seconds()

		// Service status tracking
		type serviceStatus struct {
			gateway  string
			provider string
			cache    string
			eventbus string
		}
		status := serviceStatus{
			gateway:  "healthy",
			provider: "healthy",
			cache:    "healthy",
			eventbus: "not_configured",
		}
		isHealthy := true

		// Check Provider (Model Platform) - always healthy since it's a default client
		// In production, this would ping the actual model platform
		status.provider = "healthy"

		// Check Cache (Redis) via quotaStorage
		if quotaStorage != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := quotaStorage.Ping(ctx); err != nil {
				status.cache = "unhealthy"
				isHealthy = false
				slog.Warn("Health check: cache unhealthy", slog.Any("error", err))
			}
		} else {
			status.cache = "unhealthy"
			isHealthy = false
		}

		// Check Event Bus (NATS) - not configured in current setup
		// Mark as not_configured since it's not initialized in router
		status.eventbus = "not_configured"

		// Determine response status
		overallStatus := "healthy"
		statusCode := http.StatusOK
		if !isHealthy {
			overallStatus = "unhealthy"
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"status":"%s","version":"%s","uptime":%.0f,"services":{"gateway":"%s","provider":"%s","cache":"%s","eventbus":"%s"}}`,
			overallStatus,
			gatewayVersion,
			uptime,
			status.gateway,
			status.provider,
			status.cache,
			status.eventbus,
		)))
	})

	// Single Page Application static server
	RegisterStaticRoutes(mux)

	// Wrap routing stack in our production-grade middleware layers
	return WithRecovery(
		WithCorrelationID(
			WithTracing(
				WithStructuredLogging(
					WithRateLimiting(
						rateLimitMiddleware.Middleware(mux),
					),
				),
			),
		),
	)
}
