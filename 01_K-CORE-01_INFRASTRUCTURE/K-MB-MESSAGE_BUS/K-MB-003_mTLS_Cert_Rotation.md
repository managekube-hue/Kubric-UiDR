# K-MB-003  mTLS Certificate Rotation Runbook

**Component**: NATS 2.10 mTLS  
**Namespace**: kubric  
**Last Updated**: 2026-02-26  
**Owner**: Platform Engineering / Security Operations  

---

## Table of Contents

1. [Certificate Hierarchy](#1-certificate-hierarchy)
2. [Secret Naming Convention](#2-secret-naming-convention)
3. [cert-manager Manifests](#3-cert-manager-manifests)
4. [NATS Server TLS Configuration](#4-nats-server-tls-configuration)
5. [Client TLS Configuration](#5-client-tls-configuration)
6. [Standard Rotation Procedure](#6-standard-rotation-procedure)
7. [Emergency Rotation](#7-emergency-rotation)
8. [Verification Checklist](#8-verification-checklist)

---

## 1. Certificate Hierarchy

    Root CA (self-signed, 10yr, offline / HSM)
      |
      +-- Intermediate CA (cert-manager ClusterIssuer, 2yr)
            |
            +-- nats-server cert (90d, SANs: DNS, IP)
            |     Issued to: nats-cluster.kubric.svc.cluster.local
            |
            +-- nats-client cert (90d, CN: kubric-agent)
            |     Used by: Python agents, Go workers, KAI orchestrator
            |
            +-- nats-leafnode cert (90d, CN: kubric-leafnode)
                  Used by: edge agent leaf node connections

All certificates use:
  Key algorithm : ECDSA P-384
  Signature hash: SHA-384
  Key usage      : digitalSignature, keyEncipherment
  Extended usage : serverAuth (server cert), clientAuth (client cert)

Root CA is stored offline. The Intermediate CA private key lives in a
Kubernetes Secret (nats-intermediate-ca) and is NEVER exported.

---

## 2. Secret Naming Convention

| Secret Name | Namespace | Contents | Used By |
|-------------|-----------|----------|---------|
| nats-tls | kubric | server tls.crt, tls.key, ca.crt | NATS StatefulSet pods |
| nats-client-tls | kubric | client tls.crt, tls.key, ca.crt | Python/Go service pods |
| nats-leafnode-tls | kubric | leafnode tls.crt, tls.key, ca.crt | Edge agent pods |
| nats-intermediate-ca | kubric | ca.crt, tls.key | cert-manager ClusterIssuer |
| nats-root-ca | kubric | ca.crt only | cert-manager CA chain validation |

All secrets use defaultMode: 0400 when mounted as volumes.

---

## 3. cert-manager Manifests

### 3.1 ClusterIssuer backed by Intermediate CA

    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: kubric-nats-issuer
    spec:
      ca:
        secretName: nats-intermediate-ca

### 3.2 NATS Server Certificate (90-day, auto-renew at 67.5d)

    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: nats-server-cert
      namespace: kubric
    spec:
      secretName: nats-tls
      duration: 2160h        # 90 days
      renewBefore: 540h      # renew at 67.5 days (75% of lifetime)
      subject:
        organizations:
          - kubric
      commonName: nats-cluster.kubric.svc.cluster.local
      dnsNames:
        - nats-cluster.kubric.svc.cluster.local
        - nats-cluster-headless.kubric.svc.cluster.local
        - "*.nats-cluster-headless.kubric.svc.cluster.local"
        - localhost
      ipAddresses:
        - 127.0.0.1
      usages:
        - digital signature
        - key encipherment
        - server auth
        - client auth
      privateKey:
        algorithm: ECDSA
        size: 384
      issuerRef:
        name: kubric-nats-issuer
        kind: ClusterIssuer
        group: cert-manager.io

### 3.3 NATS Client Certificate (90-day, auto-renew at 67.5d)

    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: nats-client-cert
      namespace: kubric
    spec:
      secretName: nats-client-tls
      duration: 2160h
      renewBefore: 540h
      subject:
        organizations:
          - kubric
      commonName: kubric-agent
      usages:
        - digital signature
        - key encipherment
        - client auth
      privateKey:
        algorithm: ECDSA
        size: 384
      issuerRef:
        name: kubric-nats-issuer
        kind: ClusterIssuer
        group: cert-manager.io

### 3.4 NATS LeafNode Certificate (90-day)

    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: nats-leafnode-cert
      namespace: kubric
    spec:
      secretName: nats-leafnode-tls
      duration: 2160h
      renewBefore: 540h
      subject:
        organizations:
          - kubric
      commonName: kubric-leafnode
      usages:
        - digital signature
        - key encipherment
        - server auth
        - client auth
      privateKey:
        algorithm: ECDSA
        size: 384
      issuerRef:
        name: kubric-nats-issuer
        kind: ClusterIssuer
        group: cert-manager.io
---

## 4. NATS Server TLS Configuration

The NATS server tls block (in the nats.conf ConfigMap) configures mTLS:

    tls {
      # Paths match the Secret mount at /etc/nats-tls/
      cert_file:  "/etc/nats-tls/tls.crt"
      key_file:   "/etc/nats-tls/tls.key"
      ca_file:    "/etc/nats-tls/ca.crt"

      # verify: true  -> server verifies client certs (mTLS)
      verify: true

      # timeout: TLS handshake deadline in seconds
      timeout: 5

      # min_version restricts TLS negotiation to TLS 1.3 only
      # (TLS 1.2 may be required for some legacy NATS clients)
      # min_version: "TLS1.3"
    }

Cluster routes also require TLS when running mTLS:

    cluster {
      name: kubric-nats-cluster
      port: 6222
      routes: ["nats://nats-cluster-headless.kubric.svc.cluster.local:6222"]

      tls {
        cert_file: "/etc/nats-tls/tls.crt"
        key_file:  "/etc/nats-tls/tls.key"
        ca_file:   "/etc/nats-tls/ca.crt"
        verify:    true
        timeout:   5
      }
    }

JWT/NKey operator config (required for account resolver):

    operator: /etc/nats-operator/kubric-operator.jwt

    resolver: {
      type: full
      dir:  /data/resolver
      allow_delete: false
      interval: "2m"
    }

    system_account: SYS

---

## 5. Client TLS Configuration

### 5.1 Python (nats-py)

    import ssl
    import nats

    async def create_nats_client(tenant_id: str) -> nats.NATS:
        tls_ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
        tls_ctx.load_verify_locations("/etc/nats-client-tls/ca.crt")
        tls_ctx.load_cert_chain(
            certfile="/etc/nats-client-tls/tls.crt",
            keyfile="/etc/nats-client-tls/tls.key",
        )
        # NATS server CN must match server hostname for TLS_CLIENT
        tls_ctx.check_hostname = True
        tls_ctx.verify_mode = ssl.CERT_REQUIRED

        nc = await nats.connect(
            servers=["nats://nats-cluster.kubric.svc.cluster.local:4222"],
            tls=tls_ctx,
            tls_hostname="nats-cluster.kubric.svc.cluster.local",
        )
        return nc

Mount the nats-client-tls Secret as a volume in the agent Pod:

    volumes:
      - name: nats-client-tls
        secret:
          secretName: nats-client-tls
          defaultMode: 0400
    containers:
      - name: agent
        volumeMounts:
          - name: nats-client-tls
            mountPath: /etc/nats-client-tls
            readOnly: true

### 5.2 Go (nats.go)

    package natsclient

    import (
        "crypto/tls"
        "crypto/x509"
        "os"

        "github.com/nats-io/nats.go"
    )

    func NewNATSConn() (*nats.Conn, error) {
        caCert, err := os.ReadFile("/etc/nats-client-tls/ca.crt")
        if err != nil {
            return nil, err
        }
        caPool := x509.NewCertPool()
        caPool.AppendCertsFromPEM(caCert)

        clientCert, err := tls.LoadX509KeyPair(
            "/etc/nats-client-tls/tls.crt",
            "/etc/nats-client-tls/tls.key",
        )
        if err != nil {
            return nil, err
        }

        tlsConf := &tls.Config{
            Certificates:       []tls.Certificate{clientCert},
            RootCAs:            caPool,
            ServerName:         "nats-cluster.kubric.svc.cluster.local",
            MinVersion:         tls.VersionTLS12,
        }

        return nats.Connect(
            "nats://nats-cluster.kubric.svc.cluster.local:4222",
            nats.Secure(tlsConf),
        )
    }

### 5.3 Rust (async-nats)

    use async_nats::ConnectOptions;
    use rustls::{ClientConfig, RootCertStore};

    pub async fn connect() -> async_nats::Client {
        let opts = ConnectOptions::new()
            .add_root_certificates(std::path::Path::new("/etc/nats-client-tls/ca.crt"))
            .add_client_certificate(
                std::path::Path::new("/etc/nats-client-tls/tls.crt"),
                std::path::Path::new("/etc/nats-client-tls/tls.key"),
            );
        opts.connect("nats://nats-cluster.kubric.svc.cluster.local:4222")
            .await
            .expect("NATS connect failed")
    }
---

## 6. Standard Rotation Procedure

cert-manager performs automatic rotation at 75% of the certificate lifetime
(67.5 days for a 90-day cert). This section covers the manual rotation
procedure for planned maintenance or cert-manager failures.

### Step 1 -- Verify current cert expiry

    kubectl -n kubric get secret nats-tls -o json       | python3 -c "
    import sys, json, base64, subprocess
    d = json.load(sys.stdin)
    cert_b64 = d['data']['tls.crt']
    cert_pem = base64.b64decode(cert_b64)
    with open('/tmp/nats-tls.crt','wb') as f: f.write(cert_pem)
    "
    openssl x509 -in /tmp/nats-tls.crt -noout -dates

### Step 2 -- Trigger cert-manager renewal

cert-manager renewal is triggered by annotating the Certificate resource:

    kubectl -n kubric annotate certificate nats-server-cert       cert-manager.io/force-at="2026-02-27T04:10:11Z"       --overwrite

    kubectl -n kubric annotate certificate nats-client-cert       cert-manager.io/force-at="2026-02-27T04:10:11Z"       --overwrite

Monitor renewal:

    kubectl -n kubric get certificaterequests -w

### Step 3 -- Confirm new cert is in Secret

    kubectl -n kubric get secret nats-tls -o jsonpath='{.metadata.resourceVersion}'

The resourceVersion increments when the Secret is updated. Verify the new
cert expiry (repeat Step 1).

### Step 4 -- Rolling restart NATS pods

NATS 2.10 supports live TLS reload without restart via SIGHUP.
Send the signal through the management HTTP endpoint:

    # Signal TLS reload on each pod
    for i in 0 1 2; do
      kubectl -n kubric exec nats-cluster- --         nats-server --signal reopen
    done

If live reload fails, perform a rolling restart:

    kubectl -n kubric rollout restart statefulset nats-cluster
    kubectl -n kubric rollout status statefulset nats-cluster --timeout=5m

### Step 5 -- Restart client workloads

Client pods mount the Secret at startup. Trigger a rolling restart of
all workloads that mount nats-client-tls:

    kubectl -n kubric rollout restart deployment edr-agent
    kubectl -n kubric rollout restart deployment ndr-agent
    kubectl -n kubric rollout restart deployment kai-orchestrator

Verify connectivity after restart:

    nats server ping --count 3

---

## 7. Emergency Rotation

Use emergency rotation when a private key is suspected to be compromised.

### 7.1 Immediate Key Revocation

cert-manager does not maintain a CRL natively. For immediate revocation:

1. Delete the compromised Secret to force cert-manager to reissue:

       kubectl -n kubric delete secret nats-tls
       # cert-manager will recreate within ~30 seconds

2. For client cert compromise, delete the client Secret:

       kubectl -n kubric delete secret nats-client-tls

3. Update the Intermediate CA if the intermediate key is compromised:

       # Replace the nats-intermediate-ca Secret with new key material
       # Generated offline from the Root CA
       kubectl -n kubric create secret tls nats-intermediate-ca          --cert=/path/to/new-intermediate.crt          --key=/path/to/new-intermediate.key          --dry-run=client -o yaml          | kubectl apply -f -

### 7.2 OCSP Stapling (future)

NATS 2.10 supports OCSP stapling. Enable via nats.conf:

    ocsp: {
      mode: must_staple
    }

When enabled, NATS will automatically fetch and staple the OCSP response
for the server certificate. Configure the OCSP responder URL in the
Certificate SAN extension via cert-manager annotations.

### 7.3 Emergency Contact Matrix

| Scenario | Escalation Path |
|----------|----------------|
| NATS server cert expired | Platform-on-call pager; P1 incident |
| Client cert mass-revocation needed | Security-on-call; rotate intermediate CA |
| Root CA compromise | CISO + Platform; full PKI rebuild |
| cert-manager controller down | kubectl delete pod -n cert-manager -l app=cert-manager |

---

## 8. Verification Checklist

After any rotation event, verify the following:

- [ ] kubectl get certificate -n kubric -- all show READY=True
- [ ] kubectl get certificaterequest -n kubric -- no Pending/Failed CRs
- [ ] openssl s_client -connect nats-cluster.kubric.svc.cluster.local:4222 -- shows new cert serial
- [ ] nats server ping -- all 3 pods respond
- [ ] nats stream ls -- all 15 streams visible
- [ ] No ERROR entries in kubectl logs -n kubric -l app=nats --since=5m containing "tls"
- [ ] Prometheus alert NATSConnectionDrop is resolved
- [ ] All agent deployments show 0 CrashLoopBackOff pods
- [ ] JetStream consumer lag within normal bounds (nats consumer report)
- [ ] cert-manager logs show no reissuance failures: kubectl logs -n cert-manager -l app=cert-manager --since=10m
