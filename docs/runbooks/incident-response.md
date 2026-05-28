# Incident Response Runbook

This runbook acts as the primary triage guide for production incidents affecting the NexusAI-Gateway.

**Severity Levels**
* SEV-1 (Critical): Ingress completely down. Completions return HTTP 5xx errors. Administrative access is blocked.
* SEV-2 (High): Severe degradation. Latency exceeding `> 1s` or rate limits failing to enforce quota rules.
* SEV-3 (Medium): Isolated errors. Individual key failures or minor dashboard visualization glitches.

**Triage Protocol 1: Ingress is Down (SEV-1)**
1. Check the Ingress Controller state:
   ```bash
   kubectl get ingress nexusai-gateway -n nexusai
   kubectl describe ingress nexusai-gateway -n nexusai
   ```
2. Verify pods are running:
   ```bash
   kubectl get pods -n nexusai -l app:nexusai-gateway
   ```
3. Fetch container error logs:
   ```bash
   kubectl logs -n nexusai -l app:nexusai-gateway --tail=100
   ```

**Triage Protocol 2: High CPU or Memory Exhaustion (SEV-2)**
1. Check resource usage inside pods:
   ```bash
   kubectl top pods -n nexusai -l app:nexusai-gateway
   ```
2. Temporary Mitigation: Scale out horizontal pods to distribute load:
   ```bash
   kubectl scale deployment/nexusai-gateway --replicas=5 -n nexusai
   ```
3. Identify problematic queries or excessively large chat completion streaming chunks in the JSON access logs.

**Triage Protocol 3: Database Connectivity Failure**
* Symptom: `/readyz` shows `database: degraded` or `database: disconnected`. Logs show PostgreSQL connection timeout.
* Check database pods or external database endpoints:
  ```bash
  kubectl get pods -n nexusai -l app=postgres-nexus # If running inside the cluster
  ```
* Recovery: The gateway will automatically fall back to degraded in-memory tracking. No immediate crash occurs, but quota updates are volatile. Verify network policies are not blocking pod communication to PostgreSQL on port 5432.

**Triage Protocol 4: Upstream Provider Outage (OpenAI, Anthropic, etc.)**
* Symptom: Completions return timeouts or HTTP 502/504 errors.
* Triage: Verify upstream provider status pages (e.g. status.openai.com).
* Recovery: If primary provider is down, the administrator can login to the admin dashboard and reprioritize alternative active providers.
