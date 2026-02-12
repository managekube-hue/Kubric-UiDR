# SMP (Service Minimalism) Practice Map

Maps Kubric to minimal viable service practices.

## Minimal Configuration

| Practice | Implementation | Status |
|----------|---|---|
| **Deploy NATS cluster** | 3-node StatefulSet | ✅ |
| **Deploy ClickHouse** | 1-node (scalable) | ✅ |
| **Deploy PostgreSQL** | 1-node with RLS | ✅ |
| **Deploy API Gateway** | 3-replica Deployment | ✅ |
| **Configure Ingress** | Caddy reverse proxy | ✅ |
| **Enable TLS** | cert-manager integration | ⚙️ (Optional) |
| **Setup Monitoring** | Prometheus + OTel | ⚙️ (Optional) |
| **Configure Backup** | Restic snapshots | ⚙️ (Optional) |

## Streamlined Workflows

### Incident Response (Minimal)
```
Alert → Triage Agent → Housekeeper → Close
```

### Agent Deployment (Minimal)
```
Install Token → Register → NATS Subscribe → Start
```

### Billing (Minimal)
```
Heartbeat → Aggregate → Invoice
```

---

Generated: 2026-02-12
