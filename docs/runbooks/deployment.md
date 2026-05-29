# Deployment Runbook

This runbook covers the procedures for deploying new releases of the NexusAI-Gateway to Kubernetes clusters.

**Prerequisites**
* A configured `kubectl` pointing to the target cluster.
* Helm v3 CLI installed.
* Docker CLI installed.
* Target namespace created (default: `nexusai`).

**Step 1: Container Build and Publish**
Ensure your local workspace changes are committed and pushed to the release branch (e.g. `release/v1.0.0`). The CI/CD pipeline will automatically build and publish the container image to GHCR. To perform this manually:
```bash
docker build -t ghcr.io/thomasvnn/nexusai-gateway:v1.0.0 -f deployments/Dockerfile .
docker push ghcr.io/thomasvnn/nexusai-gateway:v1.0.0
```

**Step 2: Deployment via Helm**
Using the Helm chart under `deployments/helm/`, deploy or upgrade the service:
```bash
# Update local variables or values.yaml before running
helm upgrade --install nexusai-gateway ./deployments/helm \
  --namespace nexusai \
  --set image.tag="v1.0.0" \
  --values ./deployments/helm/values.yaml
```

**Step 3: Verification**
Verify that the pods are running and healthy:
```bash
kubectl get pods -n nexusai -l app.kubernetes.io/name=nexusai-gateway
```
Monitor the rolling update status:
```bash
kubectl rollout status deployment/nexusai-gateway -n nexusai
```

**Step 4: Post-Deployment Smoke Test**
Execute a curl against the health liveness/readiness endpoint:
```bash
curl -i http://gateway.nexusai.local/healthz
curl -i http://gateway.nexusai.local/readyz
```
Inspect the `/metrics` output to verify logging and telemetry:
```bash
curl http://gateway.nexusai.local/metrics
```
