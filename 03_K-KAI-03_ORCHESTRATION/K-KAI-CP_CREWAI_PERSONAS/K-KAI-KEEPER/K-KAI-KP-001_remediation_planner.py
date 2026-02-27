"""
K-KAI-KP-001: AI-Powered Remediation Planner
Subscribes to kubric.*.incident.* NATS subjects, generates remediation plans
via Anthropic, persists to PostgreSQL, and publishes plans to NATS.
"""

import asyncio
import json
import logging
import os
import uuid
from datetime import datetime, timezone
from typing import Any

import anthropic
import asyncpg
import nats
from nats.aio.client import Client as NATSClient

logger = logging.getLogger("K-KAI-KP-001")

NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")
DATABASE_URL: str = os.getenv("DATABASE_URL", "postgresql://kubric:kubric@localhost/kubric")
ANTHROPIC_API_KEY: str = os.getenv("ANTHROPIC_API_KEY", "")

_CREATE_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS remediation_plans (
    id            UUID PRIMARY KEY,
    incident_id   TEXT NOT NULL,
    tenant_id     TEXT NOT NULL,
    plan_steps    JSONB NOT NULL,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at  TIMESTAMPTZ,
    status        TEXT NOT NULL DEFAULT 'pending'
);
"""


class RemediationPlanner:
    """
    Listens on kubric.*.incident.* and drives the AI remediation workflow.
    """

    def __init__(self) -> None:
        self.nc: NATSClient | None = None
        self._db: asyncpg.Pool | None = None
        self._running: bool = False
        self._subscription: Any = None
        self._anthropic = anthropic.Anthropic(api_key=ANTHROPIC_API_KEY) if ANTHROPIC_API_KEY else None

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    async def start(self) -> None:
        logger.info("RemediationPlanner connecting to NATS and PostgreSQL …")
        self.nc = await nats.connect(NATS_URL, max_reconnect_attempts=-1)
        self._db = await asyncpg.create_pool(DATABASE_URL, min_size=2, max_size=10)

        async with self._db.acquire() as conn:
            await conn.execute(_CREATE_TABLE_SQL)

        self._running = True
        self._subscription = await self.nc.subscribe(
            "kubric.*.incident.*",
            cb=self._on_incident,
            queue="remediation-planners",
        )
        logger.info("RemediationPlanner subscribed to kubric.*.incident.*")

        try:
            while self._running:
                await asyncio.sleep(1)
        finally:
            await self._shutdown()

    async def _shutdown(self) -> None:
        self._running = False
        if self._subscription:
            try:
                await self._subscription.unsubscribe()
            except Exception:  # noqa: BLE001
                pass
        if self._db:
            await self._db.close()
        if self.nc and not self.nc.is_closed:
            await self.nc.drain()
            await self.nc.close()
        logger.info("RemediationPlanner stopped.")

    # ------------------------------------------------------------------
    # NATS message handler
    # ------------------------------------------------------------------

    async def _on_incident(self, msg: Any) -> None:
        try:
            incident: dict = json.loads(msg.data.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError) as exc:
            logger.error("Failed to decode incident message: %s", exc)
            return

        parts = msg.subject.split(".")
        tenant_id: str = parts[1] if len(parts) >= 3 else "default"
        incident["tenant_id"] = incident.get("tenant_id") or tenant_id

        try:
            steps = await asyncio.get_event_loop().run_in_executor(
                None, self._generate_plan, incident
            )
            plan: dict = {
                "id": str(uuid.uuid4()),
                "incident_id": incident.get("id") or incident.get("incident_id", "unknown"),
                "tenant_id": tenant_id,
                "steps": steps,
                "generated_at": datetime.now(timezone.utc).isoformat(),
                "incident_summary": str(incident.get("title") or incident.get("description", ""))[:500],
            }
            await self._save_plan(plan)
            await self._dispatch_plan(plan)
        except Exception as exc:  # noqa: BLE001
            logger.exception("Remediation planning failed for incident: %s", exc)

    # ------------------------------------------------------------------
    # Plan generation
    # ------------------------------------------------------------------

    def _generate_plan(self, incident: dict) -> list[dict]:
        """Call Anthropic to generate step-by-step remediation steps."""
        if not self._anthropic:
            logger.warning("Anthropic not configured; using static fallback plan.")
            return self._fallback_plan(incident)

        prompt = (
            "You are a Senior Incident Responder. Generate a detailed, "
            "actionable remediation plan for the following security incident.\n\n"
            f"Incident title: {incident.get('title', 'Unknown')}\n"
            f"Severity: {incident.get('severity', 'Unknown')}\n"
            f"Type: {incident.get('type', incident.get('class_name', 'Unknown'))}\n"
            f"Affected assets: {json.dumps(incident.get('assets', []))[:500]}\n"
            f"Indicators: {json.dumps(incident.get('indicators', [])[:5])}\n"
            f"Description: {str(incident.get('description', ''))[:800]}\n\n"
            "Return a JSON array of remediation steps. Each step must be an object with: "
            "step_number (int), title (str), description (str), priority (high/medium/low), "
            "owner (soc/sysadmin/management), estimated_minutes (int)."
        )

        message = self._anthropic.messages.create(
            model="claude-3-haiku-20240307",
            max_tokens=1500,
            temperature=0.1,
            messages=[{"role": "user", "content": prompt}],
        )
        raw_text: str = message.content[0].text.strip()

        # Extract JSON from markdown code block if present
        if "```" in raw_text:
            import re
            match = re.search(r"```(?:json)?\s*([\s\S]+?)\s*```", raw_text)
            raw_text = match.group(1) if match else raw_text

        try:
            steps: list[dict] = json.loads(raw_text)
            if not isinstance(steps, list):
                raise ValueError("Expected a JSON array from the model.")
            return steps
        except (json.JSONDecodeError, ValueError) as exc:
            logger.warning("Failed to parse AI remediation steps: %s", exc)
            return self._fallback_plan(incident)

    @staticmethod
    def _fallback_plan(incident: dict) -> list[dict]:
        return [
            {"step_number": 1, "title": "Containment", "description": "Isolate affected systems from the network.", "priority": "high", "owner": "soc", "estimated_minutes": 30},
            {"step_number": 2, "title": "Evidence Collection", "description": "Collect forensic artefacts before remediation.", "priority": "high", "owner": "soc", "estimated_minutes": 60},
            {"step_number": 3, "title": "Eradication", "description": "Remove malicious files, processes, and persistence mechanisms.", "priority": "high", "owner": "sysadmin", "estimated_minutes": 120},
            {"step_number": 4, "title": "Recovery", "description": "Restore systems from clean backups and verify integrity.", "priority": "medium", "owner": "sysadmin", "estimated_minutes": 240},
            {"step_number": 5, "title": "Post-Incident Review", "description": "Conduct lessons-learned review and update playbooks.", "priority": "low", "owner": "management", "estimated_minutes": 60},
        ]

    # ------------------------------------------------------------------
    # Persistence & dispatch
    # ------------------------------------------------------------------

    async def _save_plan(self, plan: dict) -> None:
        assert self._db is not None
        async with self._db.acquire() as conn:
            await conn.execute(
                """
                INSERT INTO remediation_plans (id, incident_id, tenant_id, plan_steps, status)
                VALUES ($1, $2, $3, $4::jsonb, 'pending')
                """,
                plan["id"],
                plan["incident_id"],
                plan["tenant_id"],
                json.dumps(plan["steps"]),
            )
        logger.info("Remediation plan %s saved to database.", plan["id"])

    async def _dispatch_plan(self, plan: dict) -> None:
        assert self.nc is not None
        subject = f"kubric.{plan['tenant_id']}.remediation.plan"
        payload = json.dumps(plan, default=str).encode("utf-8")
        await self.nc.publish(subject, payload)

        if self._db:
            async with self._db.acquire() as conn:
                await conn.execute(
                    "UPDATE remediation_plans SET status='published', published_at=NOW() WHERE id=$1",
                    plan["id"],
                )
        logger.info("Remediation plan %s dispatched to %s.", plan["id"], subject)


if __name__ == "__main__":
    planner = RemediationPlanner()
    asyncio.run(planner.start())
