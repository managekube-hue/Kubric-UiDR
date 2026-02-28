# K-XRO-SD-001 -- RustDesk Remote Access Sidecar

**License:** AGPL 3.0 — runs as SEPARATE process (not imported as Cargo lib).  
**Boundary:** XRO agent communicates with RustDesk via local Unix socket only. No AGPL code is compiled into Kubric binaries.

---

## 1. Architecture

```
┌──────────────────────────────────┐
│  Pod / Docker Host               │
│                                  │
│  ┌────────────┐   Unix Socket   ┌──────────────┐
│  │ XRO Agent  │◄───────────────►│  RustDesk    │
│  │ (Rust)     │ /tmp/rustdesk   │  Server      │
│  │            │   .sock         │  (AGPL 3.0)  │
│  └────────────┘                 └──────────────┘
│        │                              │
│        │ NATS                         │ TCP/21116-21119
│        ▼                              ▼
│   kubric.edr.*              Remote technician
└──────────────────────────────────┘
```

XRO agent starts/stops RustDesk via `std::process::Command`. No Rust crate import. The two processes share a Unix domain socket for control messages (session approve/deny, connection metadata).

---

## 2. RustDesk Server Docker Image

```dockerfile
# docker/rustdesk-server/Dockerfile
FROM rustdesk/rustdesk-server:1.1.11

# Custom relay configuration
ENV RELAY=kubric-relay.internal:21117
ENV ENCRYPTED_ONLY=1

# TLS certificates for encrypted connections
COPY certs/rustdesk-relay.crt /root/
COPY certs/rustdesk-relay.key /root/

EXPOSE 21115 21116 21117 21118 21119

CMD ["hbbs", "-r", "kubric-relay.internal:21117", "-k", "_"]
```

---

## 3. Docker Compose Sidecar Definition

```yaml
# docker-compose.yml (snippet)
services:
  xro-agent:
    image: ghcr.io/kubric/xro-agent:latest
    privileged: true
    network_mode: host
    volumes:
      - /var/run/rustdesk:/var/run/rustdesk
      - /etc/kubric:/etc/kubric:ro
    environment:
      NATS_URL: nats://nats:4222
      RUSTDESK_SOCK: /var/run/rustdesk/control.sock
      RUSTDESK_ALLOWLIST: "10.0.0.0/8,172.16.0.0/12"
    depends_on:
      - rustdesk-sidecar

  rustdesk-sidecar:
    image: rustdesk/rustdesk-server:1.1.11
    restart: unless-stopped
    volumes:
      - /var/run/rustdesk:/var/run/rustdesk
      - rustdesk-data:/root
      - ./certs/rustdesk:/certs:ro
    environment:
      ENCRYPTED_ONLY: "1"
      ALWAYS_USE_RELAY: "Y"
    ports:
      - "21115:21115"   # NAT test
      - "21116:21116"   # TCP hole punching
      - "21117:21117"   # Relay
      - "21118:21118"   # WebSocket for web client
    command: ["hbbs", "-r", "localhost:21117", "-k", "_"]
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "21116"]
      interval: 30s
      timeout: 5s
      retries: 3

  rustdesk-relay:
    image: rustdesk/rustdesk-server:1.1.11
    restart: unless-stopped
    volumes:
      - ./certs/rustdesk:/certs:ro
    command: ["hbbr", "-k", "_"]
    ports:
      - "21117:21117"

volumes:
  rustdesk-data:
```

---

## 4. Kubernetes Sidecar Container Spec

```yaml
# deployments/k8s/xro-agent-daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: xro-agent
  namespace: kubric-agents
  labels:
    app.kubernetes.io/name: xro-agent
    app.kubernetes.io/component: endpoint-agent
spec:
  selector:
    matchLabels:
      app: xro-agent
  template:
    metadata:
      labels:
        app: xro-agent
    spec:
      hostPID: true
      hostNetwork: true
      serviceAccountName: xro-agent
      containers:
        # --- Primary XRO Agent ---
        - name: xro-agent
          image: ghcr.io/kubric/xro-agent:latest
          securityContext:
            privileged: true
          env:
            - name: NATS_URL
              valueFrom:
                secretKeyRef:
                  name: kubric-nats
                  key: url
            - name: RUSTDESK_SOCK
              value: /var/run/rustdesk/control.sock
            - name: RUSTDESK_ALLOWLIST
              value: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
          volumeMounts:
            - name: rustdesk-sock
              mountPath: /var/run/rustdesk
            - name: kubric-config
              mountPath: /etc/kubric
              readOnly: true
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi

        # --- RustDesk Sidecar (AGPL 3.0 — separate process) ---
        - name: rustdesk-sidecar
          image: rustdesk/rustdesk-server:1.1.11
          args: ["hbbs", "-r", "localhost:21117", "-k", "_"]
          env:
            - name: ENCRYPTED_ONLY
              value: "1"
          ports:
            - containerPort: 21116
              protocol: TCP
            - containerPort: 21118
              protocol: TCP
          volumeMounts:
            - name: rustdesk-sock
              mountPath: /var/run/rustdesk
            - name: rustdesk-certs
              mountPath: /certs
              readOnly: true
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 256Mi
          livenessProbe:
            tcpSocket:
              port: 21116
            initialDelaySeconds: 10
            periodSeconds: 30

      volumes:
        - name: rustdesk-sock
          emptyDir:
            medium: Memory
        - name: kubric-config
          configMap:
            name: kubric-agent-config
        - name: rustdesk-certs
          secret:
            secretName: rustdesk-tls
```

---

## 5. XRO Agent Control Interface (Rust)

The XRO agent manages RustDesk sessions through a local Unix socket control protocol. RustDesk is spawned as a child process — never linked as a Rust crate.

```rust
// agents/xro/src/sidecars/rustdesk.rs

use std::os::unix::net::UnixStream;
use std::process::{Child, Command, Stdio};
use std::io::{Read, Write};
use serde::{Deserialize, Serialize};
use tokio::sync::Mutex;
use tracing::{info, warn, error};

const RUSTDESK_SOCK: &str = "/var/run/rustdesk/control.sock";
const RUSTDESK_BIN: &str = "/usr/bin/rustdesk";

/// Allowlist for IP ranges that may connect.
#[derive(Clone)]
pub struct ConnectionAllowlist {
    pub cidrs: Vec<ipnet::IpNet>,
}

impl ConnectionAllowlist {
    pub fn from_env() -> Self {
        let raw = std::env::var("RUSTDESK_ALLOWLIST")
            .unwrap_or_else(|_| "10.0.0.0/8,172.16.0.0/12".to_string());
        let cidrs = raw
            .split(',')
            .filter_map(|s| s.trim().parse::<ipnet::IpNet>().ok())
            .collect();
        Self { cidrs }
    }

    pub fn is_allowed(&self, addr: &std::net::IpAddr) -> bool {
        self.cidrs.iter().any(|net| net.contains(addr))
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SessionRequest {
    pub remote_ip: String,
    pub technician_id: String,
    pub reason: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SessionResponse {
    pub approved: bool,
    pub session_id: Option<String>,
    pub deny_reason: Option<String>,
}

pub struct RustDeskSidecar {
    child: Mutex<Option<Child>>,
    allowlist: ConnectionAllowlist,
}

impl RustDeskSidecar {
    pub fn new() -> Self {
        Self {
            child: Mutex::new(None),
            allowlist: ConnectionAllowlist::from_env(),
        }
    }

    /// Start the RustDesk server process (AGPL boundary — separate process).
    pub async fn start(&self) -> anyhow::Result<()> {
        let mut guard = self.child.lock().await;
        if guard.is_some() {
            warn!("RustDesk sidecar already running");
            return Ok(());
        }

        let child = Command::new(RUSTDESK_BIN)
            .arg("--server")
            .arg("--socket")
            .arg(RUSTDESK_SOCK)
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .spawn()?;

        info!(pid = child.id(), "RustDesk sidecar started");
        *guard = Some(child);
        Ok(())
    }

    /// Stop the RustDesk process.
    pub async fn stop(&self) -> anyhow::Result<()> {
        let mut guard = self.child.lock().await;
        if let Some(ref mut child) = *guard {
            child.kill()?;
            child.wait()?;
            info!("RustDesk sidecar stopped");
        }
        *guard = None;
        Ok(())
    }

    /// Approve or deny a remote session based on allowlist.
    pub async fn handle_session_request(
        &self,
        req: SessionRequest,
    ) -> SessionResponse {
        let ip: std::net::IpAddr = match req.remote_ip.parse() {
            Ok(ip) => ip,
            Err(_) => {
                return SessionResponse {
                    approved: false,
                    session_id: None,
                    deny_reason: Some("Invalid IP address".into()),
                };
            }
        };

        if !self.allowlist.is_allowed(&ip) {
            warn!(
                remote_ip = %req.remote_ip,
                technician = %req.technician_id,
                "Connection DENIED — not in allowlist"
            );
            return SessionResponse {
                approved: false,
                session_id: None,
                deny_reason: Some(format!(
                    "IP {} not in allowlist", req.remote_ip
                )),
            };
        }

        let session_id = uuid::Uuid::new_v4().to_string();
        info!(
            session_id = %session_id,
            remote_ip = %req.remote_ip,
            technician = %req.technician_id,
            "Remote session APPROVED"
        );

        SessionResponse {
            approved: true,
            session_id: Some(session_id),
            deny_reason: None,
        }
    }

    /// Send control message via Unix socket.
    pub fn send_control(&self, msg: &[u8]) -> anyhow::Result<Vec<u8>> {
        let mut stream = UnixStream::connect(RUSTDESK_SOCK)?;
        stream.write_all(msg)?;
        stream.flush()?;
        let mut buf = vec![0u8; 4096];
        let n = stream.read(&mut buf)?;
        buf.truncate(n);
        Ok(buf)
    }
}
```

---

## 6. Security Configuration

### 6.1 Certificate Pinning

```yaml
# config/rustdesk/pinning.yaml
tls:
  ca_cert: /certs/kubric-ca.pem
  server_cert: /certs/rustdesk-relay.crt
  server_key: /certs/rustdesk-relay.key
  pin_sha256:
    - "sha256//BASE64_ENCODED_PUBLIC_KEY_HASH_1="
    - "sha256//BASE64_ENCODED_PUBLIC_KEY_HASH_2="
  min_tls_version: "1.3"

connection_policy:
  encrypted_only: true
  allowlist_only: true
  max_concurrent_sessions: 3
  session_timeout_minutes: 60
  require_mfa: true
```

### 6.2 Generate Pinning Hash

```bash
# Extract public key pin from certificate
openssl x509 -in /certs/rustdesk-relay.crt -pubkey -noout \
  | openssl pkey -pubin -outform der \
  | openssl dgst -sha256 -binary \
  | openssl enc -base64
```

### 6.3 Firewall Rules

```bash
# Only allow relay traffic from allowlisted ranges
iptables -A INPUT -p tcp --dport 21116:21119 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 21116:21119 -s 172.16.0.0/12 -j ACCEPT
iptables -A INPUT -p tcp --dport 21116:21119 -j DROP
```

---

## 7. NATS Event Publishing

When a remote session is established, the XRO agent publishes an audit event:

```rust
// Publish session event to NATS
let event = serde_json::json!({
    "class_uid": 3002,  // OCSF AuthenticationActivity
    "activity_id": 1,   // Logon
    "category_uid": 3,  // Identity & Access Management
    "severity_id": 3,   // Medium
    "time": chrono::Utc::now().to_rfc3339(),
    "actor": {
        "user": { "name": &req.technician_id }
    },
    "src_endpoint": {
        "ip": &req.remote_ip
    },
    "dst_endpoint": {
        "hostname": hostname::get()?.to_string_lossy()
    },
    "metadata": {
        "product": { "name": "RustDesk Sidecar", "vendor_name": "Kubric" },
        "version": "1.0.0"
    },
    "status_id": if approved { 1 } else { 2 },  // Success / Failure
    "unmapped": {
        "session_id": &session_id,
        "method": "rustdesk_remote"
    }
});

nats_client
    .publish(
        format!("kubric.edr.remote.{}", tenant_id),
        serde_json::to_vec(&event)?.into(),
    )
    .await?;
```

---

## 8. Operational Notes

| Item | Detail |
|------|--------|
| License boundary | AGPL 3.0 — process isolation, no crate import |
| Communication | Unix domain socket only |
| Encryption | TLS 1.3 enforced, certificate pinning |
| Access control | IP allowlist + MFA required |
| Audit | All sessions logged to kubric.edr.remote.> |
| Ports | 21115-21119 (NAT, TCP, relay, WS) |
| Health check | TCP probe on 21116 every 30s |
