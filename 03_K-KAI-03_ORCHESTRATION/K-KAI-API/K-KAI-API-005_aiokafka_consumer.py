"""
K-KAI-API-005: aiokafka Kafka consumer for security event ingestion.
Subscribes to kubric-events, kubric-alerts, kubric-metrics topics.
Routes consumed messages to KAI persona handlers based on topic and
OCSF class_uid.
"""

import asyncio
import logging
import os
from typing import Callable, Awaitable

import orjson
from aiokafka import AIOKafkaConsumer
from aiokafka.errors import KafkaConnectionError

logger = logging.getLogger("kai.kafka")

# ---------------------------------------------------------------------------
# OCSF class_uid routing map  (class_uid -> persona)
# ---------------------------------------------------------------------------
OCSF_CLASS_PERSONA: dict[int, str] = {
    1001: "TRIAGE",    # Security Finding
    2001: "ANALYST",   # Authentication Audit
    4001: "HUNTER",    # Network Activity
    4003: "HUNTER",    # DNS Activity
    4007: "HUNTER",    # FTP Activity
    6001: "RISK",      # Vulnerability Finding
    6002: "RISK",      # Compliance Finding
    1002: "SENTINEL",  # Malware Finding
    1003: "INVEST",    # Incident Finding
    2004: "KEEPER",    # Account Change
}

_TOPICS_DEFAULT = ["kubric-events", "kubric-alerts", "kubric-metrics"]

# Registry of persona handler callables: handler(tenant_id, event) -> None
_persona_handlers: dict[str, list[Callable[[str, dict], Awaitable[None]]]] = {}


def register_persona_handler(
    persona: str,
    handler: Callable[[str, dict], Awaitable[None]],
) -> None:
    """Register an async handler for a given persona."""
    _persona_handlers.setdefault(persona.upper(), []).append(handler)


# ---------------------------------------------------------------------------
# KAIKafkaConsumer
# ---------------------------------------------------------------------------
class KAIKafkaConsumer:
    """
    aiokafka-based consumer for KAI security event ingestion.
    Routes each message to the appropriate persona handler based on
    topic and OCSF class_uid.
    """

    def __init__(self) -> None:
        self._consumer: AIOKafkaConsumer | None = None
        self._running = False
        self._task: asyncio.Task | None = None

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------
    async def start(
        self,
        topics: list[str] | None = None,
        group_id: str = "kai-consumer-group",
    ) -> None:
        """Start consuming from the given topics."""
        topics = topics or _TOPICS_DEFAULT
        brokers = os.environ.get("KAFKA_BROKERS", "localhost:9092")

        self._consumer = AIOKafkaConsumer(
            *topics,
            bootstrap_servers=brokers,
            group_id=group_id,
            auto_offset_reset="latest",
            enable_auto_commit=True,
            value_deserializer=lambda m: m,  # raw bytes; parse in handler
        )

        # Retry connection
        for attempt in range(1, 6):
            try:
                await self._consumer.start()
                logger.info(
                    "Kafka consumer started | group=%s topics=%s brokers=%s",
                    group_id,
                    topics,
                    brokers,
                )
                break
            except KafkaConnectionError as exc:
                wait = min(2 ** attempt, 30)
                logger.warning(
                    "Kafka connect attempt %d/5 failed: %s — retrying in %ds",
                    attempt,
                    exc,
                    wait,
                )
                if attempt == 5:
                    raise
                await asyncio.sleep(wait)

        self._running = True
        self._task = asyncio.create_task(self._consume_loop())

    async def stop(self) -> None:
        """Gracefully stop the consumer."""
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        if self._consumer:
            await self._consumer.stop()
            logger.info("Kafka consumer stopped")

    # ------------------------------------------------------------------
    # Consume loop
    # ------------------------------------------------------------------
    async def _consume_loop(self) -> None:
        assert self._consumer is not None
        async for msg in self._consumer:
            if not self._running:
                break
            try:
                await self._process_message(msg)
            except Exception as exc:
                logger.error(
                    "Error processing Kafka message topic=%s partition=%d offset=%d: %s",
                    msg.topic,
                    msg.partition,
                    msg.offset,
                    exc,
                )

    # ------------------------------------------------------------------
    # Per-message routing
    # ------------------------------------------------------------------
    async def _process_message(self, msg) -> None:
        try:
            event: dict = orjson.loads(msg.value)
        except Exception as exc:
            logger.warning("Failed to deserialize Kafka message: %s", exc)
            return

        tenant_id: str = event.get("tenant_id") or event.get("metadata", {}).get("tenant_id", "unknown")
        class_uid: int = event.get("class_uid", 0)
        topic: str = msg.topic

        # Determine target persona
        persona = OCSF_CLASS_PERSONA.get(class_uid)
        if persona is None:
            # Fallback by topic
            if "alert" in topic:
                persona = "TRIAGE"
            elif "metric" in topic:
                persona = "ANALYST"
            else:
                persona = "TRIAGE"

        handlers = _persona_handlers.get(persona, [])
        if not handlers:
            logger.debug(
                "No handler registered for persona=%s class_uid=%d topic=%s",
                persona,
                class_uid,
                topic,
            )
            return

        for handler in handlers:
            try:
                await handler(tenant_id, event)
            except Exception as exc:
                logger.error(
                    "Persona handler %s raised: %s", persona, exc
                )


# ---------------------------------------------------------------------------
# Module-level singleton
# ---------------------------------------------------------------------------
_consumer_singleton = KAIKafkaConsumer()


async def start_consuming(
    topics: list[str] | None = None,
    group_id: str = "kai-consumer-group",
) -> None:
    await _consumer_singleton.start(topics, group_id)


async def stop_consuming() -> None:
    await _consumer_singleton.stop()
