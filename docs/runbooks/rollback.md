# Rollback Runbook

This guide describes how to safely roll back a deployment of NexusAI-Gateway in the event of post-deployment failures or regressions.

**Rollback Trigger Conditions**
You must trigger a rollback if any of the following occur:
1. The deployment fails to reach a ready state within 10 minutes (pods stuck in `CrashLoopBackOff` or `ImagePullBackOff`).
2. Custom latency metrics show overhead increase exceeding `+150ms`.
3. Error rates on `/v1/chat/completions` exceed `1%` of total ingress requests.
4. Unresolved connection spikes or memory leaks cause continuous OOM pod crashes.

**Step 1: Identify Last Stable Release**
Check the Helm release history to find the last known stable revision number:
```bash
helm history nexusai-gateway -n nexusai
```

**Step 2: Execute Helm Rollback**
Roll back to the identified stable revision (e.g. revision `2`):
```bash
# Helm rollback format: helm rollback <release-name> <revision-number> [flags]
helm rollback nexusai-gateway 2 --namespace nexusai
```

**Step 3: Monitor Rollback Status**
Monitor the rollback progress to ensure pods scale up cleanly:
```bash
kubectl rollout status deployment/nexusai-gateway -n nexusai
```

**Step 4: Database Schema Audit**
Since the gateway bootstraps its tables automatically on startup, check if any breaking changes occurred in tables (like `registered_keys` or `usage_records`).
* Note: The gateway DB schema has backward compatibility by design. Added columns will not break older binaries.
* In the case of severe database table corruption, restore from the last standard database backup.

**Step 5: Rollback Post-Audit**
Verify that liveness and readiness endpoints have recovered:
```bash
curl -i http://gateway.nexusai.local/healthz
curl -i http://gateway.nexusai.local/readyz
```
