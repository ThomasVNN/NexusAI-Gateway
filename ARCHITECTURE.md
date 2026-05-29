# NexusAI Gateway Architecture

NexusAI-Gateway is the core API gateway and routing edge for the NexusAI ecosystem.

**Core Principles**
1. Stateless operation for horizontal scale.
2. Clean architecture separating domain rules, application flow, and infrastructure.
3. Fail-safe degraded mode execution when the backing database is unreachable.
4. API key validation, telemetry collection, and PII scrubbing at the edge.

**Directory Structure**
* `cmd/gateway`: Entry point for server bootstrapping, database connection, and graceful shutdown.
* `configs`: Configuration management, linter settings, and pre-commit hook files.
* `deployments`: Docker, Helm charts, and Kubernetes resource definitions.
* `docs`: Architectural, API, logging, and operational runbooks.
* `internal`: Encapsulated business domain, storage adapters, and router definitions.
* `pkg`: Shared packages designed to be safely imported by other systems.
* `web`: Embedded operational React web dashboard compiled into the Go binary.

**Layer Boundaries**
* Domain Layer: Defines key, usage, and provider entities along with interface contracts under `internal/domain`.
* Storage Layer: Implements database adapters for Postgres and local in-memory fallbacks under `internal/storage`.
* Gateway Layer: Configures handlers, routes, request models, and transport boundaries under `internal/gateway`.
* Privacy Layer: Implements PII detection and content sanitization logic under `internal/privacy`.

**Web Integration**
The React admin interface under `web/` is compiled using Vite during build. The static assets are embedded into the Go binary using the Go `embed` directive in `web/web.go`, allowing a single-binary deployment.

For detailed architectural specifications, see these documents:
* Architectural Overview: `docs/architecture/overview.md`
* Logging Standard: `docs/standards/logging-standard.md`
* API Standard: `docs/standards/api-standard.md`
* Scalability Review: `docs/architecture/scalability-review.md`
