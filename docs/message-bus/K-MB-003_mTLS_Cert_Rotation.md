# K-MB-003 — mTLS Certificate Rotation

## Overview

All NATS cluster communication uses mutual TLS (mTLS). Certificates are issued by cert-manager via Vault PKI and auto-rotated before expiry.

## Certificate Chain

```
Kubric Root CA (Vault PKI)
    │
    ├── NATS Cluster CA (Intermediate)
    │   ├── nats-0.kubric.svc (server cert)
    │   ├── nats-1.kubric.svc (server cert)
    │   └── nats-2.kubric.svc (server cert)
    │
    └── NATS Client CA (Intermediate)
        ├── coresec-agent (client cert)
        ├── netguard-agent (client cert)
        ├── kai-core (client cert)
        ├── k-svc (client cert)
        └── vdr (client cert)
```

## cert-manager Integration

```yaml
# Issuer backed by Vault PKI
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: kubric-vault-pki
spec:
  vault:
    server: https://vault.kubric.svc.cluster.local:8200
    path: pki/sign/kubric-nats
    auth:
      kubernetes:
        role: cert-manager
        mountPath: /v1/auth/kubernetes
        serviceAccountRef:
          name: cert-manager

---
# NATS server certificate (auto-renewed)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: nats-tls
  namespace: kubric
spec:
  secretName: nats-tls
  issuerRef:
    name: kubric-vault-pki
    kind: ClusterIssuer
  duration: 720h      # 30 days
  renewBefore: 168h    # Renew 7 days before expiry
  commonName: nats.kubric.svc.cluster.local
  dnsNames:
    - nats.kubric.svc.cluster.local
    - nats-headless.kubric.svc.cluster.local
    - "*.nats-headless.kubric.svc.cluster.local"
    - nats-0.nats-headless.kubric.svc.cluster.local
    - nats-1.nats-headless.kubric.svc.cluster.local
    - nats-2.nats-headless.kubric.svc.cluster.local
  usages:
    - server auth
    - client auth
```

## Rotation Procedure

1. cert-manager checks certificate expiry every 24h
2. When `renewBefore` threshold is reached (7 days before expiry), cert-manager requests a new cert from Vault PKI
3. New cert is written to the `nats-tls` K8s Secret
4. NATS server detects TLS secret update and reloads certificates without restart (NATS supports hot TLS reload via SIGHUP)
5. Prometheus alert fires if cert renewal fails:

```yaml
- alert: NATSTLSCertExpiringSoon
  expr: (certmanager_certificate_expiration_timestamp_seconds - time()) < 172800
  for: 1h
  labels:
    severity: critical
  annotations:
    summary: "NATS TLS certificate expires in less than 48h"
```

## Manual Rotation (Emergency)

```bash
# Force cert-manager to renew immediately
kubectl delete secret nats-tls -n kubric
# cert-manager will recreate with a fresh cert

# Verify new cert
kubectl get certificate nats-tls -n kubric -o jsonpath='{.status.conditions}'

# Reload NATS without restart
kubectl exec nats-0 -n kubric -- nats-server --signal reload
```
