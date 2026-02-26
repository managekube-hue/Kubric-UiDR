# K-MAP-020 — License & Compliance Management (LCM)
**Discipline:** License & Compliance Management
**Abbreviation:** LCM
**Kubric Reference:** K-MAP-020
**Last Updated:** 2026-02-26

---

## 1. Discipline Definition

License and Compliance Management (LCM) ensures that all software used within an organization is properly licensed, that open-source dependencies comply with their copyright obligations, and that the organization's own software products are protected from unauthorized use. LCM also encompasses the broader compliance posture: ensuring that all regulatory, contractual, and policy obligations are demonstrated, tracked, and evidenced. Kubric's LCM capability covers open-source license compliance through the Go module system and Cargo workspace (all dependencies audited), SBOM (Software Bill of Materials) generation, and the full regulatory compliance evidence chain (SOC 2, ISO 27001) backed by ClickHouse and the KAI Bill audit agent.

---

## 2. Kubric Modules

| LCM Sub-Capability | Module | File Path |
|---|---|---|
| Go Dependency Management | go.mod / go.sum | `go.mod`, `go.sum` |
| Go Vendor / SBOM | tools.go + vendor pull | `tools.go`, `scripts/vendor-pull.sh` |
| Rust Dependency Audit | Cargo.toml per agent | `agents/*/Cargo.toml` |
| KAI Python Dependencies | pyproject.toml | `kai/pyproject.toml` |
| Vendor Asset Pull | vendor-pull.sh | `scripts/vendor-pull.sh` |
| Compliance Evidence Store | KAI Bill ClickHouse audit | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py` |
| SOC2 / ISO Crosswalk | K-ITIL-AUD-002 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs` |
| Control Evidence Map | K-ITIL-AUD-001 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map.md` |
| ITIL Compliance Mapping | K-ITIL-MAT-001 to 009 | `10_K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/` |
| Compliance Reporting | KAI Bill invoice renderer | `03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py` |

---

## 3. Open-Source License Inventory

| Component | Language | Primary License | License Risk |
|---|---|---|---|
| CoreSec agent | Rust | MIT / Apache 2.0 | Low |
| NetGuard agent | Rust | MIT / Apache 2.0 | Low |
| PerfTrace agent | Rust | MIT / Apache 2.0 | Low |
| Watchdog agent | Rust | MIT / Apache 2.0 | Low |
| YARA-X | Rust | MIT | Low |
| nDPI | C | LGPL v3 | Medium (dynamic link required) |
| libpcap | C | BSD-3-Clause | Low |
| DPDK (optional) | C | BSD | Low |
| KAI API | Python | MIT / Apache 2.0 | Low |
| FastAPI | Python | MIT | Low |
| CrewAI | Python | MIT | Low |
| asyncpg | Python | Apache 2.0 | Low |
| ClickHouse Connect | Python | Apache 2.0 | Low |
| aiokafka | Python | Apache 2.0 | Low |
| Polars | Python | MIT | Low |
| PyArrow | Python | Apache 2.0 | Low |
| pyFAIR | Python | MIT | Low |
| Sigma rule set | YAML | DRL / MIT | Low (rules, not code) |
| YARA rules (vendor) | YARA | Various (CC BY, MIT) | Low (review per rule set) |
| Nuclei templates (vendor) | YAML | MIT | Low |
| Suricata rules (vendor) | Rules | MIT / Emerging-GPLv2 | Medium (ET rules: check license) |
| ipsum blocklist | Text | CC BY 4.0 | Low (attribution required) |

---

## 4. SBOM Generation Workflow

```
Per-Release SBOM Generation
│
├── Rust agents (syft or cargo-sbom):
│   cargo sbom --output-format spdx-json > sbom/coresec.spdx.json
│   cargo sbom --output-format spdx-json > sbom/netguard.spdx.json
│   cargo sbom --output-format spdx-json > sbom/perftrace.spdx.json
│   cargo sbom --output-format spdx-json > sbom/watchdog.spdx.json
│
├── Go tools (syft):
│   syft go.mod -o spdx-json > sbom/go-tools.spdx.json
│
├── Python KAI stack (pip-licenses or cyclonedx-bom):
│   cyclonedx-py poetry -o sbom/kai.cyclonedx.json
│
└── Vendor assets:
    scripts/vendor-pull.sh documents all pinned versions
    → sbom/vendor-versions.txt

SBOMs stored in git under sbom/ and uploaded to
artifact registry on each release.
```

---

## 5. Regulatory Compliance Obligations

| Regulation / Standard | Obligation | Kubric Evidence |
|---|---|---|
| SOC 2 Type II | Annual audit with 12-month evidence period | ClickHouse event tables (13-month TTL) |
| ISO/IEC 27001:2022 | Annual surveillance audit; triennial recertification | K-ITIL-AUD-001, K-ITIL-AUD-002 |
| GDPR / UK GDPR | Data subject rights; lawful processing; DPA | ClickHouse TTL (data minimization); Vault (encryption at rest) |
| HIPAA (if applicable) | PHI protection; audit trail; BAA | ClickHouse event audit; Vault encryption |
| PCI DSS (if Stripe scope) | Cardholder data protection; log retention | Authentik + Vault + ClickHouse |
| NIS2 (EU critical infrastructure) | Incident reporting ≤ 24h; risk management | TheHive incident timeline; KAI Risk SSVC |

---

## 6. Compliance Evidence Lifecycle

```
Evidence Creation (continuous)
  K-XRO agents → NATS → ClickHouse
  (OCSF-normalized events, 13-month TTL)

Evidence Accumulation
  Vault audit log → durable storage
  Authentik events → ClickHouse
  ArgoCD sync history → Git (indefinite)
  TheHive cases → database (5 years)

Evidence Packaging (annual audit)
  KAI Bill: ClickHouse audit query
  → Compliance report (PDF via invoice renderer)
  → Auditor export package

Evidence Disposal
  ClickHouse: TTL expressions enforce retention limits
  (prevents over-retention creating liability)
```

---

## 7. License Compliance KPIs

| KPI | Target | Source |
|---|---|---|
| Components with known license | 100% | SBOM per release |
| GPL/AGPL dependencies (copyleft risk) | 0 in proprietary components | Cargo audit + SBOM review |
| nDPI: dynamic-link compliance | Yes (LGPL satisfied) | Build system verification |
| Emerging Threats rule license review | Annual | Vendor pull documentation |
| SBOM generation per release | 100% of releases | CI/CD pipeline |
| Obsolete dependency (CVE in dep) | 0 with EPSS > 0.1 | cargo audit + pip-audit |

---

## 8. Integration Points

| System | LCM Role |
|---|---|
| **ClickHouse** | Compliance evidence store (13-month retention) |
| **Git** | Immutable change history; SBOM artifact storage |
| **KAI Bill** | Compliance report generation and delivery |
| **Vault** | Encryption key management (GDPR encryption obligation) |
| **TheHive** | Incident records for regulatory reporting |
| **Grafana** | Compliance posture dashboard; license risk heat map |
| **CI/CD pipeline** | SBOM generation on every release |

---

## 9. Deployment Prerequisites

| Requirement | Detail |
|---|---|
| cargo-audit | Run `cargo audit` on each Rust agent on every PR |
| pip-audit | Run `pip-audit` on KAI Python deps on every PR |
| syft / cyclonedx-bom | SBOM generation tools installed in CI |
| ClickHouse TTL | Verify TTL expressions match retention obligations |
| Legal review | Annual review of vendor rule set licenses (ET rules, YARA rules) |
| SBOM storage | SBOMs committed to git under `sbom/` and pinned to releases |
