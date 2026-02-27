"""
K-KAI-LIBS-004: orjson fast JSON serializer for security event processing.
Handles datetime ISO format, UUID str conversion, numpy float32/64 coercion.
OPT_NON_STR_KEYS and OPT_SERIALIZE_NUMPY options enabled.
"""

import datetime
import logging
import uuid
from typing import Any

import orjson

logger = logging.getLogger("kai.libs.orjson")


# ---------------------------------------------------------------------------
# Module-level helpers
# ---------------------------------------------------------------------------
def dumps(obj: Any, pretty: bool = False) -> bytes:
    """
    Serialize *obj* to JSON bytes using orjson.
    OPT_SERIALIZE_NUMPY ensures numpy scalars/arrays are handled.
    OPT_NON_STR_KEYS allows integer/UUID dict keys.
    """
    options = orjson.OPT_SERIALIZE_NUMPY | orjson.OPT_NON_STR_KEYS
    if pretty:
        options |= orjson.OPT_INDENT_2
    return orjson.dumps(obj, default=_default_handler, option=options)


def loads(data: bytes | str) -> Any:
    """Deserialize JSON bytes or str back to a Python object."""
    if isinstance(data, str):
        data = data.encode()
    return orjson.loads(data)


def dumps_str(obj: Any, pretty: bool = False) -> str:
    """Serialize *obj* to a JSON string (convenience wrapper)."""
    return dumps(obj, pretty=pretty).decode("utf-8")


# ---------------------------------------------------------------------------
# Default handler for non-serializable types
# ---------------------------------------------------------------------------
def _default_handler(obj: Any) -> Any:
    """
    Called by orjson for types it cannot serialize natively.
    Handles: datetime, date, UUID, bytes, set, frozenset, objects with
    __dict__, and any __str__-able type as final fallback.
    """
    if isinstance(obj, (datetime.datetime, datetime.date, datetime.time)):
        return obj.isoformat()
    if isinstance(obj, uuid.UUID):
        return str(obj)
    if isinstance(obj, bytes):
        return obj.hex()
    if isinstance(obj, (set, frozenset)):
        return list(obj)
    if hasattr(obj, "__dict__"):
        return obj.__dict__
    if hasattr(obj, "__iter__"):
        return list(obj)
    return str(obj)


# ---------------------------------------------------------------------------
# OCSFSerializer
# ---------------------------------------------------------------------------
class OCSFSerializer:
    """
    Serializer tuned for OCSF (Open Cybersecurity Schema Framework) events.
    Handles all common security event types emitted by KAI agents.
    """

    def serialize_event(self, event: dict) -> bytes:
        """
        Serialize an OCSF event dict to bytes.
        - datetime fields -> ISO-8601 strings
        - UUID fields -> str
        - numpy float32/float64 -> Python float
        - Unknown types -> str(obj)
        """
        sanitized = self._sanitize(event)
        return orjson.dumps(
            sanitized,
            default=_default_handler,
            option=orjson.OPT_SERIALIZE_NUMPY | orjson.OPT_NON_STR_KEYS,
        )

    def deserialize_event(self, data: bytes) -> dict:
        """
        Deserialize bytes back to an OCSF event dict.
        Raises orjson.JSONDecodeError on malformed input.
        """
        return orjson.loads(data)

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------
    def _sanitize(self, obj: Any) -> Any:
        """
        Recursively sanitize an object tree so orjson can serialize it
        even when OCSF fields contain exotic Python types.
        """
        if isinstance(obj, dict):
            return {k: self._sanitize(v) for k, v in obj.items()}
        if isinstance(obj, (list, tuple)):
            return [self._sanitize(v) for v in obj]
        if isinstance(obj, (datetime.datetime, datetime.date, datetime.time)):
            return obj.isoformat()
        if isinstance(obj, uuid.UUID):
            return str(obj)
        if isinstance(obj, bytes):
            return obj.hex()
        # Handle numpy types without importing numpy (check class name)
        cls_name = type(obj).__name__
        if cls_name in ("float32", "float64", "float16"):
            return float(obj)
        if cls_name in ("int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64"):
            return int(obj)
        return obj


# ---------------------------------------------------------------------------
# Module-level singleton serializer
# ---------------------------------------------------------------------------
_serializer = OCSFSerializer()


def serialize_event(event: dict) -> bytes:
    return _serializer.serialize_event(event)


def deserialize_event(data: bytes) -> dict:
    return _serializer.deserialize_event(data)
