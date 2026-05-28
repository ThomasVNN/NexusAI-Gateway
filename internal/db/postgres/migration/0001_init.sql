-- Initial schema for NexusAI-Gateway (PostgreSQL)

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

CREATE INDEX IF NOT EXISTS idx_usage_key_created ON usage_records(key_id, created_at);
