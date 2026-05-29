# Logging Standards and Protocols

This document defines the strict logging specifications for the NexusAI ecosystem, ensuring all microservices emit consistent, secure, and searchable logs.

**Log Format**
* All logs must be output to stdout/stderr in structured JSON format.
* Plain text console logs are prohibited in production environments.

**Schema Fields**
* `time`: Timestamp in UTC encoded as ISO-8601 (RFC3339).
* `level`: Log level representation: `DEBUG`, `INFO`, `WARN`, `ERROR`.
* `msg`: A brief human-readable description of the log event.
* `service`: System identifier, set to `nexusai-gateway`.
* `correlation_id`: Context correlation token (`X-Correlation-ID`) received/generated on ingress.
* `trace_id`: The OpenTelemetry unique trace identifier (if active).

**Levels Usage**
1. `DEBUG`: Exhaustive debugging details (e.g. prompt token calculations, DB query plans).
2. `INFO`: Successful standard operations (e.g. starting server, completed transactions).
3. `WARN`: Recoverable anomalies or non-critical state changes (e.g. database retry success, degraded fallback activations).
4. `ERROR`: Operational failure requiring immediate response (e.g. database down, writing to disk failures).

**Security Constraints**
* Never log client secrets, user authentication tokens, database passwords, SSL private keys, or plain-text API keys.
* Never log raw user prompts or raw LLM completions.
* If diagnostic prints are necessary, scrub parameters using the `internal/privacy` scrubbing logic before printing.

**Correlation IDs**
* The ingress gateway must inject a unique `X-Correlation-ID` header into every request.
* All subsequent downstream services in the request flow must parse and forward this header.
* Logging adapters must automatically include the correlation ID in every log statement.
