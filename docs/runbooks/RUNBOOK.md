# Operational Master Runbook

This manual covers production operations, performance tuning, architecture checkpoints, and recovery strategies for the NexusAI-Gateway.

**Service Architecture Checkpoints**
* Ingress Port: `8080` (HTTP)
* Outbound Calls: Upstream LLM providers (e.g. OpenAI, Anthropic, Gemini) and MCP host servers.
* Health Check Endpoint: `/healthz` (liveness) and `/readyz` (readiness).
* Persistent Backends: PostgreSQL (primary store).

**Procedures**
1. System Provisioning:
   * Populate Helm values file with target configurations.
   * Run Helm upgrade/install commands.
   * Monitor CPU/Memory scaling during startup.
2. Graceful Maintenance:
   * Drain traffic using Kubernetes ingress controllers.
   * Verify connections have dropped.
   * Terminate pods gracefully (configured with a 15-second shutdown context timeout).

**Metrics and Performance Audits**
* Target response latency: `< 50ms` overhead on completions validation.
* Connection limits: Max idle connection timeout set to 60 seconds.
* CPU Threshold: Benchmark scales horizontally at 75% CPU load.
* Memory Threshold: Memory footprint scales at 80% RAM limit.

**Backup and Restore Operations**
* All critical state (API keys and usage quotas) resides in PostgreSQL.
* Implement standard PostgreSQL WAL archiving and snapshots.
* Degraded fallback stores do not require backing up as they reside entirely in ephemeral RAM.

**Troubleshooting Playbooks**
* Connection Pool Exhaustion:
  * Symptoms: Database connection failures or high latency in logging handler.
  * Mitigation: Check database logs, review maximum open connections config, scale database replica read-only pools.
* Memory Spikes:
  * Symptoms: Pod crashes with OOMKilled status.
  * Mitigation: Analyze logs for extremely large chat completions payloads, inspect PII scrubber memory consumption, limit maximum request body sizes.
