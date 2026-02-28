# K-HV-LXC-004 — Woodpecker CI

> **Container ID:** 113  
> **Resources:** 4 vCPU, 8 GB RAM, 100 GB disk  
> **Node:** pve-kubric-01  
> **IP:** 10.0.50.34/24  
> **Access:** https://ci.kubric.local  
> **SCM:** Gitea at git.kubric.local

---

## 1. LXC Container Creation

```bash
pct create 113 local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst \
  --hostname woodpecker-kubric \
  --cores 4 \
  --memory 8192 \
  --swap 2048 \
  --rootfs kubric-ceph:100 \
  --net0 name=eth0,bridge=vmbr2,ip=10.0.50.34/24,gw=10.0.50.1 \
  --nameserver 10.0.100.1 \
  --searchdomain kubric.local \
  --features nesting=1,keyctl=1 \
  --unprivileged 1 \
  --onboot 1 \
  --start 1
```

---

## 2. Container Setup

```bash
pct enter 113

apt-get update && apt-get upgrade -y
apt-get install -y curl wget ca-certificates gnupg git

# Install Docker + Compose
curl -fsSL https://get.docker.com | sh
apt-get install -y docker-compose-plugin
```

---

## 3. Gitea OAuth2 Application

Before deploying Woodpecker, create an OAuth2 app in Gitea:

```bash
GITEA_URL="http://10.0.50.31:3000"
GITEA_TOKEN="<Gitea admin API token>"

# Create OAuth2 application for Woodpecker
curl -s -X POST "${GITEA_URL}/api/v1/user/applications/oauth2" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Woodpecker CI",
    "redirect_uris": ["https://ci.kubric.local/authorize"]
  }'

# Response includes client_id and client_secret — save these!
# {
#   "id": 1,
#   "name": "Woodpecker CI",
#   "client_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
#   "client_secret": "gto_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
#   ...
# }
```

---

## 4. Woodpecker Deployment

### 4.1 Directory Structure

```bash
mkdir -p /opt/woodpecker/{server-data,agent-data}
```

### 4.2 Docker Compose

File: `/opt/woodpecker/docker-compose.yml`

```yaml
version: "3.8"

services:
  woodpecker-server:
    image: woodpeckerci/woodpecker-server:v2.7
    container_name: woodpecker-server
    restart: unless-stopped
    ports:
      - "8000:8000"    # HTTP UI
      - "9000:9000"    # gRPC (agent communication)
    environment:
      # Gitea OAuth2
      - WOODPECKER_GITEA=true
      - WOODPECKER_GITEA_URL=http://10.0.50.31:3000
      - WOODPECKER_GITEA_CLIENT=${GITEA_OAUTH2_CLIENT_ID}
      - WOODPECKER_GITEA_SECRET=${GITEA_OAUTH2_CLIENT_SECRET}
      - WOODPECKER_GITEA_SKIP_VERIFY=true

      # Server config
      - WOODPECKER_HOST=https://ci.kubric.local
      - WOODPECKER_OPEN=false
      - WOODPECKER_ADMIN=kubric-admin

      # Agent authentication
      - WOODPECKER_AGENT_SECRET=${WOODPECKER_AGENT_SECRET}

      # Database (SQLite for simplicity, or Postgres)
      - WOODPECKER_DATABASE_DRIVER=sqlite3
      - WOODPECKER_DATABASE_DATASOURCE=/var/lib/woodpecker/woodpecker.sqlite

      # Logging
      - WOODPECKER_LOG_LEVEL=info

      # Limits
      - WOODPECKER_MAX_PROCS=4
    volumes:
      - /opt/woodpecker/server-data:/var/lib/woodpecker
    networks:
      - woodpecker-net

  woodpecker-agent:
    image: woodpeckerci/woodpecker-agent:v2.7
    container_name: woodpecker-agent
    restart: unless-stopped
    depends_on:
      - woodpecker-server
    environment:
      - WOODPECKER_SERVER=woodpecker-server:9000
      - WOODPECKER_AGENT_SECRET=${WOODPECKER_AGENT_SECRET}
      - WOODPECKER_MAX_WORKFLOWS=3
      - WOODPECKER_BACKEND_DOCKER_VOLUMES=/opt/woodpecker/agent-cache:/cache
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/woodpecker/agent-data:/var/lib/woodpecker
      - /opt/woodpecker/agent-cache:/cache
    networks:
      - woodpecker-net

networks:
  woodpecker-net:
    driver: bridge
```

### 4.3 Environment File

File: `/opt/woodpecker/.env`

```bash
GITEA_OAUTH2_CLIENT_ID=<from Gitea OAuth2 app>
GITEA_OAUTH2_CLIENT_SECRET=<from Gitea OAuth2 app>
WOODPECKER_AGENT_SECRET=<generate with: openssl rand -hex 32>
```

### 4.4 Start Woodpecker

```bash
cd /opt/woodpecker
docker compose up -d

docker logs -f woodpecker-server
# Wait for "Starting Woodpecker server"
```

---

## 5. Pipeline Definitions

### 5.1 Main Pipeline — `.woodpecker.yml`

Place in repository root:

```yaml
# .woodpecker.yml — Kubric UiDR CI/CD Pipeline

when:
  event: [push, pull_request, tag]

steps:
  # ─── Go Backend ───
  go-lint:
    image: golangci/golangci-lint:v1.61
    commands:
      - golangci-lint run --timeout 5m ./...
    when:
      path:
        include: ["cmd/**", "internal/**", "services/**", "*.go"]

  go-test:
    image: golang:1.23-bookworm
    environment:
      - CGO_ENABLED=1
      - GOFLAGS=-count=1
    commands:
      - go mod download
      - go test -v -race -coverprofile=coverage.out ./...
      - go tool cover -func=coverage.out | tail -1
    when:
      path:
        include: ["cmd/**", "internal/**", "services/**", "*.go"]

  go-build:
    image: golang:1.23-bookworm
    commands:
      - go build -ldflags="-s -w -X main.version=${CI_COMMIT_TAG:-dev}" -o /cache/kubric-api ./cmd/api
      - go build -ldflags="-s -w" -o /cache/kubric-ksvc ./cmd/ksvc
      - go build -ldflags="-s -w" -o /cache/kubric-vdr ./cmd/vdr
      - go build -ldflags="-s -w" -o /cache/kubric-kic ./cmd/kic
      - go build -ldflags="-s -w" -o /cache/kubric-noc ./cmd/noc

  # ─── Rust Agent ───
  rust-build:
    image: rust:1.82-bookworm
    commands:
      - cd agents
      - cargo build --release
      - cp target/release/kubric-agent /cache/kubric-agent
    when:
      path:
        include: ["agents/**", "Cargo.toml", "Cargo.lock"]

  rust-test:
    image: rust:1.82-bookworm
    commands:
      - cd agents
      - cargo test --release
    when:
      path:
        include: ["agents/**"]

  # ─── Python KAI ───
  python-lint:
    image: python:3.12-bookworm
    commands:
      - pip install ruff mypy
      - ruff check kai/
      - mypy kai/ --ignore-missing-imports
    when:
      path:
        include: ["kai/**", "requirements.txt"]

  python-test:
    image: python:3.12-bookworm
    commands:
      - pip install -r requirements.txt
      - pip install pytest pytest-asyncio
      - pytest kai/tests/ -v
    when:
      path:
        include: ["kai/**"]

  # ─── Docker Images ───
  build-api-image:
    image: plugins/docker
    settings:
      repo: 10.0.50.23:5000/kubric/api
      registry: 10.0.50.23:5000
      insecure: true
      dockerfile: Dockerfile.api
      tags:
        - ${CI_COMMIT_TAG:-${CI_COMMIT_SHA:0:8}}
        - latest
      cache_from: 10.0.50.23:5000/kubric/api:latest
    when:
      event: [push, tag]
      branch: main

  build-agent-image:
    image: plugins/docker
    settings:
      repo: 10.0.50.23:5000/kubric/agent
      registry: 10.0.50.23:5000
      insecure: true
      dockerfile: Dockerfile.agents
      tags:
        - ${CI_COMMIT_TAG:-${CI_COMMIT_SHA:0:8}}
        - latest
    when:
      event: [push, tag]
      branch: main

  build-web-image:
    image: plugins/docker
    settings:
      repo: 10.0.50.23:5000/kubric/web
      registry: 10.0.50.23:5000
      insecure: true
      dockerfile: Dockerfile.web
      tags:
        - ${CI_COMMIT_TAG:-${CI_COMMIT_SHA:0:8}}
        - latest
    when:
      event: [push, tag]
      branch: main

  # ─── Deploy (tag only) ───
  deploy:
    image: bitnami/kubectl:1.30
    commands:
      - kubectl --kubeconfig /cache/kubeconfig set image deployment/kubric-api kubric-api=10.0.50.23:5000/kubric/api:${CI_COMMIT_TAG}
      - kubectl --kubeconfig /cache/kubeconfig rollout status deployment/kubric-api --timeout=120s
    when:
      event: tag

  # ─── Upload Artifacts to MinIO ───
  upload-artifacts:
    image: minio/mc:latest
    commands:
      - mc alias set kubric http://10.0.50.23:9000 kubric-admin ${MINIO_ROOT_PASS}
      - mc cp /cache/kubric-api kubric/kubric-artifacts/${CI_COMMIT_SHA}/kubric-api
      - mc cp /cache/kubric-agent kubric/kubric-artifacts/${CI_COMMIT_SHA}/kubric-agent
    when:
      event: [push, tag]
      branch: main
    secrets: [minio_root_pass]
```

### 5.2 Database Migration Pipeline

File: `.woodpecker/migrate.yml`

```yaml
when:
  event: push
  path:
    include: ["db/migrations/**"]

steps:
  migrate-check:
    image: migrate/migrate:v4.17.0
    commands:
      - migrate -database "postgres://kubric:${PG_KUBRIC_PASS}@10.0.50.22:5432/kubric_core?sslmode=disable" -path db/migrations version
    secrets: [pg_kubric_pass]

  migrate-up:
    image: migrate/migrate:v4.17.0
    commands:
      - migrate -database "postgres://kubric:${PG_KUBRIC_PASS}@10.0.50.22:5432/kubric_core?sslmode=disable" -path db/migrations up
    secrets: [pg_kubric_pass]
    when:
      branch: main
```

---

## 6. Woodpecker Secrets

```bash
# Add secrets via Woodpecker CLI or API
# Install CLI
wget https://github.com/woodpecker-ci/woodpecker/releases/download/v2.7.0/woodpecker-cli_linux_amd64.tar.gz
tar xzf woodpecker-cli_linux_amd64.tar.gz
mv woodpecker-cli /usr/local/bin/

# Configure CLI
export WOODPECKER_SERVER=https://ci.kubric.local
export WOODPECKER_TOKEN="<from Settings → API → Personal Token>"

# Add org secrets
woodpecker-cli secret add --organization kubric --name pg_kubric_pass --value "${PG_KUBRIC_PASS}"
woodpecker-cli secret add --organization kubric --name minio_root_pass --value "${MINIO_ROOT_PASS}"
woodpecker-cli secret add --organization kubric --name clickhouse_pass --value "${CLICKHOUSE_PASS}"
```

---

## 7. Docker-in-Docker (DinD)

The agent already mounts `/var/run/docker.sock`. For isolated builds:

```bash
# Alternative: Use DinD sidecar
# In docker-compose.yml agent section, add:
#   privileged: true  # Required for DinD
# This is already handled by the plugins/docker step
```

---

## 8. Artifact Storage on MinIO

```bash
# Verify artifact bucket exists
mc ls kubric/kubric-artifacts/

# List build artifacts
mc ls kubric/kubric-artifacts/${CI_COMMIT_SHA}/
```

---

## 9. Verification

```bash
# Container
pct status 113    # running

# Woodpecker UI
curl -s http://10.0.50.34:8000/healthz
# Expected: OK

# Agent connected
docker logs woodpecker-agent | grep "connected to server"

# Gitea integration
# Navigate to https://ci.kubric.local → should redirect to Gitea OAuth

# Trigger test build
cd /path/to/kubric-uidr
echo "# test" >> README.md
git add -A && git commit -m "ci: test pipeline"
git push gitea main

# Check build status
woodpecker-cli build ls kubric/kubric-uidr
```
