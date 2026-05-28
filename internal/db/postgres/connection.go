package postgres

import (
	"context"
	"database/sql"
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

	return &DB{db}, nil
}
