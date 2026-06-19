# NexusAI Gateway API Documentation

**Version:** 1.1.0
**Base URL:** `http://localhost:20129`

---

## Overview

The NexusAI Gateway provides unified access to multiple AI providers with intelligent routing, cost optimization, and safety features.

## Authentication

All API requests require an `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:20129/api/v1/providers
```

---

## Health Endpoints

### GET /healthz

Liveness probe for Kubernetes.

**Response:**
```json
{
  "status": "ok"
}
```

### GET /health

Readiness check with full status.

**Response:**
```json
{
  "status": "ok",
  "version": "1.1.0",
  "uptime": 3600
}
```

---

## Provider Endpoints

### GET /api/v1/providers

List all AI providers.

**Response:**
```json
{
  "providers": [
    {
      "id": "provider-openai",
      "name": "OpenAI",
      "category": "openai",
      "enabled": true,
      "healthy": true,
      "tier": "shared",
      "rate_limit": 500,
      "capabilities": ["streaming", "vision", "function_calling"],
      "models": [...]
    }
  ],
  "count": 16
}
```

### GET /api/v1/providers/{id}

Get a specific provider.

### GET /api/v1/providers/{id}/enable

Enable a provider.

### GET /api/v1/providers/{id}/disable

Disable a provider.

---

## Model Endpoints

### GET /api/v1/models

List all available models across all providers.

**Response:**
```json
{
  "models": [
    {
      "id": "gpt-4o",
      "name": "GPT-4o",
      "provider": "openai",
      "context_window": 128000,
      "input_cost_per_1k": 0.005,
      "output_cost_per_1k": 0.015,
      "supports_streaming": true,
      "supports_vision": true,
      "available": true
    }
  ],
  "count": 85
}
```

### GET /api/v1/models/{id}

Get a specific model.

---

## Route Endpoints

### GET /api/v1/routes

List all routing rules.

**Response:**
```json
{
  "routes": [
    {
      "id": "route-1",
      "name": "Premium to GPT-4",
      "priority": 10,
      "conditions": [
        {"field": "user.tier", "operator": "equals", "value": "premium"}
      ],
      "target": {"provider": "openai", "model": "gpt-4"},
      "enabled": true
    }
  ],
  "count": 15
}
```

### POST /api/v1/routes

Create a new routing rule.

**Request:**
```json
{
  "name": "Fast Response Route",
  "priority": 5,
  "conditions": [
    {"field": "request.latency_sla", "operator": "lt", "value": "100"}
  ],
  "target": {"provider": "openai", "model": "gpt-4o-mini"},
  "enabled": true
}
```

### PUT /api/v1/routes/{id}

Update a routing rule.

### DELETE /api/v1/routes/{id}

Delete a routing rule.

### GET /api/v1/routes/{id}/metrics

Get metrics for a specific route.

**Response:**
```json
{
  "total_requests": 5420,
  "success_count": 5380,
  "failure_count": 40,
  "avg_latency_ms": 45.2,
  "success_rate": 99.26
}
```

### POST /api/v1/routes/{id}/test

Test a routing rule against sample data.

---

## Combo Endpoints

### GET /api/v1/combos

List all combo configurations (model chains).

### POST /api/v1/combos

Create a new combo.

---

## Compression Endpoints

### POST /api/v1/compression/tokenize

Tokenize text for cost estimation.

### POST /api/v1/compression/compress

Apply compression to reduce token count.

### POST /api/v1/compression/decompress

Decompress previously compressed text.

---

## Circuit Breaker Endpoints

### GET /api/v1/circuit-breakers

List all circuit breakers.

**Response:**
```json
{
  "breakers": [
    {
      "name": "openai",
      "state": "closed",
      "failure_count": 0,
      "success_count": 100,
      "total_requests": 100,
      "last_failure": null
    }
  ],
  "count": 16
}
```

### GET /api/v1/circuit-breakers/{name}

Get a specific circuit breaker.

### POST /api/v1/circuit-breakers/{name}/reset

Reset a circuit breaker.

### POST /api/v1/circuit-breakers/reset-all

Reset all circuit breakers.

### POST /api/v1/circuit-breakers/{name}/check

Check if a request is allowed through the breaker.

---

## Safety Endpoints

*Note: Safety endpoints are served by the Platform service at port 8084.*

### GET /api/v1/safety

Get safety overview.

### GET /api/v1/safety/rules

List all safety rules.

### POST /api/v1/safety/evaluate

Evaluate content for safety.

**Request:**
```json
{
  "content": "Hello world"
}
```

**Response:**
```json
{
  "passed": true,
  "blocked": false,
  "flagged": false,
  "score": 0.1,
  "reasons": []
}
```

---

## Billing Endpoints

### GET /api/billing/summary

Get billing summary.

**Response:**
```json
{
  "total_cost": 145.67,
  "total_tokens": 1500000,
  "by_provider": {
    "openai": 89.45,
    "anthropic": 56.22
  },
  "period": "2024-06"
}
```

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request",
    "details": ["field 'name' is required"]
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `AUTHENTICATION_FAILED` | Invalid or missing API key |
| `RATE_LIMIT_EXCEEDED` | Too many requests |
| `PROVIDER_ERROR` | External provider error |
| `CIRCUIT_OPEN` | Circuit breaker is open |
| `INTERNAL_ERROR` | Internal server error |

---

## Rate Limits

| Tier | Requests/Minute |
|------|---------------|
| Free | 60 |
| Pro | 500 |
| Enterprise | 5000 |

---

## Changelog

### v1.1.0 (June 2026)
- Added Safety SDK support
- Added Route SDK support
- Added Agent SDK support
- Added Knowledge SDK support
- Added Telemetry SDK support
- Added Meta AI provider (Llama 4 models)
- Added circuit breaker management endpoints
- Added compression endpoints
- Improved rate limiting middleware

### v1.0.0 (Initial Release)
- Basic provider management
- Model listing
- Routing configuration
- Combo configurations
