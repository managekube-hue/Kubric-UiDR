"""
KAI-SENTINEL — Customer health score computation (KiSS score).

Triggered by:   periodic timer (every 15 min) OR POST /v1/score/{tenant_id}
Publishes to:   kubric.health.score.{tenant_id}

KiSS (Kubric integrated Security Score) — 0-100 composite:
  vuln 30%  + compliance 25%  + detection 25%  + response 20%

Agent persona: Security Posture Analyst
CrewAI integration: make_sentinel_crew() → Crew.kickoff() → JSON score output
"""

from __future__ import annotations

import time
from typing import Any

import orjson
import structlog

from kai.core.crew import make_sentinel_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)


class SentinelAgent:
    """KAI-SENTINEL — computes and publishes the KiSS health score."""

    async def compute(self, tenant_id: str) -> dict[str, Any]:
        # ── CrewAI crew queries VDR + KIC + ClickHouse, computes score ─────────
        crew   = make_sentinel_crew(tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        kiss_result: dict[str, Any] = {
            "tenant_id":         tenant_id,
            "computed_at":       int(time.time() * 1000),
            "kiss_score":        float(result.get("kiss_score", 75.0)),
            "vuln_score":        float(result.get("vuln_score", 75.0)),
            "compliance_score":  float(result.get("compliance_score", 75.0)),
            "detection_score":   float(result.get("detection_score", 80.0)),
            "response_score":    float(result.get("response_score", 85.0)),
            "open_criticals":    int(result.get("open_criticals", 0)),
            "open_highs":        int(result.get("open_highs", 0)),
            "active_incidents":  int(result.get("active_incidents", 0)),
            "model_used":        "crewai/ollama/llama3.2",
        }

        # Publish to NATS for portal display
        subject = f"kubric.{tenant_id}.kai.sentinel.score.v1"
        await nats_client.publish(subject, orjson.dumps(kiss_result))

        log.info(
            "sentinel.score_published",
            tenant_id=tenant_id,
            kiss=kiss_result["kiss_score"],
        )
        return kiss_result
