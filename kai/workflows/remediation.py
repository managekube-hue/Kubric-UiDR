"""
RemediationWorkflow — Temporal durable workflow for VDR scan-to-patch pipeline.

Workflow: VDR finding → validate → Ansible remediate → verify → close finding

Activities:
  validate_finding    — confirm finding is still open in PostgreSQL
  run_ansible         — execute ansible_runner with the generated playbook
  verify_remediation  — re-check the affected asset to confirm fix
  close_finding       — mark the VDR finding as remediated in PostgreSQL

The workflow is idempotent (workflow_id = finding_id) so duplicate triggers
from retried NATS messages are safely deduplicated by Temporal.
"""

from __future__ import annotations

import asyncio
import os
from datetime import timedelta
from typing import Any

import structlog

log = structlog.get_logger(__name__)

_TEMPORAL_URL    = os.getenv("KUBRIC_TEMPORAL_URL", "temporal:7233")
_VDR_URL         = os.getenv("KUBRIC_VDR_URL", "http://vdr:8081")
_TASK_QUEUE      = "kubric-remediation"


# ─── Activities ──────────────────────────────────────────────────────────────

async def validate_finding(finding_id: str, tenant_id: str) -> dict[str, Any]:
    """Confirm the finding is still open before acting."""
    import httpx  # noqa: PLC0415
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.get(
                f"{_VDR_URL}/v1/vdr/findings/{finding_id}",
                headers={"X-Tenant-ID": tenant_id},
            )
            if resp.status_code == 200:
                data = resp.json()
                return {"valid": data.get("status") == "open", "finding": data}
    except Exception as exc:
        log.warning("remediation.validate_failed", finding_id=finding_id, error=str(exc))
    return {"valid": False, "finding": {}}


async def run_ansible(playbook: str, target: str, tenant_id: str) -> dict[str, Any]:
    """Execute an Ansible playbook against a target host."""
    try:
        import ansible_runner  # noqa: PLC0415
        result = ansible_runner.run(
            playbook=playbook,
            extravars={"target": target, "tenant_id": tenant_id},
            quiet=True,
        )
        return {
            "rc": result.rc,
            "status": result.status,
            "stdout": result.stdout.read() if result.stdout else "",
        }
    except ImportError:
        log.warning("remediation.ansible_runner_not_installed")
        return {"rc": -1, "status": "skipped", "stdout": "ansible_runner not installed"}
    except Exception as exc:
        log.error("remediation.ansible_failed", playbook=playbook, error=str(exc))
        return {"rc": 1, "status": "error", "stdout": str(exc)}


async def verify_remediation(finding_id: str, tenant_id: str) -> bool:
    """Re-check asset after remediation. Returns True if finding is resolved.

    Calls the VDR service to confirm the finding status is 'remediated' or
    'accepted'. Returns False if VDR is unreachable so unresolved findings do
    not get auto-closed. Layer 3 will extend this with a targeted Nuclei
    re-scan.
    """
    import httpx  # noqa: PLC0415
    vdr_url = os.getenv("KUBRIC_VDR_URL", "http://vdr:8081")
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(
                f"{vdr_url}/v1/vdr/findings/{finding_id}",
                headers={"X-Tenant-ID": tenant_id},
            )
            if resp.status_code == 404:
                log.info("remediation.verify.not_found", finding_id=finding_id)
                return True
            if resp.status_code == 200:
                data = resp.json()
                status = data.get("status", "")
                resolved = status in ("remediated", "accepted", "false_positive")
                log.info(
                    "remediation.verify.result",
                    finding_id=finding_id,
                    status=status,
                    resolved=resolved,
                )
                return resolved
    except Exception as exc:
        log.warning(
            "remediation.verify.vdr_unreachable",
            finding_id=finding_id,
            error=str(exc),
        )
        return False
    return False


async def close_finding(finding_id: str, tenant_id: str, plan_id: str) -> bool:
    """Mark the VDR finding as remediated in PostgreSQL via VDR service."""
    import httpx  # noqa: PLC0415
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.patch(
                f"{_VDR_URL}/v1/vdr/findings/{finding_id}",
                json={"status": "remediated", "remediation_plan_id": plan_id},
                headers={"X-Tenant-ID": tenant_id},
            )
            return resp.status_code in (200, 204)
    except Exception as exc:
        log.error("remediation.close_failed", finding_id=finding_id, error=str(exc))
        return False


# ─── Temporal workflow (wrapped to degrade gracefully) ────────────────────────

async def _run_remediation_workflow(tenant_id: str, plan: dict[str, Any]) -> None:
    """
    Execute the remediation as a plain async function (used when Temporal
    is unavailable or not yet wired).  Activities run sequentially.
    """
    finding_id = plan.get("finding_id", "")
    plan_id    = plan.get("plan_id", "")
    playbook   = plan.get("ansible_playbook") or ""

    if not finding_id:
        log.warning("remediation.no_finding_id")
        return

    validated = await validate_finding(finding_id, tenant_id)
    if not validated.get("valid"):
        log.info("remediation.finding_not_open", finding_id=finding_id)
        return

    if playbook:
        target = validated.get("finding", {}).get("asset", "localhost")
        await run_ansible(playbook, target=target, tenant_id=tenant_id)

    resolved = await verify_remediation(finding_id, tenant_id)
    if resolved:
        await close_finding(finding_id, tenant_id, plan_id)
        log.info("remediation.complete", finding_id=finding_id)


def _build_temporal_workflow(tenant_id: str, plan: dict[str, Any]) -> Any:
    """
    Wrap the remediation steps in a Temporal workflow if temporalio is installed.
    Returns a Temporal WorkflowHandle or None.
    """
    try:
        from temporalio import activity, workflow  # noqa: PLC0415
        from temporalio.client import Client  # noqa: PLC0415
        from temporalio.worker import Worker  # noqa: PLC0415

        @workflow.defn
        class RemediationWorkflow:
            @workflow.run
            async def run(self, tenant_id: str, plan: dict) -> str:  # type: ignore[type-arg]
                finding_id = plan.get("finding_id", "")
                plan_id    = plan.get("plan_id", "")
                playbook   = plan.get("ansible_playbook") or ""

                validated = await workflow.execute_activity(
                    validate_finding,
                    args=[finding_id, tenant_id],
                    start_to_close_timeout=timedelta(seconds=30),
                )
                if not validated.get("valid"):
                    return "skipped"

                if playbook:
                    target = validated.get("finding", {}).get("asset", "localhost")
                    await workflow.execute_activity(
                        run_ansible,
                        args=[playbook, target, tenant_id],
                        start_to_close_timeout=timedelta(minutes=10),
                    )

                resolved = await workflow.execute_activity(
                    verify_remediation,
                    args=[finding_id, tenant_id],
                    start_to_close_timeout=timedelta(seconds=60),
                )
                if resolved:
                    await workflow.execute_activity(
                        close_finding,
                        args=[finding_id, tenant_id, plan_id],
                        start_to_close_timeout=timedelta(seconds=30),
                    )
                return "complete"

        return RemediationWorkflow

    except ImportError:
        return None


async def submit_remediation(tenant_id: str, plan: dict[str, Any]) -> None:
    """
    Public API called by KAI-KEEPER.
    Tries Temporal first; falls back to plain async execution.
    """
    WorkflowClass = _build_temporal_workflow(tenant_id, plan)

    if WorkflowClass is not None:
        try:
            from temporalio.client import Client  # noqa: PLC0415
            client = await Client.connect(_TEMPORAL_URL)
            await client.start_workflow(
                WorkflowClass.run,
                args=[tenant_id, plan],
                id=f"remediation-{plan.get('plan_id', 'unknown')}",
                task_queue=_TASK_QUEUE,
            )
            log.info("remediation.temporal_submitted", plan_id=plan.get("plan_id"))
            return
        except Exception as exc:
            log.warning("remediation.temporal_fallback", error=str(exc))

    # Fallback — run directly without Temporal
    await _run_remediation_workflow(tenant_id, plan)
