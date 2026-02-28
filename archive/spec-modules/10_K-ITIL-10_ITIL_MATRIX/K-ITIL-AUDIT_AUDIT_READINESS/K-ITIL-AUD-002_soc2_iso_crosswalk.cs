// K-ITIL-AUD-002 — SOC 2 / ISO 27001 Crosswalk
// Document Code : K-ITIL-AUD-002
// Practice      : Audit Readiness
// Kubric Ref    : K-ITIL-AUDIT_AUDIT_READINESS
// Last Updated  : 2026-02-26
//
// Purpose
// ───────
// This C# module provides a structured crosswalk between:
//   • SOC 2 Type II Trust Services Criteria (2017)
//   • ISO/IEC 27001:2022 Annex A Controls
//   • Kubric Infrastructure Controls (KIC)
//   • ITIL 4 Practices
//
// Usage
// ─────
// The CrosswalkRegistry can be consumed by audit tooling, GRC dashboards,
// or exported to JSON via CrosswalkRegistry.ToJson() for integration with
// external compliance platforms.

using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Linq;

namespace Kubric.Audit
{
    // ─────────────────────────────────────────────────────────────────────────
    // Domain models
    // ─────────────────────────────────────────────────────────────────────────

    public record KubricControl(
        string KicId,
        string Description,
        string KubricModule,
        string EvidenceLocation,
        string RetentionPeriod
    );

    public record Soc2Criterion(
        string TscCategory,
        string CriterionId,
        string Description
    );

    public record Iso27001Control(
        string Clause,
        string ControlId,
        string ControlTitle
    );

    public record Itil4Practice(
        string PracticeCode,
        string PracticeName,
        string KubricDocRef
    );

    public record CrosswalkEntry(
        KubricControl KubricControl,
        List<Soc2Criterion> Soc2Criteria,
        List<Iso27001Control> Iso27001Controls,
        List<Itil4Practice> Itil4Practices,
        string Notes
    );

    // ─────────────────────────────────────────────────────────────────────────
    // Crosswalk Registry
    // ─────────────────────────────────────────────────────────────────────────

    public static class CrosswalkRegistry
    {
        public static readonly IReadOnlyList<CrosswalkEntry> Entries = BuildEntries();

        private static List<CrosswalkEntry> BuildEntries() => new()
        {
            // ── KIC-001: Asset Inventory ──────────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-001",
                    Description: "All managed hosts inventoried via Watchdog heartbeats",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs",
                    EvidenceLocation: "ClickHouse: kubric.agent_status_history",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC6", "CC6.1",
                        "The entity implements logical access security software, infrastructure, and architectures")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.1", "Inventory of assets"),
                    new Iso27001Control("A.8", "A.8.9", "Configuration management")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("TMP2", "Infrastructure and Platform Management",
                        "K-ITIL-MAT-007_TMP2_Infrastructure.md"),
                    new Itil4Practice("SMP7", "Service Level Management",
                        "K-ITIL-MAT-009_SMP7_ServiceLevel.md")
                },
                Notes: "Watchdog publishes agent.status.v1 every 15 s. Fleet inventory query: " +
                       "SELECT agent_id, agent_type, MAX(timestamp) FROM kubric.agent_status_history GROUP BY agent_id, agent_type"
            ),

            // ── KIC-002: File Integrity Monitoring ────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-002",
                    Description: "Critical file paths monitored with inotify + BLAKE3 baseline",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/",
                    EvidenceLocation: "NATS: kubric.{tenant}.endpoint.fim.v1; ClickHouse: kubric.fim_events",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.2",
                        "The entity monitors system components and the operation of those controls"),
                    new Soc2Criterion("CC7", "CC7.3",
                        "The entity evaluates security events to determine whether they could impair service commitments")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.8",  "Management of technical vulnerabilities"),
                    new Iso27001Control("A.8", "A.8.15", "Logging"),
                    new Iso27001Control("A.8", "A.8.16", "Monitoring activities")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("SMP1", "Incident Management",
                        "K-ITIL-MAT-004_SMP1_Incident.md")
                },
                Notes: "OCSF class_uid=4010 (File Activity). Evidence query: " +
                       "SELECT path, old_hash, new_hash, severity, timestamp " +
                       "FROM kubric.fim_events WHERE timestamp BETWEEN :start AND :end ORDER BY timestamp"
            ),

            // ── KIC-003: Process Execution Monitoring ─────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-003",
                    Description: "All new process executions captured as OCSF-4007 events",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs",
                    EvidenceLocation: "NATS: kubric.{tenant}.endpoint.process.v1; ClickHouse: kubric.process_events",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.2", "System monitoring"),
                    new Soc2Criterion("CC7", "CC7.4",
                        "The entity responds to identified security incidents")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.15", "Logging"),
                    new Iso27001Control("A.8", "A.8.16", "Monitoring activities"),
                    new Iso27001Control("A.8", "A.8.7",  "Protection against malware")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("SMP1", "Incident Management",
                        "K-ITIL-MAT-004_SMP1_Incident.md"),
                    new Itil4Practice("SMP3", "Problem Management",
                        "K-ITIL-MAT-008_SMP3_Problem.md")
                },
                Notes: "eBPF hook (execve) on Linux; sysinfo poll (5 s) on all platforms. " +
                       "OCSF fields: tenant_id, agent_id, class_uid=4007, pid, ppid, executable, cmdline, blake3_hash"
            ),

            // ── KIC-004: Network Traffic Analysis ─────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-004",
                    Description: "Network flows captured; IDS and TLS anomaly detection active",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/",
                    EvidenceLocation: "NATS: kubric.{tenant}.network.flow.v1, network.ids.alert.v1",
                    RetentionPeriod: "Flows: 90 days; Alerts: 13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.2", "System monitoring"),
                    new Soc2Criterion("CC6", "CC6.6",
                        "The entity implements controls to prevent or detect unauthorized access attempts")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.20", "Networks security"),
                    new Iso27001Control("A.8", "A.8.21", "Security of network services"),
                    new Iso27001Control("A.8", "A.8.16", "Monitoring activities")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("TMP2", "Infrastructure and Platform Management",
                        "K-ITIL-MAT-007_TMP2_Infrastructure.md")
                },
                Notes: "Bidirectional 5-tuple flow analysis. TLS SNI inspected by K-XRO-NG-PCAP-002. " +
                       "IDS rules loaded from vendor/suricata-rules by K-XRO-NG-IDS-001."
            ),

            // ── KIC-005: Malware Detection ─────────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-005",
                    Description: "YARA rules + ML candle inference for malware and anomaly detection",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/",
                    EvidenceLocation: "NATS: kubric.{tenant}.endpoint.malware.v1; ClickHouse: kubric.malware_events",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.1", "Vulnerability identification and monitoring"),
                    new Soc2Criterion("CC7", "CC7.2", "System monitoring")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.7",  "Protection against malware"),
                    new Iso27001Control("A.8", "A.8.8",  "Management of technical vulnerabilities")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("GMP5", "Risk Management",
                        "K-ITIL-MAT-002_GMP5_Risk.md")
                },
                Notes: "YARA-X compiler with multi-rule bundles. ML inference via Candle framework " +
                       "(TinyLlama anomaly model). YARA rule source: vendor/yara-rules."
            ),

            // ── KIC-006: Access Control ────────────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-006",
                    Description: "JWT/RBAC enforced on all API routes via Authentik OIDC",
                    KubricModule: "services/k-svc/ (JWT middleware); Authentik IDP",
                    EvidenceLocation: "Authentik event log; PostgreSQL: kubric.api_access_log",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC6", "CC6.1",
                        "Logical access security implemented"),
                    new Soc2Criterion("CC6", "CC6.2",
                        "Authentication of users and systems"),
                    new Soc2Criterion("CC6", "CC6.3",
                        "Authorization of access to information assets")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.9",  "A.9.1",  "Access control policy"),
                    new Iso27001Control("A.9",  "A.9.2",  "User access management"),
                    new Iso27001Control("A.9",  "A.9.4",  "System and application access control"),
                    new Iso27001Control("A.8",  "A.8.2",  "Privileged access rights")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("GMP1", "Strategy Management",
                        "K-ITIL-MAT-001_GMP1_Strategy.md")
                },
                Notes: "Authentik configured as OIDC provider. JWT validated per-request in FastAPI middleware. " +
                       "Role assignments: admin, analyst, viewer, integration."
            ),

            // ── KIC-007: Secret Management ─────────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-007",
                    Description: "All secrets stored in Vault; injected at runtime via ESO",
                    KubricModule: "03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py",
                    EvidenceLocation: "Vault audit log (syslog); Kubernetes ESO sync events",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC6", "CC6.1", "Logical access security implemented"),
                    new Soc2Criterion("CC6", "CC6.7",
                        "The entity restricts the transmission, movement, and removal of information to authorized parties")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8",  "A.8.24", "Use of cryptography"),
                    new Iso27001Control("A.9",  "A.9.4",  "System and application access control"),
                    new Iso27001Control("A.8",  "A.8.10", "Information deletion")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP6", "Information Security Management",
                        "K-ITIL-MAT-003_GMP6_InfoSec.md"),
                    new Itil4Practice("SMP12", "Deployment Management",
                        "K-ITIL-MAT-006_SMP12_Deployment.md")
                },
                Notes: "Vault KV v2 for static secrets; Vault PKI for certificate issuance. " +
                       "ESO syncs Vault secrets to Kubernetes Secrets with automatic rotation on TTL expiry."
            ),

            // ── KIC-008: Change Control ────────────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-008",
                    Description: "All production changes via Git PR review + ArgoCD GitOps",
                    KubricModule: "03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/",
                    EvidenceLocation: "Git history (immutable); ClickHouse: kubric.change_audit",
                    RetentionPeriod: "Indefinite (Git); 13 months (ClickHouse)"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC8", "CC8.1",
                        "The entity authorizes, designs, develops, acquires, implements, operates, approves, " +
                        "maintains, and enhances software, infrastructure, and procedures")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8",  "A.8.32", "Change management"),
                    new Iso27001Control("A.8",  "A.8.25", "Secure development lifecycle"),
                    new Iso27001Control("A.14", "A.8.29", "Security testing in development and acceptance")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("SMP10", "Change Enablement",
                        "K-ITIL-MAT-005_SMP10_Change.md"),
                    new Itil4Practice("SMP12", "Deployment Management",
                        "K-ITIL-MAT-006_SMP12_Deployment.md")
                },
                Notes: "ArgoCD enforces desired state from Git. Criticality 5 guardrail (K-KAI-GD-003) " +
                       "blocks auto-deploy to production-critical workloads without human approval."
            ),

            // ── KIC-009: Vulnerability Management ─────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-009",
                    Description: "CVEs enriched with EPSS, triaged via SSVC, tracked to remediation",
                    KubricModule: "03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/",
                    EvidenceLocation: "ClickHouse: kubric.vuln_findings; KAI Risk SSVC records",
                    RetentionPeriod: "13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.1", "Vulnerability identification and monitoring")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8", "A.8.8",  "Management of technical vulnerabilities"),
                    new Iso27001Control("A.8", "A.8.19", "Installation of software on operational systems")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("GMP5", "Risk Management",
                        "K-ITIL-MAT-002_GMP5_Risk.md"),
                    new Itil4Practice("SMP3", "Problem Management",
                        "K-ITIL-MAT-008_SMP3_Problem.md")
                },
                Notes: "EPSS v3 API polled daily. SSVC decision outputs: Immediate, Out-of-Cycle, " +
                       "Scheduled, Defer. KAI Keeper creates TheHive tasks for Immediate findings."
            ),

            // ── KIC-010: Incident Management ──────────────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-010",
                    Description: "Security incidents detected, triaged, and tracked in TheHive",
                    KubricModule: "03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/",
                    EvidenceLocation: "TheHive case records; ClickHouse: kubric.incident_events",
                    RetentionPeriod: "5 years (TheHive); 13 months (ClickHouse)"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("CC7", "CC7.3", "Security event evaluation"),
                    new Soc2Criterion("CC7", "CC7.4", "Security incident response"),
                    new Soc2Criterion("CC7", "CC7.5",
                        "The entity identifies, develops, and implements activities to recover from identified security incidents")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.5",  "A.5.24", "Information security incident management planning and preparation"),
                    new Iso27001Control("A.5",  "A.5.25", "Assessment and decision on information security events"),
                    new Iso27001Control("A.5",  "A.5.26", "Response to information security incidents"),
                    new Iso27001Control("A.5",  "A.5.27", "Learning from information security incidents")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("SMP1", "Incident Management",
                        "K-ITIL-MAT-004_SMP1_Incident.md"),
                    new Itil4Practice("SMP3", "Problem Management",
                        "K-ITIL-MAT-008_SMP3_Problem.md")
                },
                Notes: "KAI Triage auto-creates TheHive cases with OCSF-normalized evidence. " +
                       "P1 incidents trigger VAPI phone and Twilio SMS escalation."
            ),

            // ── KIC-011: Availability and Performance ─────────────────────────
            new CrosswalkEntry(
                KubricControl: new KubricControl(
                    KicId: "KIC-011",
                    Description: "Fleet availability SLOs measured continuously by PerfTrace and Watchdog",
                    KubricModule: "02_K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/ + K-XRO-WD_WATCHDOG/",
                    EvidenceLocation: "Prometheus TSDB (30 days); ClickHouse: kubric.agent_status_history",
                    RetentionPeriod: "Prometheus: 30 days; ClickHouse: 13 months"
                ),
                Soc2Criteria: new()
                {
                    new Soc2Criterion("A1", "A1.1",
                        "The entity maintains, monitors, and evaluates current processing capacity and use of system components"),
                    new Soc2Criterion("A1", "A1.2",
                        "The entity authorizes, designs, develops, implements, operates, approves, maintains, and enhances " +
                        "environmental controls"),
                    new Soc2Criterion("A1", "A1.3",
                        "The entity tests recovery plan procedures supporting system recovery commitments")
                },
                Iso27001Controls: new()
                {
                    new Iso27001Control("A.8",  "A.8.6",  "Capacity management"),
                    new Iso27001Control("A.8",  "A.8.14", "Redundancy of information processing facilities"),
                    new Iso27001Control("A.17", "A.8.13", "Information backup")
                },
                Itil4Practices: new()
                {
                    new Itil4Practice("TMP2", "Infrastructure and Platform Management",
                        "K-ITIL-MAT-007_TMP2_Infrastructure.md"),
                    new Itil4Practice("SMP7", "Service Level Management",
                        "K-ITIL-MAT-009_SMP7_ServiceLevel.md")
                },
                Notes: "PerfTrace collects CPU, memory, disk IO, NIC stats and Linux perf_event counters. " +
                       "Watchdog publishes agent health every 15 s. SLO: ≥99.5% fleet availability per month."
            )
        };

        // ─────────────────────────────────────────────────────────────────────
        // Query helpers
        // ─────────────────────────────────────────────────────────────────────

        /// <summary>Returns all KIC entries that satisfy a given SOC 2 criterion.</summary>
        public static IEnumerable<CrosswalkEntry> BySoc2Criterion(string criterionId) =>
            Entries.Where(e => e.Soc2Criteria.Any(c => c.CriterionId == criterionId));

        /// <summary>Returns all KIC entries that map to a given ISO 27001 control ID.</summary>
        public static IEnumerable<CrosswalkEntry> ByIso27001Control(string controlId) =>
            Entries.Where(e => e.Iso27001Controls.Any(c => c.ControlId == controlId));

        /// <summary>Returns all KIC entries linked to a given ITIL 4 practice code.</summary>
        public static IEnumerable<CrosswalkEntry> ByItil4Practice(string practiceCode) =>
            Entries.Where(e => e.Itil4Practices.Any(p => p.PracticeCode == practiceCode));

        /// <summary>Serialises the full crosswalk to indented JSON.</summary>
        public static string ToJson() =>
            JsonSerializer.Serialize(
                Entries,
                new JsonSerializerOptions { WriteIndented = true, DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull }
            );

        /// <summary>Prints a markdown summary table to stdout.</summary>
        public static void PrintSummaryTable()
        {
            Console.WriteLine("| KIC ID | Description | SOC 2 Criteria | ISO 27001 Controls | ITIL 4 Practices |");
            Console.WriteLine("|--------|-------------|----------------|--------------------|------------------|");
            foreach (var entry in Entries)
            {
                var soc2 = string.Join(", ", entry.Soc2Criteria.Select(c => c.CriterionId));
                var iso  = string.Join(", ", entry.Iso27001Controls.Select(c => c.ControlId));
                var itil = string.Join(", ", entry.Itil4Practices.Select(p => p.PracticeCode));
                Console.WriteLine(
                    $"| {entry.KubricControl.KicId} | {entry.KubricControl.Description} | {soc2} | {iso} | {itil} |");
            }
        }
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Entry point for ad-hoc crosswalk export
    // ─────────────────────────────────────────────────────────────────────────

    internal static class Program
    {
        internal static void Main(string[] args)
        {
            if (args.Length > 0 && args[0] == "--json")
            {
                Console.WriteLine(CrosswalkRegistry.ToJson());
                return;
            }

            Console.WriteLine("Kubric SOC2 / ISO 27001 / ITIL 4 Crosswalk");
            Console.WriteLine(new string('=', 60));
            CrosswalkRegistry.PrintSummaryTable();
        }
    }
}
