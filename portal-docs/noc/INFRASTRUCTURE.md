# NOC - Infrastructure Management

**Module:** K-NOC-05  
**Purpose:** Network Operations Center - Infrastructure health and monitoring

---

## Overview

NOC manages cluster health, agent lifecycle, and infrastructure monitoring across the Kubric platform.

---

## Components

### 1. Cluster Management
- Proxmox cluster orchestration
- Ceph storage management
- Network topology monitoring
- Resource allocation

### 2. Agent Management
- Agent deployment and provisioning
- Health monitoring and heartbeats
- OTA updates via Watchdog
- Configuration management

### 3. Performance Monitoring
- PerfTrace metrics collection
- Resource utilization tracking
- Capacity planning
- Alerting and escalation

---

## API Endpoints

**Base URL:** `http://noc:8083`

### Health Check
```
GET /healthz
```

### Agent Management
```
GET  /api/v1/agents              # List all agents
GET  /api/v1/agents/{id}         # Get agent details
POST /api/v1/agents/{id}/restart # Restart agent
```

### Cluster Status
```
GET /api/v1/cluster/status       # Cluster health
GET /api/v1/cluster/nodes        # Node list
GET /api/v1/cluster/resources    # Resource usage
```

---

## NATS Subjects

### Agent Heartbeats
```
kubric.{tenant_id}.agent.heartbeat.v1
```

### Performance Metrics
```
kubric.{tenant_id}.perf.host.v1
```

### Cluster Events
```
kubric.{tenant_id}.cluster.node.{up|down}
kubric.{tenant_id}.cluster.resource.{warning|critical}
```

---

## Monitoring

### Grafana Dashboards
- NOC Overview Dashboard
- Agent Health Dashboard
- Cluster Resources Dashboard

### Alerts
- Agent offline > 5 minutes
- CPU usage > 80%
- Memory usage > 90%
- Disk usage > 85%

---

## Related Documentation

- [Agent Management](AGENTS.md)
- [PerfTrace Agent](../architecture/ARCHITECTURE.md#perftrace)
- [Watchdog OTA Updates](../architecture/ARCHITECTURE.md#watchdog)
