# NexusAI Gateway Runbook

This runbook provides basic instructions for running, managing, and troubleshooting the NexusAI-Gateway service.

**Quick Commands**
* Build the service: `make build`
* Start local dev: `make dev`
* Run unit tests: `make test`
* Check logs: `docker-compose logs -f`

**Configuration Management**
The service is configured entirely via environment variables. The primary config file for local development is `.env` (copied from `.env.example`).
* `PORT`: Port the HTTP server listens on (default: `8080`).
* `DATABASE_URL`: Connection string for PostgreSQL.
* `REDIS_URL`: Connection string for Redis.
* `INITIAL_PASSWORD`: Bootstrapping credentials for the admin dashboard.

**Common Operational Workflows**
1. Starting up in local mode:
   * Verify Docker is running.
   * Run `make dev` at the root.
   * Open `http://localhost:8080` in your browser.
2. Deploying to production:
   * Build the production Docker image.
   * Apply Helm chart templates or K8s configurations.
   * Verify the `/healthz` or `/livez` endpoint returns a HTTP `200` status code.
3. Scaling up:
   * The gateway is completely stateless and can be scaled horizontally.
   * Ensure that your PostgreSQL connection pool can handle the additional instances.

**Troubleshooting Failures**
* Unreachable Database:
  * Check PostgreSQL logs.
  * If PostgreSQL is down, the gateway will start in "Fallback degraded mode" using local memory.
  * Quota changes will not be saved permanently in this mode.
* Missing API Keys:
  * Ensure the initial database migration was executed successfully.
  * You can create a new developer API key from the embedded React Admin dashboard.

For more detailed runbooks, see these files:
* Deployment Runbook: `docs/runbooks/deployment.md`
* Rollback Runbook: `docs/runbooks/rollback.md`
* Incident Response: `docs/runbooks/incident-response.md`
* Local Development: `docs/runbooks/local-development.md`
* Environment Management: `docs/runbooks/environment-management.md`
