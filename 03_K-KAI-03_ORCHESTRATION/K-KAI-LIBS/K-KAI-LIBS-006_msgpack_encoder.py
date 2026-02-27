"""
K-KAI-LIBS-006: msgpack encoder/decoder for NATS JetStream binary payloads.
datetime-aware ext_hook. Also provides MsgpackNATSCodec for nats.py integration.
"""

import datetime
import logging
from typing import Any, Callable, Awaitable

import msgpack

logger = logging.getLogger("kai.libs.msgpack")

# ---------------------------------------------------------------------------
# Custom extension type IDs
# ---------------------------------------------------------------------------
_EXT_DATETIME = 1   # msgpack Ext type for datetime objects


# ---------------------------------------------------------------------------
# encode / decode
# ---------------------------------------------------------------------------
def _default_encoder(obj: Any) -> Any:
    """
    Called by msgpack.packb for objects it can't natively serialize.
    Encodes datetime -> msgpack Ext(1, iso_bytes).
    """
    if isinstance(obj, (datetime.datetime, datetime.date, datetime.time)):
        iso = obj.isoformat().encode("utf-8")
        return msgpack.ExtType(_EXT_DATETIME, iso)
    raise TypeError(f"msgpack: unsupported type {type(obj)}")


def _ext_hook(code: int, data: bytes) -> Any:
    """
    Called by msgpack.unpackb for Ext-typed values.
    Decodes Ext(1, iso_bytes) -> datetime.
    """
    if code == _EXT_DATETIME:
        iso = data.decode("utf-8")
        try:
            return datetime.datetime.fromisoformat(iso)
        except ValueError:
            return iso   # return raw string on parse failure
    return msgpack.ExtType(code, data)


def encode(obj: dict) -> bytes:
    """Encode a dict (or list-like) to msgpack bytes."""
    return msgpack.packb(
        obj,
        use_bin_type=True,
        default=_default_encoder,
    )


def decode(data: bytes) -> dict:
    """Decode msgpack bytes back to a Python dict."""
    return msgpack.unpackb(
        data,
        raw=False,
        ext_hook=_ext_hook,
        timestamp=3,  # decode Timestamp ext to Python datetime
    )


def encode_batch(items: list[dict]) -> bytes:
    """Encode a list of dicts as a single msgpack array."""
    return msgpack.packb(
        items,
        use_bin_type=True,
        default=_default_encoder,
    )


def decode_batch(data: bytes) -> list[dict]:
    """Decode a msgpack array of dicts."""
    result = msgpack.unpackb(
        data,
        raw=False,
        ext_hook=_ext_hook,
        timestamp=3,
    )
    if not isinstance(result, list):
        raise TypeError(f"Expected msgpack array, got {type(result)}")
    return result


# ---------------------------------------------------------------------------
# MsgpackNATSCodec
# ---------------------------------------------------------------------------
class MsgpackNATSCodec:
    """
    Wraps a nats.py client to transparently encode/decode all messages
    with msgpack.  Use instead of the bare nats client when binary
    efficiency matters (e.g., high-frequency network flow events).

    Usage::

        import nats
        nc = await nats.connect("nats://localhost:4222")
        codec = MsgpackNATSCodec(nc)
        await codec.publish("kubric.events", {"class_uid": 4001, "src_ip": "1.2.3.4"})
        await codec.subscribe("kubric.events", handler)
    """

    def __init__(self, nats_client: Any) -> None:
        self._nc = nats_client

    async def publish(self, subject: str, data: dict) -> None:
        """Encode *data* with msgpack and publish to *subject*."""
        payload = encode(data)
        await self._nc.publish(subject, payload)
        logger.debug("MsgpackNATSCodec publish subject=%s bytes=%d", subject, len(payload))

    async def subscribe(
        self,
        subject: str,
        handler: Callable[[dict], Awaitable[None]],
        queue: str = "",
    ) -> None:
        """Subscribe to *subject*; decode each message before passing to handler."""

        async def _wrapped(msg):
            try:
                data = decode(msg.data)
            except Exception as exc:
                logger.warning(
                    "MsgpackNATSCodec decode error on %s: %s", subject, exc
                )
                return
            try:
                await handler(data)
            except Exception as exc:
                logger.error(
                    "MsgpackNATSCodec handler error on %s: %s", subject, exc
                )

        await self._nc.subscribe(subject, queue=queue, cb=_wrapped)
        logger.info("MsgpackNATSCodec subscribed subject=%s queue=%r", subject, queue)

    async def jetstream_publish(
        self,
        js_context: Any,
        subject: str,
        data: dict,
    ) -> None:
        """Encode and publish to a JetStream subject."""
        payload = encode(data)
        await js_context.publish(subject, payload)
        logger.debug(
            "MsgpackNATSCodec JetStream publish subject=%s bytes=%d",
            subject,
            len(payload),
        )
