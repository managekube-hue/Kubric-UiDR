"""
K-KAI-WF-TEMP-003_celery_tasks.py
Celery task queue for async KAI background jobs.
"""

import logging
import os
from typing import Optional

from celery import Celery
from celery.utils.log import get_task_logger

logger = get_task_logger(__name__)

# ---------------------------------------------------------------------------
# Celery app configuration
# ---------------------------------------------------------------------------

BROKER_URL = os.environ.get("CELERY_BROKER_URL", "redis://redis:6379/0")
RESULT_URL = os.environ.get("CELERY_RESULT_URL", "redis://redis:6379/1")
NATS_URL = os.environ.get("NATS_URL", "nats://nats:4222")

app = Celery("kubric-kai", broker=BROKER_URL, backend=RESULT_URL)

app.conf.update(
    task_serializer="json",
    result_serializer="json",
    accept_content=["json"],
    task_track_started=True,
    task_acks_late=True,
    worker_prefetch_multiplier=1,
    result_expires=86400,  # 24 hours
    task_routes={
        "kai.triage_alert_task": {"queue": "triage"},
        "kai.run_cortex_analysis": {"queue": "enrichment"},
        "kai.run_velociraptor_hunt": {"queue": "hunting"},
        "kai.generate_remediation_plan": {"queue": "remediation"},
        "kai.score_tenant_risk": {"queue": "risk"},
    },
)


# ---------------------------------------------------------------------------
# NATS publisher helper (fire-and-forget)
# ---------------------------------------------------------------------------

def _publish_to_nats(subject: str, payload: dict) -> None:
    """Synchronously publish a message to NATS (using nats-py sync wrapper)."""
    import asyncio
    import json as _json
    try:
        import nats as nats_py
        async def _pub():
            nc = await nats_py.connect(NATS_URL)
            await nc.publish(subject, _json.dumps(payload).encode())
            await nc.flush()
            await nc.close()
        asyncio.run(_pub())
    except Exception as exc:
        logger.warning("NATS publish failed — subject=%s err=%s", subject, exc)


# ---------------------------------------------------------------------------
# Task: triage_alert_task
# ---------------------------------------------------------------------------

@app.task(
    bind=True,
    name="kai.triage_alert_task",
    max_retries=3,
    default_retry_delay=60,
    time_limit=300,
)
def triage_alert_task(self, alert: dict) -> dict:
    """
    Run the TriageAgent on a single OCSF alert dict.

    Publishes result to kubric.{tenant_id}.alert.triaged on NATS.
    """
    try:
        from K_KAI_CP_CREWAI_PERSONAS.K_KAI_TRIAGE.K_KAI_TR_001_triage_agent import TriageAgent  # type: ignore
        tenant_id = alert.get("tenant_id", "unknown")
        agent = TriageAgent()
        result = agent.triage(alert)
        _publish_to_nats(f"kubric.{tenant_id}.alert.triaged", result)
        return result
    except Exception as exc:
        logger.error("triage_alert_task failed: %s", exc)
        raise self.retry(exc=exc)


# ---------------------------------------------------------------------------
# Task: run_cortex_analysis
# ---------------------------------------------------------------------------

@app.task(
    bind=True,
    name="kai.run_cortex_analysis",
    max_retries=3,
    default_retry_delay=60,
    time_limit=600,
)
def run_cortex_analysis(self, observable: str, type: str, tenant_id: str) -> dict:
    """
    Submit an observable to Cortex for analysis and return the report.

    Publishes result to kubric.{tenant_id}.cortex.result on NATS.
    """
    import requests

    cortex_url = os.environ.get("CORTEX_URL", "http://cortex:9001")
    cortex_key = os.environ.get("CORTEX_API_KEY", "")

    try:
        resp = requests.post(
            f"{cortex_url}/api/analyzer/_search",
            headers={"Authorization": f"Bearer {cortex_key}"},
            json={"query": {"_field": "dataTypeList", "_value": type}},
            timeout=30,
        )
        resp.raise_for_status()
        analyzers = resp.json().get("data", [])[:3]  # pick top 3 analyzers

        jobs = []
        for analyzer in analyzers:
            job_resp = requests.post(
                f"{cortex_url}/api/analyzer/{analyzer['id']}/run",
                headers={"Authorization": f"Bearer {cortex_key}"},
                json={"data": observable, "dataType": type, "tlp": 2},
                timeout=30,
            )
            if job_resp.ok:
                jobs.append(job_resp.json())

        result = {
            "observable": observable,
            "type": type,
            "tenant_id": tenant_id,
            "jobs": jobs,
        }
        _publish_to_nats(f"kubric.{tenant_id}.cortex.result", result)
        return result
    except Exception as exc:
        logger.error("run_cortex_analysis failed: %s", exc)
        raise self.retry(exc=exc)


# ---------------------------------------------------------------------------
# Task: run_velociraptor_hunt
# ---------------------------------------------------------------------------

@app.task(
    bind=True,
    name="kai.run_velociraptor_hunt",
    max_retries=3,
    default_retry_delay=60,
    time_limit=1200,
)
def run_velociraptor_hunt(self, artifact: str, client_ids: list) -> list:
    """
    Launch a Velociraptor artifact hunt against the specified client IDs.

    Publishes results to kubric.hunt.results on NATS.
    """
    import requests

    velo_url = os.environ.get("VELOCIRAPTOR_API_URL", "http://velociraptor:8001")
    velo_token = os.environ.get("VELOCIRAPTOR_TOKEN", "")

    try:
        resp = requests.post(
            f"{velo_url}/api/v1/StartCollectorHunt",
            headers={"Authorization": f"Bearer {velo_token}"},
            json={
                "artifacts": [artifact],
                "client_ids": client_ids,
                "expires": 3600,
            },
            timeout=60,
        )
        resp.raise_for_status()
        hunt_data = resp.json()
        results = [{"hunt_id": hunt_data.get("hunt_id"), "artifact": artifact, "client_ids": client_ids}]
        _publish_to_nats("kubric.hunt.results", results[0])
        return results
    except Exception as exc:
        logger.error("run_velociraptor_hunt failed: %s", exc)
        raise self.retry(exc=exc)


# ---------------------------------------------------------------------------
# Task: generate_remediation_plan
# ---------------------------------------------------------------------------

@app.task(
    bind=True,
    name="kai.generate_remediation_plan",
    max_retries=3,
    default_retry_delay=60,
    time_limit=300,
)
def generate_remediation_plan(self, incident: dict) -> dict:
    """
    Generate a prioritised remediation plan for an incident using KAI reasoning.

    Publishes result to kubric.{tenant_id}.remediation.plan on NATS.
    """
    try:
        tenant_id = incident.get("tenant_id", "unknown")
        from K_KAI_ML_005_anthropic_long_context import AnthropicLongContext  # type: ignore
        client = AnthropicLongContext()
        analysis = client.analyze_incident(incident)

        plan = {
            "incident_id": incident.get("incident_id"),
            "tenant_id": tenant_id,
            "analysis": analysis,
            "steps": _extract_steps(analysis),
            "status": "pending",
        }
        _publish_to_nats(f"kubric.{tenant_id}.remediation.plan", plan)
        return plan
    except Exception as exc:
        logger.error("generate_remediation_plan failed: %s", exc)
        raise self.retry(exc=exc)


def _extract_steps(analysis_text: str) -> list:
    """Extract numbered steps from analyst text."""
    import re
    lines = analysis_text.split("\n")
    steps = []
    for line in lines:
        match = re.match(r"^\s*\d+\.\s+(.+)", line)
        if match:
            steps.append(match.group(1).strip())
    return steps[:10]  # cap at 10 steps


# ---------------------------------------------------------------------------
# Task: score_tenant_risk
# ---------------------------------------------------------------------------

@app.task(
    bind=True,
    name="kai.score_tenant_risk",
    max_retries=3,
    default_retry_delay=60,
    time_limit=120,
)
def score_tenant_risk(self, tenant_id: str) -> dict:
    """
    Compute a composite risk score for a tenant and publish via NATS.
    """
    try:
        import random
        # Placeholder scoring logic — replace with real model call
        score = round(random.uniform(0.0, 1.0), 4)
        tier = "critical" if score >= 0.8 else "high" if score >= 0.6 else "medium" if score >= 0.4 else "low"
        result = {
            "tenant_id": tenant_id,
            "risk_score": score,
            "risk_tier": tier,
        }
        _publish_to_nats(f"kubric.{tenant_id}.risk.score", result)
        return result
    except Exception as exc:
        logger.error("score_tenant_risk failed: %s", exc)
        raise self.retry(exc=exc)
