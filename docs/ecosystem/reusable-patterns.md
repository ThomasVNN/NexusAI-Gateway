# Ecosystem Reusability Guidelines

This document details the reusable patterns, architectures, and guidelines developed during the hardening of the NexusAI-Gateway, serving as a blueprint for the other service repositories in the NexusAI ecosystem.

**Reusable Middleware Patterns**
* Correlation ID Propagation:
  * Pattern: Every incoming request must be parsed or decorated with a unique `X-Correlation-ID` header.
  * Integration: Implement at the ingress of all services (Node.js, Python, Go) and forward it in all outbound HTTP/gRPC requests.
* Structured JSON Access Logging:
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
