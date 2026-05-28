# Environment Management Runbook

This document details the configuration standards, environmental boundaries, secrets injection, and rotation strategies for NexusAI-Gateway.

**Environmental Separation**
We run four isolated environments across the cluster lifecycle:
1. Local Development (`local`): Running on developer workstations using standard `.env` variables and docker-compose.
2. Staging (`staging`): Cloud environment mirroring production, deployed using Helm in the `nexusai-staging` namespace.
3. Production (`production`): High-availability enterprise cluster, deployed using Helm in the `nexusai` namespace.

**Secrets Injection**
* Local: Injected via `.env` at the root of `NexusAI-Gateway`.
* Kubernetes: Injected via sealed secrets or external secrets managers (e.g. HashiCorp Vault) linked to a `Secret` resource.
* CI/CD: Injected via GitHub Actions Repository Secrets.

**Configuration Variables Reference**
* `PORT`: Server bind port (Default: `20129`).
* `DATABASE_URL`: PostgreSQL connection string. Should include SSL mode enabled for staging/production.
* `REDIS_URL`: Redis connectivity details for token buckets and caching.
* `OIDC_ISSUER`: Issuer URL for validation of identity providers.
* `INITIAL_PASSWORD` / `OMNIROUTE_ADMIN_KEY`: Security bootstrap passwords used during database schema creation.

**Secrets Rotation Playbook**
When rotating database credentials or admin passwords:
1. Generate the new credentials in the PostgreSQL cluster.
2. Update the Kubernetes `Secret` definition or external vault values.
3. Perform a rolling restart of the deployment: `kubectl rollout restart deployment/nexusai-gateway -n nexusai`.
4. Monitor logs for connection status changes. The pods will handle transition gracefully using new connection pools during lifecycle rotation.
