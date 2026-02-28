# K-NET-004 — HAProxy Load Balancer Configuration

> **Deployment:** Proxmox LXC or dedicated VM on management network  
> **Primary IP:** Bound to Keepalived VIPs (10.0.100.100, 10.0.100.101)  
> **Stats:** http://10.0.100.100:9999/stats  
> **SSL Termination:** Let's Encrypt via Certbot/Caddy (see K-HV-LXC-003)

---

## 1. Installation

```bash
# On the HAProxy host (all 3 nodes for active/standby with Keepalived)
apt-get update && apt-get install -y haproxy hatop

# Enable HAProxy
systemctl enable haproxy

# Verify version
haproxy -v
# Expected: HA-Proxy version 2.6+ (Bookworm default)
```

---

## 2. HAProxy Configuration

File: `/etc/haproxy/haproxy.cfg`

```ini
# ══════════════════════════════════════════════════
# Kubric UiDR — HAProxy Load Balancer Configuration
# ══════════════════════════════════════════════════

global
    log /dev/log    local0
    log /dev/log    local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners
    stats timeout 30s
    user    haproxy
    group   haproxy
    daemon

    # SSL tuning
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets
    tune.ssl.default-dh-param 2048

    # Performance
    maxconn 50000
    tune.bufsize 32768

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    option  http-server-close
    option  forwardfor except 127.0.0.0/8
    retries 3
    timeout connect     5s
    timeout client      30s
    timeout server      30s
    timeout check       5s
    timeout http-request 10s
    timeout http-keep-alive 5s
    timeout queue       30s

    # Error pages
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

# ──────────────────────────────────────────────────
# STATS PAGE
# ──────────────────────────────────────────────────

listen stats
    bind *:9999
    mode http
    stats enable
    stats uri /stats
    stats refresh 10s
    stats show-legends
    stats show-node
    stats auth kubric-admin:${HAPROXY_STATS_PASS}
    stats admin if TRUE

# ──────────────────────────────────────────────────
# FRONTEND: K8s API Server (TCP passthrough)
# ──────────────────────────────────────────────────

frontend ft_k8s_api
    bind 10.0.100.100:6443
    mode tcp
    option tcplog
    default_backend bk_k8s_api

    # Rate limiting: 100 connections per source IP per 10s
    stick-table type ip size 100k expire 10s store conn_cur,conn_rate(10s)
    tcp-request connection track-sc0 src
    tcp-request connection reject if { sc0_conn_rate gt 100 }

backend bk_k8s_api
    mode tcp
    balance roundrobin
    option tcp-check

    # K8s API servers on all control plane nodes
    server k8s-api-01 10.0.100.21:6443 check inter 3s fall 3 rise 2
    server k8s-api-02 10.0.100.22:6443 check inter 3s fall 3 rise 2 backup
    server k8s-api-03 10.0.100.23:6443 check inter 3s fall 3 rise 2 backup

# ──────────────────────────────────────────────────
# FRONTEND: Kubric API Gateway (HTTP/HTTPS)
# ──────────────────────────────────────────────────

frontend ft_kubric_api
    bind 10.0.100.101:80
    bind 10.0.100.101:443 ssl crt /etc/haproxy/certs/kubric.pem alpn h2,http/1.1

    # Redirect HTTP → HTTPS
    http-request redirect scheme https unless { ssl_fc }

    # Security headers
    http-response set-header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
    http-response set-header X-Frame-Options DENY
    http-response set-header X-Content-Type-Options nosniff
    http-response set-header X-XSS-Protection "1; mode=block"
    http-response set-header Referrer-Policy strict-origin-when-cross-origin

    # Rate limiting: 200 req/s per source
    stick-table type ip size 200k expire 30s store http_req_rate(10s)
    http-request track-sc0 src
    http-request deny deny_status 429 if { sc_http_req_rate(0) gt 200 }

    # ACL routing based on Host header or path
    acl is_grafana hdr(host) -i grafana.kubric.local
    acl is_minio   hdr(host) -i minio.kubric.local
    acl is_gitea   hdr(host) -i git.kubric.local
    acl is_n8n     hdr(host) -i n8n.kubric.local
    acl is_api     hdr(host) -i api.kubric.local

    use_backend bk_grafana  if is_grafana
    use_backend bk_minio    if is_minio
    use_backend bk_gitea    if is_gitea
    use_backend bk_n8n      if is_n8n
    default_backend bk_kubric_api

# ──────────────────────────────────────────────────
# BACKEND: Kubric API (ksvc — Go service)
# ──────────────────────────────────────────────────

backend bk_kubric_api
    mode http
    balance roundrobin
    option httpchk GET /healthz
    http-check expect status 200

    # Connection limits per server
    default-server maxconn 1000 inter 5s fall 3 rise 2

    server ksvc-01 10.0.50.21:8080 check
    server ksvc-02 10.0.50.22:8080 check
    server ksvc-03 10.0.50.23:8080 check backup

# ──────────────────────────────────────────────────
# BACKEND: NATS Message Bus (TCP)
# ──────────────────────────────────────────────────

frontend ft_nats
    bind 10.0.100.101:4222
    mode tcp
    option tcplog
    default_backend bk_nats

backend bk_nats
    mode tcp
    balance roundrobin
    option tcp-check

    server nats-01 10.0.50.21:4222 check inter 5s fall 3 rise 2
    server nats-02 10.0.50.22:4222 check inter 5s fall 3 rise 2
    server nats-03 10.0.50.23:4222 check inter 5s fall 3 rise 2

# ──────────────────────────────────────────────────
# BACKEND: ClickHouse HTTP (8123) and Native (9000)
# ──────────────────────────────────────────────────

frontend ft_clickhouse_http
    bind 10.0.100.101:8123
    mode http
    option httplog
    default_backend bk_clickhouse_http

backend bk_clickhouse_http
    mode http
    balance roundrobin
    option httpchk GET /ping
    http-check expect string Ok.

    server ch-shard1 10.0.50.21:8123 check inter 5s fall 3 rise 2
    server ch-shard2 10.0.50.22:8123 check inter 5s fall 3 rise 2

frontend ft_clickhouse_native
    bind 10.0.100.101:9440
    mode tcp
    option tcplog
    default_backend bk_clickhouse_native

backend bk_clickhouse_native
    mode tcp
    balance roundrobin
    option tcp-check

    # Native protocol on port 9000 (remapped to 9440 externally to avoid MinIO conflict)
    server ch-native-1 10.0.50.21:9000 check inter 5s
    server ch-native-2 10.0.50.22:9000 check inter 5s

# ──────────────────────────────────────────────────
# BACKEND: PostgreSQL (TCP)
# ──────────────────────────────────────────────────

frontend ft_postgres
    bind 10.0.100.101:5432
    mode tcp
    option tcplog
    default_backend bk_postgres

backend bk_postgres
    mode tcp
    balance roundrobin
    option pgsql-check user haproxy

    # Primary only — no read replicas yet
    server pg-primary 10.0.50.22:5432 check inter 5s fall 3 rise 2

# ──────────────────────────────────────────────────
# BACKEND: Grafana (HTTP)
# ──────────────────────────────────────────────────

backend bk_grafana
    mode http
    balance roundrobin
    option httpchk GET /api/health
    http-check expect status 200

    server grafana-01 10.0.50.23:3000 check inter 10s

# ──────────────────────────────────────────────────
# BACKEND: MinIO S3 API (HTTP)
# ──────────────────────────────────────────────────

frontend ft_minio_s3
    bind 10.0.100.101:9000
    mode http
    option httplog
    default_backend bk_minio_s3

backend bk_minio_s3
    mode http
    balance roundrobin
    option httpchk GET /minio/health/live
    http-check expect status 200
    timeout server 300s

    server minio-01 10.0.50.23:9000 check inter 10s

backend bk_minio
    mode http
    balance roundrobin
    option httpchk GET /minio/health/live
    http-check expect status 200

    server minio-console 10.0.50.23:9001 check inter 10s

# ──────────────────────────────────────────────────
# BACKEND: Gitea (HTTP)
# ──────────────────────────────────────────────────

backend bk_gitea
    mode http
    balance roundrobin
    option httpchk GET /
    http-check expect status 200

    server gitea-01 10.0.50.21:3000 check inter 10s

# ──────────────────────────────────────────────────
# BACKEND: n8n (HTTP)
# ──────────────────────────────────────────────────

backend bk_n8n
    mode http
    balance roundrobin
    option httpchk GET /healthz
    http-check expect status 200
    timeout server 120s

    server n8n-01 10.0.50.22:5678 check inter 10s
```

**Environment Variable:**
```
HAPROXY_STATS_PASS=<generate with: openssl rand -base64 16>
```

---

## 3. SSL Certificate Setup

### 3.1 Let's Encrypt via Certbot

```bash
apt-get install -y certbot

# Obtain certificate (standalone or DNS challenge)
certbot certonly --standalone -d api.kubric.local -d grafana.kubric.local \
  -d minio.kubric.local -d git.kubric.local -d n8n.kubric.local \
  --email infra@kubric.local --agree-tos --non-interactive

# Combine cert + key for HAProxy
cat /etc/letsencrypt/live/api.kubric.local/fullchain.pem \
    /etc/letsencrypt/live/api.kubric.local/privkey.pem \
    > /etc/haproxy/certs/kubric.pem

chmod 600 /etc/haproxy/certs/kubric.pem
```

### 3.2 Auto-Renewal Cron

```bash
cat > /etc/cron.d/certbot-haproxy <<'EOF'
0 3 * * 1 root certbot renew --post-hook "cat /etc/letsencrypt/live/api.kubric.local/fullchain.pem /etc/letsencrypt/live/api.kubric.local/privkey.pem > /etc/haproxy/certs/kubric.pem && systemctl reload haproxy"
EOF
```

### 3.3 Self-Signed Certificate (Alternative for Internal)

```bash
mkdir -p /etc/haproxy/certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/haproxy/certs/kubric.key \
  -out /etc/haproxy/certs/kubric.crt \
  -subj "/CN=*.kubric.local/O=Kubric/C=US"

cat /etc/haproxy/certs/kubric.crt /etc/haproxy/certs/kubric.key \
  > /etc/haproxy/certs/kubric.pem
chmod 600 /etc/haproxy/certs/kubric.pem
```

---

## 4. PostgreSQL HAProxy Health Check User

```sql
-- Create read-only user for HAProxy checks
CREATE USER haproxy WITH LOGIN;
GRANT CONNECT ON DATABASE kubric_core TO haproxy;
-- No password needed — pgsql-check uses protocol-level check
```

---

## 5. Apply & Validate

### 5.1 Syntax Check

```bash
haproxy -c -f /etc/haproxy/haproxy.cfg
# Expected: Configuration file is valid

# Reload without downtime
systemctl reload haproxy
```

### 5.2 Service Verification

```bash
# Stats page
curl -u kubric-admin:${HAPROXY_STATS_PASS} http://10.0.100.100:9999/stats

# K8s API through load balancer
curl -sk https://10.0.100.100:6443/healthz
# Expected: ok

# Kubric API
curl -s http://10.0.100.101:8080/healthz
# Expected: {"status":"ok"}

# ClickHouse
curl -s "http://10.0.100.101:8123/ping"
# Expected: Ok.

# PostgreSQL
psql -h 10.0.100.101 -U kubric -d kubric_core -c "SELECT 1"
# Expected: 1

# NATS
nats server check -s nats://10.0.100.101:4222

# Grafana
curl -s http://grafana.kubric.local/api/health | jq .
```

### 5.3 Backend Health Check

```bash
# Check backend status via HAProxy socket
echo "show stat" | socat /run/haproxy/admin.sock stdio | \
  awk -F',' '{printf "%-20s %-15s %-10s\n", $1, $2, $18}' | head -30

# Expected: All backends showing "UP"
```

---

## 6. Connection & Rate Limits

### 6.1 Global Limits

```ini
# In haproxy.cfg global section:
maxconn 50000                      # Total max concurrent connections

# Per-frontend:
# ft_kubric_api: 200 req/10s per IP (HTTP layer)
# ft_k8s_api: 100 conn/10s per IP (TCP layer)
```

### 6.2 Backend Connection Limits

```ini
# In each backend, default-server sets:
default-server maxconn 1000        # Max connections per backend server

# Queue overflow behavior:
# timeout queue 30s               # Wait up to 30s in queue before 503
```

### 6.3 DDoS Mitigation

```ini
# Add to frontend ft_kubric_api:
# Tarpit aggressive clients
http-request tarpit if { sc_http_req_rate(0) gt 500 }
timeout tarpit 30s

# Block known bad user agents
acl bad_bot hdr_sub(User-Agent) -i scanner nikto sqlmap
http-request deny if bad_bot
```

---

## 7. Logging

### 7.1 rsyslog Configuration

```bash
cat > /etc/rsyslog.d/49-haproxy.conf <<'EOF'
local0.* /var/log/haproxy/haproxy.log
local1.notice /var/log/haproxy/haproxy-admin.log
EOF

mkdir -p /var/log/haproxy
systemctl restart rsyslog
systemctl reload haproxy
```

### 7.2 Log Format for ClickHouse Ingestion

```ini
# Custom log format in defaults section:
log-format '{"timestamp":"%T","client":"%ci","client_port":%cp,"frontend":"%f","backend":"%b","server":"%s","status":%ST,"bytes_read":%B,"req_time":%TR,"resp_time":%Tr,"active_conn":%ac,"method":"%HM","uri":"%HP","http_version":"%HV"}'
```

---

## 8. Monitoring

```bash
# HAProxy Prometheus exporter (built-in since 2.4)
# Add to global section:
#   stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners

# Enable Prometheus endpoint:
# Add a new frontend:
cat >> /etc/haproxy/haproxy.cfg <<'EOF'

frontend prometheus
    bind *:8405
    mode http
    http-request use-service prometheus-exporter if { path /metrics }
    no log
EOF

# Scrape from Prometheus:
# - job_name: haproxy
#   static_configs:
#     - targets: ['10.0.100.21:8405','10.0.100.22:8405','10.0.100.23:8405']
```
