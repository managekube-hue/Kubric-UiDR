# K-MAP-001 — Endpoint Detection & Response (EDR)
**Discipline:** Endpoint Detection & Response
**Abbreviation:** EDR
**Kubric Reference:** K-MAP-001
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Endpoint Detection and Response (EDR) continuously monitors endpoint devices — servers, workstations, and containers — to detect, investigate, and respond to cyber threats. EDR solutions provide deep telemetry across process execution, file system activity, registry modifications, network connections, and memory. Responses range from alerting and investigation assist to automated containment. Kubric implements EDR through the CoreSec agent (Rust), which uses eBPF hooks, BLAKE3-based FIM, Sigma rule evaluation, YARA malware scanning, and on-device ML inference, all with events published to NATS in OCSF-normalized JSON.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Process Monitoring (kernel) | CoreSec eBPF execve hook | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs` |
| File Open Monitoring (kernel) | CoreSec eBPF openat2 hook | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-002_openat2_hook.rs` |
| eBPF Map Pressure | CoreSec eBPF map pressure | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs` |
| File Integrity Monitoring | CoreSec FIM inotify | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| FIM Baseline Hashing | CoreSec FIM BLAKE3 | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs` |
| Rule-Based Detection | CoreSec Sigma evaluator | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |
| OCSF Event Normalization | CoreSec OCSF bridge | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs` |
| Malware Scanning | CoreSec YARA compiler | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs` |
| Malware Detection | CoreSec YARA scanner | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| ML Anomaly Inference | CoreSec Candle ML | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs` |
| ML Model Loading | CoreSec TinyLlama loader | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-002_tinyllama_loader.rs` |
| Memory Forensics | CoreSec memory snapshot | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs` |
| Emit Rate Control | CoreSec Governor token bucket | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs` |
| Agent Orchestration | CoreSec main entry point | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs` |
| Alert Triage | KAI Triage Agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| LLM Reasoning | KAI llama3 reasoner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-002_llama3_reasoning.py` |
| Forensic Hunting | KAI Hunter Velociraptor | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py` |

---

## 3. Data Flow Diagram

```
Endpoint (Linux Host)
│
├── Kernel Space
│   ├── eBPF execve hook  ──► process event ring buffer
│   └── eBPF openat2 hook ──► file-open event ring buffer
│
├── User Space (CoreSec Agent)
│   ├── FIM: inotify events
│   │     └── BLAKE3 hash comparison (old vs new)
│   ├── Process: sysinfo poll (5 s fallback)
│   ├── Sigma rule evaluator
│   │     └── Matches against ProcessEvent fields
│   ├── YARA-X compiler + scanner
│   │     └── Multi-rule malware detection
│   ├── ML: Candle inference + TinyLlama
│   │     └── Anomaly score per process
│   └── Governor (token-bucket 50 EPS / 200 burst)
│
├── NATS JetStream
│   ├── kubric.{tenant}.endpoint.process.v1  (OCSF 4007)
│   ├── kubric.{tenant}.endpoint.fim.v1       (OCSF 4010)
│   └── kubric.{tenant}.endpoint.malware.v1
│
└── KAI Triage Agent
    ├── llama3 reasoning
    ├── OCSF class analyzer
    ├── KISS priority score
    └── TheHive case creation
         ├── Cortex analyzer chain
         └── MISP IOC enrichment
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Execution | T1059 | Command and Scripting Interpreter | Sigma: cmdline pattern match |
| Execution | T1059.003 | Windows Command Shell | Sigma: cmd.exe parent-child |
| Execution | T1059.004 | Unix Shell | eBPF execve + Sigma |
| Execution | T1204 | User Execution | eBPF execve hook |
| Persistence | T1543 | Create or Modify System Process | Sigma: systemd service creation |
| Persistence | T1053 | Scheduled Task/Job | CoreSec process monitoring |
| Persistence | T1098 | Account Manipulation | FIM: /etc/passwd, /etc/shadow |
| Privilege Escalation | T1068 | Exploitation for Privilege Escalation | eBPF openat2 + ML anomaly |
| Defense Evasion | T1027 | Obfuscated Files or Information | YARA pattern matching |
| Defense Evasion | T1055 | Process Injection | ML anomaly + eBPF |
| Defense Evasion | T1070 | Indicator Removal | FIM: log file deletion |
| Collection | T1005 | Data from Local System | FIM: sensitive path access |
| Impact | T1486 | Data Encrypted for Impact | YARA: ransomware signatures |
| Impact | T1565 | Data Manipulation | FIM: integrity change alert |

---

## 5. Integration Points

| System | EDR Role |
|---|---|
| **TheHive** | Case creation for all Sigma/YARA confirmed detections |
| **Cortex** | Analyzer chain: VirusTotal, Cuckoo sandbox, file hash lookup |
| **MISP** | IOC enrichment; process hash correlation against known malware |
| **Wazuh** | Supplementary host agent; OSSEC rules complement Sigma |
| **Velociraptor** | Forensic artifact collection on P1 EDR incidents |
| **ClickHouse** | Long-term event store for process and FIM history |
| **Vault** | Runtime injection of `KUBRIC_SIGMA_DIR`, `KUBRIC_YARA_DIR`, `KUBRIC_MODEL_PATH` |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| OS | Linux x86_64 (eBPF); any OS (sysinfo fallback) |
| Kernel | ≥ 5.4 for eBPF; ≥ 3.10 for sysinfo |
| Capabilities | CAP_BPF, CAP_PERFMON (eBPF mode); none for sysinfo fallback |
| Environment Variables | `KUBRIC_TENANT_ID`, `KUBRIC_NATS_URL`, `KUBRIC_AGENT_ID` (required); `KUBRIC_FIM_PATHS`, `KUBRIC_SIGMA_DIR`, `KUBRIC_YARA_DIR`, `KUBRIC_MODEL_PATH` (optional with defaults) |
| NATS | JetStream-enabled NATS ≥ 2.10 at `KUBRIC_NATS_URL` |
| Disk | ≥ 500 MB for YARA rules + model weights |
| Memory | ≥ 256 MB resident; ML inference adds ~128 MB for model |
| Event Rate Limit | 50 EPS sustained, 200 EPS burst (configurable via Governor) |
