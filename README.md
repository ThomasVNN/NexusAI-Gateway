# NexusAI-Gateway

> **Bounded Context:** `Gateway` · **Primary Owner:** Dev Agent · **Supporting:** SA Agent, AI Agent, Platform Agent
> **Repository Role:** Runtime service · **Product Status:** Standalone product · **Version:** 2.x
> **Backlog & Status:** [Tasks Backlog](https://app.notion.com/p/c3b268b26842457a93fbad7fc5b1b710?pvs=1) · [SDLC V3 Control Center](https://app.notion.com/p/3843b1d5683e816c8899debb443e33c5?pvs=1)

The single ingress point for every AI request in the NexusAI ecosystem. NexusAI-Gateway authenticates callers, enforces policy, redacts PII, routes to upstream models, and exposes an MCP transport for agent tooling. It is the only context in the platform that talks directly to external model providers.

![NexusAI-Gateway — AI Request Lifecycle](./docs/architecture/gateway.svg)

---

## Table of Contents

- [Bounded Context](#bounded-context)
- [What It Does](#what-it-does)
- [What It Does NOT Do](#what-it-does-not-do)
- [Architecture](#architecture)
- [Public API Surface](#public-api-surface)
- [Tech Stack](#tech-stack)
- [Repository Layout](#repository-layout)
- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [Testing & Quality Gates](#testing--quality-gates)
- [Deployment](#deployment)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [Governance](#governance)

---

## Bounded Context

| Attribute | Value |
|---|---|
| Context name | `Gateway` |
| Primary owner | Dev Agent |
| Supporting owners | SA Agent, AI Agent, Platform Agent |
| Repository | `NexusAI-Gateway` (this repo) |
| Client libraries | `NexusAI-SDK` (Go, Python, TypeScript) |
| Bounded contexts that depend on it | `Chat`, `Skills`, `Control` (NCC), `Platform` (kill-switch) |
| Bounded contexts it depends on | `Platform` (safety eval, kill-switch, model config) |

**Why this context exists:** All AI traffic in NexusAI must enter through one audited, policy-enforced chokepoint. The Gateway owns authentication, quota, PII, routing, and provider abstraction. Other services must never call upstream providers directly.

---

## What It Does

- 🔐 **Authenticates** API access using hashed gateway keys (Bearer tokens)
- 🛡️ **Enforces policy** — quota, rate limiting, kill-switch awareness, safety evaluation hooks
- 🕵️ **Redacts PII** before any prompt leaves the trust boundary
- 🔀 **Routes** OpenAI-compatible chat completions to one of 20+ upstream providers and 60+ models
- 📡 **Exposes MCP transport** — SSE stream, JSON-RPC message, tool list for agent runtimes
- 📊 **Tracks usage** — tokens, cost, latency, errors — published to PostgreSQL and NATS
- 🖥️ **Serves an embedded admin UI** (`web/`) for legacy key management (migrating to NCC)
- 💾 **Falls back to in-memory state** when the database is unavailable so the service stays operable

## What It Does NOT Do

| Concern | Owned by |
|---|---|
| Document retrieval / RAG | `Knowledge` context |
| Skill registration / MCP tool registry | `Skills` context |
| Model registry / evaluation / safety scoring | `Platform` context |
| Chat UI / conversation storage | `Chat` context |
| Unified admin UI (post-migration) | `Control` (NCC) |

If a feature request asks Gateway to do any of the above, route it to the correct context — that is a **boundary violation**.

---

## Architecture

```
┌──────────────────┐
│ Clients / SDKs   │
│ NCC · Chat ·     │
│ External Apps    │
└────────┬─────────┘
         │ HTTPS + Bearer
         ▼
┌──────────────────────────────────────────────┐
│              NexusAI-Gateway                 │
│                                              │
│  ① Ingress  →  ② Policy  →  ③ Routing  →  ④ Upstream
│  OpenAI/MCP   auth/quota   rule engine      providers
│               PII/killsw.  providers        (OpenAI, Anthropic,
│               safety hook  fallback          Google, Mistral...)
│                                              │
│  ┌────────────┐ ┌────────┐ ┌────────────┐    │
│  │PostgreSQL │ │ Redis  │ │   NATS     │    │
│  │keys+usage │ │ limits │ │   events   │    │
│  └────────────┘ └────────┘ └────────────┘    │
└──────────────────────────────────────────────┘
         │
         ▼
   External / self-hosted model providers
```

See [`docs/architecture/ARCHITECTURE.md`](docs/architecture/ARCHITECTURE.md) for the full design.

---

## Public API Surface

### OpenAI-compatible (clients)

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/chat/completions` | Chat completion (streaming supported) |
| `POST` | `/v1/embeddings` | Embeddings |
| `GET`  | `/v1/models` | List available models |

### MCP transport (agents)

| Method | Endpoint | Description |
|---|---|---|
| `GET`  | `/mcp/v1/tools` | List available MCP tools |
| `GET`  | `/mcp/v1/stream` | SSE stream |
| `POST` | `/mcp/v1/message` | JSON-RPC message |

### Admin (NCC consumes)

| Method | Endpoint | Description |
|---|---|---|
| `GET`  | `/api/v1/keys` | List API keys |
| `POST` | `/api/v1/keys` | Create API key |
| `GET`  | `/api/v1/routes` | List routing rules |
| `POST` | `/api/v1/routes` | Create routing rule |
| `GET`  | `/api/v2/health` | Service health with components |
| `GET`  | `/api/v2/killswitch` | Kill-switch status |

> See the OpenAPI spec at `docs/openapi.yaml` (canonical contract for SDKs).

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| HTTP | `net/http` standard library |
| Database | PostgreSQL via `database/sql` + `github.com/lib/pq` |
| Cache | Redis |
| Event bus | NATS |
| Admin UI | React 18 + Vite + TypeScript (legacy, migrating to NCC) |
| Container | Docker (multi-stage, multi-platform) |
| Lint | `golangci-lint` |
| Tracing | OpenTelemetry |

---

## Repository Layout

```text
cmd/gateway/              Go entrypoint and server bootstrap
internal/auth/            API key parsing and hashing helpers
internal/config/          Environment configuration loading
internal/db/postgres/     PostgreSQL connection bootstrap and schema init
internal/domain/          Domain models, repositories, gateway service logic
internal/gateway/         HTTP handlers, router, and MCP protocol handling
internal/privacy/         PII detection and redaction
internal/storage/         PostgreSQL and in-memory repository implementations
deployments/              Docker, Docker-Compose, Kubernetes, Helm charts
web/                      Embedded admin dashboard (migrating to NCC)
docs/                     Architecture, ADRs, runbooks, standards
```

---

## Quick Start

### Prerequisites

- Go 1.22.0
- Node.js 22+ (for the embedded admin UI)
- Docker and Docker Compose

### Bootstrap

```bash
make bootstrap     # local master bootstrap
make dev-env-up    # PostgreSQL, Redis, NATS containers
make dev           # run the gateway on :20129
```

The gateway listens on `PORT` (default `20129`).

---

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `PORT` | HTTP listen port | `20129` |
| `DATABASE_URL` | PostgreSQL connection string | Local fallback |
| `REDIS_URL` | Redis connection string | Local fallback |
| `NATS_URL` | NATS event bus URL | Local fallback |
| `OIDC_ISSUER` | OIDC issuer base URL (compatibility flows) | `http://localhost:20129` |
| `INITIAL_PASSWORD` | Initial admin password for bootstrap | Falls back to `OMNIROUTE_ADMIN_KEY` |
| `OMNIROUTE_ADMIN_KEY` | Legacy bootstrap fallback | Not set |
| `UPSTREAM_API_URL` | Upstream chat completion endpoint | Not set |
| `UPSTREAM_API_KEY` | Upstream provider bearer token | Not set |

See `internal/config/` for the authoritative list.

---

## Testing & Quality Gates

```bash
make test           # unit + integration
make lint           # golangci-lint
make security       # Trivy + Gitleaks
```

**Merge gates** (per global SDLC V2):
- All CI checks passing
- SA Agent approval on architectural changes
- QA Agent validation
- Branch up to date with `main`
- Docs updated

---

## Deployment

- **Docker:** `deployments/Dockerfile` — multi-stage, multi-platform
- **Kubernetes:** Kustomize manifests in `deployments/k8s/` and Helm chart in `deployments/helm/`
- **Local dev:** `docker compose up` in the workspace root
- **Production runtime topology:** see [`NexusAI-Infra`](https://github.com/ThomasVNN/NexusAI-Infra)

---

## Documentation

| Topic | Path |
|---|---|
| Architecture | `docs/architecture/ARCHITECTURE.md` |
| Scalability review | `docs/architecture/scalability-review.md` |
| ADRs | `docs/adr/ADR-0001..0004` |
| Runbooks | `docs/runbooks/` (deploy, rollback, incident, env, local) |
| Standards | `docs/standards/` (branch, commits, PR, logging, API) |
| OpenAPI | `docs/openapi.yaml` |

---

## Contributing

1. Branch from `main` using the convention `feature/<ticket>-<description>`
2. Follow [Conventional Commits](https://www.conventionalcommits.org/)
3. Open a PR using `.github/PULL_REQUEST_TEMPLATE.md`
4. Wait for SA Agent review → QA Agent validation → Release Manager merge
5. Never commit directly to `main`

See `CONTRIBUTING.md` for the full workflow.

---

## Governance

| Attribute | Value |
|---|---|
| Document owner | Dev Agent |
| Review cadence | Monthly |
| Last updated | June 20, 2026 |
| License | Internal — NexusAI Platform |

---

*This README is part of the NexusAI Platform documentation set. The canonical source for product context is `docs/bounded-contexts/gateway.md` in the workspace root. Notion mirrors these pages; this file is the immutable published version.*
