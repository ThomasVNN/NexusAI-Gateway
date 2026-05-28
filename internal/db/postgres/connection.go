package postgres

import (
	"context"
	"database/sql"
	"log"
	"time"

	// Import PostgreSQL driver
	_ "github.com/lib/pq"
)

// DB wraps the standard sql.DB connection pool
type DB struct {
	*sql.DB
}

// Connect establishes a connection pool to the PostgreSQL instance
func Connect(ctx context.Context, connStr string) (*DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Set connection pool boundaries
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Validate connectivity
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// Execute automatic table schema creation
	if err := bootstrapSchema(ctx, db); err != nil {
		log.Printf("Warning: Failed to bootstrap PostgreSQL database schema: %v", err)
	} else {
		log.Println("PostgreSQL database schema bootstrapped successfully")
	}

	return &DB{db}, nil
}

func bootstrapSchema(ctx context.Context, db *sql.DB) error {
	schemaQuery := `
	CREATE TABLE IF NOT EXISTS registered_keys (
	    id VARCHAR(255) PRIMARY KEY,
	    key_hash VARCHAR(64) NOT NULL UNIQUE,
	    name VARCHAR(255) NOT NULL,
	    source_app VARCHAR(50) NOT NULL,
	    daily_quota INT NOT NULL DEFAULT 500,
	    hourly_quota INT NOT NULL DEFAULT 100,
	    active BOOLEAN NOT NULL DEFAULT TRUE,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS usage_records (
	    id SERIAL PRIMARY KEY,
	    key_id VARCHAR(255) REFERENCES registered_keys(id),
	    model_id VARCHAR(255) NOT NULL,
	    prompt_tokens INT NOT NULL DEFAULT 0,
	    completion_tokens INT NOT NULL DEFAULT 0,
	    latency_ms INT NOT NULL DEFAULT 0,
	    source_app VARCHAR(50) NOT NULL,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS provider_connections (
	    id VARCHAR(255) PRIMARY KEY,
	    provider VARCHAR(100) NOT NULL,
	    name VARCHAR(255) NOT NULL,
	    api_key TEXT,
	    endpoint TEXT,
	    is_active BOOLEAN NOT NULL DEFAULT TRUE,
	    priority INT NOT NULL DEFAULT 0,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_usage_key_created ON usage_records(key_id, created_at);
	`
	_, err := db.ExecContext(ctx, schemaQuery)
	return err
}
