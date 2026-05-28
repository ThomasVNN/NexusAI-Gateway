# ADR 0002: Observability and Telemetry Strategy

**Status**
Accepted

**Context**
In a distributed microservice ecosystem, isolating performance bottlenecks, request failures, and key usage patterns requires comprehensive tracing, structured logs, and metrics. Without an observability framework, tracking streaming SSE completions and MCP request-reply latency across multiple network boundaries is extremely difficult.

**Decision**
We will implement an OpenTelemetry-first observability strategy across the NexusAI-Gateway.

This strategy includes:
1. Structured Logging: Emit all logs in structured JSON format with predefined severity levels. Include correlation IDs (`x-correlation-id`) and OpenTelemetry trace/span IDs in every log statement.
2. Metrics Collection: Expose an HTTP `/metrics` endpoint in Prometheus exposition format to capture service metrics (request count, duration, error rate, memory utilization, DB pool usage).
3. Distributed Tracing: Integrate OpenTelemetry instrumentation to trace ingress requests, database calls, and outgoing proxy completions.
4. Health Checking: Implement standard `/healthz` and `/readyz` endpoints for Kubernetes probes.

**Consequences**
* Pros:
  * Eliminates dark corners in production by providing full end-to-end request tracing.
  * Standardizes telemetry formats across the entire NexusAI ecosystem.
  * Enables proactive alerting and monitoring using Grafana, Loki, Prometheus, and Tempo.
* Cons:
  * Slight performance overhead due to JSON marshaling and span generation.
  * Storage overhead for telemetry backends (Prometheus, Loki, Tempo).

**Alternatives Considered**
* Standard Go `log` Output: Rejected because unstructured logs are very difficult to query and aggregate at scale.
* Jaeger-specific Tracing: Rejected because OpenTelemetry is the CNCF standard and provides cloud-agnostic vendor independence.
