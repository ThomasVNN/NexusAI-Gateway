package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/router"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/observability"
)

var startTime = time.Now()

func main() {
	// Configure global slog JSON logger as the standard logging facility
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("Starting NexusAI-Gateway with custom runtime configs...",
		slog.String("service", "nexusai-gateway"),
	)

	// 1. Load configuration
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("FATAL: Configuration validation failed", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Configuration loaded successfully",
		slog.String("port", cfg.Port),
		slog.String("env", cfg.AppEnv),
		slog.Bool("sandbox_fallback", cfg.EnableSandboxFallback),
	)

	// 2. Initialize observability (OpenTelemetry + Prometheus)
	obsCtx, obsCancel := context.WithCancel(context.Background())
	defer obsCancel()

	if err := observability.Init(obsCtx, observability.Config{
		ServiceName:    cfg.OTELServiceName,
		ServiceVersion: "1.0.0",
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Enabled:        cfg.ObservabilityEnabled,
	}); err != nil {
		slog.Warn("Observability initialization failed (continuing without)", slog.Any("error", err))
	}

	// 3. Initialize PostgreSQL connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := postgres.Connect(dbCtx, cfg.PostgresURL)
	if err != nil {
		slog.Warn("Failed to connect to PostgreSQL at startup (Using sandbox fallback mode)", slog.Any("error", err))
	} else {
		defer db.Close()
		slog.Info("Connected to PostgreSQL successfully")
	}

	// Start background metrics updater
	dbChecker := func() bool {
		if db == nil {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		return db.PingContext(ctx) == nil
	}
	observability.StartObservability(obsCtx, startTime, dbChecker)

	// 4. Setup HTTP router and routes
	r := router.New(db, cfg)
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 4. Graceful shutdown orchestration
	go func() {
		slog.Info("Server listening", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("NexusAI-Gateway stopped")
}
