# ADR 0001: Architectural Role of the API Gateway

**Status**
Accepted

**Context**
The NexusAI ecosystem uses a microservices architecture to process large-scale AI requests. We need a single, highly performant gateway at the edge to manage traffic routing, rate limiting, quota enforcement, API key validation, PII redaction, and Model Context Protocol (MCP) stream orchestration. This prevents backend services (like Chat, Skills, and Knowledge) from duplicating these concern-heavy infrastructure layers.

**Decision**
We will implement the NexusAI-Gateway as an independent, lightweight Go service running at the ingress edge of the cluster.

The gateway's core features are:
1. Translating public-facing OpenAI-compatible request formats to internal backend APIs.
2. Managing system access through API key verification.
3. Scrubbing input and output payloads for PII using an embedded regex and NLP engine.
4. Hosting a lightweight, embedded React dashboard for immediate administrative insight.
5. Providing a Model Context Protocol (MCP) server integration to coordinate plugins and LLM actions.

**Consequences**
* Pros:
  * Centralizes ingress security, authentication, and compliance.
  * Reduces microservices latency and footprint by stripping off infrastructure concerns.
  * Decouples administrative controls and rate limits from core AI execution logic.
* Cons:
  * Introduces a single point of failure for cluster ingress.
  * Adds an additional network hop for incoming traffic.

**Alternatives Considered**
* Kong or Envoy Gateway with Custom Plugins: Rejected because writing custom Go plugins inside Kong/Envoy added significant operational overhead and made native Go-based MCP streaming integrations overly complex.
* Decoupled Middleware in Each Service: Rejected because it violates domain separation and requires duplicating security/quota compliance logic across Node.js, Python, and Go microservices.
