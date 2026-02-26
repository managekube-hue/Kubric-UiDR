"""
KAI Security Tools — CrewAI @tool definitions.

Each function is decorated with @tool so CrewAI agents can invoke them
autonomously.  All tools are synchronous wrappers around async helpers
(httpx / nats / clickhouse-connect) that are safe to call from within a
CrewAI task execution context.

Tools available:
  • get_vdr_summary        — open vulnerability counts from VDR service
  • get_kic_summary        — latest compliance pass-rate from KIC service
  • query_recent_alerts    — alert volume from ClickHouse (last N hours)
  • publish_nats_event     — publish an enriched event to NATS
  • trigger_remediation    — submit a RemediationWorkflow via Temporal
  • forward_to_n8n         — push an incident payload to the n8n webhook
"""

from __future__ import annotations

import asyncio
import json
from typing import Any

import httpx
import structlog
from crewai.tools import tool

from kai.config import settings

log = structlog.get_logger(__name__)

# ─── Internal ports (Docker Compose service names) ────────────────────────────
_VDR_BASE  = "http://vdr:8081"
_KIC_BASE  = "http://kic:8082"
_NOC_BASE  = "http://noc:8083"
_KSVC_BASE = "http://ksvc:8080"


# =============================================================================
# Vulnerability tools
# =============================================================================

@tool("get_vdr_summary")
def get_vdr_summary(tenant_id: str) -> str:
    """
    Returns a JSON string with open vulnerability counts for a tenant.
    Example output: {"critical": 3, "high": 12, "medium": 45, "low": 20}
    Use this to understand the current vulnerability posture.
    """
    try:
        resp = httpx.get(
            f"{_VDR_BASE}/findings",
            params={"tenant_id": tenant_id, "status": "open"},
            timeout=5.0,
        )
        if resp.status_code == 200:
            findings = resp.json()
            counts: dict[str, int] = {"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
            for f in (findings if isinstance(findings, list) else findings.get("items", [])):
                sev = str(f.get("severity", "")).lower()
                counts[sev] = counts.get(sev, 0) + 1
            return json.dumps(counts)
    except Exception as exc:
        log.debug("tool.get_vdr_summary.failed", error=str(exc))
    return json.dumps({"critical": 0, "high": 0, "medium": 0, "low": 0, "error": "unavailable"})


# =============================================================================
# Compliance tools
# =============================================================================

@tool("get_kic_summary")
def get_kic_summary(tenant_id: str) -> str:
    """
    Returns a JSON string with the latest compliance assessment summary.
    Example output: {"framework": "CIS_v8", "pass_rate": 87.5, "failed": 12}
    Use this to understand the tenant's compliance posture.
    """
    try:
        resp = httpx.get(
            f"{_KIC_BASE}/assessments",
            params={"tenant_id": tenant_id},
            timeout=5.0,
        )
        if resp.status_code == 200:
            data = resp.json()
            assessments = data if isinstance(data, list) else data.get("items", [])
            if assessments:
                latest = assessments[0]  # assume sorted by date desc
                return json.dumps({
                    "framework":  latest.get("framework", "unknown"),
                    "pass_rate":  latest.get("pass_rate", 0.0),
                    "failed":     latest.get("failed", 0),
                    "total":      latest.get("total_controls", 0),
                })
    except Exception as exc:
        log.debug("tool.get_kic_summary.failed", error=str(exc))
    return json.dumps({"framework": "unknown", "pass_rate": 75.0, "failed": 0, "error": "unavailable"})


# =============================================================================
# Threat intelligence tools
# =============================================================================

@tool("query_recent_alerts")
def query_recent_alerts(tenant_id: str, hours: int = 24) -> str:
    """
    Returns a JSON string with alert counts by severity from ClickHouse
    for the last N hours.  Use this to understand alert velocity and trends.
    Example output: {"CRITICAL": 2, "HIGH": 8, "MEDIUM": 31, "total": 41}
    """
    try:
        import clickhouse_connect  # type: ignore[import-untyped]
        client = clickhouse_connect.get_client(
            host="clickhouse", port=8123, database="kubric"
        )
        rows = client.query(
            """
            SELECT severity, count() AS cnt
            FROM kubric.kai_triage_results
            WHERE tenant_id = {tenant_id:String}
              AND timestamp >= now() - INTERVAL {hours:UInt32} HOUR
            GROUP BY severity
            """,
            parameters={"tenant_id": tenant_id, "hours": hours},
        ).result_rows
        counts: dict[str, Any] = {row[0]: row[1] for row in rows}
        counts["total"] = sum(counts.values())
        return json.dumps(counts)
    except Exception as exc:
        log.debug("tool.query_recent_alerts.failed", error=str(exc))
    return json.dumps({"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "total": 0, "error": "unavailable"})


# =============================================================================
# NATS publishing tools
# =============================================================================

@tool("publish_nats_event")
def publish_nats_event(subject: str, json_payload: str) -> str:
    """
    Publishes an enriched event to the NATS message bus.
    subject: the full NATS subject string e.g. kubric.tenant-id.kai.triage.enriched
    json_payload: a valid JSON string representing the event payload.
    Returns 'published' on success or an error description.
    """
    from kai.core.nats_client import nats_client  # noqa: PLC0415

    async def _pub() -> None:
        await nats_client.publish(subject, json_payload.encode())

    try:
        asyncio.get_event_loop().run_until_complete(_pub())
        return "published"
    except RuntimeError:
        # running inside an already-running event loop (e.g. FastAPI)
        import concurrent.futures
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as executor:
            future = executor.submit(asyncio.run, _pub())
            future.result(timeout=10)
        return "published"
    except Exception as exc:
        log.warning("tool.publish_nats_event.failed", error=str(exc))
        return f"error: {exc}"


# =============================================================================
# Remediation tools
# =============================================================================

@tool("trigger_remediation")
def trigger_remediation(tenant_id: str, plan_json: str) -> str:
    """
    Submits a RemediationWorkflow to Temporal for durable execution.
    tenant_id: the target tenant
    plan_json: JSON string with remediation plan (steps, ansible_playbook, etc.)
    Returns 'submitted', 'executed', or an error description.
    """
    async def _submit() -> str:
        from kai.workflows.remediation import submit_remediation  # noqa: PLC0415
        plan = json.loads(plan_json)
        await submit_remediation(tenant_id=tenant_id, plan=plan)
        return "submitted"

    try:
        return asyncio.run(_submit())
    except Exception as exc:
        log.warning("tool.trigger_remediation.failed", error=str(exc))
        return f"error: {exc}"


# =============================================================================
# n8n / ITSM tools
# =============================================================================

@tool("forward_to_n8n")
def forward_to_n8n(incident_json: str) -> str:
    """
    Forwards a security incident payload to the n8n ITSM webhook so that
    n8n can route it to Slack, PagerDuty, email, or other downstream systems.
    incident_json: JSON string with incident details (tenant_id, severity, summary, etc.)
    Returns 'forwarded' on success or an error description.
    """
    n8n_url = f"{settings.n8n_base_url}/webhook/kubric-incident"
    try:
        resp = httpx.post(
            n8n_url,
            content=incident_json,
            headers={"Content-Type": "application/json"},
            timeout=5.0,
        )
        resp.raise_for_status()
        return "forwarded"
    except Exception as exc:
        log.warning("tool.forward_to_n8n.failed", url=n8n_url, error=str(exc))
        return f"error: {exc}"
