# ADR 0003: Authentication and Access Control Strategy

**Status**
Accepted

**Context**
The API gateway must authenticate every incoming request before routing or calculating quotas. Storing plain-text API keys in the database poses a significant security risk if the database is compromised. Additionally, executing database calls for every token check would create a bottleneck under heavy streaming loads.

**Decision**
We will implement a secure and high-performance API key authentication pattern.

The strategy includes:
1. Cryptographic Hash: Save API keys using SHA-256 hashes instead of plain text. The raw key is only shown once to the administrator during generation.
2. Authorization Header: Require client requests to pass the key via `Authorization: Bearer <key>`.
3. In-Memory Key Caching: Cache key records in memory using an LRU cache to avoid constant database lookups.
4. Resilient Fallback Store: If PostgreSQL is offline, the gateway falls back to an in-memory storage mode to allow local testing and prevent immediate production blackouts.

**Consequences**
* Pros:
  * Restricts access to credential stores, conforming to OWASP guidelines.
  * Reduces latency to sub-millisecond ranges for key lookups.
  * Degraded mode increases service reliability during temporary network cuts.
* Cons:
  * Keys cannot be recovered if lost; they must be rotated and regenerated.
  * In-memory fallbacks do not persist quota updates, leading to quota drift when degraded.

**Alternatives Considered**
* Plaintext Key Storage: Rejected because it violates our basic security standards and compliance guidelines.
* Stateless JWT Tokens: Rejected because the gateway must support instant key revocation and dynamic rate limits, which are easier to manage with server-side key validation.
