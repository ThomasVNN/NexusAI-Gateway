# Technical Architecture Blueprint

This document details the software design, layer boundaries, data flow, and technology integration inside NexusAI-Gateway.

**Design Patterns**
* Clean Architecture: Logic is decoupled from protocols and external services.
* Dependency Injection: Dependencies are injected during startup in `cmd/gateway/main.go`.
* Degraded Mode Resilience: If PostgreSQL is offline, the gateway switches to safe, in-memory rate limiting and token bucket tracking.
* Single Binary Deployment: Front-end React assets are built and embedded in the Go binary using `go:embed`.

**Layer Boundaries**
1. Domain Layer (`internal/domain/`):
   * Models: Holds entities such as `Key`, `Usage`, `Provider` under `internal/domain/model`.
   * Repositories: Interface contracts (e.g. `KeyRepository`, `UsageRepository`) under `internal/domain/repository`.
   * Services: Domain-specific rules for quota auditing and gateway compliance under `internal/domain/service`.
2. Storage Layer (`internal/storage/`):
   * Postgres: Implements repository interfaces using SQL queries under `internal/storage/postgres`.
   * Memory: In-memory fail-safe store implementing key/usage interfaces under `internal/storage/memory`.
3. Gateway Layer (`internal/gateway/`):
   * HTTP Handlers: Standard `http.Handler` implementations under `internal/gateway/http/handler`.
   * Router: Mux setup and static asset registration under `internal/gateway/http/router`.
   * MCP Handler: Model Context Protocol SSE and JSON-RPC implementations under `internal/gateway/mcp`.
4. Privacy Layer (`internal/privacy/`):
   * Detector: RegEx and NLP boundaries for matching credit cards, credentials, and PII under `internal/privacy/detector.go`.
   * Engine: Redaction orchestrator replacing matched values with placeholders under `internal/privacy/engine.go`.

**Data Flows**
* Client Ingress Flow:
  1. Client sends an HTTP request to `/v1/chat/completions`.
  2. Router directs the request to `ChatHandler`.
  3. `ChatHandler` extracts the API key and performs validation via `KeyRepository`.
  4. Quota auditor verifies token counts and rate limits via `UsageRepository`.
  5. The request body is scrubbed for PII through the `PrivacyEngine`.
  6. The gateway proxies the request to the upstream AI provider and streams back the chunked response.
  7. Upon completion, actual token counts are recorded in `UsageRepository`.
* Admin Management Flow:
  1. An administrator authenticates via `/api/auth/login`.
  2. Key CRUD operations are triggered on `/api/admin/keys`, writing to the persistence storage.
  3. Usage metrics are parsed from `/api/admin/usage` or `/api/provider-metrics`.

**Front-End Embedding**
* React Source: The Single Page Application is located in `web/`.
* Compilation: Compiled with Vite using `npm run build` which outputs to `web/dist/`.
* Embed Directive: The compiled files are compiled into the binary with Go's `embed` package in `web/web.go`.
* Fallback Routing: Unknown routes are routed back to `index.html` to support React client-side routing.
