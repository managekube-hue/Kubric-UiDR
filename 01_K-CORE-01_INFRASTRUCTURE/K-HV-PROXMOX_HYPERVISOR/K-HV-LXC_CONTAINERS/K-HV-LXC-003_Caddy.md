# K-HV-LXC-003 — Caddy Reverse Proxy

> **Container ID:** 112  
> **Resources:** 1 vCPU, 2 GB RAM, 10 GB disk  
> **Node:** pve-kubric-01  
> **IP:** 10.0.50.33/24  
> **Purpose:** TLS termination, reverse proxy, security headers for all Kubric services

---

## 1. LXC Container Creation

```bash
pct create 112 local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst \
  --hostname caddy-kubric \
  --cores 1 \
  --memory 2048 \
  --swap 512 \
  --rootfs kubric-ceph:10 \
  --net0 name=eth0,bridge=vmbr2,ip=10.0.50.33/24,gw=10.0.50.1 \
  --nameserver 10.0.100.1 \
  --searchdomain kubric.local \
  --features nesting=1 \
  --unprivileged 1 \
  --onboot 1 \
  --start 1
```

---

## 2. Install Caddy

```bash
pct enter 112

apt-get update && apt-get upgrade -y
apt-get install -y curl wget debian-keyring debian-archive-keyring apt-transport-https

# Install Caddy from official repo
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | \
  gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg

curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | \
  tee /etc/apt/sources.list.d/caddy-stable.list

apt-get update
apt-get install -y caddy

# Verify
caddy version
# v2.8.x
```

---

## 3. Caddyfile Configuration

File: `/etc/caddy/Caddyfile`

```
# ══════════════════════════════════════════════════
# Kubric UiDR — Caddy Reverse Proxy Configuration
# ══════════════════════════════════════════════════

# Global options
{
    # Email for Let's Encrypt ACME
    email infra@kubric.local

    # For internal domains without public DNS, use internal CA
    # or disable auto-HTTPS for .local domains
    local_certs

    # Logging
    log {
        output file /var/log/caddy/access.log {
            roll_size 100mb
            roll_keep 10
        }
        format json
    }

    # Admin API
    admin off

    # OCSP stapling
    ocsp_stapling on

    # Rate limiting (requires caddy-ratelimit plugin or order directive)
    order rate_limit before basicauth
}

# ─────────────────────────────────────────────────
# Kubric API Gateway
# ─────────────────────────────────────────────────
api.kubric.local {
    # TLS with auto-generated internal cert
    tls internal

    # Security headers
    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
        X-XSS-Protection "1; mode=block"
        Referrer-Policy strict-origin-when-cross-origin
        Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"
        Permissions-Policy "camera=(), microphone=(), geolocation=()"
        -Server
        -X-Powered-By
    }

    # Rate limiting: 100 requests per second per IP
    rate_limit {
        zone api_zone {
            key {remote_host}
            events 100
            window 1s
        }
    }

    # Reverse proxy to Kubric API (ksvc)
    reverse_proxy 10.0.50.21:8080 10.0.50.22:8080 10.0.50.23:8080 {
        lb_policy round_robin
        health_uri /healthz
        health_interval 10s
        health_timeout 3s
        health_status 200

        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    # Access logging
    log {
        output file /var/log/caddy/api.log {
            roll_size 50mb
            roll_keep 5
        }
        format json
    }
}

# ─────────────────────────────────────────────────
# Grafana Dashboard
# ─────────────────────────────────────────────────
grafana.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Frame-Options SAMEORIGIN
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.23:3000 {
        health_uri /api/health
        health_interval 30s
    }
}

# ─────────────────────────────────────────────────
# Gitea
# ─────────────────────────────────────────────────
git.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.31:3000 {
        health_uri /
        health_interval 30s
    }
}

# ─────────────────────────────────────────────────
# n8n Workflow Automation
# ─────────────────────────────────────────────────
n8n.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Frame-Options SAMEORIGIN
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.32:5678 {
        health_uri /healthz
        health_interval 30s
    }
}

# ─────────────────────────────────────────────────
# MinIO Console
# ─────────────────────────────────────────────────
minio.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.23:9001 {
        health_uri /minio/health/live
        health_interval 30s
    }
}

# ─────────────────────────────────────────────────
# MinIO S3 API
# ─────────────────────────────────────────────────
s3.kubric.local {
    tls internal

    reverse_proxy 10.0.50.23:9000 {
        health_uri /minio/health/live
        health_interval 30s
        # Large uploads
        transport http {
            response_header_timeout 300s
        }
    }
}

# ─────────────────────────────────────────────────
# Woodpecker CI
# ─────────────────────────────────────────────────
ci.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Frame-Options SAMEORIGIN
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.34:8000 {
        health_uri /healthz
        health_interval 30s
    }
}

# ─────────────────────────────────────────────────
# ClickHouse HTTP Interface (read-only dashboard queries)
# ─────────────────────────────────────────────────
clickhouse.kubric.local {
    tls internal

    # Restrict to GET only (read-only HTTP queries)
    @not_get not method GET
    respond @not_get "Method not allowed" 405

    header {
        X-Content-Type-Options nosniff
        -Server
    }

    reverse_proxy 10.0.50.21:8123 10.0.50.22:8123 {
        lb_policy round_robin
        health_uri /ping
        health_interval 15s
    }
}

# ─────────────────────────────────────────────────
# Proxmox Web UI (management access)
# ─────────────────────────────────────────────────
pve.kubric.local {
    tls internal

    header {
        Strict-Transport-Security "max-age=63072000"
        -Server
    }

    reverse_proxy https://10.0.100.21:8006 {
        transport http {
            tls_insecure_skip_verify
        }
    }
}
```

---

## 4. Access Logging to ClickHouse

### 4.1 Caddy JSON Log Format

Caddy's JSON log output at `/var/log/caddy/access.log` contains:

```json
{
  "level": "info",
  "ts": 1709136000.123,
  "logger": "http.log.access",
  "msg": "handled request",
  "request": {
    "remote_ip": "10.0.50.1",
    "method": "GET",
    "uri": "/api/v1/events",
    "host": "api.kubric.local",
    "proto": "HTTP/2.0"
  },
  "duration": 0.015234,
  "size": 4521,
  "status": 200,
  "resp_headers": {}
}
```

### 4.2 Vector Log Shipper to ClickHouse

```bash
# Install Vector for log shipping
apt-get install -y vector

# /etc/vector/vector.toml
cat > /etc/vector/vector.toml <<'EOF'
[sources.caddy_logs]
type = "file"
include = ["/var/log/caddy/*.log"]
read_from = "beginning"

[transforms.parse_caddy]
type = "remap"
inputs = ["caddy_logs"]
source = '''
. = parse_json!(.message)
.timestamp = to_timestamp!(.ts)
.remote_ip = .request.remote_ip
.method = .request.method
.uri = .request.uri
.host = .request.host
.status = .status
.duration_ms = .duration * 1000
.response_size = .size
'''

[sinks.clickhouse]
type = "clickhouse"
inputs = ["parse_caddy"]
endpoint = "http://10.0.50.21:8123"
database = "kubric_telemetry"
table = "http_access_log"
compression = "lz4"
auth.strategy = "basic"
auth.user = "kubric"
auth.password = "${CLICKHOUSE_PASS}"
EOF

systemctl enable --now vector
```

---

## 5. Apply & Verify

```bash
# Validate Caddyfile syntax
caddy validate --config /etc/caddy/Caddyfile

# Reload
systemctl reload caddy

# Or format + reload
caddy fmt --overwrite /etc/caddy/Caddyfile
systemctl reload caddy

# Check status
systemctl status caddy
journalctl -u caddy --since "5 minutes ago" --no-pager

# Test each service
curl -sk https://api.kubric.local/healthz --resolve "api.kubric.local:443:10.0.50.33"
curl -sk https://grafana.kubric.local/api/health --resolve "grafana.kubric.local:443:10.0.50.33"
curl -sk https://git.kubric.local/ --resolve "git.kubric.local:443:10.0.50.33"

# Check security headers
curl -skI https://api.kubric.local/ --resolve "api.kubric.local:443:10.0.50.33" | \
  grep -iE "strict-transport|x-frame|x-content-type|referrer-policy"

# Check TLS certificate
openssl s_client -connect 10.0.50.33:443 -servername api.kubric.local </dev/null 2>/dev/null | \
  openssl x509 -noout -subject -dates
```

---

## 6. DNS Configuration

Add to internal DNS or `/etc/hosts` on client machines:

```
10.0.50.33  api.kubric.local
10.0.50.33  grafana.kubric.local
10.0.50.33  git.kubric.local
10.0.50.33  n8n.kubric.local
10.0.50.33  minio.kubric.local
10.0.50.33  s3.kubric.local
10.0.50.33  ci.kubric.local
10.0.50.33  clickhouse.kubric.local
10.0.50.33  pve.kubric.local
```
