# Kubric Quick Start Guide

**Get Kubric running in 5 minutes**

---

## Prerequisites

### Required
- **Docker Desktop** 24+ with Compose V2
- **RAM:** 16 GB minimum
- **CPU:** 4 cores minimum
- **Disk:** 100 GB free space

### Supported Platforms
- ✅ Linux (Ubuntu 22.04+, Debian 12+, RHEL 9+)
- ✅ macOS (Intel & Apple Silicon)
- ✅ Windows 11 with WSL2

---

## Installation

### 1. Clone Repository
```bash
git clone https://github.com/managekube-hue/Kubric-UiDR.git
cd Kubric-UiDR
```

### 2. Configure Environment
```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your settings
nano .env
```

**Minimum required variables:**
```bash
KUBRIC_DB_PASSWORD=your_secure_password
KUBRIC_DEFAULT_TENANT=your-company
```

### 3. Start Infrastructure
```bash
# Start core infrastructure only
docker compose up -d nats postgres clickhouse redis vault

# Wait 30 seconds for services to initialize
sleep 30

# Verify health
docker compose ps
```

### 4. Start Observability
```bash
# Start monitoring stack
docker compose up -d prometheus loki grafana

# Access Grafana
open http://localhost:3000
# Login: admin / kubric-grafana
```

### 5. Start ITSM
```bash
# Start ERPNext
docker compose up -d erpnext-db erpnext

# Wait 60 seconds for initialization
sleep 60

# Access ERPNext
open http://localhost:8000
# Login: Administrator / admin
```

### 6. Start Full Stack (Optional)
```bash
# Start all application services
docker compose --profile app up -d

# This includes:
# - K-SVC (tenant management)
# - VDR (vulnerability detection)
# - KIC (compliance)
# - NOC (infrastructure)
# - KAI (AI orchestration)
```

---

## Verify Installation

### Check Service Health
```bash
# All services should show "healthy"
docker compose ps

# Check logs
docker compose logs -f --tail=50
```

### Test Endpoints
```bash
# NATS
curl http://localhost:8222/healthz

# ClickHouse
curl http://localhost:8123/ping

# Grafana
curl http://localhost:3000/api/health

# ERPNext
curl http://localhost:8000
```

---

## Access Points

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / kubric-grafana |
| ERPNext | http://localhost:8000 | Administrator / admin |
| n8n | http://localhost:5678 | (no auth in dev) |
| Temporal UI | http://localhost:8233 | (no auth) |
| Prometheus | http://localhost:9090 | (no auth) |

---

## Next Steps

### Deploy Your First Agent
```bash
# Generate install script
curl -X POST http://localhost:8083/api/v1/agents/install \
  -H "Content-Type: application/json" \
  -d '{"platform": "linux", "tenant_id": "your-company"}'

# Run on target host
bash install-agent.sh
```

### Configure Detection Rules
```bash
# Sigma rules
ls vendor/sigma/rules/

# YARA rules
ls vendor/yara-rules/

# Suricata rules
ls vendor/suricata-rules/
```

### View Documentation
```bash
# Open portal docs
cd portal-docs
cat INDEX.md
```

---

## Troubleshooting

### Services Won't Start
```bash
# Check Docker resources
docker system df

# Increase Docker memory to 8GB minimum
# Docker Desktop → Settings → Resources

# Restart Docker
docker compose down
docker compose up -d
```

### Port Conflicts
```bash
# Check what's using ports
netstat -tulpn | grep -E '3000|8000|5432|9000'

# Stop conflicting services or change ports in docker-compose.yml
```

### Database Connection Errors
```bash
# Reset databases
docker compose down -v
docker compose up -d postgres clickhouse

# Wait for initialization
sleep 30
```

---

## Production Deployment

For production, use `docker-compose.prod.yml`:

```bash
# Production stack with replicas and resource limits
docker compose -f docker-compose.prod.yml up -d

# Includes:
# - 3 replicas for API services
# - Resource limits (CPU/memory)
# - Health checks
# - Watchtower auto-updates
# - JSON logging
```

---

## Support

- **Documentation:** [portal-docs/INDEX.md](portal-docs/INDEX.md)
- **Issues:** https://github.com/managekube-hue/Kubric-UiDR/issues
- **Discussions:** https://github.com/managekube-hue/Kubric-UiDR/discussions

---

## License

See [LICENSE](LICENSE) file for details.
