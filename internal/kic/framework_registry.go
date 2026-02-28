package kic

// ─── GRC Framework Registry ────────────────────────────────────────────────────
// 200 compliance, regulatory, and industry frameworks supported by Kubric GRC.
//
// This is the canonical framework list consumed by:
//   - Assessment intake validation (handler_assessment.go)
//   - CISO-Assistant framework listing (handler_ciso.go)
//   - Compliance posture dashboard (frontend)
//   - OSCAL ingestion pipeline (K-GRC-OSCAL)
//
// Aligned with the Kubric unified platform vision:
//   PSA (business/ITSM) + NOC (infrastructure ops) + SOC (security ops)
//   + GRC (governance/risk/compliance) + KAI (AI orchestration)
//
// Categories:
//   federal        — US federal government frameworks
//   international  — Non-US government / multinational
//   industry       — Sector-specific standards
//   privacy        — Data protection / privacy regulations
//   cloud          — Cloud security frameworks
//   infrastructure — Infrastructure & K8s hardening
//   risk           — Risk management frameworks
//   supply_chain   — Software supply chain security
//   iot_ics        — IoT / Industrial Control Systems
//   healthcare     — Healthcare-specific compliance
//   financial      — Financial services regulations
//   telecom        — Telecommunications standards
//   energy         — Energy / utilities sector
//   education      — Education sector compliance
//   ai_ml          — AI / ML governance
//   cyber_insurance — Cyber insurance frameworks
//   audit          — Audit & assurance standards

// Framework represents a compliance framework in the registry.
type Framework struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Version  string `json:"version"`
	OSCAL    bool   `json:"oscal"` // has OSCAL catalog available
}

// FrameworkRegistry is the full set of 200 GRC frameworks.
var FrameworkRegistry = []Framework{
	// ── US Federal (1–25) ─────────────────────────────────────────────────────
	{ID: "NIST-800-53", Name: "NIST SP 800-53 Rev 5", Category: "federal", Version: "Rev5", OSCAL: true},
	{ID: "NIST-800-171", Name: "NIST SP 800-171 Rev 3", Category: "federal", Version: "Rev3", OSCAL: true},
	{ID: "NIST-CSF-2.0", Name: "NIST Cybersecurity Framework 2.0", Category: "federal", Version: "2.0", OSCAL: true},
	{ID: "NIST-800-218", Name: "NIST SP 800-218 SSDF", Category: "federal", Version: "1.1", OSCAL: true},
	{ID: "NIST-800-82", Name: "NIST SP 800-82 Rev 3 (ICS)", Category: "federal", Version: "Rev3", OSCAL: true},
	{ID: "NIST-800-190", Name: "NIST SP 800-190 Container Security", Category: "federal", Version: "1.0", OSCAL: true},
	{ID: "NIST-800-63", Name: "NIST SP 800-63B Digital Identity", Category: "federal", Version: "Rev4", OSCAL: true},
	{ID: "NIST-AI-100-1", Name: "NIST AI Risk Management Framework", Category: "federal", Version: "1.0", OSCAL: true},
	{ID: "FedRAMP-High", Name: "FedRAMP High Baseline", Category: "federal", Version: "Rev5", OSCAL: true},
	{ID: "FedRAMP-Moderate", Name: "FedRAMP Moderate Baseline", Category: "federal", Version: "Rev5", OSCAL: true},
	{ID: "FedRAMP-Low", Name: "FedRAMP Low Baseline", Category: "federal", Version: "Rev5", OSCAL: true},
	{ID: "FedRAMP-LI-SaaS", Name: "FedRAMP LI-SaaS", Category: "federal", Version: "Rev5", OSCAL: true},
	{ID: "FISMA", Name: "Federal Information Security Modernization Act", Category: "federal", Version: "2014", OSCAL: true},
	{ID: "FIPS-140-3", Name: "FIPS 140-3 Cryptographic Modules", Category: "federal", Version: "2019", OSCAL: false},
	{ID: "FIPS-199", Name: "FIPS 199 Security Categorization", Category: "federal", Version: "2004", OSCAL: true},
	{ID: "CISA-CPG", Name: "CISA Cross-Sector CPGs", Category: "federal", Version: "1.0", OSCAL: false},
	{ID: "CISA-ZTA", Name: "CISA Zero Trust Maturity Model", Category: "federal", Version: "2.0", OSCAL: false},
	{ID: "DOD-CMMC-2.0", Name: "DoD CMMC 2.0", Category: "federal", Version: "2.0", OSCAL: true},
	{ID: "DOD-STIG", Name: "DoD Security Technical Implementation Guides", Category: "federal", Version: "2024", OSCAL: true},
	{ID: "DOD-SRG", Name: "DoD Cloud Computing SRG", Category: "federal", Version: "1.6", OSCAL: true},
	{ID: "ITAR", Name: "International Traffic in Arms Regulations", Category: "federal", Version: "2024", OSCAL: false},
	{ID: "EAR", Name: "Export Administration Regulations", Category: "federal", Version: "2024", OSCAL: false},
	{ID: "FAR-52.204-21", Name: "FAR Basic Safeguarding", Category: "federal", Version: "2024", OSCAL: false},
	{ID: "DFARS-252.204-7012", Name: "DFARS Cyber Incident Reporting", Category: "federal", Version: "2024", OSCAL: false},
	{ID: "StateRAMP", Name: "StateRAMP", Category: "federal", Version: "2024", OSCAL: true},

	// ── International Government (26–55) ─────────────────────────────────────
	{ID: "ISO-27001", Name: "ISO/IEC 27001:2022", Category: "international", Version: "2022", OSCAL: true},
	{ID: "ISO-27002", Name: "ISO/IEC 27002:2022", Category: "international", Version: "2022", OSCAL: true},
	{ID: "ISO-27017", Name: "ISO/IEC 27017 Cloud Security", Category: "international", Version: "2015", OSCAL: true},
	{ID: "ISO-27018", Name: "ISO/IEC 27018 Cloud PII Protection", Category: "international", Version: "2019", OSCAL: true},
	{ID: "ISO-27701", Name: "ISO/IEC 27701 Privacy Information Mgmt", Category: "international", Version: "2019", OSCAL: true},
	{ID: "ISO-27005", Name: "ISO/IEC 27005 Information Security Risk Mgmt", Category: "international", Version: "2022", OSCAL: true},
	{ID: "ISO-22301", Name: "ISO 22301 Business Continuity", Category: "international", Version: "2019", OSCAL: true},
	{ID: "ISO-31000", Name: "ISO 31000 Risk Management", Category: "international", Version: "2018", OSCAL: true},
	{ID: "ISO-42001", Name: "ISO/IEC 42001 AI Management System", Category: "international", Version: "2023", OSCAL: true},
	{ID: "ISO-27799", Name: "ISO 27799 Health Informatics", Category: "international", Version: "2016", OSCAL: false},
	{ID: "UK-NCSC-CAF", Name: "UK NCSC Cyber Assessment Framework", Category: "international", Version: "3.2", OSCAL: false},
	{ID: "UK-Cyber-Essentials", Name: "UK Cyber Essentials Plus", Category: "international", Version: "2024", OSCAL: false},
	{ID: "EU-NIS2", Name: "EU NIS2 Directive", Category: "international", Version: "2022", OSCAL: false},
	{ID: "EU-DORA", Name: "EU Digital Operational Resilience Act", Category: "international", Version: "2025", OSCAL: false},
	{ID: "EU-CRA", Name: "EU Cyber Resilience Act", Category: "international", Version: "2024", OSCAL: false},
	{ID: "EU-AI-Act", Name: "EU Artificial Intelligence Act", Category: "international", Version: "2024", OSCAL: false},
	{ID: "BSI-IT-Grundschutz", Name: "BSI IT-Grundschutz (Germany)", Category: "international", Version: "2023", OSCAL: false},
	{ID: "ANSSI-PGSSI-S", Name: "ANSSI PGSSI-S (France)", Category: "international", Version: "2024", OSCAL: false},
	{ID: "CSA-STAR", Name: "Cloud Security Alliance STAR", Category: "international", Version: "4.0", OSCAL: false},
	{ID: "CSA-CCM", Name: "Cloud Security Alliance CCM v4", Category: "international", Version: "4.0", OSCAL: true},
	{ID: "ENISA-EUCS", Name: "ENISA EU Cloud Certification Scheme", Category: "international", Version: "2024", OSCAL: false},
	{ID: "IRAP", Name: "Australia IRAP / ISM", Category: "international", Version: "2024", OSCAL: false},
	{ID: "ACSC-Essential-8", Name: "Australia ACSC Essential Eight", Category: "international", Version: "2024", OSCAL: false},
	{ID: "PDPA-Singapore", Name: "Singapore PDPA", Category: "international", Version: "2021", OSCAL: false},
	{ID: "PIPL-China", Name: "China Personal Information Protection Law", Category: "international", Version: "2021", OSCAL: false},
	{ID: "APRA-CPS-234", Name: "APRA CPS 234 Information Security (Australia)", Category: "international", Version: "2019", OSCAL: false},
	{ID: "MAS-TRM", Name: "MAS Technology Risk Management (Singapore)", Category: "international", Version: "2021", OSCAL: false},
	{ID: "OSFI-B-13", Name: "OSFI B-13 Technology Risk (Canada)", Category: "international", Version: "2024", OSCAL: false},
	{ID: "PIPEDA", Name: "Canada PIPEDA", Category: "international", Version: "2024", OSCAL: false},
	{ID: "LGPD-Brazil", Name: "Brazil Lei Geral de Proteção de Dados", Category: "international", Version: "2020", OSCAL: false},

	// ── Industry Standards (56–95) ───────────────────────────────────────────
	{ID: "SOC2", Name: "SOC 2 Type II (Trust Services Criteria)", Category: "industry", Version: "2017", OSCAL: true},
	{ID: "SOC1", Name: "SOC 1 Type II (ISAE 3402 / SSAE 18)", Category: "industry", Version: "2017", OSCAL: false},
	{ID: "SOC3", Name: "SOC 3 (General Use Report)", Category: "industry", Version: "2017", OSCAL: false},
	{ID: "SOC-CSP", Name: "SOC for Cybersecurity", Category: "industry", Version: "2017", OSCAL: false},
	{ID: "PCI-DSS-4.0", Name: "PCI DSS v4.0", Category: "industry", Version: "4.0", OSCAL: true},
	{ID: "PCI-PIN", Name: "PCI PIN Security", Category: "industry", Version: "3.1", OSCAL: false},
	{ID: "PCI-P2PE", Name: "PCI Point-to-Point Encryption", Category: "industry", Version: "3.1", OSCAL: false},
	{ID: "PCI-3DS", Name: "PCI 3-D Secure", Category: "industry", Version: "2.3", OSCAL: false},
	{ID: "PCI-SSF", Name: "PCI Software Security Framework", Category: "industry", Version: "1.2", OSCAL: false},
	{ID: "PCI-TSP", Name: "PCI Token Service Provider", Category: "industry", Version: "2.0", OSCAL: false},
	{ID: "SWIFT-CSCF", Name: "SWIFT Customer Security Controls Framework", Category: "industry", Version: "2024", OSCAL: false},
	{ID: "COBIT-2019", Name: "COBIT 2019", Category: "industry", Version: "2019", OSCAL: false},
	{ID: "ITIL-4", Name: "ITIL 4", Category: "industry", Version: "4.0", OSCAL: false},
	{ID: "TOGAF", Name: "TOGAF 10", Category: "industry", Version: "10", OSCAL: false},
	{ID: "SABSA", Name: "SABSA Security Architecture", Category: "industry", Version: "2024", OSCAL: false},
	{ID: "OWASP-ASVS", Name: "OWASP Application Security Verification Standard", Category: "industry", Version: "4.0", OSCAL: false},
	{ID: "OWASP-SAMM", Name: "OWASP Software Assurance Maturity Model", Category: "industry", Version: "2.0", OSCAL: false},
	{ID: "OWASP-Top-10", Name: "OWASP Top 10", Category: "industry", Version: "2021", OSCAL: false},
	{ID: "OWASP-MASVS", Name: "OWASP Mobile Application Security", Category: "industry", Version: "2.1", OSCAL: false},
	{ID: "OWASP-API-Top10", Name: "OWASP API Security Top 10", Category: "industry", Version: "2023", OSCAL: false},
	{ID: "BSIMM", Name: "Building Security In Maturity Model", Category: "industry", Version: "14", OSCAL: false},
	{ID: "CIS-Controls-v8", Name: "CIS Controls v8.1", Category: "industry", Version: "8.1", OSCAL: true},
	{ID: "CIS-K8s-1.8", Name: "CIS Kubernetes Benchmark v1.8", Category: "infrastructure", Version: "1.8", OSCAL: true},
	{ID: "CIS-Docker", Name: "CIS Docker Benchmark", Category: "infrastructure", Version: "1.6", OSCAL: true},
	{ID: "CIS-Linux", Name: "CIS Linux Benchmarks (RHEL/Ubuntu/Debian)", Category: "infrastructure", Version: "2024", OSCAL: true},
	{ID: "CIS-Windows", Name: "CIS Windows Server Benchmark", Category: "infrastructure", Version: "2024", OSCAL: true},
	{ID: "CIS-AWS", Name: "CIS Amazon Web Services Foundations Benchmark", Category: "infrastructure", Version: "3.0", OSCAL: true},
	{ID: "CIS-Azure", Name: "CIS Microsoft Azure Foundations Benchmark", Category: "infrastructure", Version: "2.1", OSCAL: true},
	{ID: "CIS-GCP", Name: "CIS Google Cloud Platform Benchmark", Category: "infrastructure", Version: "2.0", OSCAL: true},
	{ID: "CIS-Oracle-Cloud", Name: "CIS Oracle Cloud Infrastructure Benchmark", Category: "infrastructure", Version: "2.0", OSCAL: true},
	{ID: "CIS-M365", Name: "CIS Microsoft 365 Benchmark", Category: "infrastructure", Version: "3.1", OSCAL: true},
	{ID: "CIS-PostgreSQL", Name: "CIS PostgreSQL Benchmark", Category: "infrastructure", Version: "1.0", OSCAL: true},
	{ID: "CIS-Apache", Name: "CIS Apache HTTP Server Benchmark", Category: "infrastructure", Version: "2.5", OSCAL: true},
	{ID: "CIS-NGINX", Name: "CIS NGINX Benchmark", Category: "infrastructure", Version: "2.1", OSCAL: true},
	{ID: "CIS-MongoDB", Name: "CIS MongoDB Benchmark", Category: "infrastructure", Version: "1.0", OSCAL: true},
	{ID: "CIS-Redis", Name: "CIS Redis Benchmark", Category: "infrastructure", Version: "1.0", OSCAL: true},
	{ID: "SANS-Top-20", Name: "SANS/CIS Critical Security Controls", Category: "industry", Version: "8.0", OSCAL: false},
	{ID: "MITRE-ATT&CK", Name: "MITRE ATT&CK Enterprise", Category: "industry", Version: "15.1", OSCAL: false},
	{ID: "MITRE-D3FEND", Name: "MITRE D3FEND", Category: "industry", Version: "1.0", OSCAL: false},
	{ID: "MITRE-ATLAS", Name: "MITRE ATLAS (AI Threat Landscape)", Category: "industry", Version: "4.0", OSCAL: false},

	// ── Privacy & Data Protection (96–125) ──────────────────────────────────
	{ID: "GDPR", Name: "EU General Data Protection Regulation", Category: "privacy", Version: "2018", OSCAL: false},
	{ID: "CCPA-CPRA", Name: "California Consumer Privacy Act / CPRA", Category: "privacy", Version: "2023", OSCAL: false},
	{ID: "HIPAA", Name: "HIPAA Security Rule", Category: "healthcare", Version: "2013", OSCAL: true},
	{ID: "HIPAA-Privacy", Name: "HIPAA Privacy Rule", Category: "healthcare", Version: "2013", OSCAL: false},
	{ID: "HITRUST-CSF", Name: "HITRUST CSF v11", Category: "healthcare", Version: "11.3", OSCAL: false},
	{ID: "HITRUST-i1", Name: "HITRUST i1 Assessment", Category: "healthcare", Version: "2024", OSCAL: false},
	{ID: "HITRUST-e1", Name: "HITRUST e1 Assessment", Category: "healthcare", Version: "2024", OSCAL: false},
	{ID: "FDA-21-CFR-11", Name: "FDA 21 CFR Part 11 (Electronic Records)", Category: "healthcare", Version: "2024", OSCAL: false},
	{ID: "HICP", Name: "HHS Health Industry Cybersecurity Practices", Category: "healthcare", Version: "2023", OSCAL: false},
	{ID: "NYDFS-500", Name: "NYDFS 23 NYCRR 500 Cybersecurity", Category: "financial", Version: "2023", OSCAL: false},
	{ID: "GLBA", Name: "Gramm-Leach-Bliley Act Safeguards Rule", Category: "financial", Version: "2023", OSCAL: false},
	{ID: "SOX-IT", Name: "Sarbanes-Oxley IT General Controls", Category: "financial", Version: "2002", OSCAL: false},
	{ID: "FFIEC-CAT", Name: "FFIEC Cybersecurity Assessment Tool", Category: "financial", Version: "2024", OSCAL: false},
	{ID: "FFIEC-BSA-AML", Name: "FFIEC BSA/AML Examination Manual", Category: "financial", Version: "2024", OSCAL: false},
	{ID: "SEC-Cyber-Rule", Name: "SEC Cybersecurity Disclosure Rules", Category: "financial", Version: "2023", OSCAL: false},
	{ID: "Basel-III-Ops", Name: "Basel III Operational Risk", Category: "financial", Version: "2023", OSCAL: false},
	{ID: "FERPA", Name: "FERPA Student Privacy", Category: "education", Version: "2024", OSCAL: false},
	{ID: "COPPA", Name: "COPPA Children's Online Privacy", Category: "privacy", Version: "2024", OSCAL: false},
	{ID: "POPIA-ZA", Name: "South Africa POPIA", Category: "privacy", Version: "2021", OSCAL: false},
	{ID: "PDPB-India", Name: "India Digital Personal Data Protection Act", Category: "privacy", Version: "2023", OSCAL: false},
	{ID: "APEC-CBPR", Name: "APEC Cross-Border Privacy Rules", Category: "privacy", Version: "2024", OSCAL: false},
	{ID: "PA-DSS", Name: "PA-DSS Payment Application", Category: "industry", Version: "3.2", OSCAL: false},
	{ID: "TISAX", Name: "TISAX Trusted Information Security (Automotive)", Category: "industry", Version: "6.0", OSCAL: false},
	{ID: "VDMA", Name: "VDMA Industrial Security (Manufacturing)", Category: "industry", Version: "2024", OSCAL: false},
	{ID: "CPS-003-AU", Name: "APRA CPS 003 Outsourcing (Australia)", Category: "financial", Version: "2024", OSCAL: false},
	{ID: "Colorado-Privacy", Name: "Colorado Privacy Act", Category: "privacy", Version: "2024", OSCAL: false},
	{ID: "Virginia-CDPA", Name: "Virginia Consumer Data Protection Act", Category: "privacy", Version: "2023", OSCAL: false},
	{ID: "Connecticut-CTDPA", Name: "Connecticut Data Privacy Act", Category: "privacy", Version: "2023", OSCAL: false},
	{ID: "Utah-UCPA", Name: "Utah Consumer Privacy Act", Category: "privacy", Version: "2023", OSCAL: false},
	{ID: "Texas-TDPSA", Name: "Texas Data Privacy and Security Act", Category: "privacy", Version: "2024", OSCAL: false},

	// ── Cloud & Infrastructure (126–155) ─────────────────────────────────────
	{ID: "AWS-Well-Architected", Name: "AWS Well-Architected Security Pillar", Category: "cloud", Version: "2024", OSCAL: false},
	{ID: "Azure-Security-Benchmark", Name: "Azure Security Benchmark v3", Category: "cloud", Version: "3.0", OSCAL: false},
	{ID: "GCP-Security-Foundations", Name: "GCP Enterprise Security Blueprint", Category: "cloud", Version: "2024", OSCAL: false},
	{ID: "NIST-800-144", Name: "NIST SP 800-144 Cloud Computing Guidelines", Category: "cloud", Version: "2011", OSCAL: false},
	{ID: "NIST-800-145", Name: "NIST SP 800-145 Cloud Definition", Category: "cloud", Version: "2011", OSCAL: false},
	{ID: "K8s-Pod-Security", Name: "Kubernetes Pod Security Standards", Category: "infrastructure", Version: "1.31", OSCAL: false},
	{ID: "K8s-RBAC-Best", Name: "Kubernetes RBAC Best Practices", Category: "infrastructure", Version: "1.31", OSCAL: false},
	{ID: "K8s-Network-Policy", Name: "Kubernetes Network Policy Baseline", Category: "infrastructure", Version: "1.31", OSCAL: false},
	{ID: "OPA-Gatekeeper", Name: "OPA Gatekeeper Policy Library", Category: "infrastructure", Version: "3.16", OSCAL: false},
	{ID: "Kyverno-Best", Name: "Kyverno Best Practices Policy Set", Category: "infrastructure", Version: "1.12", OSCAL: false},
	{ID: "Falco-Rules", Name: "Falco Default + Kubric Custom Rules", Category: "infrastructure", Version: "0.38", OSCAL: false},
	{ID: "Istio-Security", Name: "Istio Service Mesh Security Policy", Category: "infrastructure", Version: "1.22", OSCAL: false},
	{ID: "Cilium-Network", Name: "Cilium Network Policy Baseline", Category: "infrastructure", Version: "1.16", OSCAL: false},
	{ID: "Vault-Security", Name: "HashiCorp Vault Security Hardening", Category: "infrastructure", Version: "1.17", OSCAL: false},
	{ID: "Terraform-Security", Name: "Terraform Security Best Practices", Category: "infrastructure", Version: "1.9", OSCAL: false},
	{ID: "Ansible-Hardening", Name: "Ansible STIG Hardening Roles", Category: "infrastructure", Version: "2024", OSCAL: false},
	{ID: "Docker-Content-Trust", Name: "Docker Content Trust / Notary v2", Category: "infrastructure", Version: "2024", OSCAL: false},
	{ID: "OCI-Image-Spec", Name: "OCI Image Security Spec", Category: "infrastructure", Version: "1.1", OSCAL: false},
	{ID: "Sigstore-Cosign", Name: "Sigstore/Cosign Image Signing", Category: "supply_chain", Version: "2.4", OSCAL: false},
	{ID: "SLSA-L3", Name: "SLSA (Supply-chain Levels for Software Artifacts) L3", Category: "supply_chain", Version: "1.0", OSCAL: false},
	{ID: "SSDF-NIST", Name: "NIST SSDF (Secure Software Development)", Category: "supply_chain", Version: "1.1", OSCAL: true},
	{ID: "OpenSSF-Scorecard", Name: "OpenSSF Scorecard", Category: "supply_chain", Version: "5.0", OSCAL: false},
	{ID: "CycloneDX-SBOM", Name: "CycloneDX SBOM Standard", Category: "supply_chain", Version: "1.6", OSCAL: false},
	{ID: "SPDX-SBOM", Name: "SPDX SBOM Standard", Category: "supply_chain", Version: "2.3", OSCAL: false},
	{ID: "in-toto", Name: "in-toto Supply Chain Layout", Category: "supply_chain", Version: "1.0", OSCAL: false},
	{ID: "SCVS", Name: "OWASP Software Component Verification Standard", Category: "supply_chain", Version: "1.0", OSCAL: false},

	// ── IoT / ICS / OT (156–175) ────────────────────────────────────────────
	{ID: "IEC-62443", Name: "IEC 62443 Industrial Cybersecurity", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "NERC-CIP", Name: "NERC CIP (Critical Infrastructure Protection)", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "NERC-CIP-013", Name: "NERC CIP-013 Supply Chain Risk Mgmt", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "TSA-Pipeline", Name: "TSA Pipeline Cybersecurity Directive", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "NIST-IR-8259", Name: "NIST IR 8259 IoT Device Cybersecurity", Category: "iot_ics", Version: "2020", OSCAL: false},
	{ID: "NIST-800-183", Name: "NIST SP 800-183 Networks of Things", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "ETSI-EN-303645", Name: "ETSI EN 303 645 Consumer IoT Security", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "UL-2900", Name: "UL 2900 Software Cybersecurity for IoT", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "ISA-99", Name: "ISA/IEC 62443 (ISA-99) Industrial Automation", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "CISA-ICS-CERT", Name: "CISA ICS-CERT Advisories Compliance", Category: "iot_ics", Version: "2024", OSCAL: false},
	{ID: "IEC-61850", Name: "IEC 61850 Power Utility Communications Security", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "IEEE-1686", Name: "IEEE 1686 Substation IED Security", Category: "energy", Version: "2019", OSCAL: false},
	{ID: "API-1164", Name: "API 1164 Pipeline SCADA Security", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "NIST-1800-10", Name: "NIST SP 1800-10 Financial Services IT Security", Category: "financial", Version: "2024", OSCAL: false},
	{ID: "ITU-X.805", Name: "ITU-T X.805 Telecom Security Architecture", Category: "telecom", Version: "2024", OSCAL: false},
	{ID: "3GPP-SCAS", Name: "3GPP SCAS (Security Assurance Specification)", Category: "telecom", Version: "2024", OSCAL: false},
	{ID: "GSMA-NESAS", Name: "GSMA NESAS Network Equipment Security", Category: "telecom", Version: "2024", OSCAL: false},
	{ID: "IEEE-2030.5", Name: "IEEE 2030.5 Smart Energy Security", Category: "energy", Version: "2024", OSCAL: false},
	{ID: "NISTIR-8183", Name: "NISTIR 8183 CSF Manufacturing Profile", Category: "iot_ics", Version: "2017", OSCAL: false},
	{ID: "C2M2", Name: "Cyber Capability Maturity Model (C2M2)", Category: "energy", Version: "2.1", OSCAL: false},

	// ── Risk, Governance & Audit (176–195) ──────────────────────────────────
	{ID: "FAIR", Name: "Factor Analysis of Information Risk", Category: "risk", Version: "2024", OSCAL: false},
	{ID: "COSO-ERM", Name: "COSO Enterprise Risk Management", Category: "risk", Version: "2017", OSCAL: false},
	{ID: "OCTAVE-Allegro", Name: "OCTAVE Allegro Risk Assessment", Category: "risk", Version: "2024", OSCAL: false},
	{ID: "TARA", Name: "Threat Agent Risk Assessment", Category: "risk", Version: "2024", OSCAL: false},
	{ID: "NIST-800-30", Name: "NIST SP 800-30 Risk Assessment Guide", Category: "risk", Version: "Rev1", OSCAL: true},
	{ID: "NIST-800-37", Name: "NIST SP 800-37 Risk Management Framework", Category: "risk", Version: "Rev2", OSCAL: true},
	{ID: "NIST-800-39", Name: "NIST SP 800-39 Managing Information Security Risk", Category: "risk", Version: "2011", OSCAL: true},
	{ID: "ISACA-RISK-IT", Name: "ISACA Risk IT Framework", Category: "risk", Version: "2024", OSCAL: false},
	{ID: "ISA-5500", Name: "ISO/IEC 27005 + ISMS Risk Treatment", Category: "risk", Version: "2022", OSCAL: false},
	{ID: "NIST-Privacy-FW", Name: "NIST Privacy Framework 1.0", Category: "privacy", Version: "1.0", OSCAL: true},
	{ID: "AICPA-TSC", Name: "AICPA Trust Services Criteria (SOC 2 Source)", Category: "audit", Version: "2017", OSCAL: false},
	{ID: "SSAE-18", Name: "SSAE 18 (SOC Reporting Standard)", Category: "audit", Version: "2017", OSCAL: false},
	{ID: "ISAE-3402", Name: "ISAE 3402 (International SOC Equivalent)", Category: "audit", Version: "2024", OSCAL: false},
	{ID: "ISA-3000", Name: "ISAE 3000 Non-Financial Assurance", Category: "audit", Version: "2024", OSCAL: false},
	{ID: "PCAOB-AS-2201", Name: "PCAOB AS 2201 IT Audit Standard", Category: "audit", Version: "2024", OSCAL: false},
	{ID: "IIA-IPPF", Name: "IIA International Professional Practices Framework", Category: "audit", Version: "2024", OSCAL: false},
	{ID: "GDPR-Art-32", Name: "GDPR Article 32 Technical Measures", Category: "privacy", Version: "2018", OSCAL: false},
	{ID: "GDPR-Art-35", Name: "GDPR Article 35 DPIA Requirements", Category: "privacy", Version: "2018", OSCAL: false},
	{ID: "SCF", Name: "Secure Controls Framework", Category: "industry", Version: "2024", OSCAL: true},
	{ID: "NIST-CSF-1.1", Name: "NIST Cybersecurity Framework 1.1 (Legacy)", Category: "federal", Version: "1.1", OSCAL: true},

	// ── AI / ML Governance (196–204) ────────────────────────────────────────
	{ID: "NIST-AI-600-1", Name: "NIST AI 600-1 GenAI Profile", Category: "ai_ml", Version: "1.0", OSCAL: false},
	{ID: "ISO-23894", Name: "ISO/IEC 23894 AI Risk Management", Category: "ai_ml", Version: "2023", OSCAL: false},
	{ID: "IEEE-7000", Name: "IEEE 7000 Ethically Aligned Design", Category: "ai_ml", Version: "2021", OSCAL: false},
	{ID: "OECD-AI-Principles", Name: "OECD AI Principles", Category: "ai_ml", Version: "2024", OSCAL: false},
	{ID: "Singapore-AI-Gov", Name: "Singapore Model AI Governance Framework", Category: "ai_ml", Version: "2024", OSCAL: false},
	{ID: "ISO-27035", Name: "ISO/IEC 27035 Incident Management", Category: "international", Version: "2023", OSCAL: false},
	{ID: "ISO-27032", Name: "ISO/IEC 27032 Cybersecurity Guidelines", Category: "international", Version: "2023", OSCAL: false},
	{ID: "NIST-800-61", Name: "NIST SP 800-61 Computer Security Incident Handling", Category: "federal", Version: "Rev3", OSCAL: true},
	{ID: "CYBER-INSURANCE-COALITION", Name: "Coalition Cyber Insurance Controls", Category: "cyber_insurance", Version: "2024", OSCAL: false},
}

// frameworkIndex is a lookup map built at init time.
var frameworkIndex map[string]Framework

func init() {
	frameworkIndex = make(map[string]Framework, len(FrameworkRegistry))
	for _, fw := range FrameworkRegistry {
		frameworkIndex[fw.ID] = fw
	}
}

// LookupFramework returns the Framework by ID, or false if not found.
func LookupFramework(id string) (Framework, bool) {
	fw, ok := frameworkIndex[id]
	return fw, ok
}

// IsValidFramework returns true if the framework ID is in the registry.
func IsValidFramework(id string) bool {
	_, ok := frameworkIndex[id]
	return ok
}

// FrameworksByCategory returns all frameworks matching the given category.
func FrameworksByCategory(category string) []Framework {
	var result []Framework
	for _, fw := range FrameworkRegistry {
		if fw.Category == category {
			result = append(result, fw)
		}
	}
	return result
}

// FrameworkCategories returns the distinct category list.
func FrameworkCategories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, fw := range FrameworkRegistry {
		if !seen[fw.Category] {
			seen[fw.Category] = true
			cats = append(cats, fw.Category)
		}
	}
	return cats
}

// FrameworkCount returns the total number of registered frameworks.
func FrameworkCount() int { return len(FrameworkRegistry) }
