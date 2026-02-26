# K-SEC-004 — Certificate Authority Setup

## Overview

Kubric uses Vault's PKI secrets engine as an internal Certificate Authority
for mTLS between all services, agents, and data stores.

## PKI Hierarchy

```
Root CA (offline, air-gapped)
  └── Intermediate CA (Vault PKI engine: pki_int/)
        ├── kubric-service   → Go service mTLS certs (720h TTL)
        ├── kubric-agent     → Rust agent mTLS certs (720h TTL)
        └── kubric-nats      → NATS cluster JetStream TLS (720h TTL)
```

## Setup Commands

```bash
# Enable PKI engines
vault secrets enable -path=pki pki
vault secrets tune -max-lease-ttl=87600h pki
vault secrets enable -path=pki_int pki
vault secrets tune -max-lease-ttl=43800h pki_int

# Generate root CA
vault write pki/root/generate/internal \
  common_name="Kubric Root CA" ttl=87600h

# Generate + sign intermediate
vault write -format=json pki_int/intermediate/generate/internal \
  common_name="Kubric Intermediate CA" | jq -r '.data.csr' > pki_int.csr
vault write -format=json pki/root/sign-intermediate \
  csr=@pki_int.csr format=pem_bundle ttl=43800h | \
  jq -r '.data.certificate' > signed_intermediate.pem
vault write pki_int/intermediate/set-signed certificate=@signed_intermediate.pem

# Create issuance roles
vault write pki_int/roles/kubric-service \
  allowed_domains="kubric.svc.cluster.local" allow_subdomains=true max_ttl=720h
vault write pki_int/roles/kubric-agent \
  allowed_domains="kubric.io" allow_subdomains=true max_ttl=720h
vault write pki_int/roles/kubric-nats \
  allowed_domains="nats.kubric.svc.cluster.local" allow_subdomains=true max_ttl=720h
```

## Runtime Certificate Issuance

Via `internal/security/vault_k8s_auth.go`:

```go
cert, key, ca, err := vc.IssueCertificate(ctx, "kubric-service", "k-svc.kubric.svc.cluster.local", nil)
```

## Implementation References

- **Vault PKI client**: `internal/security/vault_k8s_auth.go` — `IssueCertificate()`
- **Vault policies**: `config/vault/policies.hcl` — `pki_int/issue/*` paths
- **cert-manager**: `infra/k8s/base/` — ArgoCD manages cert-manager Helm chart
- **NATS mTLS**: `infra/k8s/statefulsets/nats-statefulset.yaml` — TLS volume mounts
