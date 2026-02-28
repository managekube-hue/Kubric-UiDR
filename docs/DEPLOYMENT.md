# Kubric-UiDR Deployment Guide

**Single source of truth for deploying Kubric in any environment.**

---

## Quick Start (Local Development)

```bash
# 1. Clone and enter repo
git clone https://github.com/managekube-hue/Kubric-UiDR.git
cd Kubric-UiDR

# 2. Copy environment template
cp .env.example .env

# 3. Start infrastructure only
docker compose up -d

# 4. Start full stack (including app services)
docker compose --profile app up -d

# 5. Verify health
docker compose ps
curl http://localhost:8080/healthz  # K-SVC
curl http://localhost:8081/healthz  # VDR
curl http://localhost:8082/healthz  # KIC
curl http://localhost:8083/healthz  # NOC
curl http://localhost:3001/         # Frontend
```

**Access:**
- Frontend: http://localhost:3001
- Grafana: http://localhost:3000 (admin/kubric-grafana)
- Temporal UI: http://localhost:8233
- NATS Monitoring: http://localhost:8222

---

## Production Deployment (Docker Compose)

```bash
# 1. Prepare environment
cp .env.example .env.production
# Edit .env.production with production secrets

# 2. Build images
docker compose -f docker-compose.prod.yml build

# 3. Deploy with 3 replicas
docker compose -f docker-compose.prod.yml up -d

# 4. Verify all services healthy
docker compose -f docker-compose.prod.yml ps
```

**Production Features:**
- 3 replicas for stateless services (ksvc, vdr, kic, noc)
- Resource limits enforced (CPU/memory)
- Automatic restarts on failure
- Log rotation configured
- Health checks every 10s
- Watchtower auto-updates enabled

---

## AWS Deployment (Amplify + ECS)

```bash
# 1. Configure AWS credentials
aws configure

# 2. Set environment variables
export AWS_REGION=us-east-1
export ECR_REGISTRY=123456789.dkr.ecr.us-east-1.amazonaws.com
export DOMAIN=kubric.security

# 3. Run deployment script
bash scripts/deploy-aws.sh

# 4. Outputs
# API: https://api.kubric.security
# Portal: https://app.kubric.security
```

**AWS Architecture:**
- Frontend: Amplify Hosting (CDN + auto-scaling)
- Backend: ECS Fargate (3 tasks per service)
- Database: RDS Aurora Serverless v2 (PostgreSQL)
- Cache: ElastiCache Redis
- Storage: S3 + CloudFront
- Monitoring: CloudWatch + Grafana

**Estimated Cost:** $500-800/month for 10k endpoints

---

## Agent Deployment

### Linux (systemd)

```bash
# 1. Download agent installer
curl -fsSL https://install.kubric.security/linux.sh | bash

# 2. Configure tenant
export KUBRIC_TENANT_ID=your-tenant-id
export KUBRIC_NATS_URL=nats://your-nats-server:4222

# 3. Start agents
systemctl enable --now kubric-coresec
systemctl enable --now kubric-netguard
systemctl enable --now kubric-perftrace
systemctl enable --now kubric-watchdog

# 4. Verify
systemctl status kubric-coresec
journalctl -u kubric-coresec -f
```

### Windows (Service)

```powershell
# 1. Download agent installer
Invoke-WebRequest -Uri https://install.kubric.security/windows.ps1 -OutFile install.ps1
.\install.ps1

# 2. Configure tenant
$env:KUBRIC_TENANT_ID = "your-tenant-id"
$env:KUBRIC_NATS_URL = "nats://your-nats-server:4222"

# 3. Start agents
Start-Service KubricCoreSec
Start-Service KubricPerfTrace
Start-Service KubricWatchdog

# 4. Verify
Get-Service Kubric*
Get-EventLog -LogName Application -Source Kubric* -Newest 10
```

---

## KAI AI Autonomy

### Enable Auto-Scaling

```bash
# KAI DEPLOY persona will auto-scale based on load
docker exec -it kubric-kai-python-1 kai deploy enable-autoscale

# View current scaling rules
docker exec -it kubric-kai-python-1 kai deploy rules
```

### Enable Auto-Remediation

```bash
# KAI HOUSE persona will restart failed services
docker exec -it kubric-kai-python-1 kai house enable-remediation

# View infrastructure health
docker exec -it kubric-kai-python-1 kai house status
```

---

## Backup & Disaster Recovery

### Automated Backups

```bash
# Backups run automatically every hour (ClickHouse) and daily (PostgreSQL, Neo4j, Vault)
# Stored in MinIO S3-compatible storage

# Manual backup trigger
docker exec -it kubric-noc-1 /app/backup-all.sh

# List backups
docker exec -it kubric-minio-1 mc ls local/kubric-backups/

# Restore from backup
docker exec -it kubric-noc-1 /app/restore-from-backup.sh 2026-02-27-12-00
```

### Disaster Recovery Test

```bash
# Run quarterly DR drill
make restore-drill

# Expected: All data restored, services healthy within 5 minutes
```

---

## Monitoring & Observability

### Grafana Dashboards

- **Platform Health**: http://localhost:3000/d/kubric-platform
- **Tenant View**: http://localhost:3000/d/kubric-tenant
- **Agent Metrics**: http://localhost:3000/d/kubric-agents
- **KAI Performance**: http://localhost:3000/d/kubric-kai

### Alerts

Configured alerts (Slack/PagerDuty/Email):
- NATS JetStream lag > 10k messages
- ClickHouse ingestion rate < 1k/sec
- Vault sealed
- Agent heartbeat missing > 5 min
- Temporal workflow failures
- Pod crash loop

### Logs

```bash
# View all logs
docker compose logs -f

# View specific service
docker compose logs -f ksvc

# Search logs (Loki)
curl -G http://localhost:3100/loki/api/v1/query \
  --data-urlencode 'query={service="ksvc"}' \
  --data-urlencode 'limit=100'
```

---

## Scaling Guide

### Horizontal Scaling

```bash
# Scale specific service
docker compose -f docker-compose.prod.yml up -d --scale ksvc=5

# Or let KAI DEPLOY auto-scale
docker exec -it kubric-kai-python-1 kai deploy scale ksvc 5
```

### Vertical Scaling

Edit `docker-compose.prod.yml`:

```yaml
services:
  ksvc:
    deploy:
      resources:
        limits:
          cpus: '2.0'      # Increase from 1.0
          memory: 2048M    # Increase from 1024M
```

### Database Scaling

**PostgreSQL:**
- Enable streaming replication (3 nodes)
- Use connection pooling (PgBouncer)

**ClickHouse:**
- Add shards for horizontal scaling
- Configure distributed tables

**Neo4j:**
- Enable clustering (3+ nodes)
- Use read replicas

---

## Security Hardening

### TLS/SSL

```bash
# Generate self-signed certs (dev)
make generate-certs

# Use Let's Encrypt (prod)
# Caddy handles automatic cert renewal
```

### Secrets Management

```bash
# Store secrets in Vault
docker exec -it kubric-vault-1 vault kv put secret/kubric/stripe api_key=sk_live_...

# Rotate secrets
docker exec -it kubric-vault-1 vault kv put secret/kubric/jwt secret=$(openssl rand -base64 32)
```

### Network Security

```yaml
# docker-compose.prod.yml
networks:
  kubric:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16
    driver_opts:
      com.docker.network.bridge.enable_ip_masquerade: "true"
      com.docker.network.bridge.enable_icc: "false"  # Disable inter-container communication
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker compose logs <service>

# Check health
docker compose ps

# Restart service
docker compose restart <service>

# Full reset
docker compose down -v
docker compose up -d
```

### Agent Not Connecting

```bash
# Check NATS connectivity
docker exec -it kubric-nats-1 nats server check connection

# Check agent logs
journalctl -u kubric-coresec -f

# Verify tenant ID
echo $KUBRIC_TENANT_ID
```

### Database Connection Issues

```bash
# PostgreSQL
docker exec -it kubric-postgres-1 psql -U kubric -d kubric -c "SELECT 1;"

# ClickHouse
docker exec -it kubric-clickhouse-1 clickhouse-client --query "SELECT 1;"

# Neo4j
docker exec -it kubric-neo4j-1 cypher-shell -u neo4j -p kubric-neo4j "RETURN 1;"
```

### Performance Issues

```bash
# Check resource usage
docker stats

# Check KAI HOUSE recommendations
docker exec -it kubric-kai-python-1 kai house analyze

# Enable debug logging
export KUBRIC_LOG_LEVEL=debug
docker compose restart
```

---

## Upgrade Procedure

### Rolling Update (Zero Downtime)

```bash
# 1. Pull new images
docker compose -f docker-compose.prod.yml pull

# 2. Update one service at a time
docker compose -f docker-compose.prod.yml up -d --no-deps ksvc

# 3. Verify health before next service
curl http://localhost:8080/healthz

# 4. Continue for all services
docker compose -f docker-compose.prod.yml up -d --no-deps vdr
docker compose -f docker-compose.prod.yml up -d --no-deps kic
docker compose -f docker-compose.prod.yml up -d --no-deps noc
```

### Rollback

```bash
# Rollback to previous version
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml up -d --force-recreate

# Or use KAI DEPLOY
docker exec -it kubric-kai-python-1 kai deploy rollback ksvc
```

---

## Support

- **Documentation**: https://docs.kubric.security
- **GitHub Issues**: https://github.com/managekube-hue/Kubric-UiDR/issues
- **Discord**: https://discord.gg/kubric
- **Email**: support@kubric.security

---

**Last Updated:** 2026-02-27  
**Version:** 1.0.0
