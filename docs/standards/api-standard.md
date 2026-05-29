# API Standards and Design Guidelines

This document establishes the REST and gRPC API specifications for the NexusAI-Gateway, aligning with the ecosystem-wide standards.

**Response Contracts**
All APIs must return responses in a standardized wrapper format:
```json
{
  "success": true,
  "data": {},
  "meta": {},
  "error": null
}
```
* `success`: Boolean indicating if the request completed without errors.
* `data`: Payload object containing the response resources.
* `meta`: Diagnostic metadata (e.g. page counts, latency metrics).
* `error`: Standardized error payload if `success` is false.

**Error Formats**
Error responses must use this structured layout:
```json
{
  "success": false,
  "data": null,
  "meta": {},
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "The requested API key does not exist.",
    "details": []
  }
}
```
* `code`: Machine-readable unique uppercase error string.
* `message`: Human-readable error message.
* `details`: Array of field-level validation errors (if applicable).

**API Versioning**
* All public HTTP APIs must use URL path versioning (e.g. `/v1/chat/completions`).
* Administrative APIs must be prefix-versioned under `/api/admin/` or `/api/v1/admin/`.

**Ingress and In-Transit Headers**
* Ingress Authorization: `Authorization: Bearer <API_KEY>`
* Request Correlation: `X-Correlation-ID` (used to link logs across microservices).
* Request Tracing: OpenTelemetry W3C trace context headers (`traceparent`, `tracestate`).

**Rate Limiting and Quota Headers**
The gateway injects access quota compliance headers on API completions:
* `X-RateLimit-Limit-Hourly`: Permitted hourly requests.
* `X-RateLimit-Remaining-Hourly`: Available requests remaining in current hour.
* `X-RateLimit-Reset`: Duration in seconds until quota count resets.
