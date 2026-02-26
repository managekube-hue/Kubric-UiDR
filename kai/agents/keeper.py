"""
KAI-KEEPER — Remediation plan generation and execution.

Subscribes to:  kubric.{tenant}.vdr.vuln.*   (vulnerability findings from VDR)
                kubric.{tenant}.grc.drift.*  (config drift from KIC)
Triggers:       Temporal RemediationWorkflow (via trigger_remediation @tool)
Publishes to:   kubric.kai.keeper.plan

Agent persona: DevSecOps Remediation Engineer
CrewAI integration: make_keeper_crew() → Crew.kickoff() → JSON remediation plan
"""

from __future__ import annotations

import time
import uuid
from typing import Any

import orjson
import structlog

from kai.core.crew import make_keeper_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

import os
_AUTO_REMEDIATE = os.getenv("KUBRIC_AUTO_REMEDIATE", "false").lower() == "true"


class KeeperAgent:
    """KAI-KEEPER agent — drafts and optionally executes remediation plans."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        tenant_id    = _extract_tenant(subject)
        plan_id      = str(uuid.uuid4())
        finding_json = orjson.dumps(event).decode()

        # ── CrewAI crew generates remediation plan ────────────────────────────
        crew   = make_keeper_crew(finding_json, tenant_id)
        output = crew.kickoff()
        plan   = parse_crew_output(output)

        result: dict[str, Any] = {
            "plan_id":            plan_id,
            "tenant_id":          tenant_id,
            "source_subject":     subject,
            "finding_id":         event.get("event_id", event.get("id", "")),
            "timestamp":          int(time.time() * 1000),
            "remediation_type":   plan.get("remediation_type", "manual_review"),
            "steps":              plan.get("steps", []),
            "ansible_playbook":   plan.get("ansible_playbook"),
            "estimated_risk":     plan.get("estimated_risk", "medium"),
            "auto_safe":          bool(plan.get("auto_safe", False)),
            "status":             "pending",
            "model_used":         "crewai/ollama/llama3.2",
        }

        # Trigger Temporal workflow if auto-remediation enabled AND plan is safe
        if _AUTO_REMEDIATE and result["auto_safe"]:
            await _trigger_temporal_workflow(tenant_id, result)
            result["status"] = "submitted"

        await nats_client.publish("kubric.kai.keeper.plan", orjson.dumps(result))

        log.info(
            "keeper.plan_published",
            plan_id=plan_id,
            tenant_id=tenant_id,
            type=result["remediation_type"],
            auto=result["status"] == "submitted",
        )
        return result


# ─── helpers ─────────────────────────────────────────────────────────────────

async def _trigger_temporal_workflow(tenant_id: str, plan: dict[str, Any]) -> None:
    try:
        from kai.workflows.remediation import submit_remediation  # noqa: PLC0415
        await submit_remediation(tenant_id=tenant_id, plan=plan)
    except ImportError:
        log.warning("keeper.temporal_unavailable")
    except Exception as exc:
        log.error("keeper.temporal_submit_failed", error=str(exc))


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
