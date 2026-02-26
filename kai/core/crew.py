"""
KAI CrewAI Crew factories — L2 AI orchestration layer.

Five security expert personas, each backed by an Ollama (local) LLM
with cloud fallback via LiteLLM (OpenAI / Anthropic).

Personas
--------
TRIAGE    — Senior SOC Analyst: severity + MITRE ATT&CK mapping
SENTINEL  — Security Posture Analyst: KiSS score computation
KEEPER    — DevSecOps Remediation Engineer: fix-plan generation
COMM      — Security Communications Officer: escalation / notifications
FORESIGHT — Threat Intelligence Analyst: predictive risk scoring

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
