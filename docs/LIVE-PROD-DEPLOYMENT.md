# Live Production Deployment

This is the durable deployment path for production-like environments.

## What it does

- Applies `infra/k8s/overlays/prod`
- Pins deployment images to explicit tags
- Waits for statefulsets and deployments to roll out
- Runs runtime smoke checks via `ops-batch-04` (service API-proxy probes)

## One-command path

```powershell
make deploy-live-prod
```

## Tagged image deployment

Use explicit immutable tags for each service:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/deploy-live-prod.ps1 `
  -Namespace kubric `
  -KSvcTag <tag> `
  -VdrTag <tag> `
  -KicTag <tag> `
  -NocTag <tag> `
  -KaiTag <tag> `
  -WebTag <tag> `
  -N8nTag <tag>
```

## Preconditions

- `kubectl` authenticated to target cluster/context
- Required secrets/config for prod overlay already provisioned
- Container images pushed and pullable by cluster

## Post-deploy checks

```powershell
kubectl get pods -n kubric
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-04-runtime-smoke.ps1 -Namespace kubric
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-05-docker-libs.ps1
```
