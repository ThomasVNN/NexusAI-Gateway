# NexusAI-Gateway

NexusAI-Gateway is a high-performance, enterprise-grade AI API gateway and control plane built in Go, designed to replace the Node.js/Next.js OmniRoute microservice within the LocalAgent platform. 

It provides sub-millisecond API proxying, robust Server-Sent Events (SSE) streaming, fine-grained quota enforcement, central model routing, and native support for the Model Context Protocol (MCP) — all backed by a resilient PostgreSQL database.

## Key Features

- **High-Performance Go Lcore**: Low-latency, concurrency-first routing engine designed to process thousands of simultaneous LLM streams with minimal memory overhead.
- **PostgreSQL Persistence**: Replaces local SQLite files to deliver high concurrent write throughput, ensuring fail-safe usage accounting and audit logging.
- **Model Context Protocol (MCP)**: Native stream endpoints `/api/mcp/stream` allowing coding agents (like OpenClaude) to seamlessly explore and execute system tools.
- **PII Privacy Redaction**: Advanced high-speed regex-based prompt sanitization engine running directly in-memory before sending payloads to external providers.
- **Embedded Admin Dashboard**: Modern single-page application built using Vite, fully compiled and embedded directly into the Go binary for single-asset deployment.

## Architecture Map

```
                     Browser / Client (OpenWebUI, OpenClaude, SDKs)
                                      │
                                      ▼
                               [ Traefik (L0) ]
                                      │
                                      ▼
                        [ NexusAI-Gateway (Go Core) ] ── [ PostgreSQL ]
                         ├── SSE Chat Completions
                         ├── OIDC / JWT SSO
                         └── MCP Stream Endpoint
                                      │
                                      ▼
                        [ AI Model Providers (SaaS/Local) ]
```

## Directory Structure

```text
├── cmd/
│   └── gateway/             # App entrypoint (main.go) and dependency injection setup
├── internal/
│   ├── config/              # Configuration loader for environment variables
│   ├── domain/              # DDD Layer: core models, repo interface, and business service
│   ├── gateway/             # Ingress controllers: HTTP handlers, SSE routers, and MCP registry
│   ├── privacy/             # In-memory PII detection and regex-based redaction engine
│   ├── db/                  # PostgreSQL connection pool and migration schemas
│   └── auth/                # Identity providers, OIDC SSO, and API key management
├── pkg/                     # Reusable utilities (SSE packets, MCP message structures)
├── web/                     # Admin Dashboard Frontend (React / Svelte + Vite)
└── deployments/             # Docker configuration files (Dockerfile, Compose)
```

## Getting Started

### Prerequisites

- Go 1.22+ (to build locally)
- Node.js 22+ (to build frontend assets)
- Docker & Docker Compose (recommended)

### Running with Docker

You can spin up the gateway along with a dedicated PostgreSQL database:

```bash
cd deployments
docker compose up --build
```

The gateway will be accessible at `http://localhost:20129`.

### Building Manually

1. Compile the frontend:
   ```bash
   cd web && npm install && npm run build
   ```
2. Run tests:
   ```bash
   go test -v ./...
   ```
3. Compile the Go binary:
   ```bash
   make build
   ```
4. Execute:
   ```bash
   ./bin/nexusai-gateway
   ```

## Development Guidelines

Please review [DEVELOPMENT.md](DEVELOPMENT.md) for coding standards, Git branch naming rules, linter configuration, and commit conventions before submitting a Pull Request.
