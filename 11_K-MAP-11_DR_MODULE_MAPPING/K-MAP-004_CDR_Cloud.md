# K-MAP-004 — Cloud Detection & Response (CDR)
**Discipline:** Cloud Detection & Response
**Abbreviation:** CDR
**Kubric Reference:** K-MAP-004
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Cloud Detection and Response (CDR) identifies and responds to security threats within cloud infrastructure — encompassing cloud misconfigurations, IAM policy abuse, abnormal API activity, workload compromise, and cloud-native lateral movement. CDR requires visibility into cloud control plane events, workload runtime behavior, and cloud service configurations. Kubric's CDR capability leverages Kubernetes-native deployment via ArgoCD, runtime monitoring of containerized workloads through CoreSec and NetGuard agents, Vault-enforced secret hygiene, and the KAI Risk agent for cloud posture scoring through pyFAIR and SSVC.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Container Runtime Monitoring | CoreSec eBPF (execve, openat2) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/` |
| Container FIM | CoreSec FIM inotify | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs` |
| Cloud Workload Network | NetGuard flow analysis | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs` |
| Cloud IAM Enforcement | Authentik OIDC + JWT | `services/k-svc/` |
| Secret Management | HashiCorp Vault + ESO | KAI Keeper `K-KAI-KP-003_vault_secret_fetcher.py` |
| GitOps Deployment State | ArgoCD | `services/k-svc/` manifests |
| Cloud Posture Scoring | KAI Risk pyFAIR | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py` |
| CVE Risk in Cloud Workloads | KAI Risk EPSS scorer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py` |
| Deploy Security Gate | KAI Deploy + criticality check | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py` |
| Cloud Incident Triage | KAI Triage agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| Kubernetes Threat Detection | CoreSec Sigma (k8s patterns) | `02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs` |

---

## 3. Data Flow Diagram

```
Kubernetes Cluster
│
├── Nodes (CoreSec agent per node)
│   ├── eBPF: container process events
│   ├── FIM: ConfigMap mounts, /etc, /var
│   ├── Sigma: kubectl exec, privilege escalation
│   └── NATS: kubric.{tenant}.endpoint.process.v1
│
├── Network (NetGuard agent per node / CNI tap)
│   ├── Pod-to-pod flow analysis
│   ├── Egress traffic inspection
│   └── NATS: kubric.{tenant}.network.flow.v1
│
├── Control Plane
│   ├── Authentik: OIDC events → ClickHouse
│   └── ArgoCD: manifest drift detection → alert
│
├── Secrets Plane
│   └── Vault audit log → ClickHouse
│
└── KAI Orchestration
    ├── Triage: container escape, privileged pod detection
    ├── Risk: pyFAIR cloud risk score
    ├── Keeper: Vault secret remediation
    └── Deploy: GitOps change enforcement
```

---

## 4. MITRE ATT&CK Coverage

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Initial Access | T1190 | Exploit Public-Facing App | IDS + NetGuard flow |
| Execution | T1609 | Container Administration Command | eBPF: kubectl exec detection |
| Execution | T1610 | Deploy Container | Sigma: unexpected container deployment |
| Persistence | T1525 | Implant Internal Image | ArgoCD drift + FIM |
| Privilege Escalation | T1611 | Escape to Host | eBPF execve: host PID namespace |
| Defense Evasion | T1578 | Modify Cloud Infrastructure | ArgoCD drift detection |
| Credential Access | T1552.007 | Container API (secret from env) | Vault audit + FIM |
| Discovery | T1613 | Container and Resource Discovery | Sigma: kubectl get, describe |
| Lateral Movement | T1021 | Remote Services (pod-to-pod) | NetGuard flow east-west |
| Exfiltration | T1537 | Transfer Data to Cloud Account | NetGuard flow: egress volume |
| Impact | T1496 | Resource Hijacking (cryptomining) | PerfTrace CPU spike + Sigma |

---

## 5. Integration Points

| System | CDR Role |
|---|---|
| **ArgoCD** | GitOps enforcement; drift from desired state is a CDR signal |
| **Vault** | Secret lifecycle; unauthorized access is a CDR signal |
| **Authentik** | Cloud IAM; unusual permission grants or policy changes |
| **cert-manager** | Certificate anomalies (unexpected issuance) as CDR signals |
| **TheHive** | Cloud incident case management |
| **MISP** | Cloud-targeting threat group intelligence |
| **Grafana** | Cloud workload health dashboards; resource anomaly charts |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Kubernetes | ≥ 1.27 with ArgoCD installed |
| CoreSec | Deployed as DaemonSet on all Kubernetes nodes |
| NetGuard | Deployed per node or via CNI mirror/tap |
| Vault | Kubernetes auth method enabled; ESO syncing secrets |
| Authentik | OIDC provider configured for all platform services |
| ArgoCD | Applications tracking production manifests |
