"""
KAI-FORESIGHT — Predictive risk modelling and network baseline.

Runs as:   periodic async task (every 30 min by default)
Reads:     ClickHouse via query_recent_alerts @tool + get_vdr_summary @tool
Publishes: kubric.kai.foresight.risk.{tenant_id} (per-tenant risk forecast)

Agent persona: Threat Intelligence Analyst
CrewAI integration: make_foresight_crew() → Crew.kickoff() → JSON risk score

Layer 3 will replace the heuristic baseline with a trained LSTM model (HIKARI-2021).
"""

from __future__ import annotations

import asyncio
import os
import time
from typing import Any

import orjson
import structlog

from kai.core.crew import make_foresight_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

_FORESIGHT_INTERVAL_SEC = int(os.getenv("KUBRIC_FORESIGHT_INTERVAL", "1800"))  # 30 min


class ForesightAgent:
    """KAI-FORESIGHT — predictive risk scoring via CrewAI TI Analyst persona."""

    async def run_once(self, tenant_id: str) -> dict[str, Any]:
        # ── CrewAI crew queries ClickHouse + VDR, computes risk score ─────────
        crew   = make_foresight_crew(tenant_id)
        output = crew.kickoff()
        result = parse_crew_output(output)

        forecast: dict[str, Any] = {
            "tenant_id":         tenant_id,
            "computed_at":       int(time.time() * 1000),
            "risk_score":        float(result.get("risk_score", 20.0)),
            "alert_velocity":    float(result.get("alert_velocity", 0.0)),
            "vuln_density":      float(result.get("vuln_density", 0.0)),
            "lateral_movement":  bool(result.get("lateral_movement", False)),
            "data_exfil_signal": bool(result.get("data_exfil_signal", False)),
            "top_risk_factors":  result.get("top_risk_factors", []),
            "recommendations":   result.get("recommendations", []),
            "model_used":        "crewai/ollama/llama3.2",
            "horizon_hours":     24,
        }

        subject = f"kubric.kai.foresight.risk.{tenant_id}"
        await nats_client.publish(subject, orjson.dumps(forecast))
        log.info("foresight.published", tenant_id=tenant_id, risk=forecast["risk_score"])
        return forecast


async def run_foresight_loop(tenant_ids: list[str]) -> None:
    """
    Background task: compute forecasts for all known tenants periodically.
    tenant_ids seeded at startup from KUBRIC_KNOWN_TENANTS env var.
    Layer 3 will query this list from PostgreSQL tenants table dynamically.
    """
    agent = ForesightAgent()
    while True:
        for tid in tenant_ids:
            try:
                await agent.run_once(tid)
            except Exception as exc:
                log.error("foresight.loop_error", tenant_id=tid, error=str(exc))
        await asyncio.sleep(_FORESIGHT_INTERVAL_SEC)
