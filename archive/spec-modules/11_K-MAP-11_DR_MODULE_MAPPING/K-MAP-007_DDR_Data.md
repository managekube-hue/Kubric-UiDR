# K-MAP-007 — Data Detection & Response (DDR)
**Discipline:** Data Detection & Response
**Abbreviation:** DDR
**Kubric Reference:** K-MAP-007
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Data Detection and Response (DDR) focuses on protecting data assets from unauthorized access, exfiltration, encryption (ransomware), and modification. DDR monitors data stores, file systems, and data-in-transit to detect policy violations and data-centric attacks. Kubric implements DDR primarily through the CoreSec FIM subsystem (real-time file access monitoring with BLAKE3 integrity hashing), YARA scanning for ransomware indicators, ClickHouse-based data access audit trails, and the KAI Keeper remediation agent for automated data protection response.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| File Integrity Monitoring | CoreSec FIM inotify watcher | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| File Hash Baseline | CoreSec FIM BLAKE3 | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs` |
| Ransomware Detection | CoreSec YARA compiler + scanner | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs` |
| Ransomware Detection | CoreSec YARA malware scanner | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs` |
| Anomalous Data Access (ML) | CoreSec Candle inference | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs` |
| Memory Forensics | CoreSec memory snapshot | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs` |
| Data Exfil Detection (network) | NetGuard flow analyzer | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs` |
| Data Exfil Detection (TLS) | NetGuard TLS SNI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs` |
| Data Access Audit Store | ClickHouse connector | `03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py` |
| Data Protection Response | KAI Keeper remediation planner | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py` |
| Automated Response | KAI Keeper Cortex subprocess | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-002_cortex_subprocess.py` |
| Parquet Export / Data Pipeline | KAI Libs PyArrow | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-002_pyarrow_parquet.py` |
| Data Analytics | KAI Libs Polars | `03_K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-001_polars_dataframe.py` |

---

## 3. Data Flow Diagram

```
Data Assets
│
├── File System (CoreSec FIM)
│   ├── inotify: CREATE, MODIFY, DELETE events
│   ├── BLAKE3: hash baseline comparison
│   │    └── old_hash != new_hash → FIM event
│   ├── Sigma: sensitive path access patterns
│   ├── YARA: ransomware file signature scan
│   │    └── encrypted-extension patterns, note drop
│   └── NATS: kubric.{tenant}.endpoint.fim.v1
│                kubric.{tenant}.endpoint.malware.v1
│
├── Data-in-Transit (NetGuard)
│   ├── Flow: large outbound transfers (exfil indicator)
│   ├── TLS SNI: data to unexpected destinations
│   └── NATS: kubric.{tenant}.network.flow.v1
│
├── Data Stores (ClickHouse / PostgreSQL)
│   ├── Query audit via FastAPI access log
│   │    └── PostgreSQL: kubric.api_access_log
│   └── ClickHouse TTL policies enforce retention limits
│
└── KAI Response
    ├── Triage: ransomware → P1 case → TheHive
    ├── Keeper: isolation playbook (Cortex subprocess)
    └── Memory snapshot: capture before process termination
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Collection | T1005 | Data from Local System | FIM: mass file read on sensitive paths |
| Collection | T1039 | Data from Network Shared Drive | NetGuard: large SMB flow |
| Collection | T1074 | Data Staged | FIM: large temporary archive file creation |
| Exfiltration | T1041 | Exfiltration Over C2 | NetGuard: flow + ipsum lookup |
| Exfiltration | T1048 | Exfiltration Over Alternative Protocol | NetGuard DPI: FTP, DNS tunnel |
| Exfiltration | T1567 | Exfiltration to Cloud Storage | NetGuard: egress to cloud provider IP |
| Impact | T1486 | Data Encrypted for Impact (Ransomware) | YARA: ransomware signatures + FIM mass change |
| Impact | T1485 | Data Destruction | FIM: mass DELETE events; BLAKE3 all-zeros |
| Impact | T1565 | Data Manipulation | FIM: BLAKE3 hash change on DB files |
| Defense Evasion | T1070.004 | File Deletion (log wipe) | FIM: /var/log deletion events |
| Persistence | T1025 | Data from Removable Media | FIM: mount point activity |

---

## 5. Integration Points

| System | DDR Role |
|---|---|
| **TheHive** | Data incident cases (ransomware, exfiltration, unauthorized access) |
| **Cortex** | Automated isolation responders triggered by KAI Keeper |
| **ClickHouse** | FIM event archive; retroactive data-access forensics |
| **Vault** | Data encryption key management; KAI Keeper accesses for re-encryption |
| **MISP** | Ransomware family intelligence; extortion group TTPs |
| **Polars / PyArrow** | Large-scale FIM event analytics via data pipeline |
| **Grafana** | Data modification rate, suspicious delete volume dashboards |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| CoreSec FIM paths | Must include data directories: `/data`, `/var/lib`, application data roots |
| YARA ransomware rules | Available in `vendor/yara-rules/ransomware/` |
| BLAKE3 baseline | Run on initial deployment; re-baseline after planned changes |
| ClickHouse retention | FIM events: 13 months; set via TTL expression per table |
| Cortex responders | Isolation playbook installed and tested in Cortex admin |
| NetGuard flow timeout | 120 s idle for detecting slow exfil flows |
