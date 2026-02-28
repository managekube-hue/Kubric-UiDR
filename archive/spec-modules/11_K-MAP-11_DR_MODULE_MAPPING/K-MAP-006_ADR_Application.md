# K-MAP-006 — Application Detection & Response (ADR)
**Discipline:** Application Detection & Response
**Abbreviation:** ADR
**Kubric Reference:** K-MAP-006
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Application Detection and Response (ADR) monitors applications in production for runtime anomalies, injection attacks, API abuse, and business logic violations. Unlike SAST or DAST tools that operate pre-production, ADR is continuous and runtime-focused, correlating application telemetry (API calls, error rates, latency) with security signals. Kubric implements ADR through the NetGuard agent's L7 DPI capabilities (nDPI + L7 classifier), CoreSec's process and FIM monitoring of application files, the PerfTrace agent's application performance telemetry, and the KAI Analyst and Triage agents for enriched investigation.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| L7 Protocol Identification | NetGuard nDPI FFI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs` |
| Application Protocol Classification | NetGuard L7 classifier | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs` |
| HTTP Attack Detection | NetGuard IDS rules (web signatures) | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs` |
| App Process Monitoring | CoreSec eBPF execve hook | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs` |
| App File Integrity | CoreSec FIM inotify | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| App Performance Telemetry | PerfTrace host metrics | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs` |
| App Latency / OTEL | PerfTrace OTEL exporter | `02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs` |
| Alert Triage | KAI Triage agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| App Observable Enrichment | KAI Analyst observable enrichment | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py` |
| Real-Time API Monitoring | FastAPI + Socket.IO | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py` |
| API Access Logging | FastAPI + asyncpg | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py` |
| Prompt Injection Guardrail | KAI Guardrails prompt injection | `03_K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-004_prompt_injection.py` |

---

## 3. Data Flow Diagram

```
Application Layer
│
├── HTTP/S Traffic (NetGuard DPI)
│   ├── nDPI: identify HTTP, REST, GraphQL, gRPC
│   ├── L7 classifier: application label assignment
│   ├── IDS rules: SQLi, XSS, LFI, RCE pattern match
│   └── NATS: kubric.{tenant}.network.ids.alert.v1
│
├── Application Process (CoreSec)
│   ├── eBPF execve: unexpected child processes (web shell indicator)
│   ├── FIM: application binary changes, config file modifications
│   └── NATS: endpoint.process.v1, endpoint.fim.v1
│
├── Application Performance (PerfTrace)
│   ├── CPU / memory per top processes
│   ├── Custom OTEL spans from application instrumentation
│   └── Prometheus: :9090/metrics
│
├── Kubric API Self-Monitoring
│   ├── FastAPI: per-request latency, error rate, auth failures
│   ├── Socket.IO: real-time event push to frontend
│   └── Guardrails: prompt injection detection on AI inputs
│
└── KAI Analysis
    ├── Triage: web attack alert → TheHive case
    ├── Analyst: IP/domain enrichment
    └── Risk: CVE in application dependencies (EPSS scoring)
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Initial Access | T1190 | Exploit Public-Facing Application | IDS: SQLi, XSS, RCE, LFI patterns |
| Execution | T1059 | Command and Scripting Interpreter | eBPF: web server spawning shell |
| Execution | T1203 | Exploitation for Client Execution | IDS rules: deserialization attacks |
| Persistence | T1505.003 | Server-Side Backdoor (web shell) | eBPF + FIM: PHP/JSP file creation + exec |
| Defense Evasion | T1036 | Masquerading | eBPF: renamed binary execution |
| Credential Access | T1212 | Exploitation for Credential Access | IDS: auth bypass patterns |
| Discovery | T1083 | File and Directory Discovery | FIM: mass read on app config paths |
| Collection | T1005 | Data from Local System | FIM: sensitive file access |
| Impact | T1495 | Firmware Corruption | FIM: app binary replacement |

---

## 5. Integration Points

| System | ADR Role |
|---|---|
| **TheHive** | Application attack cases; web shell incidents |
| **Cortex** | URL scan, IP reputation on application attack sources |
| **MISP** | Web-targeting threat actor intelligence |
| **OpenTelemetry** | Application trace correlation with security events |
| **Grafana** | Application error rate, latency, and security event overlay |
| **Prometheus** | Application performance metrics from PerfTrace |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| NetGuard | DPI enabled; IDS rule set includes OWASP Top 10 signatures |
| CoreSec FIM paths | Must include application root directories and config paths |
| PerfTrace | OTEL endpoint configured for application trace correlation |
| Application instrumentation | Optional; OpenTelemetry SDK recommended for trace correlation |
| IDS rules | Web application rules from `vendor/suricata-rules/http/` |
| Guardrails | AI input validation enabled for all KAI agent endpoints |
