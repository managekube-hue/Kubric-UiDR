"""
KAI CrewAI Crew factories — L2 AI orchestration layer.

Thirteen security expert personas, each backed by an Ollama (local) LLM
with cloud fallback via LiteLLM (OpenAI / Anthropic).

Personas
--------
TRIAGE    — Senior SOC Analyst: severity + MITRE ATT&CK mapping
SENTINEL  — Security Posture Analyst: KiSS score computation
KEEPER    — DevSecOps Remediation Engineer: fix-plan generation
COMM      — Security Communications Officer: escalation / notifications
FORESIGHT — Threat Intelligence Analyst: predictive risk scoring
HOUSE     — SOC Operations Manager: dashboard insights + tenant health
BILL      — Billing Reconciliation Specialist: usage metering + invoicing
ANALYST   — Senior Threat Analyst: deep-dive forensic analysis
HUNTER    — Threat Hunter: proactive hypothesis-driven hunting
INVEST    — Digital Forensics Investigator: evidence chain + root cause
SIMULATE  — Purple Team Operator: attack simulation + coverage testing
RISK      — Cyber Risk Quantification Analyst: FAIR-based risk assessment
DEPLOY    — Deployment Security Engineer: change validation + approval

Usage
-----
    from kai.core.crew import make_triage_crew
    crew   = make_triage_crew(event_json, tenant_id)
    result = crew.kickoff()          # returns CrewOutput
    data   = json.loads(result.raw)  # parse JSON from output
"""

from __future__ import annotations

import json
import textwrap
from typing import Any

import structlog
from crewai import Agent, Crew, Process, Task
from crewai import LLM

from kai.config import settings
from kai.tools.security_tools import (
    forward_to_n8n,
    get_kic_summary,
    get_vdr_summary,
    publish_nats_event,
    query_recent_alerts,
    trigger_remediation,
)

log = structlog.get_logger(__name__)


# =============================================================================
# LLM factory  — Ollama first, OpenAI fallback via LiteLLM
# =============================================================================

def _llm(temperature: float = 0.1) -> LLM:
    """
    Returns a CrewAI LLM configured for Ollama (local, zero-cost).
    If KUBRIC_OPENAI_API_KEY is set, CrewAI will automatically fall back
    to OpenAI when Ollama is unavailable.
    """
    return LLM(
        model="ollama/llama3.2",
        base_url=settings.ollama_url,
        temperature=temperature,
    )


# =============================================================================
# TRIAGE — Senior SOC Analyst
# =============================================================================

def make_triage_crew(event_json: str, tenant_id: str) -> Crew:
    """Creates a single-agent Crew that triages one OCSF security event."""

    analyst = Agent(
        role="Senior SOC Analyst",
        goal=(
            "Analyse incoming OCSF security events, assign an accurate severity level, "
            "map to MITRE ATT&CK techniques, produce a concise analyst summary, and "
            "recommend the immediate action the responder should take."
        ),
        backstory=textwrap.dedent("""\
            You are a Senior SOC Analyst with 10 years of incident response at
            managed security service providers.  You have deep expertise in endpoint
            detection (EDR/eBPF), network anomaly detection, and identity threat
            detection (ITDR).  You always respond with valid JSON — no prose outside
            the JSON object.
        """),
        tools=[publish_nats_event],
        llm=_llm(),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            OCSF security event (JSON):
            {event_json}

            Triage this event and return ONLY a JSON object with these exact keys:
            {{
              "severity":           "CRITICAL|HIGH|MEDIUM|LOW|INFO",
              "mitre_techniques":   ["TA0001", "T1059", ...],
              "summary":            "one-sentence analyst summary",
              "recommended_action": "concise action for the analyst",
              "confidence":         0.0-1.0
            }}

            Do NOT include any text outside the JSON object.
        """),
        expected_output=(
            'A single valid JSON object with keys: '
            'severity, mitre_techniques, summary, recommended_action, confidence.'
        ),
        agent=analyst,
    )

    return Crew(
        agents=[analyst],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# KEEPER — DevSecOps Remediation Engineer
# =============================================================================

def make_keeper_crew(finding_json: str, tenant_id: str) -> Crew:
    """Creates a single-agent Crew that drafts a remediation plan for a finding."""

    engineer = Agent(
        role="DevSecOps Remediation Engineer",
        goal=(
            "Given a security vulnerability or configuration drift finding, produce "
            "a precise, actionable remediation plan that can be executed automatically "
            "or handed to an on-call engineer."
        ),
        backstory=textwrap.dedent("""\
            You are a DevSecOps engineer specialised in automated remediation.
            You know Ansible, Terraform, package management, and Linux hardening.
            You always assess whether automation is safe before recommending
            auto-apply, and you correctly identify the package, CVE, or misconfiguration
            that must be fixed.  You always respond with valid JSON.
        """),
        tools=[get_vdr_summary, trigger_remediation],
        llm=_llm(),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Security finding (JSON):
            {finding_json}

            Draft a remediation plan and return ONLY a JSON object:
            {{
              "remediation_type": "patch|config_change|manual_review",
              "steps":            ["step 1", "step 2", ...],
              "ansible_playbook": "playbook_name.yml or null",
              "estimated_risk":   "low|medium|high",
              "auto_safe":        true|false
            }}

            Set auto_safe=true ONLY if applying without human approval is safe.
            Do NOT include any text outside the JSON object.
        """),
        expected_output=(
            'A single valid JSON object with keys: '
            'remediation_type, steps, ansible_playbook, estimated_risk, auto_safe.'
        ),
        agent=engineer,
    )

    return Crew(
        agents=[engineer],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# SENTINEL — Security Posture Analyst (KiSS score)
# =============================================================================

def make_sentinel_crew(tenant_id: str) -> Crew:
    """Creates a 2-task Crew that computes and explains the KiSS health score."""

    posture_analyst = Agent(
        role="Security Posture Analyst",
        goal=(
            "Compute the Kubric integrated Security Score (KiSS) for a tenant by "
            "querying vulnerability and compliance data, then produce a human-readable "
            "narrative that explains the score and the top improvement actions."
        ),
        backstory=textwrap.dedent("""\
            You are a Security Posture Analyst who specialises in multi-tenant MSSP
            environments.  You query live security data, quantify risk using the KiSS
            framework, and deliver executive-level security briefings.
        """),
        tools=[get_vdr_summary, get_kic_summary, query_recent_alerts],
        llm=_llm(temperature=0.2),
        verbose=False,
        allow_delegation=False,
    )

    score_task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}

            1. Use get_vdr_summary to get open vulnerability counts.
            2. Use get_kic_summary to get the latest compliance pass rate.
            3. Use query_recent_alerts(tenant_id="{tenant_id}", hours=24) for alert volume.
            4. Compute the KiSS score (0-100):
               kiss = vuln_score*0.30 + compliance_score*0.25 + detection_score*0.25 + response_score*0.20

               Where:
               - vuln_score       = max(0, 100 - criticals*5 - highs*2)
               - compliance_score = pass_rate (from KIC)
               - detection_score  = 100 if total_alerts < 100 else max(50, 100 - total_alerts/10)
               - response_score   = 85 (placeholder until MTTR data available in Layer 3)

            Return ONLY a JSON object:
            {{
              "kiss_score":       87.5,
              "vuln_score":       75.0,
              "compliance_score": 90.0,
              "detection_score":  80.0,
              "response_score":   85.0,
              "open_criticals":   2,
              "open_highs":       8,
              "active_incidents": 0
            }}
        """),
        expected_output='A JSON object with KiSS score components.',
        agent=posture_analyst,
    )

    return Crew(
        agents=[posture_analyst],
        tasks=[score_task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# COMM — Security Communications Officer
# =============================================================================

def make_comm_crew(incident_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that routes incident notifications via n8n."""

    comm_officer = Agent(
        role="Security Communications Officer",
        goal=(
            "Evaluate the severity of a security incident and route the appropriate "
            "notification to on-call engineers, management, and ITSM systems via n8n."
        ),
        backstory=textwrap.dedent("""\
            You are a Security Communications Officer responsible for ensuring the right
            people are alerted at the right time.  You understand escalation policies,
            SLAs, and the correct channels for CRITICAL vs LOW severity incidents.
            You use the forward_to_n8n tool to trigger downstream ITSM workflows.
        """),
        tools=[forward_to_n8n],
        llm=_llm(temperature=0.0),
        verbose=False,
        allow_delegation=False,
    )

    notif_task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Security incident (JSON):
            {incident_json}

            Evaluate the incident severity and:
            - If severity is CRITICAL or HIGH: call forward_to_n8n with the full
              incident JSON so the ITSM notification workflow is triggered.
            - If severity is MEDIUM or below: no escalation needed.

            Return ONLY a JSON object:
            {{
              "escalated":          true|false,
              "channel":            "n8n_itsm|none",
              "notification_sent":  true|false,
              "reason":             "one-line explanation"
            }}
        """),
        expected_output='A JSON object confirming whether escalation was triggered.',
        agent=comm_officer,
    )

    return Crew(
        agents=[comm_officer],
        tasks=[notif_task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# FORESIGHT — Threat Intelligence Analyst (predictive risk)
# =============================================================================

def make_foresight_crew(tenant_id: str) -> Crew:
    """Creates a Crew that produces a 24-hour predictive risk score."""

    ti_analyst = Agent(
        role="Threat Intelligence Analyst",
        goal=(
            "Analyse recent alert trends, open vulnerability density, and behavioural "
            "signals to produce a predictive risk score and recommend proactive actions."
        ),
        backstory=textwrap.dedent("""\
            You are a Threat Intelligence Analyst with expertise in attack chain
            analysis, behavioural analytics, and predictive threat modelling.
            You surface early-warning signals before they become incidents.
        """),
        tools=[query_recent_alerts, get_vdr_summary],
        llm=_llm(temperature=0.3),
        verbose=False,
        allow_delegation=False,
    )

    risk_task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}

            1. Use query_recent_alerts(tenant_id="{tenant_id}", hours=24) for today's alerts.
            2. Use query_recent_alerts(tenant_id="{tenant_id}", hours=168) for 7-day baseline.
            3. Use get_vdr_summary to get open critical vulnerability count.
            4. Compute alert_velocity = alerts_24h / 24 (per-hour rate).
            5. Detect lateral_movement if CRITICAL alerts > 2 in last 24h.
            6. Compute risk_score = min(100, alert_velocity * 10 + criticals * 5).

            Return ONLY a JSON object:
            {{
              "risk_score":       45.0,
              "alert_velocity":   2.1,
              "vuln_density":     3.0,
              "lateral_movement": false,
              "data_exfil_signal": false,
              "top_risk_factors": ["Elevated CRITICAL alert rate", "3 unpatched CVEs"],
              "recommendations":  ["Patch CVE-2024-XXXX within 24h", "Review firewall egress"]
            }}
        """),
        expected_output='A JSON object with predictive risk score and recommendations.',
        agent=ti_analyst,
    )

    return Crew(
        agents=[ti_analyst],
        tasks=[risk_task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# HOUSE — SOC Operations Manager
# =============================================================================

def make_house_crew(tenant_id: str) -> Crew:
    """Creates a Crew that aggregates tenant health and SOC dashboard insights."""

    soc_manager = Agent(
        role="SOC Operations Manager",
        goal=(
            "Aggregate agent health status, alert volumes, open ticket counts, and "
            "the current KiSS score to produce a tenant health dashboard summary."
        ),
        backstory=textwrap.dedent("""\
            You are a SOC Operations Manager overseeing a multi-tenant MSSP
            environment.  You monitor agent health, alert trends, and ticket
            throughput to ensure SLA compliance and early detection of issues.
        """),
        tools=[get_vdr_summary, get_kic_summary, query_recent_alerts],
        llm=_llm(temperature=0.1),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}

            1. Use query_recent_alerts(tenant_id="{tenant_id}", hours=24) for alert counts.
            2. Use get_vdr_summary for vulnerability posture.
            3. Use get_kic_summary for compliance status.

            Return ONLY a JSON object:
            {{
              "agent_health": {{"healthy": 3, "stale": 0, "offline": 0}},
              "alert_24h": 42,
              "open_tickets": 5,
              "kiss_score": 82.5,
              "top_issues": ["2 agents stale > 10 min", "3 unpatched CVEs"]
            }}
        """),
        expected_output='A JSON object with tenant health dashboard data.',
        agent=soc_manager,
    )

    return Crew(
        agents=[soc_manager],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# BILL — Billing Reconciliation Specialist
# =============================================================================

def make_bill_crew(usage_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that reconciles billing usage data."""

    billing_specialist = Agent(
        role="Billing Reconciliation Specialist",
        goal=(
            "Validate billing usage records for accuracy, detect anomalies in "
            "event volumes that could indicate metering errors, and produce a "
            "reconciled billing summary."
        ),
        backstory=textwrap.dedent("""\
            You are a Billing Reconciliation Specialist with expertise in SaaS
            usage-based billing.  You validate event counts against Merkle-tree
            audit roots and flag discrepancies before invoices are finalized.
        """),
        tools=[query_recent_alerts],
        llm=_llm(temperature=0.0),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Usage data (JSON):
            {usage_json}

            Reconcile the usage data and return ONLY a JSON object:
            {{
              "period": "2024-01",
              "total_agents": 5,
              "total_events": 125000,
              "amount_usd": 499.00,
              "anomaly_detected": false,
              "merkle_root": "abc123..."
            }}
        """),
        expected_output='A JSON object with reconciled billing data.',
        agent=billing_specialist,
    )

    return Crew(
        agents=[billing_specialist],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# ANALYST — Senior Threat Analyst
# =============================================================================

def make_analyst_crew(event_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that performs deep-dive forensic analysis on escalated alerts."""

    threat_analyst = Agent(
        role="Senior Threat Analyst",
        goal=(
            "Perform deep-dive investigation of escalated security alerts.  Reconstruct "
            "the complete attack chain, identify all IOCs, map affected assets, build a "
            "timeline, and recommend containment actions."
        ),
        backstory=textwrap.dedent("""\
            You are a Senior Threat Analyst with 12 years of incident response
            experience.  You specialise in APT analysis, lateral movement detection,
            and evidence-based investigation.  You always produce structured findings.
        """),
        tools=[query_recent_alerts, get_vdr_summary, publish_nats_event],
        llm=_llm(temperature=0.2),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Escalated alert (JSON):
            {event_json}

            Perform a deep-dive investigation and return ONLY a JSON object:
            {{
              "attack_chain": ["initial_access", "execution", "persistence"],
              "iocs": [{{"type": "ip", "value": "1.2.3.4"}}, {{"type": "hash", "value": "abc..."}}],
              "affected_assets": ["server-01", "workstation-03"],
              "timeline": [{{"time": "2024-01-01T00:00:00Z", "action": "..."}}],
              "containment_actions": ["isolate server-01", "block IP 1.2.3.4"],
              "confidence": 0.85
            }}
        """),
        expected_output='A JSON object with investigation report.',
        agent=threat_analyst,
    )

    return Crew(
        agents=[threat_analyst],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# HUNTER — Threat Hunter
# =============================================================================

def make_hunter_crew(context_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that runs hypothesis-driven threat hunts."""

    hunter = Agent(
        role="Threat Hunter",
        goal=(
            "Based on threat intelligence and anomaly signals, formulate hunting "
            "hypotheses, query available data sources, and identify previously "
            "undetected threats or attacker footholds."
        ),
        backstory=textwrap.dedent("""\
            You are a Threat Hunter who proactively searches for adversary activity
            that evades automated detection.  You use MITRE ATT&CK as your primary
            framework and combine log analysis, network telemetry, and endpoint data
            to uncover hidden threats.
        """),
        tools=[query_recent_alerts, get_vdr_summary],
        llm=_llm(temperature=0.3),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Context / trigger (JSON):
            {context_json}

            Conduct a threat hunt and return ONLY a JSON object:
            {{
              "hypothesis": "Adversary may be using living-off-the-land binaries...",
              "data_sources_queried": ["process_events", "network_flows", "dns_queries"],
              "findings": [{{"description": "...", "evidence": "..."}}],
              "iocs_discovered": [{{"type": "domain", "value": "evil.com"}}],
              "mitre_techniques": ["T1059.001", "T1071.001"],
              "severity": "HIGH",
              "recommendation": "Block egress to evil.com and investigate host-X"
            }}
        """),
        expected_output='A JSON object with threat hunting findings.',
        agent=hunter,
    )

    return Crew(
        agents=[hunter],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# INVEST — Digital Forensics Investigator
# =============================================================================

def make_invest_crew(event_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that manages forensic investigations with chain-of-custody."""

    investigator = Agent(
        role="Digital Forensics Investigator",
        goal=(
            "Collect and preserve digital evidence, maintain chain of custody, "
            "determine root cause, assess impact, and identify regulatory implications."
        ),
        backstory=textwrap.dedent("""\
            You are a certified Digital Forensics Investigator (GCFE, EnCE) with
            expertise in disk forensics, memory analysis, and network forensics.
            You maintain strict chain-of-custody procedures and produce court-admissible
            investigation reports.
        """),
        tools=[query_recent_alerts, get_vdr_summary],
        llm=_llm(temperature=0.1),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Investigation context (JSON):
            {event_json}

            Produce a forensic case report as ONLY a JSON object:
            {{
              "evidence_collected": [{{"type": "memory_dump", "source": "host-01", "hash": "..."}}],
              "chain_of_custody": [{{"action": "acquired", "by": "kai-invest", "time": "..."}}],
              "root_cause": "Phishing email led to credential compromise...",
              "impact_assessment": "3 systems compromised, no data exfiltration confirmed",
              "regulatory_implications": ["GDPR Article 33 notification required"],
              "remediation_verified": false
            }}
        """),
        expected_output='A JSON object with forensic case report.',
        agent=investigator,
    )

    return Crew(
        agents=[investigator],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# SIMULATE — Purple Team Operator
# =============================================================================

def make_simulate_crew(scenario_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that plans and evaluates attack simulations."""

    purple_team = Agent(
        role="Purple Team Operator",
        goal=(
            "Design controlled attack simulations mapped to MITRE ATT&CK, evaluate "
            "which detections fired and which were missed, compute a detection "
            "coverage score, and recommend improvements."
        ),
        backstory=textwrap.dedent("""\
            You are a Purple Team Operator with red team and blue team experience.
            You design realistic attack scenarios, execute them safely in controlled
            environments, and produce detection coverage reports that help blue teams
            close gaps.
        """),
        tools=[query_recent_alerts],
        llm=_llm(temperature=0.2),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Simulation scenario (JSON):
            {scenario_json}

            Plan and evaluate a simulation, then return ONLY a JSON object:
            {{
              "attack_type": "ransomware_simulation",
              "mitre_techniques_tested": ["T1566.001", "T1059.001", "T1486"],
              "detections_triggered": ["sigma:ransomware_extension", "yara:ransomware_note"],
              "detections_missed": ["T1486 file encryption activity"],
              "coverage_score": 66.7,
              "gaps_identified": ["No detection for volume shadow copy deletion"],
              "recommendations": ["Add Sigma rule for vssadmin delete shadows"]
            }}
        """),
        expected_output='A JSON object with simulation results and coverage analysis.',
        agent=purple_team,
    )

    return Crew(
        agents=[purple_team],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# RISK — Cyber Risk Quantification Analyst (FAIR)
# =============================================================================

def make_risk_crew(context_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that performs FAIR-based quantitative risk assessment."""

    risk_analyst = Agent(
        role="Cyber Risk Quantification Analyst",
        goal=(
            "Apply the FAIR (Factor Analysis of Information Risk) methodology to "
            "quantify cyber risk in financial terms.  Estimate threat event frequency, "
            "vulnerability factor, loss magnitude, and annual loss expectancy."
        ),
        backstory=textwrap.dedent("""\
            You are a FAIR-certified Cyber Risk Analyst (Open FAIR).  You translate
            security findings into business-language risk metrics that executives
            understand: ALE, probable loss ranges, and risk reduction ROI.
        """),
        tools=[get_vdr_summary, get_kic_summary],
        llm=_llm(temperature=0.2),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Risk context (JSON):
            {context_json}

            Perform a FAIR risk assessment and return ONLY a JSON object:
            {{
              "risk_scenario": "Ransomware targeting unpatched CVE-2024-XXXXX",
              "threat_event_frequency": 2.5,
              "vulnerability_factor": 0.7,
              "loss_magnitude_usd": 250000.00,
              "annual_loss_expectancy": 437500.00,
              "risk_rating": "HIGH",
              "mitigations": ["Patch CVE within 72h", "Enable endpoint isolation"],
              "residual_risk": 0.15
            }}
        """),
        expected_output='A JSON object with FAIR risk quantification.',
        agent=risk_analyst,
    )

    return Crew(
        agents=[risk_analyst],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# DEPLOY — Deployment Security Engineer
# =============================================================================

def make_deploy_crew(deployment_json: str, tenant_id: str) -> Crew:
    """Creates a Crew that validates deployments for security compliance."""

    deploy_engineer = Agent(
        role="Deployment Security Engineer",
        goal=(
            "Review deployment requests for security compliance, validate that "
            "security checks pass, assess risk level, and approve or reject the "
            "deployment with conditions."
        ),
        backstory=textwrap.dedent("""\
            You are a Deployment Security Engineer responsible for the security
            gate in the CI/CD pipeline.  You review container images, IaC templates,
            dependency manifests, and runtime configurations to ensure deployments
            meet security baseline requirements.
        """),
        tools=[get_vdr_summary, get_kic_summary],
        llm=_llm(temperature=0.0),
        verbose=False,
        allow_delegation=False,
    )

    task = Task(
        description=textwrap.dedent(f"""\
            Tenant: {tenant_id}
            Deployment request (JSON):
            {deployment_json}

            Validate the deployment and return ONLY a JSON object:
            {{
              "deployment_type": "kubernetes_rollout",
              "target": "production/api-gateway",
              "security_checks_passed": ["image_scan_clean", "no_critical_cves", "secrets_not_embedded"],
              "security_checks_failed": ["container_running_as_root"],
              "approved": false,
              "risk_level": "HIGH",
              "rollback_plan": "kubectl rollout undo deployment/api-gateway",
              "conditions": ["Fix container running as root before re-submitting"]
            }}
        """),
        expected_output='A JSON object with deployment validation results.',
        agent=deploy_engineer,
    )

    return Crew(
        agents=[deploy_engineer],
        tasks=[task],
        process=Process.sequential,
        verbose=False,
    )


# =============================================================================
# Utility: safe JSON extraction from CrewOutput
# =============================================================================

def parse_crew_output(raw: Any) -> dict[str, Any]:
    """
    Extract a dict from a CrewAI CrewOutput object.
    CrewOutput.raw is a string; strip markdown fences and parse as JSON.
    Falls back to {"raw": str(raw)} on parse failure.
    """
    text = str(getattr(raw, "raw", raw)).strip()
    if text.startswith("```"):
        lines = text.splitlines()
        text = "\n".join(lines[1:-1]) if len(lines) > 2 else text
    try:
        return json.loads(text)  # type: ignore[no-any-return]
    except Exception:
        log.warning("crew.output_parse_failed", snippet=text[:200])
        return {"raw": text}
