# K-GRC-FW-000 — Framework Registry Index

> **200 compliance, regulatory, and industry frameworks** imported into the Kubric GRC module.

This index is the source of truth for framework coverage. The programmatic registry
lives in [`internal/kic/framework_registry.go`](../../internal/kic/framework_registry.go).

## Platform Vision

Kubric unifies **PSA + NOC + SOC + GRC + AI Orchestration** into a single platform:

| Pillar | Module | Purpose |
|--------|--------|---------|
| **PSA** | K-PSA-06 | Professional Services Automation — billing, ITSM, customer portal |
| **NOC** | K-NOC-05 | Network Operations Center — infrastructure health, agent management |
| **SOC** | K-SOC-04 | Security Operations Center — EDR, NDR, ITDR, threat intel, forensics |
| **GRC** | K-GRC-07 | Governance, Risk & Compliance — 200 frameworks, OSCAL, evidence vault |
| **KAI** | K-KAI-03 | AI Orchestration — CISO-Assistant, ML pipelines, RAG, model tiering |

## Framework Coverage by Category

| # | Category | Count | Examples |
|---|----------|-------|----------|
| 1 | Federal (US) | 25 | NIST 800-53, FedRAMP, CMMC, FISMA, DOD STIG |
| 2 | International | 30 | ISO 27001, ISO 42001, NIS2, DORA, BSI, IRAP |
| 3 | Industry Standards | 40 | SOC 2, CIS Controls, OWASP, COBIT, MITRE ATT&CK |
| 4 | Privacy & Data Protection | 20 | GDPR, CCPA, LGPD, PDPA, POPIA, state privacy laws |
| 5 | Healthcare | 7 | HIPAA, HITRUST CSF, FDA 21 CFR, HICP |
| 6 | Financial Services | 10 | PCI DSS, NYDFS 500, GLBA, SOX, FFIEC, SEC |
| 7 | Cloud & Infrastructure | 36 | AWS/Azure/GCP benchmarks, CIS K8s/Docker/Linux, OPA, Kyverno |
| 8 | Supply Chain | 8 | SLSA, SBOM (CycloneDX/SPDX), Sigstore, OpenSSF |
| 9 | IoT / ICS / OT | 10 | IEC 62443, NIST IR 8259, ETSI EN 303 645 |
| 10 | Energy & Utilities | 8 | NERC CIP, C2M2, IEC 61850, API 1164 |
| 11 | Telecom | 3 | ITU X.805, 3GPP SCAS, GSMA NESAS |
| 12 | Education | 1 | FERPA |
| 13 | Risk Management | 9 | FAIR, COSO ERM, OCTAVE, NIST 800-30/37/39 |
| 14 | Audit & Assurance | 6 | AICPA TSC, SSAE 18, ISAE 3402, PCAOB |
| 15 | AI / ML Governance | 5 | NIST AI RMF, EU AI Act, ISO 42001, IEEE 7000 |
| **Total** | | **200** | |

## OSCAL Support

Frameworks with `oscal: true` have machine-readable OSCAL catalogs available for:
- Automated control ingestion via `K-GRC-OSCAL-001_nist_ingest.py`
- Cross-framework mapping via `K-GRC-OSCAL-002_soc2_mapper.py`
- Compliance-as-code via `K-GRC-OSCAL-004_compliance_trestle.py`

**OSCAL-enabled frameworks:** ~40 out of 200 (federal + major industry standards).

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ciso/frameworks` | List all 200 frameworks |
| GET | `/ciso/frameworks?category=federal` | Filter by category |
| GET | `/ciso/posture` | Tenant compliance posture across all frameworks |
| POST | `/ciso/ask` | AI-grounded compliance Q&A |
| POST | `/assessments` | Record assessment result for any framework |
| GET | `/assessments?framework=NIST-800-53` | List assessments by framework |

## Docker

The KIC service image (`kubric/kic`) includes the full 200-framework registry at compile time.
No runtime data download required — frameworks are embedded in the Go binary.

```bash
docker build -t kubric/kic --target kic -f Dockerfile.api .
# or via docker-compose:
docker compose --profile app up kic
```

## Evidence Chain

Assessment results flow through:
1. **Intake** → `POST /assessments` (any of 200 frameworks)
2. **Storage** → PostgreSQL `kic_assessments` table (RLS per tenant)
3. **Evidence** → BLAKE3-signed audit trail (`K-GRC-EV-002_blake3_signer.go`)
4. **Events** → NATS `kubric.grc.drift.v1.<tenant>` + `kubric.grc.ciso.v1.<tenant>`
5. **Dashboard** → Customer portal `/compliance` page
6. **AI** → CISO-Assistant RAG answers grounded in posture data
