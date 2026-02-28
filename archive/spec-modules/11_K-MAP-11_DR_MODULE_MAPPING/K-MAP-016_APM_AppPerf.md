# K-MAP-016 — Application Performance Management (APM)
**Discipline:** Application Performance Management
**Abbreviation:** APM
**Kubric Reference:** K-MAP-016
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Application Performance Management (APM) monitors and manages the performance and availability of software applications to detect performance degradation, errors, and slow transactions before they impact users. APM encompasses distributed tracing, metrics collection, log aggregation, and synthetic monitoring. In a security platform context, APM signals (unusual latency spikes, error rate increases, throughput drops) can indicate active attacks, overloaded detection pipelines, or resource-abuse incidents. Kubric implements APM through the PerfTrace agent (Linux perf_event counters, sysinfo metrics, Prometheus exposition, and OpenTelemetry export), the FastAPI server's built-in request instrumentation, and Socket.IO for real-time performance event streaming.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Host Performance Collection | PerfTrace sysinfo host metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |
| Hardware Counter Collection | PerfTrace perf_event_open | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs` |
| Prometheus Metrics Exposition | PerfTrace Prometheus registry | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs` |
| OpenTelemetry Trace + Metrics | PerfTrace OTEL exporter | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs` |
| API Request Performance | KAI API FastAPI server | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py` |
| Real-Time Performance Events | KAI API Socket.IO | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py` |
| Async ASGI Performance | KAI API anyio backend | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-007_anyio_backend.py` |
| Database Performance | KAI API asyncpg client | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py` |
| ClickHouse Query Performance | KAI API ClickHouse connect | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |
| NATS Performance | KAI API nats-py client | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-006_nats_py_client.py` |
| ML Model Performance | KAI ML TensorBoard logger | `03_K-KAI-03_ORCHESTRATION/K-KAI-ML-001_tensorboard_logger.py` |
| Process CPU Profiling | PerfTrace top-50 process CPU | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |

---

## 3. Data Flow Diagram

```
Application Services
│
├── KAI FastAPI (Python + anyio)
│   ├── Per-request: latency, status code, path
│   ├── AsyncPG: DB query latency
│   ├── ClickHouse: query execution time
│   └── NATS publish latency
│
├── K-XRO Agents (Rust)
│   ├── PerfTrace: collection cycle duration histogram
│   ├── CoreSec: event emit rate, governor drops
│   └── NetGuard: flow events/s, capture drop rate
│
PerfTrace (per host, every 15 s):
├── CPU: usage_pct, per-core breakdown
├── Memory: usage_pct, swap
├── Process: top-50 by CPU (histogram kubric_process_cpu_pct_bucket)
├── NIC: rx_bytes, tx_bytes, drops
└── perf_event_open (Linux):
     IPC, cache-miss, branch-miss, instructions, cycles

┌──────────────────────────────────────────────────────┐
│  Observability Backends                             │
│                                                    │
│  Prometheus    ◄── :9090/metrics (per host)         │
│    └── Grafana dashboards                          │
│                                                    │
│  OTEL Collector ◄── PerfTrace gRPC export          │
│    ├── Trace spans → Jaeger / Tempo                │
│    └── Metrics → Prometheus remote write          │
│                                                    │
│  TensorBoard   ◄── KAI ML model training logs      │
│  Socket.IO     ◄── Real-time dashboard push        │
└──────────────────────────────────────────────────────┘
```

---

## 4. APM Metrics Reference

| Metric | Type | Description | Source |
|---|---|---|---|
| `kubric_collection_duration_seconds` | Histogram | PerfTrace collection cycle latency | PerfTrace |
| `kubric_nats_publish_errors_total` | Counter | NATS publish failures | PerfTrace |
| `kubric_cpu_usage_pct` | Gauge | Host CPU utilization | PerfTrace sysinfo |
| `kubric_memory_usage_pct` | Gauge | Host memory utilization | PerfTrace sysinfo |
| `kubric_process_cpu_pct_bucket` | Histogram | Per-process CPU distribution | PerfTrace sysinfo |
| `kubric_net_rx_bytes_total` | Counter | NIC receive bytes | PerfTrace sysinfo |
| `kubric_perf_ipc` | Gauge | Instructions per clock cycle | perf_event_open |
| `kubric_perf_cache_miss_rate` | Gauge | Last-level cache miss rate | perf_event_open |
| `kubric_api_request_duration_seconds` | Histogram | FastAPI per-route latency | FastAPI middleware |
| `kubric_api_requests_total` | Counter | Total API requests by status | FastAPI middleware |

---

## 5. APM Health SLOs

| SLI | SLO |
|---|---|
| PerfTrace collection p99 | < 1 s |
| FastAPI API p99 response time | < 200 ms |
| ClickHouse query p99 | < 50 ms |
| NATS publish failure rate | < 0.1% |
| Host CPU utilization p95 | < 70% |
| Host memory utilization p95 | < 80% |

---

## 6. Integration Points

| System | APM Role |
|---|---|
| **Prometheus** | Metrics scraping; alerting rules for SLO thresholds |
| **Grafana** | APM dashboards; p99 latency, error rate, resource utilization |
| **OpenTelemetry** | Distributed tracing and metrics export |
| **TensorBoard** | ML model training performance tracking |
| **Socket.IO** | Real-time performance event push to dashboard frontend |
| **NATS** | Message throughput as an APM signal |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| PerfTrace | Deployed on every host; `COLLECTION_INTERVAL_SECS=15` default |
| Prometheus | Scrape config targeting all PerfTrace `:9090/metrics` instances |
| OTEL endpoint | `OTEL_ENDPOINT` env var set on PerfTrace if using trace export |
| Linux perf_event | Kernel ≥ 3.10; CAP_PERFMON or perf_event_paranoid ≤ 1 |
| Grafana | Dashboards importing Prometheus datasource |
| TensorBoard | Optional; enable via `TENSORBOARD_LOG_DIR` on KAI training nodes |
