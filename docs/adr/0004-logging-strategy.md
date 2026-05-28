# ADR 0004: Standardized Logging Strategy

**Status**
Accepted

**Context**
Production log files must be parsed, structured, and audit-friendly. Inadvertently writing sensitive credentials, user prompts, PII, or API keys into logs violates privacy standards and can compromise system security. We need a strict, structured logging standard across the ecosystem.

**Decision**
We will implement standard structured JSON logging using the Go standard library's `log/slog` or a high-performance library like `uber-go/zap` wrapped in our logging adapters.

Guidelines:
1. Structured Logs: All output must be written in JSON format to stdout/stderr.
2. Mandatory Attributes: Every log entry must include `time`, `level`, `msg`, `service` (set to `nexusai-gateway`), and optionally `correlation_id` and `trace_id`.
3. Log Levels: Use `DEBUG` for low-level diagnostic logs, `INFO` for routing outcomes, `WARN` for database retries or degraded modes, and `ERROR` for crashes.
4. Security Safeguards: Under no circumstances should logs contain authorization tokens, database passwords, user API keys, user chat prompts, or raw LLM completions.

**Consequences**
* Pros:
  * Fast query parsing in Loki, Elasticsearch, or Datadog.
  * Prevents critical security disclosures in log aggregation systems.
  * Easy log correlation across microservices.
* Cons:
  * Developers must use structured logging methods instead of standard `fmt.Printf`.
  * Logs are harder to read in raw shell outputs without formatting tools (like `jq`).

**Alternatives Considered**
* Standard Go `log` Output: Rejected due to lacking JSON serialization and trace context support.
* Third-party log collectors parsing stdout strings: Rejected as fragile and error-prone due to schema inconsistency.
