package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/config"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/db/postgres"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/gateway/http/router"
)

func main() {
	log.Println("Starting NexusAI-Gateway with custom runtime configs...")

	// 1. Load configuration
	cfg := config.Load()
	log.Printf("Loaded Port: %s", cfg.Port)
	log.Printf("Loaded Database URL: %s", cfg.PostgresURL)
	log.Printf("Loaded Redis URL: %s", cfg.RedisURL)

	// 2. Initialize PostgreSQL connection
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := postgres.Connect(dbCtx, cfg.PostgresURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL at startup: %v (Using sandbox fallback mode)", err)
	} else {
		defer db.Close()
		log.Println("Connected to PostgreSQL successfully")
	}

	// 3. Setup HTTP router and routes
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
		log.Printf("Server listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("NexusAI-Gateway stopped")
}
