# K-SEC-004 — Internal CA Setup

## Overview

Kubric operates its own internal Certificate Authority (CA) backed by HashiCorp Vault PKI secrets engine. This CA issues TLS certificates for all internal services, NATS mTLS, agent-to-server authentication, and inter-service communication.

## CA Hierarchy

```
Kubric Root CA (offline, Vault PKI)
  │  Validity: 10 years
  │  Key: RSA 4096 or ECDSA P-384
  │  Stored: Vault transit backend (or TPM-sealed)
  │
  ├── Kubric Intermediate CA (online, Vault PKI)
  │   │  Validity: 3 years
  │   │  Auto-rotated by Vault
  │   │
  │   ├── Service Certificates (short-lived)
  │   │   ├── k-svc.kubric.svc.cluster.local    (30d, auto-renewed)
  │   │   ├── kai-core.kubric.svc.cluster.local  (30d, auto-renewed)
  │   │   ├── vdr.kubric.svc.cluster.local       (30d, auto-renewed)
  │   │   └── kic.kubric.svc.cluster.local       (30d, auto-renewed)
  │   │
  │   ├── NATS Cluster Certificates
  │   │   ├── nats-0.nats-headless.kubric.svc    (30d, auto-renewed)
  │   │   ├── nats-1.nats-headless.kubric.svc    (30d, auto-renewed)
  │   │   └── nats-2.nats-headless.kubric.svc    (30d, auto-renewed)
  │   │
  │   └── Agent Client Certificates
  │       ├── coresec-agent-{asset_id}           (7d, auto-renewed)
  │       ├── netguard-agent-{asset_id}          (7d, auto-renewed)
  │       └── perftrace-agent-{asset_id}         (7d, auto-renewed)
  │
  └── Customer Portal CA (separate intermediate)
      └── *.kubric.io                            (90d, Let's Encrypt or Vault)
```

## Vault PKI Setup

### 1. Enable PKI Secrets Engine

```bash
# Root CA
vault secrets enable -path=pki pki
vault secrets tune -max-lease-ttl=87600h pki  # 10 years

# Generate root certificate
vault write pki/root/generate/internal \
  common_name="Kubric Root CA" \
  ttl=87600h \
  key_type=ec \
  key_bits=384

# Intermediate CA
vault secrets enable -path=pki_int pki
vault secrets tune -max-lease-ttl=26280h pki_int  # 3 years

# Generate intermediate CSR
vault write pki_int/intermediate/generate/internal \
  common_name="Kubric Intermediate CA" \
  key_type=ec \
  key_bits=384

# Sign intermediate with root
vault write pki/root/sign-intermediate \
  csr=@intermediate.csr \
  ttl=26280h

# Set signed certificate
vault write pki_int/intermediate/set-signed \
  certificate=@signed_intermediate.pem
```

### 2. Create Roles

```bash
# Service certificates (30 day, SAN-based)
vault write pki_int/roles/kubric-service \
  allowed_domains="kubric.svc.cluster.local" \
  allow_subdomains=true \
  max_ttl=720h \
  key_type=ec \
  key_bits=256

# Agent certificates (7 day, short-lived)
vault write pki_int/roles/kubric-agent \
  allowed_domains="agent.kubric.internal" \
  allow_subdomains=true \
  max_ttl=168h \
  key_type=ec \
  key_bits=256

# NATS cluster certificates
vault write pki_int/roles/kubric-nats \
  allowed_domains="kubric.svc.cluster.local" \
  allow_subdomains=true \
  allow_glob_domains=true \
  max_ttl=720h \
  key_type=ec \
  key_bits=256
```

### 3. Issue Certificate

```bash
vault write pki_int/issue/kubric-service \
  common_name="k-svc.kubric.svc.cluster.local" \
  alt_names="k-svc,localhost" \
  ttl=720h
```

## cert-manager Integration

cert-manager auto-requests and rotates certificates from Vault PKI:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: kubric-vault-pki
spec:
  vault:
    server: https://vault.kubric.svc.cluster.local:8200
    path: pki_int/sign/kubric-service
    auth:
      kubernetes:
        role: cert-manager
        mountPath: /v1/auth/kubernetes
        serviceAccountRef:
          name: cert-manager
```

## Certificate Revocation

```bash
# Revoke a compromised certificate
vault write pki_int/revoke \
  serial_number="39:dd:2e:90:..."

# Tidy revoked certificates
vault write pki_int/tidy \
  tidy_cert_store=true \
  tidy_revoked_certs=true \
  safety_buffer=72h
```

## Monitoring

| Metric | Alert Threshold |
|--------|----------------|
| Certificate expiry | < 48h remaining |
| CRL size | > 1000 entries |
| Issuance rate | > 100/hour (anomaly) |
| Root CA access | Any access (critical) |
