"""
K-KAI-LIBS-005: ujson fallback serializer with auto-backend selection.
Same interface as orjson_serializer. Auto-selects orjson when available.
Sanitizes NaN / Inf values and handles datetime ISO serialization.
"""

import datetime
import logging
import math
import uuid
from typing import Any

logger = logging.getLogger("kai.libs.ujson_fallback")

# ---------------------------------------------------------------------------
# Backend selection
# ---------------------------------------------------------------------------
try:
    import orjson as _orjson
    _has_orjson = True
    logger.debug("ujson_fallback: orjson available, using it as backend")
except ImportError:
    _has_orjson = False
    logger.debug("ujson_fallback: orjson not available, using ujson")

try:
    import ujson as _ujson
    _has_ujson = True
except ImportError:
    _has_ujson = False
    import json as _json_std


# ---------------------------------------------------------------------------
# Custom exception
# ---------------------------------------------------------------------------
class JSONSerializationError(Exception):
    """Raised when serialization or deserialization fails."""


# ---------------------------------------------------------------------------
# NaN / Inf sanitizer
# ---------------------------------------------------------------------------
def _sanitize(obj: Any) -> Any:
    """
    Recursively sanitize *obj*:
    - float NaN / Inf -> None
    - datetime / date / time -> ISO string
    - UUID -> str
    - bytes -> hex str
    - set / frozenset -> list
    - numpy scalar types -> Python native (checked by class name)
    """
    if obj is None:
        return None
    if isinstance(obj, bool):
        return obj
    if isinstance(obj, float):
        if math.isnan(obj) or math.isinf(obj):
            return None
        return obj
    if isinstance(obj, int):
        return obj
    if isinstance(obj, str):
        return obj
    if isinstance(obj, dict):
        return {str(k): _sanitize(v) for k, v in obj.items()}
    if isinstance(obj, (list, tuple)):
        return [_sanitize(item) for item in obj]
    if isinstance(obj, (datetime.datetime, datetime.date, datetime.time)):
        return obj.isoformat()
    if isinstance(obj, uuid.UUID):
        return str(obj)
    if isinstance(obj, bytes):
        return obj.hex()
    if isinstance(obj, (set, frozenset)):
        return [_sanitize(item) for item in obj]
    # numpy scalar check without importing numpy
    cls_name = type(obj).__name__
    if cls_name in ("float32", "float64", "float16"):
        v = float(obj)
        return None if (math.isnan(v) or math.isinf(v)) else v
    if cls_name in ("int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64"):
        return int(obj)
    if hasattr(obj, "__dict__"):
        return _sanitize(obj.__dict__)
    return str(obj)


# ---------------------------------------------------------------------------
# dumps
# ---------------------------------------------------------------------------
def dumps(obj: Any, pretty: bool = False) -> str:
    """
    Serialize *obj* to a JSON string.
    Uses orjson if available (fastest), then ujson, then stdlib json.
    Returns str (not bytes) for universal compatibility.
    """
    sanitized = _sanitize(obj)
    try:
        if _has_orjson:
            opts = _orjson.OPT_NON_STR_KEYS | _orjson.OPT_SERIALIZE_NUMPY
            if pretty:
                opts |= _orjson.OPT_INDENT_2
            return _orjson.dumps(sanitized, option=opts).decode("utf-8")
        if _has_ujson:
            indent = 2 if pretty else 0
            return _ujson.dumps(sanitized, indent=indent, ensure_ascii=False)
        indent = 2 if pretty else None
        return _json_std.dumps(sanitized, ensure_ascii=False, indent=indent)
    except Exception as exc:
        raise JSONSerializationError(f"JSON serialization failed: {exc}") from exc


# ---------------------------------------------------------------------------
# loads
# ---------------------------------------------------------------------------
def loads(data: bytes | str) -> Any:
    """
    Deserialize JSON bytes or str.
    Uses orjson if available, then ujson, then stdlib json.
    """
    try:
        if _has_orjson:
            return _orjson.loads(data)
        if _has_ujson:
            return _ujson.loads(data)
        if isinstance(data, bytes):
            data = data.decode("utf-8")
        return _json_std.loads(data)
    except Exception as exc:
        raise JSONSerializationError(f"JSON deserialization failed: {exc}") from exc


# ---------------------------------------------------------------------------
# Convenience: safe round-trip
# ---------------------------------------------------------------------------
def safe_loads(data: bytes | str, default: Any = None) -> Any:
    """Like loads but returns *default* on error instead of raising."""
    try:
        return loads(data)
    except JSONSerializationError:
        return default


def safe_dumps(obj: Any, pretty: bool = False) -> str:
    """Like dumps but returns empty JSON object '{}' on error instead of raising."""
    try:
        return dumps(obj, pretty=pretty)
    except JSONSerializationError:
        return "{}"
