# No-Vault Full Deployment Workaround

This profile is the fastest path to full platform deployment when Vault and External Secrets Operator are not ready.

## What this does

- Uses plain Kubernetes Secrets and ConfigMaps (no Vault dependency)
- Keeps base manifests intact
- Deploys via a dedicated overlay: infra/k8s/overlays/sandbox-no-vault
- Reduces moving parts to a minimum reproducible stack

## Why this is the right workaround

- You avoid hard blockers from Vault auth bootstrapping and sealed-secret generation.
- You can validate full service topology and runtime behavior first.
- You can migrate to Vault later with zero service code changes.

## Deploy

Windows PowerShell:

```powershell
./scripts/bootstrap/deploy-sandbox-no-vault.ps1
```

Manual equivalent:

```powershell
kubectl apply -k infra/k8s/overlays/sandbox-no-vault
kubectl rollout status deployment/k-svc -n kubric --timeout=300s
kubectl rollout status deployment/vdr -n kubric --timeout=300s
kubectl rollout status deployment/kic -n kubric --timeout=300s
kubectl rollout status deployment/noc -n kubric --timeout=300s
kubectl rollout status deployment/kai-core -n kubric --timeout=300s
kubectl rollout status deployment/web -n kubric --timeout=300s
kubectl rollout status deployment/n8n -n kubric --timeout=300s
```

## Workaround scope

This overlay provides the required runtime objects for:

- postgresql-credentials
- postgresql-tls
- postgresql-config
- postgresql-init-scripts
- kubric-db-credentials
- kubric-auth-secrets
- nats-config
- nats-tls
- clickhouse-config
- clickhouse-users

## Security note

This is for sandbox/staging acceleration only. Replace placeholder secrets before production traffic.

## Upgrade path to enterprise hardened mode

1. Keep this overlay for recovery/smoke testing.
2. Add Vault/ESO overlay in parallel.
3. Switch deployment target from sandbox-no-vault to prod overlay once secret delivery is stable.
