# K-MAP-015 — Mobile Device Management (MDM)
**Discipline:** Mobile Device Management
**Abbreviation:** MDM
**Kubric Reference:** K-MAP-015
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

Mobile Device Management (MDM) enforces security policies on smartphones, tablets, and other mobile endpoints used to access organizational resources. MDM capabilities include device enrollment, remote wipe, app management, compliance enforcement, and conditional access. While Kubric does not include a native MDM agent (mobile OS kernels do not support eBPF in the same way as Linux servers), it integrates with mobile device posture through its identity layer (Authentik conditional access policies), credential health monitoring (HIBP), device compliance signals fed into the Sentinel health score, and network-side monitoring of mobile device traffic via NetGuard.

---

## 2. Kubric Modules

| Sub-Capability | Module | File Path |
|---|---|---|
| Mobile Identity / Conditional Access | Authentik OIDC device policies | `services/k-svc/` |
| Mobile Credential Breach Detection | KAI Sentinel HIBP | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py` |
| Mobile Health Score | KAI Sentinel health score | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py` |
| Mobile Network Traffic | NetGuard flow analyzer + DPI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs` |
| Mobile TLS Anomaly | NetGuard TLS SNI | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs` |
| Mobile C2 Detection | NetGuard IP threat intel | `02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs` |
| Mobile Risk Scoring | KAI Risk SSVC + EPSS | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py` |
| Mobile Incident Triage | KAI Triage agent | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py` |
| Mobile Churn / Anomaly | KAI Sentinel churn risk | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py` |

---

## 3. Integration Architecture

```
Mobile Devices
│
├── Authentication → Authentik OIDC
│   ├── Device compliance policy (certificate or device enrollment check)
│   ├── Conditional access: block non-compliant devices
│   └── Event log → ClickHouse: kubric.auth_events
│
├── Network Traffic (via corporate Wi-Fi / VPN)
│   └── NetGuard (deployed at network ingress):
│        ├── Flow analysis: mobile device IP flows
│        ├── TLS SNI: mobile app destination inspection
│        ├── ipsum lookup: mobile C2 detection
│        └── NATS: kubric.{tenant}.network.flow.v1
│
├── HIBP Credential Check (user accounts on mobile)
│   └── KAI Sentinel → NATS: kubric.{tenant}.sla.health.v1
│
└── 3rd-Party MDM Integration (Microsoft Intune / Mosyle / Jamf)
    └── Compliance posture feed → Authentik device trust
         └── Non-compliant → block access to Kubric-protected services
```

---

## 4. MITRE ATT&CK Coverage (Mobile)

| Tactic | Technique ID | Technique Name | Kubric Detection |
|---|---|---|---|
| Initial Access | T1476 | Deliver Malicious App via App Store | NetGuard: C2 traffic from device |
| Command and Control | T1481 | Web Service (mobile C2) | NetGuard TLS SNI + ipsum |
| Credential Access | T1417 | Input Capture (keylogger) | HIBP credential monitoring |
| Exfiltration | T1532 | Archive Collected Data | NetGuard: large upload from mobile IP |
| Defense Evasion | T1406 | Obfuscated Files or Information | NetGuard DPI: obfuscated protocol |

---

## 5. Integration Points

| System | MDM Role |
|---|---|
| **Authentik** | Conditional access based on device compliance posture |
| **HIBP API** | Credential breach check for mobile user accounts |
| **NetGuard** | Network-side mobile traffic monitoring |
| **External MDM** | Microsoft Intune / Jamf / Mosyle feed device compliance to Authentik |
| **TheHive** | Mobile security incident cases |
| **ClickHouse** | Mobile authentication audit trail |

---

## 6. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| Authentik | Device policies configured; support for OIDC device_authorization_grant |
| External MDM | Optional but recommended: device compliance signals push to Authentik |
| NetGuard | Deployed at Wi-Fi/VPN ingress to capture mobile traffic |
| HIBP API key | Available in Vault |
| ClickHouse | `kubric.auth_events` with device_id and compliance_status fields |

---

## 7. MDM Gap and Roadmap

Kubric does not provide a native MDM agent for iOS or Android. The current implementation relies on:
- Network-side visibility (NetGuard)
- Identity-layer enforcement (Authentik)
- Credential health monitoring (HIBP)

A future native mobile agent integration is roadmapped for the K-XRO platform using the Android eBPF capabilities available in Android 12+ and iOS network extension APIs.
