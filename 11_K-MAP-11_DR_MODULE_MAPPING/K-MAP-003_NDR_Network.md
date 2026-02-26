# K-MAP-003 — Network Detection & Response (NDR)
**Discipline:** Network Detection & Response
**Abbreviation:** NDR
**Kubric Reference:** K-MAP-003
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Network Detection and Response (NDR) analyzes network traffic to detect threats that endpoint-based controls may miss — including encrypted command-and-control channels, lateral movement between hosts, network reconnaissance, and protocol anomalies. NDR solutions use deep packet inspection, flow analysis, and behavioral baselines to identify malicious patterns without requiring host agents. Kubric's NDR capability is delivered by the NetGuard agent (Rust), which provides libpcap/AF_PACKET/DPDK packet capture, bidirectional flow aggregation, nDPI-based L7 classification, TLS SNI inspection, and a Suricata-compatible IDS rule engine.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Packet Capture | NetGuard main entry | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-001_main.rs` |
| Flow Analysis (5-tuple) | NetGuard FlowAnalyzer | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs` |
| TLS SNI Inspection | NetGuard TLS SNI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs` |
| AF_PACKET Ring Buffer | NetGuard AF_PACKET | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs` |
| DPDK High-Speed Capture | NetGuard DPDK bypass | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs` |
| Deep Packet Inspection | NetGuard nDPI FFI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs` |
| L7 Protocol Classification | NetGuard L7 classifier | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs` |
| IDS Rule Loading | NetGuard IDS rule loader | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs` |
| IDS Alert Publishing | NetGuard IDS alert publisher | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs` |
| Threat Intel IP Lookup | NetGuard ipsum lookup | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs` |
| Traffic Analysis | KAI Analyst Cortex chain | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py` |
| Observable Enrichment | KAI Analyst enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py` |
| PCAP Parsing | KAI Libs dpkt parser | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py` |
| Probe / Test Traffic | KAI Libs scapy probe | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py` |
| GeoIP Resolution | KAI Libs GeoIP2 | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py` |

---

## 3. Data Flow Diagram

```
Network Interface (promiscuous mode / SPAN port)
        |
NetGuard Agent (Rust)
   ├── Capture backend:
   │    ├── libpcap (default)
   │    ├── AF_PACKET ring buffer (high-perf)
   │    └── DPDK bypass (≥10 Gbps)
   │
   ├── Layer 3/4 processing:
   │    ├── FlowAnalyzer: bidirectional 5-tuple aggregation
   │    │    └── emit FlowEvent on TCP FIN/RST or idle timeout
   │    └── TLS SNI: extract server_name from ClientHello
   │
   ├── Layer 7 DPI:
   │    ├── nDPI FFI: protocol identification
   │    └── L7 classifier: application label (HTTP, DNS, SMTP, ...)
   │
   ├── IDS:
   │    ├── rule_loader: Suricata-compatible rules from vendor/
   │    └── alert_publisher: rule match → alert payload
   │
   └── Threat Intel:
        └── ipsum_lookup: src/dst IP against ipsum blocklist
              └── Hit → threat_score enrichment on FlowEvent

NATS JetStream
  ├── kubric.{tenant}.network.flow.v1      (FlowEvent, every close/timeout)
  ├── kubric.{tenant}.network.ids.alert.v1 (IDS matches)
  └── kubric.{tenant}.network.tls.anomaly.v1 (SNI anomalies)

KAI Analyst Agent
  ├── Cortex analyzer chain (AbuseIPDB, Shodan, PassiveDNS)
  ├── Observable enrichment (IP, domain, hash)
  └── GeoIP2 resolution
        └── TheHive case update
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Initial Access | T1190 | Exploit Public-Facing Application | IDS rules: web attack signatures |
| Initial Access | T1133 | External Remote Services | NetGuard: unexpected port scan detection |
| Command and Control | T1071 | Application Layer Protocol | DPI L7 classifier: unexpected protocol |
| Command and Control | T1071.001 | Web Protocols (C2 over HTTP) | DPI + TLS SNI: suspicious domains |
| Command and Control | T1071.004 | DNS (C2 via DNS tunneling) | DPI: DNS query size anomaly |
| Command and Control | T1090 | Proxy | Flow analysis: suspicious relay pattern |
| Command and Control | T1573 | Encrypted Channel | TLS SNI + ipsum lookup |
| Exfiltration | T1041 | Exfiltration Over C2 Channel | Flow: outbound volume spike |
| Exfiltration | T1048 | Exfiltration Over Alternative Protocol | DPI: non-standard protocol on egress |
| Discovery | T1046 | Network Service Scanning | IDS: port scan signatures |
| Lateral Movement | T1021 | Remote Services | Flow: internal east-west SSH/RDP |
| Lateral Movement | T1570 | Lateral Tool Transfer | NetGuard: large SMB/SCP flows |
| Impact | T1498 | Network Denial of Service | IDS: flood signatures; flow rate anomaly |
| Collection | T1557 | Adversary-in-the-Middle | TLS SNI mismatch; ARP anomaly (IDS) |

---

## 5. Integration Points

| System | NDR Role |
|---|---|
| **MISP** | IP/domain IOC feed; ipsum blocklist enrichment |
| **TheHive** | Cases created for IDS alert matches and TLS anomalies |
| **Cortex** | AbuseIPDB, Shodan, PassiveDNS analyzers on network observables |
| **ClickHouse** | Flow telemetry store; supports retroactive hunt queries |
| **Grafana** | Network topology dashboards; flow volume, protocol distribution |
| **GeoIP2** | Country and ASN resolution for flow enrichment |
| **Prometheus** | NetGuard capture rate, drop counter, active flow count |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| OS | Linux x86_64 |
| Interface | Promiscuous mode OR SPAN/TAP port for production |
| libpcap | ≥ 1.10; or kernel AF_PACKET support (≥ 3.3) |
| DPDK | Optional; requires NIC with DPDK PMD driver for ≥10 Gbps |
| CAP_NET_RAW | Required for libpcap / AF_PACKET capture |
| IDS rules | Suricata-compatible rules in `vendor/suricata-rules/` |
| ipsum blocklist | Updated via `scripts/vendor-pull.sh` |
| NATS | JetStream-enabled at `KUBRIC_NATS_URL` |
| nDPI | Shared library `libndpi.so` present on PATH |
