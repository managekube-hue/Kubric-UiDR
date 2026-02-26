# K-SEC-001 — HashiCorp Vault

## Overview

Kubric uses HashiCorp Vault as the single source of truth for all secrets,
credentials, and cryptographic material. No secrets are stored in environment
variables, K8s Secrets, or `.env` files in production.

## Architecture

```
┌──────────────┐     K8s SA JWT      ┌──────────────┐
│  Go Service  │  ────────────────▶  │    Vault      │
│  (k-svc)     │  ◀── dynamic creds  │  (raft HA)    │
└──────────────┘                     └──────┬───────┘
                                            │
┌──────────────┐     K8s SA JWT      ┌──────▼───────┐
│  Rust Agent  │  ────────────────▶  │  Database     │
│  (coresec)   │  ◀── NATS creds    │  Engine       │
└──────────────┘                     └──────────────┘
```

## Secret Scopes

| Path | Purpose | Consumers |
|------|---------|-----------|
| `secret/data/agents/*` | Agent NATS + config | Rust agents |
| `secret/data/nats/*` | NATS connection creds | All services |
| `database/creds/kubric-*` | Dynamic PostgreSQL creds | Go services |
| `secret/data/stripe/*` | Stripe API + webhook | K-SVC, KAI |
| `secret/data/ti/*` | Threat intel API keys | KAI-TI |
| `secret/data/llm/*` | LLM provider keys | KAI |
| `pki_int/issue/*` | mTLS certificates | All services |
| `transit/encrypt/blake3-audit` | HSM-backed signing | Audit chain |

## Deployment

```bash
# Dev mode (docker-compose)
vault server -dev -dev-root-token-id=root

# Production (K8s via Helm — ArgoCD managed)
# See infra/argocd/kubric-app-of-apps.yaml → kubric-vault application

# Apply policies
vault policy write kubric-default config/vault/policies.hcl
```

## Implementation References

- **Go Vault client**: `internal/security/vault_k8s_auth.go`
- **Vault policies**: `config/vault/policies.hcl`
- **Service credential loader**: `internal/security/config.go`
- **K8s ServiceAccount annotations**: `infra/k8s/base/*.yaml` (vault.hashicorp.com/agent-inject)
- **Dev fallback**: env vars when `KUBERNETES_SERVICE_HOST` is unset
