-- Phase 7-9: Database Parity Migration
-- Maps OmniRoute SQLite schema to NexusAI PostgreSQL
-- 
-- This migration adds:
-- 1. Routing chains and steps for combo/routing support
-- 2. Provider endpoints for multi-endpoint providers
-- 3. Provider health tracking
-- 4. Response cache with vector support
-- 5. Call logs for detailed request tracking
-- 6. Circuit breaker state persistence
-- 7. Model lockouts
-- 8. Extensions to existing tables

-- ============================================================
-- ROUTING CHAINS
-- ============================================================

CREATE TABLE IF NOT EXISTS routing_chains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    strategy VARCHAR(50) NOT NULL DEFAULT 'priority',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_routing_chains_tenant ON routing_chains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_routing_chains_strategy ON routing_chains(strategy);
CREATE INDEX IF NOT EXISTS idx_routing_chains_active ON routing_chains(tenant_id, is_active);

COMMENT ON TABLE routing_chains IS 'Routing chains/combos for multi-step model selection';
COMMENT ON COLUMN routing_chains.strategy IS 'Routing strategy: priority, round_robin, weighted, cost_optimized, latency_optimized, auto, etc.';

-- ============================================================
-- ROUTING CHAIN STEPS
-- ============================================================

CREATE TABLE IF NOT EXISTS routing_chain_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_id UUID NOT NULL REFERENCES routing_chains(id) ON DELETE CASCADE,
    step_order INTEGER NOT NULL,
    provider_id UUID REFERENCES providers(id) ON DELETE CASCADE,
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    fallback_model VARCHAR(255),
    weight FLOAT DEFAULT 1.0,
    min_latency_ms INTEGER,
    max_cost_per_1k FLOAT,
    capabilities TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(chain_id, step_order)
);

CREATE INDEX IF NOT EXISTS idx_routing_chain_steps_chain ON routing_chain_steps(chain_id);
CREATE INDEX IF NOT EXISTS idx_routing_chain_steps_provider ON routing_chain_steps(provider_id);

COMMENT ON TABLE routing_chain_steps IS 'Individual steps in a routing chain';
COMMENT ON COLUMN routing_chain_steps.weight IS 'Weight for weighted routing strategies';

-- ============================================================
-- PROVIDER ENDPOINTS
-- ============================================================

CREATE TABLE IF NOT EXISTS provider_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    api_type VARCHAR(50) DEFAULT 'openai',
    is_primary BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_endpoints_provider ON provider_endpoints(provider_id);
CREATE INDEX IF NOT EXISTS idx_provider_endpoints_active ON provider_endpoints(provider_id, is_active);

COMMENT ON TABLE provider_endpoints IS 'Multiple endpoints per provider (for regional routing, failover)';

-- ============================================================
-- PROVIDER HEALTH
-- ============================================================

CREATE TABLE IF NOT EXISTS provider_health (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    latency_ms INTEGER,
    error_rate FLOAT DEFAULT 0,
    success_rate FLOAT DEFAULT 1.0,
    checked_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_health_provider ON provider_health(tenant_id, provider_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_provider_health_status ON provider_health(tenant_id, provider_id, status);

COMMENT ON TABLE provider_health IS 'Provider health check history for circuit breaker';
COMMENT ON COLUMN provider_health.status IS 'Status: unknown, healthy, degraded, unhealthy';

-- ============================================================
-- RESPONSE CACHE
-- ============================================================

CREATE TABLE IF NOT EXISTS response_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_hash VARCHAR(64) NOT NULL,
    request_text TEXT,
    response_text TEXT,
    model_id UUID REFERENCES models(id),
    provider_id UUID REFERENCES providers(id),
    embedding VECTOR(1536),
    similarity_score FLOAT,
    ttl_seconds INTEGER DEFAULT 3600,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    hit_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_response_cache_hash ON response_cache(tenant_id, request_hash);
CREATE INDEX IF NOT EXISTS idx_response_cache_expires ON response_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_response_cache_model ON response_cache(model_id);

COMMENT ON TABLE response_cache IS 'Response caching with semantic similarity (requires pgvector)';
COMMENT ON COLUMN response_cache.embedding IS 'Vector embedding for semantic similarity search';

-- ============================================================
-- CALL LOGS
-- ============================================================

CREATE TABLE IF NOT EXISTS call_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    method VARCHAR(10),
    path VARCHAR(500),
    status_code INTEGER,
    model VARCHAR(255),
    provider VARCHAR(100),
    chain_id UUID REFERENCES routing_chains(id),
    chain_step_id UUID REFERENCES routing_chain_steps(id),
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    tokens_cached INTEGER DEFAULT 0,
    tokens_reasoning INTEGER DEFAULT 0,
    latency_ms INTEGER DEFAULT 0,
    ttft_ms INTEGER DEFAULT 0,
    cache_hit BOOLEAN DEFAULT false,
    error_code VARCHAR(50),
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_call_logs_timestamp ON call_logs(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_call_logs_model ON call_logs(tenant_id, model);
CREATE INDEX IF NOT EXISTS idx_call_logs_provider ON call_logs(tenant_id, provider);
CREATE INDEX IF NOT EXISTS idx_call_logs_status ON call_logs(tenant_id, status_code);

COMMENT ON TABLE call_logs IS 'Detailed request/response logging for analytics';

-- ============================================================
-- CIRCUIT BREAKER STATE
-- ============================================================

CREATE TABLE IF NOT EXISTS circuit_breaker_state (
    provider_id UUID PRIMARY KEY REFERENCES providers(id) ON DELETE CASCADE,
    state VARCHAR(20) NOT NULL DEFAULT 'closed',
    failures INTEGER DEFAULT 0,
    successes INTEGER DEFAULT 0,
    last_failure TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    config JSONB DEFAULT '{"failure_threshold": 5, "reset_timeout": 30000}'
);

COMMENT ON TABLE circuit_breaker_state IS 'Circuit breaker state per provider';
COMMENT ON COLUMN circuit_breaker_state.state IS 'State: closed, open, half_open';

-- ============================================================
-- MODEL LOCKOUTS
-- ============================================================

CREATE TABLE IF NOT EXISTS model_lockouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    model_id VARCHAR(255) NOT NULL,
    reason VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(provider_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_model_lockouts_expires ON model_lockouts(expires_at);
CREATE INDEX IF NOT EXISTS idx_model_lockouts_provider ON model_lockouts(provider_id);

COMMENT ON TABLE model_lockouts IS 'Temporary model lockouts (quota exceeded, etc.)';

-- ============================================================
-- API KEYS EXTENSIONS
-- ============================================================

ALTER TABLE api_keys 
    ADD COLUMN IF NOT EXISTS machine_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS scopes TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS ip_allowlist INET[],
    ADD COLUMN IF NOT EXISTS allowed_chains UUID[],
    ADD COLUMN IF NOT EXISTS max_sessions INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

COMMENT ON COLUMN api_keys.machine_id IS 'Machine identifier for device binding';
COMMENT ON COLUMN api_keys.scopes IS 'Permission scopes (read, write, admin)';
COMMENT ON COLUMN api_keys.ip_allowlist IS 'Allowed IP addresses (empty = all allowed)';
COMMENT ON COLUMN api_keys.max_sessions IS 'Max concurrent sessions (0 = unlimited)';

-- ============================================================
-- REQUESTS EXTENSIONS
-- ============================================================

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS tokens_cached_read INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tokens_reasoning INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS chain_id UUID REFERENCES routing_chains(id),
    ADD COLUMN IF NOT EXISTS cache_hit BOOLEAN DEFAULT false;

COMMENT ON COLUMN requests.tokens_cached_read IS 'Tokens read from cache';
COMMENT ON COLUMN requests.tokens_reasoning IS 'Tokens used for reasoning (e.g., DeepSeek)';
COMMENT ON COLUMN requests.chain_id IS 'Routing chain used for this request';
COMMENT ON COLUMN requests.cache_hit IS 'Whether response was served from cache';

-- ============================================================
-- MODELS EXTENSIONS
-- ============================================================

ALTER TABLE models
    ADD COLUMN IF NOT EXISTS aliases TEXT[] DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS is_deprecated BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS deprecation_note TEXT,
    ADD COLUMN IF NOT EXISTS provider_endpoint_id UUID REFERENCES provider_endpoints(id);

COMMENT ON COLUMN models.aliases IS 'Alternative model names';
COMMENT ON COLUMN models.is_deprecated IS 'Model is deprecated';
COMMENT ON COLUMN models.provider_endpoint_id IS 'Specific endpoint for this model';

-- ============================================================
-- PROVIDERS EXTENSIONS
-- ============================================================

ALTER TABLE providers
    ADD COLUMN IF NOT EXISTS tier VARCHAR(20) DEFAULT 'apikey',
    ADD COLUMN IF NOT EXISTS auth_type VARCHAR(20) DEFAULT 'apikey',
    ADD COLUMN IF NOT EXISTS has_free_tier BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS free_monthly_limit INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS free_rpm INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS free_tpm INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS context_window INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS supported_endpoints TEXT[] DEFAULT '{}';

COMMENT ON COLUMN providers.tier IS 'Provider tier: subscription, apikey, cheap, free';
COMMENT ON COLUMN providers.auth_type IS 'Auth type: apikey, oauth, basic, bearer';
COMMENT ON COLUMN providers.has_free_tier IS 'Provider has a free tier';
