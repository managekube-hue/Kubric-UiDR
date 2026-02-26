# K-ITIL-MAT-007 — ITIL 4 TMP2: Infrastructure and Platform Management
**Practice Code:** TMP2
**Practice Family:** Technical Management Practices
**Kubric Reference:** K-ITIL-MAT-007
**Last Updated:** 2026-02-26

---

## 1. ITIL 4 Practice Overview

Infrastructure and Platform Management ensures that the infrastructure and platforms used by an organization are effectively managed, supporting the delivery of services. It covers compute, network, storage, container orchestration, and the observability stack that makes infrastructure behavior visible to operations teams. Kubric's infrastructure layer is fully Kubernetes-native, with platform telemetry collected by the PerfTrace agent (Rust), orchestrated via NATS JetStream, and surfaced through Prometheus/Grafana and OpenTelemetry pipelines.

---

## 2. Kubric Platform Mapping

| ITIL Sub-Capability | Kubric Module | File Reference |
|---|---|---|
| Host Metrics Collection | PerfTrace – sysinfo host metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |
| Hardware Counter Collection | PerfTrace – perf_event_open | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs` |
| Prometheus Exposition | PerfTrace – Prometheus registry | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs` |
| OpenTelemetry Export | PerfTrace – OTEL collector | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs` |
| Agent Lifecycle Management | Watchdog – agent orchestrator | `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs` |
| eBPF Kernel Visibility | CoreSec – eBPF map pressure monitor | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs` |
| Network Flow Telemetry | NetGuard – AF_PACKET ring buffer | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs` |
| High-Perf Network Capture | NetGuard – DPDK bypass | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs` |
| Telemetry Messaging | NATS JetStream (all agents) | Kubric NATS subjects `kubric.*` |
| Platform Health Dashboard | Grafana + Prometheus | PerfTrace `:9090/metrics` endpoint |
| Columnar Event Storage | ClickHouse | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |
| Platform API | FastAPI async server | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py` |
| Async DB Access | asyncpg (PostgreSQL) | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py` |
| Event Streaming | aiokafka consumer | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-005_aiokafka_consumer.py` |
| Real-Time UI Push | Socket.IO | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py` |

---

## 3. RACI Matrix

| Activity | Platform Eng | Operations | CISO | KAI Agents |
|---|---|---|---|---|
| Define infrastructure capacity targets | **A** | R | C | I |
| Operate NATS JetStream cluster | **A** | R | I | I |
| Operate Kubernetes cluster | **A** | R | I | I |
| Collect and publish host metrics | I | C | I | **R** |
| Monitor infrastructure health | C | **A** | I | **R** |
| Respond to eBPF map pressure alerts | **A** | R | I | R |
| Tune Prometheus scrape intervals | **A** | R | I | I |
| Manage ClickHouse schema and TTL | **A** | R | I | I |
| Manage OTEL pipeline | **A** | R | I | R |

---

## 4. NATS Subjects and API Endpoints

### NATS Subjects

| Subject | Publisher | Payload |
|---|---|---|
| `kubric.{tenant}.metrics.host.v1` | PerfTrace | OCSF-5001: cpu_pct, mem_pct, process_count, disk_io |
| `kubric.{tenant}.agent.heartbeat` | All agents | `{agent_id, agent_type, uptime_secs}` |
| `kubric.{tenant}.agent.status.v1` | Watchdog | `{agent_id, agent_type, status, last_seen_secs_ago}` |
| `kubric.{tenant}.network.flow.v1` | NetGuard | FlowEvent: 5-tuple, bytes, packets, duration_ms |

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/platform/health` | GET | Aggregate platform health from Watchdog |
| `/api/v1/platform/agents` | GET | All registered agents with status and uptime |
| `/api/v1/platform/metrics/host` | GET | Latest host metrics per agent from ClickHouse |
| `/api/v1/platform/nats/status` | GET | NATS JetStream stream lag and consumer status |
| `GET :9090/metrics` | GET | Prometheus scrape endpoint (PerfTrace per host) |
| `GET :9090/healthz` | GET | PerfTrace agent liveness probe |
| `GET :9090/ready` | GET | PerfTrace readiness probe |

---

## 5. Platform Observability Stack

```
┌──────────────────────────────────────────────────────────────────┐
│  Kubric Host                                                     │
│                                                                  │
│  PerfTrace Agent (Rust)                                          │
│  ├── sysinfo: CPU, memory, disk, NIC, top-50 processes           │
│  ├── perf_event_open: IPC, cache-miss, branch-miss (Linux only)  │
│  ├── Prometheus registry ──► :9090/metrics                       │
│  └── OTEL exporter ──────────► OTLP gRPC endpoint               │
│                                                                  │
│  Published: kubric.{tenant}.metrics.host.v1 (NATS)              │
└──────────────────────────────────────────────────────────────────┘
                |                         |
       Prometheus scrape          OTEL Collector
                |                         |
        Grafana dashboards        Jaeger / Tempo traces
                |
      Alerting rules (PagerDuty / Comm Agent)
```

---

## 6. KPIs and Metrics

| KPI | Target | Source |
|---|---|---|
| Host CPU Utilisation (p95) | < 70% | PerfTrace `kubric.{tenant}.metrics.host.v1` |
| Host Memory Utilisation (p95) | < 80% | PerfTrace host metrics |
| Platform API p99 Response Time | < 200 ms | FastAPI metrics |
| NATS JetStream Consumer Lag | < 1000 messages | NATS monitoring endpoint |
| ClickHouse Insert Latency (p99) | < 50 ms | ClickHouse system.query_log |
| Agent Fleet Availability | ≥ 99.5% | Watchdog `agent.status.v1` |
| PerfTrace Collection Cycle Duration | < 1 s | `collection_duration_seconds` histogram |
| eBPF Map Pressure (% full) | < 60% | CoreSec `K-XRO-CS-EBPF-003_map_pressure.rs` |

---

## 7. Integration Points

| System | Infrastructure Role |
|---|---|
| **Kubernetes** | Container orchestration; Deployments, Services, ConfigMaps, Secrets |
| **ArgoCD** | GitOps reconciliation of all K8s manifests |
| **NATS JetStream** | Durable, ordered messaging backbone for all agents |
| **ClickHouse** | Columnar time-series store for telemetry and events |
| **PostgreSQL** | Transactional store for case data, configuration, billing |
| **Prometheus** | Metrics scraping from PerfTrace and all platform services |
| **Grafana** | Dashboards, alerting, and SLO tracking |
| **OpenTelemetry** | Distributed tracing and metrics export to backend observability platform |
| **Vault** | PKI and secrets management for platform infrastructure |
| **cert-manager** | Automated TLS certificate issuance for all services |

---

## 8. References

- ITIL 4 Foundation: *Infrastructure and Platform Management* (TMP2)
- Linux `perf_event_open(2)` man page
- OpenTelemetry spec: https://opentelemetry.io/docs/specs/otel
- NATS JetStream: https://docs.nats.io/nats-concepts/jetstream
- Kubric PerfTrace: `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/`
- Kubric Watchdog: `02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/`
