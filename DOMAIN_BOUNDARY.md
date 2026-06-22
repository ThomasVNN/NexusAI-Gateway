# NexusAI-Gateway — Domain Boundary

**Version:** 1.0
**Date:** 2026-06-22
**Owner:** Solution Architect Agent
**Classification:** Canonical — Foundation Document

---

## 1. Identity

| Attribute | Value |
|-----------|-------|
| **Repository** | `NexusAI-Gateway` |
| **Bounded Context** | Single ingress point for all AI requests; provider abstraction and routing |
| **Role** | Model Access Layer |
| **Language** | Go |
| **Type** | Runtime service |

---

## 2. Purpose

NexusAI-Gateway is the **Model Access Layer** — the single egress point for all AI model calls. It handles provider abstraction, request routing, quota management, PII redaction, and usage tracking.

NexusAI-Gateway is NOT a model provider. It is NOT a business logic layer. It is a proxy and policy enforcement point.

---

## 3. Owned Capabilities

| Capability | Description |
|-----------|-------------|
| **Auth & Security** | JWT validation, API key management |
| **Quota Management** | Per-org, per-user request limits |
| **PII Redaction** | Automatic detection and masking of sensitive data |
| **Request Routing** | Model selection based on routing policy |
| **Provider Abstraction** | Single interface across OpenAI, Anthropic, Google, Azure, etc. |
| **MCP Transport** | Model Context Protocol for tool/function calling |
| **Usage Tracking** | Cost and token recording per request |

---

## 4. Owned APIs

| Endpoint | Description |
|----------|-------------|
| `/v1/chat/completions` | OpenAI-compatible chat completions |
| `/v1/embeddings` | OpenAI-compatible embeddings |
| `/api/v1/providers/*` | Provider CRUD |
| `/api/v1/models/*` | Model registry read |
| `/api/v1/routing/*` | Routing policy management |

---

## 5. Owned Data (PostgreSQL)

| Table | Purpose |
|-------|---------|
| `providers` | AI provider configurations |
| `models` | Available models per provider |
| `routing_policies` | Request routing rules |
| `usage_logs` | Per-request usage and cost records |

---

## 6. Owned Events

| Event | Trigger |
|-------|---------|
| `request.processed` | Incoming AI request handled |
| `model.called` | External provider invoked |
| `cost.recorded` | Usage cost logged |
| `quota.exceeded` | Org/user quota limit hit |

---

## 7. Dependencies

| Dependency | Purpose |
|-----------|---------|
| `NexusAI-Platform` | Safety evaluation and kill-switch |

---

## 8. Forbidden

- **Direct call to Knowledge RAG** — must go through Knowledge API
- **Direct SDK writes**
- **Bypassing the Gateway for model calls** — all AI calls must route through Gateway
- **User authentication** — delegated to upstream

---

## 9. Integration Points

| Integration | Direction | Protocol |
|------------|----------|---------|
| All downstream services | egress | HTTP (model calls) |
| External Providers (OpenAI, Anthropic, etc.) | egress | HTTP |
| NexusAI-Platform | reads | HTTP (safety/kill-switch) |

---

## 10. Ecosystem Position

```
NexusAI (AI SDLC Runtime)
 └── NexusAI-Gateway (Model Access Layer)  ← ALL AI calls route through here
     ├── External Providers (OpenAI, Anthropic, Azure, etc.)
     ├── NexusAI-Platform (Safety evaluation)
     └── All domain services (model inference)
```

NexusAI-Gateway is the **only** egress point to external model providers.

---

**Canonical document — do not modify without Architecture Council approval.**
**Source:** [REPOSITORY_BOUNDARY_MATRIX.md](../REPOSITORY_BOUNDARY_MATRIX.md)
