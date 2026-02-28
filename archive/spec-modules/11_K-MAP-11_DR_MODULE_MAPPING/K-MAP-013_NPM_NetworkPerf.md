# K-MAP-013 — Network Performance Management (NPM)
**Discipline:** Network Performance Management
**Abbreviation:** NPM
**Kubric Reference:** K-MAP-013
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Network Performance Management (NPM) monitors, analyzes, and optimizes network performance to ensure that applications receive the bandwidth and latency characteristics they require. In a security platform context, NPM both supports operational health and provides a security signal: network performance anomalies — sudden drops in throughput, unexpected protocol distribution changes, or latency spikes — can indicate attacks such as DDoS, bandwidth-eating C2 beaconing, or misconfigured network devices. Kubric delivers NPM through the NetGuard agent (flow analysis, DPI, protocol classification) and PerfTrace (NIC statistics, host network counters), with all data surfaced via Prometheus and Grafana.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Packet Capture Rate | NetGuard AF_PACKET ring buffer | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs` |
| High-Speed Capture | NetGuard DPDK bypass | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs` |
| Bidirectional Flow Analysis | NetGuard FlowAnalyzer | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs` |
| Protocol Distribution (L7) | NetGuard nDPI FFI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs` |
| Protocol Classification | NetGuard L7 classifier | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs` |
| Host NIC Metrics | PerfTrace sysinfo host metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |
| Hardware Perf Counters | PerfTrace perf_event_open | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs` |
| Prometheus Metrics | PerfTrace Prometheus registry | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs` |
| OTEL Export | PerfTrace OTEL collector | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs` |
| Scapy Probe | KAI Libs scapy probe | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py` |
| PCAP Capture (test) | KAI Libs pcap capture | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-009_pcap_capture.py` |
| dpkt PCAP parser | KAI Libs dpkt | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py` |

---

## 3. Data Flow Diagram

```
Network Interface
│
NetGuard: packet capture
├── AF_PACKET ring: zero-copy kernel capture
├── FlowAnalyzer: bytes, packets, duration per 5-tuple
│    └── FlowEvent on close/idle timeout
└── DPI: nDPI protocol classification
     └── L7 label per flow (HTTP, DNS, SMTP, ...)

NATS: kubric.{tenant}.network.flow.v1
  └── bytes, packets, duration_ms, protocol, src/dst
       └── ClickHouse: kubric.netflow_events (TTL: 90 days)

PerfTrace: host NIC stats
├── RX/TX bytes per interface
├── Packet drop counters
└── Prometheus: kubric_net_rx_bytes_total, kubric_net_tx_bytes_total

OTEL → Grafana (traces for flow latency)
Prometheus → Grafana (NIC dashboards)

Anomaly Detection:
├── NetGuard IDS: rate-based rules (flood detection)
├── KAI Foresight LSTM: flow volume baseline deviation
└── Alert → NATS: kubric.{tenant}.network.ids.alert.v1
```

---

## 4. NPM Metrics

| Metric | Source | Prometheus Metric Name |
|---|---|---|
| Host NIC RX bytes/s | PerfTrace sysinfo | `kubric_net_rx_bytes_total` |
| Host NIC TX bytes/s | PerfTrace sysinfo | `kubric_net_tx_bytes_total` |
| Packet drop rate | PerfTrace sysinfo | `kubric_net_rx_dropped_total` |
| Active flows | NetGuard FlowAnalyzer | `kubric_netguard_active_flows` |
| Packet capture rate | NetGuard AF_PACKET | `kubric_netguard_capture_rate_pps` |
| Protocol distribution | NetGuard DPI L7 | `kubric_netguard_protocol_flows_total{proto=...}` |
| Flow duration p99 | ClickHouse query | `SELECT quantile(0.99)(duration_ms) FROM kubric.netflow_events` |

---

## 5. Security / Performance Correlation

| Performance Anomaly | Security Interpretation | Kubric Response |
|---|---|---|
| Sudden throughput drop | DDoS / routing attack | IDS flood rules → TheHive case |
| TX spike to single external IP | Data exfiltration | NetGuard ipsum + flow alert |
| CPU spike + network activity | Cryptomining (T1496) | PerfTrace CPU + Sigma rule |
| Unusual protocol appearance | New C2 channel | DPI + IDS alert → Triage |
| Flow duration >> baseline | Slow exfil / C2 keep-alive | KAI Foresight LSTM deviation |

---

## 6. Integration Points

| System | NPM Role |
|---|---|
| **Prometheus** | Primary metrics scrape from PerfTrace |
| **Grafana** | Network topology and performance dashboards |
| **ClickHouse** | Flow telemetry retention (90 days) |
| **OTEL** | Distributed tracing for application network latency |
| **TheHive** | Cases for performance-linked security events |

---

## 7. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| NetGuard | Deployed on all hosts with promiscuous NIC or SPAN tap |
| libpcap | ≥ 1.10 for standard capture; AF_PACKET for high-perf |
| DPDK | Optional; required for ≥ 10 Gbps line rate capture |
| PerfTrace | Deployed on all hosts; scrape interval ≤ 15 s |
| Prometheus | Scrape targets include all PerfTrace instances on `:9090/metrics` |
| ClickHouse | `kubric.netflow_events` table with 90-day TTL |
