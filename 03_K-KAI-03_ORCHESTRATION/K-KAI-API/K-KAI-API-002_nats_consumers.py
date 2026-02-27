"""
K-KAI-API-002: NATS JetStream pull consumer worker for the KAI Orchestration Layer.

Wires all 13 CrewAI persona agents as durable JetStream pull consumers.

Subject pattern   : kubric.<tenant_id>.agent.<persona>.trigger
Result subject    : kubric.<tenant_id>.agent.<persona>.result
Stream            : KUBRIC_AGENTS  (covers kubric.*.agent.*.trigger)

Personas handled:
  triage, analyst, hunter, keeper, risk, invest,
  sentinel, foresight, house, bill, comm, deploy, simulate

Ack policy:
  - Ack after successful handler return
  - Nak(delay=30s) on any handler exception
  - Nak immediately on JSON decode failure

Imports are lazy (inside dispatch) to avoid circular-import issues when
each persona module itself imports asyncpg / CrewAI / torch at module level.

Requirements:
  nats-py>=2.3.0, orjson, Python 3.11+
"""

from __future__ import annotations

import asyncio
import importlib
import logging
import os
import sys
from datetime import datetime, timezone
from typing import Any

import nats
import nats.errors
import orjson
from nats.aio.client import Client as NATSConnection
from nats.js import JetStreamContext
from nats.js.client import PullSubscription

logger = logging.getLogger("kai.nats_consumers")

# ---------------------------------------------------------------------------
# Path resolution
# ---------------------------------------------------------------------------
# This file lives at:
#   <repo>/03_K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_nats_consumers.py
# Persona packages live at:
#   <repo>/03_K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/<K-KAI-XXXXX>/

_THIS_DIR: str = os.path.dirname(os.path.abspath(__file__))
_ORCHESTRATION_ROOT: str = os.path.dirname(_THIS_DIR)
_PERSONAS_ROOT: str = os.path.join(_ORCHESTRATION_ROOT, "K-KAI-CP_CREWAI_PERSONAS")

# ---------------------------------------------------------------------------
# Persona registry
# Each entry: persona_key -> (package_subfolder, module_name, function_name)
#
# The module_name is the Python-importable name once the package_subfolder
# has been inserted into sys.path.  File names that use hyphens are exposed
# as underscore equivalents (Python replaces '-' with '_' for the module
# finder when the directory is on sys.path).
# ---------------------------------------------------------------------------
_PERSONA_REGISTRY: dict[str, tuple[str, str, str]] = {
    "triage":   ("K-KAI-TRIAGE",    "K_KAI_TRIAGE",    "triage_alert"),
    "analyst":  ("K-KAI-ANALYST",   "K_KAI_ANALYST",   "analyze_incident"),
    "hunter":   ("K-KAI-HUNTER",    "K_KAI_HUNTER",    "hunt_threats"),
    "keeper":   ("K-KAI-KEEPER",    "K_KAI_KEEPER",    "manage_knowledge"),
    "risk":     ("K-KAI-RISK",      "K_KAI_RISK",      "assess_risk"),
    "invest":   ("K-KAI-INVEST",    "K_KAI_INVEST",    "investigate"),
    "sentinel": ("K-KAI-SENTINEL",  "K_KAI_SENTINEL",  "monitor"),
    "foresight":("K-KAI-FORESIGHT", "K_KAI_FORESIGHT", "forecast"),
    "house":    ("K-KAI-HOUSE",     "K_KAI_HOUSE",     "maintain"),
    "bill":     ("K-KAI-BILL",      "K_KAI_BILL",      "process_billing"),
    "comm":     ("K-KAI-COMM",      "K_KAI_COMM",      "dispatch_comm"),
    "deploy":   ("K-KAI-DEPLOY",    "K_KAI_DEPLOY",    "execute_deploy"),
    "simulate": ("K-KAI-SIMULATE",  "K_KAI_SIMULATE",  "run_simulation"),
}

# All 13 persona keys in canonical order
ALL_PERSONAS: tuple[str, ...] = tuple(_PERSONA_REGISTRY)

# JetStream stream name that covers all trigger subjects
STREAM_NAME: str = "KUBRIC_AGENTS"

# Nak delay in seconds when a handler raises an exception
NAK_DELAY_SECONDS: int = 30


# ---------------------------------------------------------------------------
# Lazy handler loader
# ---------------------------------------------------------------------------

def _load_handler(persona: str) -> Any:
    """
    Lazily import the handler function for *persona*.

    Adds the persona's package directory to sys.path on first call so that
    the module's own relative imports (e.g. ``from K_KAI_TR_002_... import``)
    resolve correctly.

    Returns the callable.
    Raises ImportError or AttributeError if the module or function is absent.
    """
    pkg_subdir, module_name, func_name = _PERSONA_REGISTRY[persona]
    pkg_path: str = os.path.join(_PERSONAS_ROOT, pkg_subdir)

    # Insert once so that sibling imports inside the persona package work.
    if pkg_path not in sys.path:
        sys.path.insert(0, pkg_path)

    mod = importlib.import_module(module_name)
    handler = getattr(mod, func_name)
    logger.debug(
        "Loaded handler %s.%s for persona '%s'",
        module_name,
        func_name,
        persona,
    )
    return handler


# ---------------------------------------------------------------------------
# NatsConsumerWorker
# ---------------------------------------------------------------------------

class NatsConsumerWorker:
    """
    Routes inbound JetStream trigger messages to the correct persona handler.

    Usage (managed externally by the runner):

        worker = NatsConsumerWorker(js=js_ctx, nc=nc_conn)
        sub = await js_ctx.pull_subscribe(
            subject="kubric.*.agent.triage.trigger",
            durable="kubric-agent-triage",
            stream=STREAM_NAME,
        )
        await worker.consume_loop("triage", sub, shutdown_event)
    """

    def __init__(self, js: JetStreamContext, nc: NATSConnection) -> None:
        self._js = js
        self._nc = nc
        # Cache loaded handlers to avoid repeated importlib lookups.
        self._handler_cache: dict[str, Any] = {}

    # ------------------------------------------------------------------
    # Public: per-persona consume loop (run as an asyncio task)
    # ------------------------------------------------------------------

    async def consume_loop(
        self,
        persona: str,
        sub: PullSubscription,
        shutdown: asyncio.Event,
        *,
        fetch_timeout: float = 5.0,
        batch_size: int = 1,
    ) -> None:
        """
        Pull messages one-at-a-time from *sub* until *shutdown* is set.

        Parameters
        ----------
        persona:
            One of the 13 registered persona keys.
        sub:
            Durable pull subscription returned by ``js.pull_subscribe()``.
        shutdown:
            Async event; loop exits cleanly when it is set.
        fetch_timeout:
            Seconds to wait for the next batch before looping.
        batch_size:
            Number of messages to fetch per round (keep at 1 for ordered
            per-tenant processing; raise for high-throughput personas).
        """
        logger.info("Consumer loop STARTED  persona=%s", persona)
        while not shutdown.is_set():
            try:
                msgs = await sub.fetch(batch_size, timeout=fetch_timeout)
            except nats.errors.TimeoutError:
                # No messages within the window; poll again.
                continue
            except Exception as exc:  # noqa: BLE001
                logger.error(
                    "fetch() error persona=%s err=%s — backing off 2s",
                    persona,
                    exc,
                )
                await asyncio.sleep(2)
                continue

            for msg in msgs:
                await self._process_message(msg, persona)

        logger.info("Consumer loop STOPPED  persona=%s", persona)

    # ------------------------------------------------------------------
    # Internal: single-message dispatch
    # ------------------------------------------------------------------

    async def _process_message(self, msg: Any, persona: str) -> None:
        """
        Decode, validate, dispatch, ack/nak a single JetStream message.

        Flow:
          1. Decode orjson → dict payload
          2. Extract tenant_id / correlation_id from payload
          3. Lazy-load the persona handler
          4. Await handler(payload)
          5. Publish result to kubric.<tenant_id>.agent.<persona>.result
          6. Ack the message
          On any exception: nak with 30 s delay.
        """
        subject: str = msg.subject

        # --- 1. Decode -------------------------------------------------------
        try:
            payload: dict = orjson.loads(msg.data)
        except Exception as exc:
            logger.warning(
                "JSON decode error subject=%s err=%s — naked immediately",
                subject,
                exc,
            )
            await msg.nak()
            return

        tenant_id: str = payload.get("tenant_id", "unknown")
        correlation_id: str = payload.get("correlation_id", "")
        priority: int = int(payload.get("priority", 3))

        logger.info(
            "DISPATCH  subject=%s  tenant=%s  persona=%s  priority=%d  cid=%s",
            subject,
            tenant_id,
            persona,
            priority,
            correlation_id,
        )

        # --- 2. Lazy-load handler --------------------------------------------
        handler = self._get_cached_handler(persona)
        if handler is None:
            logger.error(
                "No handler available for persona=%s — naking (delay=%ds)",
                persona,
                NAK_DELAY_SECONDS,
            )
            await _nak_with_delay(msg, NAK_DELAY_SECONDS)
            return

        # --- 3. Execute handler ----------------------------------------------
        started_at = datetime.now(timezone.utc)
        try:
            result: Any = await _invoke_handler(handler, payload)
        except Exception as exc:  # noqa: BLE001
            logger.error(
                "Handler error persona=%s tenant=%s cid=%s err=%s — naking (delay=%ds)",
                persona,
                tenant_id,
                correlation_id,
                exc,
                NAK_DELAY_SECONDS,
                exc_info=True,
            )
            await _nak_with_delay(msg, NAK_DELAY_SECONDS)
            return

        elapsed_ms: float = (
            datetime.now(timezone.utc) - started_at
        ).total_seconds() * 1000

        # --- 4. Publish result -----------------------------------------------
        result_subject: str = f"kubric.{tenant_id}.agent.{persona}.result"
        result_envelope: dict = {
            "tenant_id": tenant_id,
            "persona": persona,
            "correlation_id": correlation_id,
            "trigger_subject": subject,
            "result": result if isinstance(result, (dict, list)) else {"output": str(result)},
            "elapsed_ms": round(elapsed_ms, 2),
            "completed_at": datetime.now(timezone.utc).isoformat(),
        }

        try:
            await self._nc.publish(
                result_subject,
                orjson.dumps(result_envelope),
            )
            logger.debug(
                "Result published subject=%s elapsed_ms=%.1f",
                result_subject,
                elapsed_ms,
            )
        except Exception as exc:  # noqa: BLE001
            logger.warning(
                "Failed to publish result subject=%s err=%s",
                result_subject,
                exc,
            )
            # Do not nak — we still processed the message successfully.

        # --- 5. Ack ----------------------------------------------------------
        await msg.ack()
        logger.info(
            "ACK       subject=%s  tenant=%s  persona=%s  elapsed_ms=%.1f",
            subject,
            tenant_id,
            persona,
            elapsed_ms,
        )

    # ------------------------------------------------------------------
    # Handler cache
    # ------------------------------------------------------------------

    def _get_cached_handler(self, persona: str) -> Any | None:
        """Return the cached handler callable, loading it on first access."""
        if persona in self._handler_cache:
            return self._handler_cache[persona]
        try:
            handler = _load_handler(persona)
            self._handler_cache[persona] = handler
            return handler
        except (ImportError, AttributeError, ModuleNotFoundError) as exc:
            logger.error(
                "Cannot load handler for persona=%s: %s",
                persona,
                exc,
            )
            return None

    # ------------------------------------------------------------------
    # Convenience: build a subject for a given persona
    # ------------------------------------------------------------------

    @staticmethod
    def trigger_subject(persona: str) -> str:
        """Return the wildcard trigger subject for *persona*."""
        return f"kubric.*.agent.{persona}.trigger"

    @staticmethod
    def durable_name(persona: str) -> str:
        """Return the JetStream durable consumer name for *persona*."""
        return f"kubric-agent-{persona}"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

async def _invoke_handler(handler: Any, payload: dict) -> Any:
    """
    Call *handler* with *payload*.

    Supports both coroutine functions and plain callables.  Plain callables
    are offloaded to the default executor so they do not block the event loop.
    """
    if asyncio.iscoroutinefunction(handler):
        return await handler(payload)
    loop = asyncio.get_running_loop()
    return await loop.run_in_executor(None, handler, payload)


async def _nak_with_delay(msg: Any, delay_seconds: int) -> None:
    """Negative-ack *msg* with a re-delivery delay."""
    try:
        await msg.nak(delay=delay_seconds)
    except Exception as exc:  # noqa: BLE001
        logger.debug("nak() failed (message may already be acked): %s", exc)
