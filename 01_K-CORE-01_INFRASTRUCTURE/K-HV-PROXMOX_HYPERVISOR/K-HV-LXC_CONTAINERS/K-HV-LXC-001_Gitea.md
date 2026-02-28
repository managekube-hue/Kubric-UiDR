# K-HV-LXC-001 — Gitea Self-Hosted Git Server

> **Container ID:** 110  
> **Resources:** 2 vCPU, 4 GB RAM, 50 GB disk  
> **Node:** pve-kubric-01  
> **IP:** 10.0.50.31/24  
> **Access:** https://git.kubric.local (via Caddy/HAProxy)

---

## 1. LXC Container Creation

```bash
# On pve-kubric-01

# Download Debian 12 template if not present
pveam update
pveam download local debian-12-standard_12.2-1_amd64.tar.zst

# Create LXC container
pct create 110 local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst \
  --hostname gitea-kubric \
  --cores 2 \
  --memory 4096 \
  --swap 1024 \
  --rootfs kubric-ceph:50 \
  --net0 name=eth0,bridge=vmbr2,ip=10.0.50.31/24,gw=10.0.50.1 \
  --nameserver 10.0.100.1 \
  --searchdomain kubric.local \
  --features nesting=1,keyctl=1 \
  --unprivileged 1 \
  --onboot 1 \
  --start 1

# Enable Docker-in-LXC (nesting feature)
pct set 110 --features nesting=1,keyctl=1
```

---

## 2. Container Setup

```bash
# Enter container
pct enter 110

# Update system
apt-get update && apt-get upgrade -y
apt-get install -y curl wget git ca-certificates gnupg lsb-release sudo

# Install Docker
curl -fsSL https://get.docker.com | sh

# Install Docker Compose
apt-get install -y docker-compose-plugin

# Create Gitea user
useradd -m -s /bin/bash -G docker gitea
```

---

## 3. Gitea Deployment via Docker Compose

### 3.1 Directory Structure

```bash
mkdir -p /opt/gitea/{data,config,postgres}
chown -R 1000:1000 /opt/gitea/data /opt/gitea/config
```

### 3.2 Docker Compose File

File: `/opt/gitea/docker-compose.yml`

```yaml
version: "3.8"

services:
  gitea-db:
    image: postgres:16-bookworm
    container_name: gitea-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: gitea
      POSTGRES_PASSWORD: "${GITEA_DB_PASS}"
      POSTGRES_DB: gitea
    volumes:
      - /opt/gitea/postgres:/var/lib/postgresql/data
    networks:
      - gitea-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gitea"]
      interval: 10s
      timeout: 5s
      retries: 5

  gitea:
    image: gitea/gitea:1.22-rootless
    container_name: gitea
    restart: unless-stopped
    depends_on:
      gitea-db:
        condition: service_healthy
    environment:
      - USER_UID=1000
      - USER_GID=1000
      - GITEA__database__DB_TYPE=postgres
      - GITEA__database__HOST=gitea-db:5432
      - GITEA__database__NAME=gitea
      - GITEA__database__USER=gitea
      - GITEA__database__PASSWD=${GITEA_DB_PASS}
    ports:
      - "3000:3000"     # HTTP
      - "2222:2222"     # SSH
    volumes:
      - /opt/gitea/data:/var/lib/gitea
      - /opt/gitea/config:/etc/gitea
    networks:
      - gitea-net

networks:
  gitea-net:
    driver: bridge
```

### 3.3 Environment File

File: `/opt/gitea/.env`

```bash
GITEA_DB_PASS=<generate with: openssl rand -base64 24>
```

### 3.4 Start Gitea

```bash
cd /opt/gitea
docker compose up -d

# Check logs
docker logs -f gitea
# Wait for "Starting new Web server" message
```

---

## 4. Gitea app.ini Configuration

After first startup, customize `/opt/gitea/config/app.ini`:

```ini
APP_NAME = Kubric UiDR — Git Repository
RUN_USER = git
RUN_MODE = prod
WORK_PATH = /var/lib/gitea

[server]
DOMAIN           = git.kubric.local
ROOT_URL         = https://git.kubric.local/
HTTP_PORT        = 3000
SSH_DOMAIN       = git.kubric.local
SSH_PORT         = 2222
SSH_LISTEN_PORT  = 2222
LFS_START_SERVER = true
OFFLINE_MODE     = false

[database]
DB_TYPE  = postgres
HOST     = gitea-db:5432
NAME     = gitea
USER     = gitea
PASSWD   = ${GITEA_DB_PASS}
SSL_MODE = disable

[repository]
ROOT                    = /var/lib/gitea/repositories
DEFAULT_BRANCH          = main
DEFAULT_REPO_UNITS      = repo.code,repo.issues,repo.pulls,repo.releases
ENABLE_PUSH_CREATE_USER = true
ENABLE_PUSH_CREATE_ORG  = true

[security]
INSTALL_LOCK       = true
SECRET_KEY         = <generate with: gitea generate secret SECRET_KEY>
INTERNAL_TOKEN     = <generate with: gitea generate secret INTERNAL_TOKEN>

[service]
DISABLE_REGISTRATION       = true
REQUIRE_SIGNIN_VIEW        = true
DEFAULT_KEEP_EMAIL_PRIVATE = true
ENABLE_NOTIFY_MAIL         = true

[mailer]
ENABLED   = true
SMTP_ADDR = 10.0.100.21
SMTP_PORT = 25
FROM      = gitea@kubric.local

[webhook]
ALLOWED_HOST_LIST = 10.0.50.0/24,10.0.100.0/24
SKIP_TLS_VERIFY   = true

[log]
MODE      = console
LEVEL     = info

[metrics]
ENABLED   = true
TOKEN     = ${GITEA_METRICS_TOKEN}

[actions]
ENABLED   = false
```

```bash
# Restart after config change
docker restart gitea
```

---

## 5. SSH Key Configuration

```bash
# Generate deploy key for Kubric CI
ssh-keygen -t ed25519 -C "kubric-ci@kubric.local" -f /root/.ssh/kubric-ci -N ""

# Add public key to Gitea via API (after creating admin user via web UI)
GITEA_URL="http://10.0.50.31:3000"
GITEA_TOKEN="<admin API token from Settings → Applications>"

# Create organization
curl -s -X POST "${GITEA_URL}/api/v1/orgs" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"username":"kubric","full_name":"Kubric UiDR","visibility":"private"}'

# Create main repository
curl -s -X POST "${GITEA_URL}/api/v1/orgs/kubric/repos" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name":"kubric-uidr","description":"Unified Detection & Response Platform","private":true,"default_branch":"main","auto_init":true}'

# Add deploy key
curl -s -X POST "${GITEA_URL}/api/v1/repos/kubric/kubric-uidr/keys" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"kubric-ci\",\"key\":\"$(cat /root/.ssh/kubric-ci.pub)\",\"read_only\":false}"
```

---

## 6. Webhook Setup for Woodpecker CI

```bash
# Add Woodpecker webhook to trigger builds on push
curl -s -X POST "${GITEA_URL}/api/v1/repos/kubric/kubric-uidr/hooks" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "gitea",
    "active": true,
    "events": ["push", "pull_request", "tag"],
    "config": {
      "url": "http://10.0.50.34:8000/api/hook",
      "content_type": "json",
      "secret": "${WOODPECKER_WEBHOOK_SECRET}"
    }
  }'
```

---

## 7. Push Existing Kubric-UiDR Repository

```bash
# From development machine
cd /path/to/Kubric-UiDR
git remote add gitea ssh://git@git.kubric.local:2222/kubric/kubric-uidr.git
git push gitea main --tags
```

---

## 8. Backup Procedure

### 8.1 Automated Backup Script

File: `/opt/gitea/backup.sh`

```bash
#!/bin/bash
BACKUP_DIR="/opt/gitea/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
mkdir -p "${BACKUP_DIR}"

# Dump Gitea (includes repos, DB, config)
docker exec gitea /usr/local/bin/gitea dump \
  -c /etc/gitea/app.ini \
  -w /tmp \
  -f "/tmp/gitea-dump-${TIMESTAMP}.zip"

docker cp "gitea:/tmp/gitea-dump-${TIMESTAMP}.zip" "${BACKUP_DIR}/"

# Also backup PostgreSQL separately
docker exec gitea-postgres pg_dump -U gitea gitea | \
  gzip > "${BACKUP_DIR}/gitea-db-${TIMESTAMP}.sql.gz"

# Upload to MinIO
mc cp "${BACKUP_DIR}/gitea-dump-${TIMESTAMP}.zip" kubric/kubric-backups/gitea/
mc cp "${BACKUP_DIR}/gitea-db-${TIMESTAMP}.sql.gz" kubric/kubric-backups/gitea/

# Cleanup local backups older than 7 days
find "${BACKUP_DIR}" -name "gitea-*" -mtime +7 -delete

echo "$(date): Gitea backup completed → gitea-dump-${TIMESTAMP}.zip"
```

```bash
chmod +x /opt/gitea/backup.sh

# Cron: daily at 2 AM
echo "0 2 * * * root /opt/gitea/backup.sh >> /var/log/gitea-backup.log 2>&1" \
  > /etc/cron.d/gitea-backup
```

---

## 9. Verification

```bash
# Container running
pct status 110    # running

# Gitea web UI
curl -s http://10.0.50.31:3000/api/v1/version | jq .
# {"version":"1.22.x"}

# SSH access
ssh -p 2222 git@10.0.50.31 -T
# "Hi there! You've successfully authenticated..."

# PostgreSQL
docker exec gitea-postgres pg_isready
# /var/run/postgresql:5432 - accepting connections

# Repository listing
curl -s -H "Authorization: token ${GITEA_TOKEN}" \
  "${GITEA_URL}/api/v1/repos/search?q=kubric" | jq '.[].full_name'

# Webhook delivery status
curl -s -H "Authorization: token ${GITEA_TOKEN}" \
  "${GITEA_URL}/api/v1/repos/kubric/kubric-uidr/hooks" | jq '.[].last_status'
```
