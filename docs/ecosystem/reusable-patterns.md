# Ecosystem Reusability Guidelines

This document details the reusable patterns, architectures, and guidelines developed during the hardening of the NexusAI-Gateway (including Go reference implementations), serving as a blueprint for the other service repositories in the NexusAI ecosystem.

**Reusable Middleware Patterns**
* Correlation ID Propagation:
* Fail-Fast Configuration Validation:
  - Pattern: Define a strict `Validate()` contract that checks if `APP_ENV` matches permitted environmental profiles (e.g. `local`, `development`, `staging`, `production`).
  - Production Rules: Fail startup immediately if unsafe credentials, sandbox configurations, or default local databases are configured in staging/production. This prevents misconfigured services from booting silently.

  * Pattern: Every incoming request must be parsed or decorated with a unique `X-Correlation-ID` header.
  * Integration: Implement at the ingress of all services (Node.js, Python, Go) and forward it in all outbound HTTP/gRPC requests.
* Structured JSON Access Logging:
* Outermost Panic Recovery (`WithRecovery`):
  - Pattern: Implement a recovery shell at the absolute ingress boundary of the HTTP/gRPC pipeline.
  - Action: Catch panics, dump stack traces via structured logs (preventing console leakage), and return a standardized API error envelope with a 500 Internal Server Error status.
* Lightweight Sliding-Window Rate Limiting (`WithRateLimiting`):
  - Pattern: Add zero-dependency token-bucket or sliding-window limits based on client IP or Authorization API key to protect backend processing resources.

  * Pattern: Access log entries must be emitted as standard JSON blocks.
  * Integration: Write standard logging middleware that records request method, path, HTTP status code, correlation ID, and latency in milliseconds.

**Reusable CI/CD Pipeline Blueprints**
* Multi-Gated CI Workflows:
  * Implement modular pipelines separated into lint, test, security, and build phases.
  * Security scans should run Gitleaks for secrets, Govulncheck/npm-audit for dependency vulnerabilities, and Trivy for container checks.
* Automated Dependency Management:
  * Use Dependabot across all repositories with uniform weekly updates for language modules, package managers, and GitHub Actions.

**Ecosystem Observability Blueprints**
* Health Checking:
* Fail-Safe Dependency Rule:
  - Pattern: If a critical datastore or dependency (e.g. PostgreSQL) is unavailable and sandbox fallback mode is deactivated, the `/readyz` probe MUST return a `503 Service Unavailable` status and report a status of `DOWN`.
  - Impact: This ensures failing instances are instantly and automatically excluded from active ingress load-balancers (e.g. Kubernetes services).

  * Maintain consistent `/healthz` (liveness) and `/readyz` (readiness) paths.
  * Readiness endpoints should actively verify backing resources (PostgreSQL, Redis, Vector indexes) and report degraded/operational statuses.
* Prometheus Telemetry:
  * Expose an endpoint `/metrics` utilizing raw Prometheus exposition formats.
  * Standard metrics to capture: database connectivity state (gauge), server uptime (gauge), request counts (counter), and response latency (histogram).

**Standardized Documentation Layout**
* Every repository must adopt the standard `/docs` folder layout:
  * `/docs/architecture/` - technical design blueprints and scalability reviews.
  * `/docs/adr/` - architectural decision records using the context-decision-consequences template.
  * `/docs/runbooks/` - step-by-step deploy, rollback, and incident recovery operations.
  * `/docs/standards/` - coding, logging, and API standard documents.

**Reusable Cloud Deployment Blueprints**
* Docker Multi-Stage Packaging:
  * Utilize multi-stage builds to optimize image size.
  * Never run containers as `root`. Specify `USER 10001:10001` with customized read-only mounts in Kubernetes.
* Helm Chart Structure:
  * Adopt a parameterized Helm layout featuring full ingress SSL/TLS controls, probe configs, liveness/readiness options, resource limits, and environment map bindings.
