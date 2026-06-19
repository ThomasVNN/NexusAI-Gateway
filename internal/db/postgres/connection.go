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

	-- new-api: Channel Management
	CREATE TABLE IF NOT EXISTS channels (
	    id SERIAL PRIMARY KEY,
	    name VARCHAR(255) NOT NULL,
	    type VARCHAR(50) NOT NULL,
	    base_url VARCHAR(500),
	    api_key_encrypted TEXT,
	    models JSONB DEFAULT '[]',
	    priority INT DEFAULT 1,
	    ratio INT DEFAULT 1,
	    is_active BOOLEAN DEFAULT true,
	    balance DECIMAL(12, 4) DEFAULT 0,
	    balance_type VARCHAR(20) DEFAULT 'prepay',
	    group_name VARCHAR(100) DEFAULT '',
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- new-api: Token Groups
	CREATE TABLE IF NOT EXISTS token_groups (
	    id SERIAL PRIMARY KEY,
	    name VARCHAR(255) NOT NULL,
	    allowed_models JSONB DEFAULT '[]',
	    daily_quota BIGINT DEFAULT 0,
	    hourly_quota BIGINT DEFAULT 0,
	    monthly_quota BIGINT DEFAULT 0,
	    used_today BIGINT DEFAULT 0,
	    used_this_hour BIGINT DEFAULT 0,
	    used_this_month BIGINT DEFAULT 0,
	    is_active BOOLEAN DEFAULT true,
	    priority INT DEFAULT 1,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- new-api: Users & Organizations
	CREATE TABLE IF NOT EXISTS organizations (
	    id SERIAL PRIMARY KEY,
	    name VARCHAR(255) NOT NULL,
	    settings JSONB DEFAULT '{}',
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
	    id SERIAL PRIMARY KEY,
	    username VARCHAR(100) UNIQUE NOT NULL,
	    email VARCHAR(255),
	    password_hash VARCHAR(255),
	    role VARCHAR(20) DEFAULT 'user',
	    organization_id INT REFERENCES organizations(id),
	    is_active BOOLEAN DEFAULT true,
	    last_login_at TIMESTAMP WITH TIME ZONE,
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_quotas (
	    user_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
	    daily_token_limit BIGINT DEFAULT 1000000,
	    monthly_token_limit BIGINT DEFAULT 10000000,
	    rate_limit_rpm INT DEFAULT 60,
	    rate_limit_rpd INT DEFAULT 10000
	);

	CREATE TABLE IF NOT EXISTS user_permissions (
	    id SERIAL PRIMARY KEY,
	    user_id INT REFERENCES users(id) ON DELETE CASCADE,
	    permission VARCHAR(100) NOT NULL,
	    resource_id INT,
	    granted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    UNIQUE(user_id, permission, resource_id)
	);

	-- new-api: Model Pricing
	CREATE TABLE IF NOT EXISTS model_pricing (
	    id SERIAL PRIMARY KEY,
	    model_name VARCHAR(100) NOT NULL UNIQUE,
	    input_price_per_1k DECIMAL(10, 6) DEFAULT 0,
	    output_price_per_1k DECIMAL(10, 6) DEFAULT 0,
	    batch_input_price_per_1k DECIMAL(10, 6) DEFAULT 0,
	    is_active BOOLEAN DEFAULT true,
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- new-api: Request Logs
	CREATE TABLE IF NOT EXISTS request_logs (
	    id SERIAL PRIMARY KEY,
	    channel_id INT,
	    model VARCHAR(100) NOT NULL,
	    token_group_id INT,
	    user_id INT,
	    api_key_id VARCHAR(255),
	    input_tokens BIGINT DEFAULT 0,
	    output_tokens BIGINT DEFAULT 0,
	    total_tokens BIGINT DEFAULT 0,
	    latency_ms INT DEFAULT 0,
	    status VARCHAR(20) DEFAULT 'success',
	    error_message TEXT,
	    provider VARCHAR(50),
	    request_id VARCHAR(100),
	    ip_address VARCHAR(50),
	    user_agent TEXT,
	    model_raw VARCHAR(100),
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- new-api: Billing
	CREATE TABLE IF NOT EXISTS billing_records (
	    id SERIAL PRIMARY KEY,
	    organization_id INT REFERENCES organizations(id),
	    user_id INT REFERENCES users(id),
	    api_key_id VARCHAR(255),
	    model VARCHAR(100) NOT NULL,
	    input_tokens BIGINT DEFAULT 0,
	    output_tokens BIGINT DEFAULT 0,
	    input_cost DECIMAL(12, 4) DEFAULT 0,
	    output_cost DECIMAL(12, 4) DEFAULT 0,
	    total_cost DECIMAL(12, 4) DEFAULT 0,
	    currency VARCHAR(10) DEFAULT 'USD',
	    channel_id INT,
	    token_group_id INT,
	    request_id VARCHAR(100),
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS balances (
	    organization_id INT REFERENCES organizations(id),
	    user_id INT REFERENCES users(id),
	    balance DECIMAL(12, 4) DEFAULT 0,
	    currency VARCHAR(10) DEFAULT 'USD',
	    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	    PRIMARY KEY (organization_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS billing_transactions (
	    id SERIAL PRIMARY KEY,
	    organization_id INT REFERENCES organizations(id),
	    user_id INT REFERENCES users(id),
	    amount DECIMAL(12, 4) NOT NULL,
	    balance_before DECIMAL(12, 4) NOT NULL,
	    balance_after DECIMAL(12, 4) NOT NULL,
	    type VARCHAR(20) NOT NULL,
	    description TEXT,
	    reference_id VARCHAR(100),
	    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_usage_key_created ON usage_records(key_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type);
	CREATE INDEX IF NOT EXISTS idx_channels_active ON channels(is_active);
	CREATE INDEX IF NOT EXISTS idx_token_groups_name ON token_groups(name);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_org ON users(organization_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_request_logs_model ON request_logs(model);
	CREATE INDEX IF NOT EXISTS idx_request_logs_channel ON request_logs(channel_id);
	CREATE INDEX IF NOT EXISTS idx_request_logs_user ON request_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_billing_records_org ON billing_records(organization_id);
	CREATE INDEX IF NOT EXISTS idx_billing_records_user ON billing_records(user_id);
	CREATE INDEX IF NOT EXISTS idx_billing_records_created ON billing_records(created_at);
	CREATE INDEX IF NOT EXISTS idx_model_pricing_name ON model_pricing(model_name);
	`
	_, err := db.ExecContext(ctx, schemaQuery)
	return err
}
