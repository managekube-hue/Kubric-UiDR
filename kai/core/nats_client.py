"""
Shared NATS connection manager for KAI.

Usage:
    from kai.core.nats_client import nats_client

    await nats_client.connect()
    await nats_client.publish("kubric.kai.triage", payload_bytes)
    await nats_client.disconnect()
"""

from __future__ import annotations

import structlog
from nats.aio.client import Client as NATSClient

from kai.config import settings

log = structlog.get_logger(__name__)


class _NATSManager:
    """Singleton wrapper around nats-py async client."""

    def __init__(self) -> None:
        self._nc: NATSClient | None = None

    @property
    def connected(self) -> bool:
        return self._nc is not None and self._nc.is_connected

    async def connect(self) -> None:
        if self.connected:
            return
        nc = NATSClient()
        try:
            await nc.connect(
                settings.nats_url,
                name="kubric-kai",
                reconnect_time_wait=2,
                max_reconnect_attempts=10,
                error_cb=_on_nats_error,
                disconnected_cb=_on_nats_disconnect,
                reconnected_cb=_on_nats_reconnect,
            )
            self._nc = nc
            log.info("nats.connected", url=settings.nats_url)
        except Exception as exc:
            log.warning("nats.connect_failed", error=str(exc))
            # KAI degrades gracefully — agents still serve HTTP even without NATS
            self._nc = None

    async def disconnect(self) -> None:
        if self._nc and self._nc.is_connected:
            await self._nc.drain()
            await self._nc.close()
            log.info("nats.disconnected")
        self._nc = None

    async def publish(self, subject: str, payload: bytes) -> None:
        if not self.connected:
            log.warning("nats.publish_skipped", subject=subject, reason="not connected")
            return
        assert self._nc is not None
        await self._nc.publish(subject, payload)

    async def subscribe(self, subject: str, cb: object) -> object | None:
        """Subscribe to a subject. Returns the subscription or None if not connected."""
        if not self.connected:
            log.warning("nats.subscribe_skipped", subject=subject, reason="not connected")
            return None
        assert self._nc is not None
        return await self._nc.subscribe(subject, cb=cb)  # type: ignore[arg-type]

    async def jetstream(self) -> object | None:
        """Return a JetStream context, or None if unavailable."""
        if not self.connected:
            return None
        assert self._nc is not None
        return self._nc.jetstream()


async def _on_nats_error(exc: Exception) -> None:
    log.error("nats.error", error=str(exc))


async def _on_nats_disconnect() -> None:
    log.warning("nats.disconnected_event")


async def _on_nats_reconnect() -> None:
    log.info("nats.reconnected")


# Module-level singleton — import this everywhere in kai/
nats_client = _NATSManager()
