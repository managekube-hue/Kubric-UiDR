"""
K-GRC-OSCAL-005_regscale_ingest.py

RegScale GRC platform integration — async httpx client.
Syncs security plans, controls, issues, and assessments between
Kubric PostgreSQL and the RegScale API.
"""
from __future__ import annotations

import asyncio
import json
import os
from dataclasses import dataclass, field
from typing import Any

import asyncpg
import httpx
import structlog

log = structlog.get_logger(__name__)

_BASE_URL = os.getenv("REGSCALE_URL", "https://app.regscale.com")
_API_KEY = os.getenv("REGSCALE_API_KEY", "")
_NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")


def _headers() -> dict[str, str]:
    return {
        "Authorization": f"Bearer {_API_KEY}",
        "Content-Type": "application/json",
        "Accept": "application/json",
    }


@dataclass
class RegScaleSecurityPlan:
    plan_id: int
    title: str
    status: str
    system_owner: str
    created_at: str


@dataclass
class RegScaleControl:
    control_id: int
    control_number: str
    title: str
    implementation_status: str
    security_plan_id: int


@dataclass
class RegScaleIssue:
    issue_id: int
    title: str
    severity: str   # critical | high | medium | low
    status: str
    security_plan_id: int
    description: str


class RegScaleClient:
    """Async httpx client for the RegScale GRC API."""

    def __init__(self, base_url: str = _BASE_URL, api_key: str = _API_KEY) -> None:
        self.base_url = base_url.rstrip("/")
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
                "Accept": "application/json",
            },
            timeout=30.0,
        )

    async def close(self) -> None:
        await self._client.aclose()

    async def _get(self, path: str, params: dict | None = None) -> Any:
        resp = await self._client.get(path, params=params)
        resp.raise_for_status()
        return resp.json()

    async def _post(self, path: str, body: dict) -> Any:
        resp = await self._client.post(path, content=json.dumps(body))
        resp.raise_for_status()
        return resp.json()

    async def _put(self, path: str, body: dict) -> Any:
        resp = await self._client.put(path, content=json.dumps(body))
        resp.raise_for_status()
        return resp.json()

    # ------------------------------------------------------------------ #
    #  Security Plans
    # ------------------------------------------------------------------ #

    async def get_security_plans(self) -> list[RegScaleSecurityPlan]:
        """Fetch all security plans from RegScale."""
        data = await self._get("/api/securityplans")
        plans: list[RegScaleSecurityPlan] = []
        for item in data if isinstance(data, list) else data.get("data", []):
            plans.append(RegScaleSecurityPlan(
                plan_id=item.get("id", 0),
                title=item.get("systemName", ""),
                status=item.get("status", ""),
                system_owner=item.get("systemOwner", ""),
                created_at=item.get("createdAt", ""),
            ))
        log.info("regscale_plans_fetched", count=len(plans))
        return plans

    # ------------------------------------------------------------------ #
    #  Controls
    # ------------------------------------------------------------------ #

    async def get_controls(self, security_plan_id: int) -> list[RegScaleControl]:
        """Fetch controls for a given security plan."""
        data = await self._get(f"/api/securitycontrols", params={"securityPlanId": security_plan_id})
        controls: list[RegScaleControl] = []
        for item in data if isinstance(data, list) else data.get("data", []):
            controls.append(RegScaleControl(
                control_id=item.get("id", 0),
                control_number=item.get("controlNumber", ""),
                title=item.get("controlTitle", ""),
                implementation_status=item.get("implementationStatus", "not_implemented"),
                security_plan_id=security_plan_id,
            ))
        return controls

    async def update_control_implementation(
        self,
        control_id: int,
        implementation_status: str,
        narrative: str,
    ) -> dict:
        """Update the implementation status and narrative for a control."""
        body = {
            "implementationStatus": implementation_status,
            "controlNarrative": narrative,
        }
        result = await self._put(f"/api/securitycontrols/{control_id}", body)
        log.info("regscale_control_updated", control_id=control_id, status=implementation_status)
        return result

    # ------------------------------------------------------------------ #
    #  Issues
    # ------------------------------------------------------------------ #

    async def create_issue(
        self,
        security_plan_id: int,
        title: str,
        severity: str,
        description: str,
    ) -> RegScaleIssue:
        """Create a new issue in RegScale."""
        body = {
            "title": title,
            "severity": severity,
            "description": description,
            "securityPlanId": security_plan_id,
            "status": "open",
        }
        data = await self._post("/api/issues", body)
        issue = RegScaleIssue(
            issue_id=data.get("id", 0),
            title=title,
            severity=severity,
            status="open",
            security_plan_id=security_plan_id,
            description=description,
        )
        log.info("regscale_issue_created", issue_id=issue.issue_id, title=title)
        return issue

    async def get_issues(self, security_plan_id: int) -> list[RegScaleIssue]:
        """Fetch issues for a security plan."""
        data = await self._get("/api/issues", params={"securityPlanId": security_plan_id})
        issues: list[RegScaleIssue] = []
        for item in data if isinstance(data, list) else data.get("data", []):
            issues.append(RegScaleIssue(
                issue_id=item.get("id", 0),
                title=item.get("title", ""),
                severity=item.get("severity", "medium"),
                status=item.get("status", "open"),
                security_plan_id=security_plan_id,
                description=item.get("description", ""),
            ))
        return issues

    # ------------------------------------------------------------------ #
    #  Sync: Kubric Assessments → RegScale
    # ------------------------------------------------------------------ #

    async def sync_assessments_to_regscale(
        self,
        tenant_id: str,
        security_plan_id: int,
        db_pool: asyncpg.Pool,
    ) -> int:
        """
        Read assessments from PostgreSQL and update corresponding RegScale controls.
        Returns the count of controls updated.
        """
        rows = await db_pool.fetch(
            """
            SELECT control_id, status, message
            FROM assessments
            WHERE tenant_id = $1
            """,
            tenant_id,
        )

        updated = 0
        controls = await self.get_controls(security_plan_id)
        # Build a lookup from control number → RegScale control_id
        ctrl_map = {c.control_number.upper(): c.control_id for c in controls}

        for row in rows:
            nist_id = str(row["control_id"]).upper()
            rs_ctrl_id = ctrl_map.get(nist_id)
            if not rs_ctrl_id:
                continue

            impl_status = "implemented" if row["status"] == "pass" else "not_implemented"
            narrative = str(row["message"] or "")
            try:
                await self.update_control_implementation(rs_ctrl_id, impl_status, narrative)
                updated += 1
            except httpx.HTTPStatusError as exc:
                log.warning("regscale_sync_error", control=nist_id, error=str(exc))

        log.info("regscale_sync_complete", tenant_id=tenant_id, updated=updated)
        return updated

    # ------------------------------------------------------------------ #
    #  Sync: RegScale Issues → Kubric KAI Alerts (via NATS)
    # ------------------------------------------------------------------ #

    async def pull_issues_from_regscale(
        self,
        security_plan_id: int,
        tenant_id: str,
        db_pool: asyncpg.Pool,
    ) -> list[RegScaleIssue]:
        """
        Pull critical/high issues from RegScale and publish kai_alerts via NATS.
        Also upserts issues into a regscale_issues PostgreSQL table for audit.
        """
        all_issues = await self.get_issues(security_plan_id)
        critical = [i for i in all_issues if i.severity.lower() in ("critical", "high")]

        for issue in critical:
            # Persist to local table
            await db_pool.execute(
                """
                INSERT INTO regscale_issues
                    (issue_id, tenant_id, security_plan_id, title, severity, status, description, synced_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
                ON CONFLICT (issue_id) DO UPDATE SET
                    status     = EXCLUDED.status,
                    synced_at  = NOW()
                """,
                issue.issue_id,
                tenant_id,
                security_plan_id,
                issue.title,
                issue.severity,
                issue.status,
                issue.description,
            )

        # Publish to NATS (best-effort, non-blocking)
        if critical:
            await _publish_kai_alerts(critical, tenant_id)

        log.info(
            "regscale_pull_issues",
            total=len(all_issues),
            critical_high=len(critical),
            tenant_id=tenant_id,
        )
        return critical


async def _publish_kai_alerts(issues: list[RegScaleIssue], tenant_id: str) -> None:
    """Publish critical/high RegScale issues to NATS as KAI alerts."""
    try:
        import nats  # type: ignore

        nc = await nats.connect(_NATS_URL)
        for issue in issues:
            subject = f"kubric.{tenant_id}.grc.regscale.alert.v1"
            payload = json.dumps({
                "type": "regscale_issue",
                "issue_id": issue.issue_id,
                "title": issue.title,
                "severity": issue.severity,
                "status": issue.status,
            }).encode()
            await nc.publish(subject, payload)
        await nc.flush()
        await nc.close()
        log.info("regscale_kai_alerts_published", count=len(issues))
    except Exception as exc:
        log.warning("regscale_nats_publish_failed", error=str(exc))


async def main() -> None:
    db_url = os.getenv("DATABASE_URL", "postgresql://localhost/kubric")
    tenant_id = os.getenv("TENANT_ID", "00000000-0000-0000-0000-000000000001")
    plan_id = int(os.getenv("REGSCALE_PLAN_ID", "1"))

    db_pool = await asyncpg.create_pool(db_url)
    client = RegScaleClient()
    try:
        plans = await client.get_security_plans()
        print(f"Security plans: {len(plans)}")
        for p in plans[:3]:
            print(f"  [{p.plan_id}] {p.title} ({p.status})")

        updated = await client.sync_assessments_to_regscale(tenant_id, plan_id, db_pool)
        print(f"Controls synced to RegScale: {updated}")

        critical = await client.pull_issues_from_regscale(plan_id, tenant_id, db_pool)
        print(f"Critical/High issues pulled: {len(critical)}")
    finally:
        await client.close()
        await db_pool.close()


if __name__ == "__main__":
    asyncio.run(main())
