"""
KAI NATS subscriber loop.

Runs as a background asyncio task (started in FastAPI lifespan).
Subscribes to the full kubric.> subject tree and dispatches each message
to the appropriate KAI agent based on the category segment.

Subject format: kubric.{tenant_id}.{category}.{event}.v1

Category dispatch (all 13 KAI personas):

  edr / ndr / itdr / cdr / sdr / adr / ddr  → KAI-TRIAGE
  vdr / grc                                  → KAI-KEEPER
  comm                                       → KAI-COMM
  billing                                    → KAI-BILL
  agent                                      → KAI-HOUSE
  ti                                         → KAI-ANALYST
  kai  (sub-route by segment 3)              → hunter / deploy / invest / risk / simulate
"""

from __future__ import annotations

import asyncio

import orjson
import structlog
from nats.aio.msg import Msg

from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

# ─── Category-based dispatch (segment index 2 of subject) ─────────────────
# Subject: kubric.{tenant_id}.{category}.{event}...
_CATEGORY_DISPATCH: dict[str, str] = {
    # Detection events → KAI-TRIAGE
    "edr":  "triage",
    "ndr":  "triage",
    "itdr": "triage",
    "cdr":  "triage",
    "sdr":  "triage",
    "adr":  "triage",
    "ddr":  "triage",
    # Vulnerability / compliance → KAI-KEEPER
    "vdr":  "keeper",
    "grc":  "keeper",
    # Communication routing → KAI-COMM
    "comm": "comm",
    # Billing events → KAI-BILL
    "billing": "bill",
    # Agent lifecycle → KAI-HOUSE
    "agent": "house",
    # Threat-intel feeds → KAI-ANALYST
    "ti": "analyst",
    # KAI sub-namespace → secondary dispatch on segment 3
    "kai": "_kai_sub",
}

# Secondary dispatch for kubric.{tenant_id}.kai.{persona}.* subjects
_KAI_SUB_DISPATCH: dict[str, str] = {
    "hunt":     "hunter",
    "hunter":   "hunter",
    "deploy":   "deploy",
    "invest":   "invest",
    "risk":     "risk",
    "simulate": "simulate",
    "analyst":  "analyst",
    "foresight": "foresight",
}


async def _route_message(msg: Msg) -> None:
    """Decode and dispatch a NATS message to the right KAI agent."""
    subject = msg.subject
    try:
        event: dict = orjson.loads(msg.data)  # type: ignore[assignment]
    except Exception:
        log.warning("subscriber.decode_error", subject=subject)
        return

    agent_name = _resolve_agent(subject)
    if agent_name is None:
        return

    try:
        await _dispatch(agent_name, subject, event)
    except Exception as exc:
        log.error("subscriber.dispatch_error", subject=subject, agent=agent_name, error=str(exc))


def _resolve_agent(subject: str) -> str | None:
    """Extract category from segment index 2, look up dispatch table."""
    parts = subject.split(".")
    if len(parts) < 3:
        return None
    category = parts[2]  # kubric.{tenant_id}.{category}.*
    agent = _CATEGORY_DISPATCH.get(category)
    if agent == "_kai_sub":
        # Secondary dispatch: kubric.{tenant_id}.kai.{persona}.*
        if len(parts) < 4:
            return None
        return _KAI_SUB_DISPATCH.get(parts[3])
    return agent


async def _dispatch(agent_name: str, subject: str, event: dict) -> None:  # type: ignore[type-arg]
    """Import agents lazily — avoids top-level import of crewai at startup."""
    if agent_name == "triage":
        from kai.agents.triage import TriageAgent  # noqa: PLC0415
        await TriageAgent().handle(subject=subject, event=event)

    elif agent_name == "keeper":
        from kai.agents.keeper import KeeperAgent  # noqa: PLC0415
        await KeeperAgent().handle(subject=subject, event=event)

    elif agent_name == "comm":
        from kai.agents.comm import CommAgent  # noqa: PLC0415
        await CommAgent().handle(subject=subject, event=event)

    elif agent_name == "bill":
        from kai.agents.bill import BillAgent  # noqa: PLC0415
        await BillAgent().handle(subject=subject, event=event)

    elif agent_name == "house":
        from kai.agents.house import HouseAgent  # noqa: PLC0415
        await HouseAgent().handle(subject=subject, event=event)

    elif agent_name == "analyst":
        from kai.agents.analyst import AnalystAgent  # noqa: PLC0415
        await AnalystAgent().handle(subject=subject, event=event)

    elif agent_name == "hunter":
        from kai.agents.hunter import HunterAgent  # noqa: PLC0415
        await HunterAgent().handle(subject=subject, event=event)

    elif agent_name == "deploy":
        from kai.agents.deploy import DeployAgent  # noqa: PLC0415
        await DeployAgent().handle(subject=subject, event=event)

    elif agent_name == "invest":
        from kai.agents.invest import InvestAgent  # noqa: PLC0415
        await InvestAgent().handle(subject=subject, event=event)

    elif agent_name == "risk":
        from kai.agents.risk import RiskAgent  # noqa: PLC0415
        await RiskAgent().handle(subject=subject, event=event)

    elif agent_name == "simulate":
        from kai.agents.simulate import SimulateAgent  # noqa: PLC0415
        await SimulateAgent().handle(subject=subject, event=event)

    elif agent_name == "foresight":
        from kai.agents.foresight import ForesightAgent  # noqa: PLC0415
        await ForesightAgent().handle(subject=subject, event=event)


async def start_subscriber() -> list[object]:
    """
    Subscribe to all kubric.* patterns.
    Returns the subscription handles (call unsubscribe on shutdown).
    """
    subs: list[object] = []

    # Single wildcard subscription — one handler routes everything
    sub = await nats_client.subscribe("kubric.>", cb=_route_message)
    if sub is not None:
        subs.append(sub)
        log.info("subscriber.started", subject="kubric.>")
    else:
        log.warning("subscriber.nats_unavailable")

    return subs


async def stop_subscriber(subs: list[object]) -> None:
    for sub in subs:
        try:
            await sub.unsubscribe()  # type: ignore[union-attr]
        except Exception:
            pass
    log.info("subscriber.stopped")
