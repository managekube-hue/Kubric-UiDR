# External Closure Runbook (One-Pass)

Date: 2026-02-27
Mode: No-Vault deployment first, Vault hardening second
Audience: Ops/SRE execution team

## Gate 0 — Start Condition

Proceed only when these are true:

1. Repo migration fixes are present and tests pass.
2. No-Vault overlay exists at `infra/k8s/overlays/sandbox-no-vault`.
3. Ops accepts sandbox-first secret posture (plain K8s secrets) for initial deployment.

If any item is false, stop and resolve before deployment.

---

## Batch 01 — Preflight (must pass)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-01-preflight.ps1 -Namespace kubric
```

Expected result: `Preflight PASSED`

---

## Batch 02 — Detection Asset Closure (must pass)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-02-detection-assets.ps1
```

Expected result:
- Sigma > 100 files
- YARA > 50 files
- Nuclei > 500 files

This closes the “empty detection assets” runtime concern.

---

## Batch 03 — Runtime Deploy (must pass)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-03-deploy-runtime.ps1 -Namespace kubric
```

Expected result:
- Deployments rolled out: `k-svc`, `vdr`, `kic`, `noc`, `kai-core`, `web`, `n8n`
- StatefulSets rolled out: `clickhouse`, `nats`, `postgresql`

---

## Batch 04 — Runtime Smoke (must pass)

```powershell
powershell -ExecutionPolicy Bypass -File scripts/bootstrap/ops-batch-04-runtime-smoke.ps1 -Namespace kubric
```

Expected result: `Runtime smoke PASSED`

This closes runtime readiness for sandbox deployment.

---

## Post-Batch Verification Commands

```powershell
kubectl get pods -n kubric
kubectl get svc -n kubric
kubectl get events -n kubric --sort-by=.lastTimestamp | Select-Object -Last 50
```

---

## Incident Triage Commands (if a batch fails)

```powershell
kubectl describe pod <pod-name> -n kubric
kubectl logs <pod-name> -n kubric --all-containers --tail=200
kubectl get deploy,sts -n kubric
```

---

## Optional Hardening Batch (after sandbox stable)

When you are ready to reintroduce secret management hardening:

```powershell
bash scripts/vault-setup.sh
```

Then migrate from `sandbox-no-vault` overlay to hardened overlay.

---

## Closure Mapping to Audit Concerns

- Secrets management blocker: closed for deployment via no-Vault workaround.
- Detection asset emptiness: closed by Batch 02.
- Runtime deployment uncertainty: closed by Batch 03 + 04.
- Vault-specific blockers: deferred by design, with explicit hardening phase.
