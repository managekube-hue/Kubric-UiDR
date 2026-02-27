"""
K-KAI-API-006: nats.py JetStream client wrapper for KAI persona message bus.
Singleton pattern with auto-reconnect and orjson serialization.
"""

import asyncio
import logging
import os
from typing import Any, Callable, Awaitable

import nats
from nats.aio.client import Client as NATSConnection
from nats.js import JetStreamContext
import orjson

logger = logging.getLogger("kai.nats")

# ---------------------------------------------------------------------------
# NATSClient singleton
# ---------------------------------------------------------------------------
class NATSClient:
    """
    Singleton JetStream-enabled NATS client.
    All publish/subscribe methods encode/decode with orjson.
    """

    _instance: "NATSClient | None" = None

    def __new__(cls) -> "NATSClient":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self) -> None:
        if hasattr(self, "_initialized"):
            return
        self._initialized = True
        self._nc: NATSConnection | None = None
        self._js: JetStreamContext | None = None
        self._subscriptions: list = []

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------
    async def connect(self, url: str | None = None) -> None:
        """Connect to NATS server with auto-reconnect enabled."""
        url = url or os.environ.get("NATS_URL", "nats://localhost:4222")

        async def _error_cb(exc: Exception) -> None:
            logger.error("NATS error: %s", exc)

        async def _reconnected_cb() -> None:
            logger.info("NATS reconnected to %s", self._nc.connected_url.netloc if self._nc else "?")

        async def _disconnected_cb() -> None:
            logger.warning("NATS disconnected")

        async def _closed_cb() -> None:
            logger.info("NATS connection closed")

        self._nc = await nats.connect(
            url,
            error_cb=_error_cb,
            reconnected_cb=_reconnected_cb,
            disconnected_cb=_disconnected_cb,
            closed_cb=_closed_cb,
            reconnect_time_wait=2,
            max_reconnect_attempts=-1,   # infinite
            ping_interval=20,
            max_outstanding_pings=3,
        )
        self._js = self._nc.jetstream()
        logger.info("NATSClient connected to %s", url)

    async def disconnect(self) -> None:
        """Drain all subscriptions and close the connection."""
        if self._nc and not self._nc.is_closed:
            await self._nc.drain()
            logger.info("NATSClient disconnected")

    # ------------------------------------------------------------------
    # Core NATS publish / subscribe
    # ------------------------------------------------------------------
    async def publish(self, subject: str, data: dict | bytes) -> None:
        """Publish a message to a core NATS subject."""
        if self._nc is None:
            raise RuntimeError("NATSClient not connected")
        payload = orjson.dumps(data) if isinstance(data, dict) else data
        await self._nc.publish(subject, payload)
        logger.debug("NATS publish subject=%s size=%d", subject, len(payload))

    async def subscribe(
        self,
        subject: str,
        handler: Callable[[dict], Awaitable[None]],
        queue: str = "",
    ) -> None:
        """Subscribe to a core NATS subject. Handler receives decoded dict."""
        if self._nc is None:
            raise RuntimeError("NATSClient not connected")

        async def _msg_handler(msg) -> None:
            try:
                data = orjson.loads(msg.data)
            except Exception as exc:
                logger.warning("NATS message decode error on %s: %s", subject, exc)
                return
            try:
                await handler(data)
            except Exception as exc:
                logger.error("NATS handler error on %s: %s", subject, exc)

        sub = await self._nc.subscribe(subject, queue=queue, cb=_msg_handler)
        self._subscriptions.append(sub)
        logger.info("NATS subscribed subject=%s queue=%r", subject, queue)

    # ------------------------------------------------------------------
    # JetStream publish
    # ------------------------------------------------------------------
    async def jetstream_publish(
        self,
        subject: str,
        data: dict | bytes,
        stream: str = "",
    ) -> None:
        """
        Publish to a JetStream stream.
        If *stream* is provided it is passed as the expected stream header.
        """
        if self._js is None:
            raise RuntimeError("NATSClient not connected (no JetStream context)")
        payload = orjson.dumps(data) if isinstance(data, dict) else data
        kwargs: dict[str, Any] = {}
        if stream:
            kwargs["stream"] = stream
        ack = await self._js.publish(subject, payload, **kwargs)
        logger.debug(
            "JetStream publish subject=%s seq=%d stream=%s",
            subject,
            ack.seq,
            ack.stream,
        )

    # ------------------------------------------------------------------
    # JetStream subscribe (push consumer)
    # ------------------------------------------------------------------
    async def jetstream_subscribe(
        self,
        subject: str,
        handler: Callable[[dict], Awaitable[None]],
        durable: str = "",
        queue: str = "",
        deliver_policy: str = "new",
    ) -> None:
        """Subscribe to a JetStream subject with an optional durable consumer."""
        if self._js is None:
            raise RuntimeError("NATSClient not connected")

        async def _js_handler(msg) -> None:
            try:
                data = orjson.loads(msg.data)
            except Exception as exc:
                logger.warning("JetStream message decode error on %s: %s", subject, exc)
                await msg.nak()
                return
            try:
                await handler(data)
                await msg.ack()
            except Exception as exc:
                logger.error("JetStream handler error on %s: %s", subject, exc)
                await msg.nak()

        kwargs: dict[str, Any] = {"cb": _js_handler}
        if durable:
            kwargs["durable"] = durable
        if queue:
            kwargs["queue"] = queue

        sub = await self._js.subscribe(subject, **kwargs)
        self._subscriptions.append(sub)
        logger.info("JetStream subscribed subject=%s durable=%r", subject, durable)

    # ------------------------------------------------------------------
    # Properties
    # ------------------------------------------------------------------
    @property
    def is_connected(self) -> bool:
        return self._nc is not None and self._nc.is_connected


# ---------------------------------------------------------------------------
# Module-level singleton accessor
# ---------------------------------------------------------------------------
_nats_client = NATSClient()


def get_nats_client() -> NATSClient:
    return _nats_client


async def ensure_connected(url: str | None = None) -> NATSClient:
    """Connect if not already connected and return the singleton."""
    if not _nats_client.is_connected:
        await _nats_client.connect(url)
    return _nats_client
