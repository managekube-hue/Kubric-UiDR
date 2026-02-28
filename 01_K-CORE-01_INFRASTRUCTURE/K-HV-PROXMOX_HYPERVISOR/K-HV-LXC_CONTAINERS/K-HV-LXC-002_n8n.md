# K-HV-LXC-002 — n8n Workflow Automation

> **Container ID:** 111  
> **Resources:** 2 vCPU, 4 GB RAM, 20 GB disk  
> **Node:** pve-kubric-02  
> **IP:** 10.0.50.32/24  
> **Access:** https://n8n.kubric.local  
> **License:** Sustainable Use License (non-production or fair-use evaluation)

---

## 1. LXC Container Creation

```bash
# On pve-kubric-02
pct create 111 local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst \
  --hostname n8n-kubric \
  --cores 2 \
  --memory 4096 \
  --swap 1024 \
  --rootfs kubric-ceph:20 \
  --net0 name=eth0,bridge=vmbr2,ip=10.0.50.32/24,gw=10.0.50.1 \
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
pct enter 111

apt-get update && apt-get upgrade -y
apt-get install -y curl wget ca-certificates gnupg lsb-release

# Install Docker
curl -fsSL https://get.docker.com | sh
apt-get install -y docker-compose-plugin
```

---

## 3. n8n Deployment

### 3.1 Directory Structure

```bash
mkdir -p /opt/n8n/{data,postgres}
```

### 3.2 Docker Compose

File: `/opt/n8n/docker-compose.yml`

```yaml
version: "3.8"

services:
  n8n-db:
    image: postgres:16-bookworm
    container_name: n8n-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: n8n
      POSTGRES_PASSWORD: "${N8N_DB_PASS}"
      POSTGRES_DB: n8n
    volumes:
      - /opt/n8n/postgres:/var/lib/postgresql/data
    networks:
      - n8n-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U n8n"]
      interval: 10s
      timeout: 5s
      retries: 5

  n8n:
    image: n8nio/n8n:1.64.3
    container_name: n8n
    restart: unless-stopped
    depends_on:
      n8n-db:
        condition: service_healthy
    ports:
      - "5678:5678"
    environment:
      # Database
      - DB_TYPE=postgresdb
      - DB_POSTGRESDB_HOST=n8n-db
      - DB_POSTGRESDB_PORT=5432
      - DB_POSTGRESDB_DATABASE=n8n
      - DB_POSTGRESDB_USER=n8n
      - DB_POSTGRESDB_PASSWORD=${N8N_DB_PASS}

      # General
      - N8N_HOST=n8n.kubric.local
      - N8N_PORT=5678
      - N8N_PROTOCOL=https
      - WEBHOOK_URL=https://n8n.kubric.local/
      - GENERIC_TIMEZONE=UTC

      # Security
      - N8N_BASIC_AUTH_ACTIVE=true
      - N8N_BASIC_AUTH_USER=kubric-admin
      - N8N_BASIC_AUTH_PASSWORD=${N8N_ADMIN_PASS}
      - N8N_ENCRYPTION_KEY=${N8N_ENCRYPTION_KEY}

      # Execution
      - EXECUTIONS_MODE=regular
      - EXECUTIONS_DATA_PRUNE=true
      - EXECUTIONS_DATA_MAX_AGE=168

      # External hooks
      - N8N_DIAGNOSTICS_ENABLED=false
      - N8N_TEMPLATES_ENABLED=true
    volumes:
      - /opt/n8n/data:/home/node/.n8n
    networks:
      - n8n-net

networks:
  n8n-net:
    driver: bridge
```

### 3.3 Environment File

File: `/opt/n8n/.env`

```bash
N8N_DB_PASS=<generate with: openssl rand -base64 24>
N8N_ADMIN_PASS=<generate with: openssl rand -base64 16>
N8N_ENCRYPTION_KEY=<generate with: openssl rand -hex 32>
```

### 3.4 Start n8n

```bash
cd /opt/n8n
docker compose up -d

# Verify
docker logs -f n8n
# Wait for "n8n ready on 0.0.0.0, port 5678"
```

---

## 4. Workflow Templates

### 4.1 O365 / Microsoft Graph Log Polling

Import via n8n UI or API:

```json
{
  "name": "O365 Audit Log Ingestion",
  "nodes": [
    {
      "name": "Schedule Trigger",
      "type": "n8n-nodes-base.scheduleTrigger",
      "parameters": {
        "rule": { "interval": [{ "field": "minutes", "minutesInterval": 5 }] }
      },
      "position": [250, 300]
    },
    {
      "name": "Microsoft Graph API",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "GET",
        "url": "https://graph.microsoft.com/v1.0/auditLogs/signIns",
        "authentication": "oAuth2",
        "queryParameters": {
          "parameters": [
            { "name": "$filter", "value": "createdDateTime ge {{$now.minus(5, 'minutes').toISO()}}" },
            { "name": "$top", "value": "100" }
          ]
        },
        "options": { "response": { "response": { "responseFormat": "json" } } }
      },
      "position": [450, 300]
    },
    {
      "name": "Transform to OCSF",
      "type": "n8n-nodes-base.code",
      "parameters": {
        "jsCode": "return items.map(item => {\n  const log = item.json;\n  return {\n    json: {\n      class_uid: 3002,\n      category_uid: 3,\n      type_uid: 300201,\n      time: log.createdDateTime,\n      severity_id: log.riskState === 'atRisk' ? 4 : 1,\n      actor: { user: { name: log.userDisplayName, email_addr: log.userPrincipalName } },\n      src_endpoint: { ip: log.ipAddress },\n      status: log.status?.errorCode === 0 ? 'Success' : 'Failure',\n      metadata: { product: { name: 'Microsoft 365' }, version: '1.1.0' }\n    }\n  };\n});"
      },
      "position": [650, 300]
    },
    {
      "name": "Push to NATS",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "POST",
        "url": "http://10.0.50.21:8222/publish/kubric.ocsf.authentication",
        "sendBody": true,
        "bodyParameters": { "parameters": [{ "name": "payload", "value": "={{JSON.stringify($json)}}" }] }
      },
      "position": [850, 300]
    }
  ],
  "connections": {
    "Schedule Trigger": { "main": [[{ "node": "Microsoft Graph API", "type": "main", "index": 0 }]] },
    "Microsoft Graph API": { "main": [[{ "node": "Transform to OCSF", "type": "main", "index": 0 }]] },
    "Transform to OCSF": { "main": [[{ "node": "Push to NATS", "type": "main", "index": 0 }]] }
  }
}
```

### 4.2 Threat Feed Ingestion (AlienVault OTX)

```json
{
  "name": "Threat Feed - AlienVault OTX",
  "nodes": [
    {
      "name": "Schedule",
      "type": "n8n-nodes-base.scheduleTrigger",
      "parameters": {
        "rule": { "interval": [{ "field": "hours", "hoursInterval": 1 }] }
      },
      "position": [250, 300]
    },
    {
      "name": "OTX Subscribed Pulses",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "GET",
        "url": "https://otx.alienvault.com/api/v1/pulses/subscribed?modified_since={{$now.minus(1,'hour').toISO()}}",
        "headerParameters": {
          "parameters": [{ "name": "X-OTX-API-KEY", "value": "${OTX_API_KEY}" }]
        }
      },
      "position": [450, 300]
    },
    {
      "name": "Extract IOCs",
      "type": "n8n-nodes-base.code",
      "parameters": {
        "jsCode": "const iocs = [];\nfor (const item of items) {\n  for (const pulse of item.json.results || []) {\n    for (const indicator of pulse.indicators || []) {\n      iocs.push({ json: {\n        type: indicator.type,\n        value: indicator.indicator,\n        pulse_name: pulse.name,\n        created: indicator.created,\n        tlp: pulse.TLP || 'white'\n      }});\n    }\n  }\n}\nreturn iocs;"
      },
      "position": [650, 300]
    },
    {
      "name": "Insert to ClickHouse",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "POST",
        "url": "http://10.0.50.21:8123/?query=INSERT INTO kubric_telemetry.threat_indicators FORMAT JSONEachRow",
        "sendBody": true,
        "bodyParameters": { "parameters": [{ "name": "", "value": "={{JSON.stringify($json)}}" }] }
      },
      "position": [850, 300]
    }
  ],
  "connections": {
    "Schedule": { "main": [[{ "node": "OTX Subscribed Pulses", "type": "main", "index": 0 }]] },
    "OTX Subscribed Pulses": { "main": [[{ "node": "Extract IOCs", "type": "main", "index": 0 }]] },
    "Extract IOCs": { "main": [[{ "node": "Insert to ClickHouse", "type": "main", "index": 0 }]] }
  }
}
```

### 4.3 Slack/Teams Alert Workflow

```json
{
  "name": "Kubric Alert → Slack + Teams",
  "nodes": [
    {
      "name": "NATS Webhook",
      "type": "n8n-nodes-base.webhook",
      "parameters": {
        "path": "kubric-alert",
        "httpMethod": "POST"
      },
      "position": [250, 300]
    },
    {
      "name": "Format Alert",
      "type": "n8n-nodes-base.code",
      "parameters": {
        "jsCode": "const alert = items[0].json;\nconst severity = ['info','low','medium','high','critical'][alert.severity_id || 0];\nreturn [{ json: {\n  text: `🚨 *Kubric Alert* [${severity.toUpperCase()}]\\n*${alert.title || 'Security Event'}*\\nSource: ${alert.src_endpoint?.ip || 'N/A'}\\nTime: ${alert.time}\\nDetails: ${alert.message || JSON.stringify(alert).slice(0,200)}`\n}}];"
      },
      "position": [450, 300]
    },
    {
      "name": "Slack",
      "type": "n8n-nodes-base.slack",
      "parameters": {
        "channel": "#kubric-alerts",
        "text": "={{$json.text}}"
      },
      "position": [650, 250]
    },
    {
      "name": "Teams Webhook",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "method": "POST",
        "url": "${TEAMS_WEBHOOK_URL}",
        "sendBody": true,
        "bodyParameters": { "parameters": [{ "name": "text", "value": "={{$json.text}}" }] }
      },
      "position": [650, 400]
    }
  ],
  "connections": {
    "NATS Webhook": { "main": [[{ "node": "Format Alert", "type": "main", "index": 0 }]] },
    "Format Alert": { "main": [[{ "node": "Slack", "type": "main", "index": 0 }, { "node": "Teams Webhook", "type": "main", "index": 0 }]] }
  }
}
```

---

## 5. NATS Webhook Trigger

Configure NATS to forward alert events to n8n:

```bash
# NATS subscription that POSTs to n8n webhook
nats sub "kubric.alerts.>" --each \
  'curl -s -X POST http://10.0.50.32:5678/webhook/kubric-alert \
    -H "Content-Type: application/json" \
    -d "$MSG"' &

# Or use a Go bridge (see kai/nats-webhook-bridge)
```

---

## 6. Sustainable Use License Notes

> **n8n uses the Sustainable Use License (SUL) since v1.0.** This means:
> - ✅ Free for internal use, self-hosted
> - ✅ Free for evaluation and testing
> - ❌ Cannot offer n8n as a managed service to third parties
> - ✅ Kubric internal automation use case is compliant
>
> If Kubric becomes a commercial product offering workflow automation to customers, evaluate the n8n Enterprise license or replace with Apache-licensed alternatives (e.g., Temporal, Prefect).

---

## 7. Verification

```bash
# Container
pct status 111    # running

# n8n web UI
curl -s http://10.0.50.32:5678/healthz
# Expected: {"status":"ok"}

# Docker containers
pct exec 111 -- docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Test webhook
curl -X POST http://10.0.50.32:5678/webhook/kubric-alert \
  -H "Content-Type: application/json" \
  -d '{"severity_id":3,"title":"Test Alert","time":"2026-02-28T12:00:00Z","message":"Test alert from verification"}'

# PostgreSQL
pct exec 111 -- docker exec n8n-postgres pg_isready
```
