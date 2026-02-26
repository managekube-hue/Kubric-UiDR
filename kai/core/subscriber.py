"""
KAI NATS subscriber loop.

Runs as a background asyncio task (started in FastAPI lifespan).
Subscribes to the full kubric.* subject tree and dispatches each message
to the appropriate KAI agent based on the subject prefix.

Subject taxonomy (mirrors KUBRIC Implementation doc §19):

  kubric.edr.*           → KAI-TRIAGE
  kubric.ndr.*           → KAI-TRIAGE + KAI-FORESIGHT
  kubric.itdr.*          → KAI-TRIAGE
  kubric.vdr.*           → KAI-KEEPER
  kubric.grc.*           → KAI-KEEPER
  kubric.billing.*       → (future KAI-CLERK)
  kubric.health.*        → publish only (KAI-SENTINEL pushes here)
  kubric.ti.*            → (future SIDR TI ingestion)
  kubric.comm.*          → KAI-COMM
"""

from __future__ import annotations

import asyncio

import orjson
import structlog
from nats.aio.msg import Msg

from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

# Lazy imports of agents to avoid circular imports at module load time
_SUBJECT_DISPATCH: dict[str, str] = {
    "kubric.edr.":    "triage",
    "kubric.itdr.":   "triage",
    "kubric.ndr.":    "triage",   # triage handles NDR too; foresight runs separately
    "kubric.vdr.":    "keeper",
    "kubric.grc.":    "keeper",
    "kubric.comm.":   "comm",
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
    for prefix, agent in _SUBJECT_DISPATCH.items():
        if subject.startswith(prefix):
            return agent
    return None


async def _dispatch(agent_name: str, subject: str, event: dict) -> None:  # type: ignore[type-arg]
    # Import agents lazily — avoids top-level import of crewai at startup
    if agent_name == "triage":
        from kai.agents.triage import TriageAgent  # noqa: PLC0415
        await TriageAgent().handle(subject=subject, event=event)

    elif agent_name == "keeper":
        from kai.agents.keeper import KeeperAgent  # noqa: PLC0415
        await KeeperAgent().handle(subject=subject, event=event)

    elif agent_name == "comm":
        from kai.agents.comm import CommAgent  # noqa: PLC0415
        await CommAgent().handle(subject=subject, event=event)


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
