# Operational Master Runbook

This manual covers production operations, performance tuning, architecture checkpoints, and recovery strategies for the NexusAI-Gateway.

**Service Architecture Checkpoints**
* Ingress Port: `8080` (HTTP)
* Outbound Calls: Upstream LLM providers (e.g. OpenAI, Anthropic, Gemini) and MCP host servers.
* Health Check Endpoint: `/healthz` (liveness) and `/readyz` (readiness).
* Persistent Backends: PostgreSQL (primary store).

**Procedures**
1. System Provisioning:
   * Populate Helm values file with target configurations.
   * Run Helm upgrade/install commands.
   * Monitor CPU/Memory scaling during startup.
2. Graceful Maintenance:
   * Drain traffic using Kubernetes ingress controllers.
   * Verify connections have dropped.
   * Terminate pods gracefully (configured with a 15-second shutdown context timeout).

**Metrics and Performance Audits**
* Target response latency: `< 50ms` overhead on completions validation.
* Connection limits: Max idle connection timeout set to 60 seconds.
* CPU Threshold: Benchmark scales horizontally at 75% CPU load.
* Memory Threshold: Memory footprint scales at 80% RAM limit.

**Backup and Restore Operations**
* All critical state (API keys and usage quotas) resides in PostgreSQL.
* Implement standard PostgreSQL WAL archiving and snapshots.
* Degraded fallback stores do not require backing up as they reside entirely in ephemeral RAM.

**Troubleshooting Playbooks**
* Connection Pool Exhaustion:
  * Symptoms: Database connection failures or high latency in logging handler.
  * Mitigation: Check database logs, review maximum open connections config, scale database replica read-only pools.
* Memory Spikes:
  * Symptoms: Pod crashes with OOMKilled status.
  * Mitigation: Analyze logs for extremely large chat completions payloads, inspect PII scrubber memory consumption, limit maximum request body sizes.

---

# Hardened Operations, Security & Observability Blueprint

Following the platform foundation hardening initiative, the NexusAI-Gateway implements rigorous production-ready standards across configuration, health checks, middleware layering, and observability.

## 1. Robust Configuration & Fail-Fast Bootstrapping
The configuration module at `internal/config/config.go` enforces strict environmental boundaries:
* **Environment Separation:** Allowed values for `APP_ENV` are strictly validated (`local`, `development`, `staging`, `production`).
* **Safe Defaults & Production Checks:** In `production` or `staging` environments:
  - Weak default passwords (e.g. `postgres_secure_pass`, `admin`, `mock-key-for-local-dev`, `change-me-before-production`) are strictly rejected.
  - Sandbox fallbacks (`ENABLE_SANDBOX_FALLBACK`) are strictly disabled.
  - PostgreSQL database connection strings (`DATABASE_URL`) cannot use local address patterns (`localhost`, `127.0.0.1`) or default password credentials.
* **Fail-Fast Startup:** On bootstrap, configuration validation is evaluated immediately. Any safety violation causes an immediate exit (`os.Exit(1)`) with detailed diagnostics output to stdout in structured JSON format.

## 2. Structured Observability & Correlation ID Mapping
* **Structured slog JSON Logging:** The service has migrated from unstructured standard logging to Go's standard library `log/slog` structured JSON framework. All logs (both request completions and internal diagnostic events) are output directly as standardized JSON blocks to standard output.
* **Correlation ID Context Propagation:** The gateway automatically injects a request-tracing Correlation ID:
  - Ingress: Tracing attempts to read `X-Correlation-ID` header. If missing, it securely generates a unique 16-byte random hex ID.
  - Context Mapping: Injected into the request context (`CorrelationIDKey`), making it retrievable by database layers, PII scrubbers, and downstream services via `GetCorrelationID(ctx)`.
  - Egress: The correlation ID is output as `correlation_id` in all slog logs, and passed back to clients in the `X-Correlation-ID` response header.

## 3. High-Safety Middleware Layering Order
The HTTP routing stack applies middleware in a strict, high-safety pipeline:
1. **Panic Recovery (`WithRecovery`):** Outermost shell. Catches downstream panics, captures raw stack traces, logs the error and stack trace under `slog.ErrorContext`, and writes a standardized error payload with HTTP 500 status.
2. **Correlation ID (`WithCorrelationID`):** Generates and registers the request correlation ID context.
3. **Structured Logging (`WithStructuredLogging`):** Measures execution duration, logging the request details (service, method, path, correlation_id, status_code, latency_ms) on completion.
4. **Rate Limiting (`WithRateLimiting`):** A high-performance, zero-dependency sliding window rate limiter that shields downstream processing from resource starvation. Rates are calculated per API Authorization Key or IP address.
5. **Business/Admin Routes:** Request flows downstream into model catalogs, MCP stream engines, or completions.

## 4. Hardened Health & Readiness Checks
* **Liveness Endpoint (`/healthz`):** Verifies the process is running. Returns `200 OK` with JSON payload: `{"status":"UP","service":"nexusai-gateway","timestamp":"..."}`.
* **Readiness Endpoint (`/readyz`):** Evaluates dependency health with strict production safety:
  - Active check: Performs a live PostgreSQL ping with a strict 2-second timeout window.
  - Fail-Safe Dependency Rule: If the database is disconnected or degraded, and `ENABLE_SANDBOX_FALLBACK` is deactivated (standard in staging/production), the endpoint returns `503 Service Unavailable` with `status: DOWN`. This blocks incorrect routing of user traffic to un-ready backend instances.
  - Sandbox Fallback: If `ENABLE_SANDBOX_FALLBACK` is activated, `/readyz` returns `200 OK` but reports `database: degraded` or `database: disconnected` so local developers can work offline.
