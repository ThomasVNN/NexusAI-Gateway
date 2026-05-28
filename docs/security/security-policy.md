# Security Architecture and Policy Guide

This guide establishes the security controls, secrets management standards, vulnerability handling procedures, and threat mitigation models for NexusAI-Gateway.

**Secrets Management Rules**
1. Never commit API keys, client secrets, passwords, or production credentials to the repository.
2. In production, load secrets exclusively from environment variables or external secrets managers (e.g. HashiCorp Vault, AWS Secrets Manager, Kubernetes Secrets).
3. Local secrets for development must reside in an uncommitted `.env` file.

**Cryptographic Hashing Standards**
* Gateway API keys must be cryptographically hashed with SHA-256 before storage.
* The plain-text key must only be exposed once (during key creation) and never written to standard logs or debug outputs.
* Salt keys using service-level salts where necessary.

**PII Scrubbing and Compliance**
* The `internal/privacy` engine parses client prompts and completions.
* PII elements such as credit card numbers, Social Security Numbers, passwords, and private keys are scrubbed and replaced with placeholders.
* Custom regular expressions and pattern extractors must be tested regularly for performance and accuracy.

**Fail-Closed and Secure Failure Principles**
* **Fail-Closed by Default**: The gateway strictly enforces fail-closed behavior for all authentication, authorization, and quota-checking failures. If the authentication datastore, PostgreSQL database, or other key repositories are unavailable, the gateway denies access immediately with `503 Service Unavailable` instead of falling back to sandbox access.
* **Provider Failure Protection**: Downstream or upstream AI provider invocation failures (such as timeout, network outage, or non-200 responses) do not fall back to insecure or mock streams. Requests are denied with `502 Bad Gateway` to prevent silent degradation or bypasses of LLM processing.
* **Configuration Security Validation**: The gateway validates all administrative configuration at startup. Unsafe defaults (e.g. default passwords or mock sandboxes) are prohibited in production environments (`APP_ENV=production`), causing immediate startup failure.
* **Zero Secret Leakage Logging**: Security-relevant errors (like authentication failure, quota exceed, or provider failure) are logged using structured JSON logs (`LogSecurityEvent`) containing correlation IDs and error classifications, with strict guarantees against logging plain-text API keys, passwords, or user prompt content.

**Threat Mitigation Models**
* Quota Violations and Denial of Service:
  * Quota auditors enforce limits at the ingress layer.
  * In-memory token bucket rate limiters protect against upstream provider floods.
* DB Compromise:
  * Since keys are stored as hashes, a database compromise does not immediately leak active user keys.
  * Secrets and database configuration strings must be rotated immediately in case of a breach.

**Vulnerability Reporting Procedures**
* If you discover a vulnerability, do not open a public issue.
* Email security issues directly to `security@nexusai.local` (or refer to the instructions in `/SECURITY.md`).
* A response and acknowledgement will be sent within 48 hours.
