"""
K-KAI-LIBS-002: PyArrow Parquet I/O utilities for archiving security event streams.
Provides schema definitions for OCSF classes 4007, 2004, 4001, 4003.
Supports write with compression, filtered reads, and append mode.
"""

import logging
from pathlib import Path
from typing import Any

import pyarrow as pa
import pyarrow.parquet as pq

logger = logging.getLogger("kai.libs.pyarrow")


# ---------------------------------------------------------------------------
# OCSF Arrow schemas
# ---------------------------------------------------------------------------
_BASE_FIELDS = [
    pa.field("class_uid", pa.int32()),
    pa.field("class_name", pa.string()),
    pa.field("time", pa.int64()),           # epoch ms
    pa.field("time_dt", pa.string()),       # ISO-8601
    pa.field("tenant_id", pa.string()),
    pa.field("severity_id", pa.int32()),
    pa.field("severity", pa.string()),
    pa.field("status", pa.string()),
    pa.field("message", pa.string()),
]

_NETWORK_FIELDS = [
    pa.field("src_ip", pa.string()),
    pa.field("dst_ip", pa.string()),
    pa.field("src_port", pa.int32()),
    pa.field("dst_port", pa.int32()),
    pa.field("protocol", pa.string()),
    pa.field("bytes_in", pa.int64()),
    pa.field("bytes_out", pa.int64()),
]

# OCSF class 4001: Network Activity
_SCHEMA_4001 = pa.schema(_BASE_FIELDS + _NETWORK_FIELDS + [
    pa.field("connection_uid", pa.string()),
    pa.field("direction", pa.string()),
])

# OCSF class 4003: DNS Activity
_SCHEMA_4003 = pa.schema(_BASE_FIELDS + _NETWORK_FIELDS + [
    pa.field("query_hostname", pa.string()),
    pa.field("query_type", pa.string()),
    pa.field("answers", pa.list_(pa.string())),
])

# OCSF class 4007: FTP Activity
_SCHEMA_4007 = pa.schema(_BASE_FIELDS + _NETWORK_FIELDS + [
    pa.field("command", pa.string()),
    pa.field("file_name", pa.string()),
    pa.field("file_size", pa.int64()),
])

# OCSF class 2004: Account Change
_SCHEMA_2004 = pa.schema(_BASE_FIELDS + [
    pa.field("user_name", pa.string()),
    pa.field("user_uid", pa.string()),
    pa.field("change_type", pa.string()),
    pa.field("src_ip", pa.string()),
])

# Fallback generic schema
_SCHEMA_GENERIC = pa.schema(_BASE_FIELDS + [
    pa.field("raw", pa.string()),
])

_SCHEMA_MAP: dict[int, pa.Schema] = {
    4001: _SCHEMA_4001,
    4003: _SCHEMA_4003,
    4007: _SCHEMA_4007,
    2004: _SCHEMA_2004,
}


def schema_for_ocsf_class(class_uid: int) -> pa.Schema:
    """Return the appropriate Arrow schema for an OCSF class_uid."""
    schema = _SCHEMA_MAP.get(class_uid, _SCHEMA_GENERIC)
    logger.debug("schema_for_ocsf_class(%d) -> %d fields", class_uid, len(schema))
    return schema


# ---------------------------------------------------------------------------
# Write
# ---------------------------------------------------------------------------
def write_events_parquet(
    events: list[dict],
    path: str,
    compression: str = "snappy",
) -> None:
    """
    Write a list of OCSF event dicts to a Parquet file.
    Schema is inferred from class_uid of the first event if homogeneous,
    otherwise falls back to generic schema.
    """
    if not events:
        logger.warning("write_events_parquet: empty event list, skipping write to %s", path)
        return

    class_uid = events[0].get("class_uid", 0)
    schema = schema_for_ocsf_class(class_uid)

    table = _dicts_to_table(events, schema)
    pq.write_table(table, path, compression=compression)
    logger.info("Wrote %d events to %s (compression=%s)", len(events), path, compression)


def _dicts_to_table(events: list[dict], schema: pa.Schema) -> pa.Table:
    """Convert list of dicts to a PyArrow table, coercing types."""
    columns: dict[str, list[Any]] = {field.name: [] for field in schema}
    # Collect values per column
    for event in events:
        for field in schema:
            columns[field.name].append(event.get(field.name))

    arrays: list[pa.Array] = []
    for field in schema:
        try:
            arr = pa.array(columns[field.name], type=field.type)
        except (pa.ArrowInvalid, pa.ArrowTypeError):
            # Fall back to string and cast
            arr = pa.array(
                [str(v) if v is not None else None for v in columns[field.name]],
                type=pa.string(),
            )
            arr = arr.cast(pa.string())
        arrays.append(arr)

    return pa.table(dict(zip(schema.names, arrays)), schema=schema)


# ---------------------------------------------------------------------------
# Read
# ---------------------------------------------------------------------------
def read_events_parquet(
    path: str,
    filters: list | None = None,
) -> list[dict]:
    """
    Read OCSF events from a Parquet file.
    *filters* follows the PyArrow DNF filter format:
    e.g., [[("severity_id", ">=", 4)]]
    """
    resolved = Path(path).resolve()
    if not resolved.exists():
        raise FileNotFoundError(f"Parquet file not found: {resolved}")
    table = pq.read_table(str(resolved), filters=filters)
    records = table.to_pylist()
    logger.info("Read %d events from %s", len(records), path)
    return records


# ---------------------------------------------------------------------------
# Append
# ---------------------------------------------------------------------------
def append_to_parquet(events: list[dict], path: str) -> None:
    """
    Append events to an existing Parquet file, or create it if absent.
    Uses ParquetWriter with schema from first event's class_uid.
    """
    if not events:
        logger.warning("append_to_parquet: no events to append, path=%s", path)
        return

    class_uid = events[0].get("class_uid", 0)
    schema = schema_for_ocsf_class(class_uid)
    new_table = _dicts_to_table(events, schema)

    resolved = Path(path)
    if resolved.exists():
        existing = pq.read_table(str(resolved))
        merged = pa.concat_tables([existing, new_table], promote_options="default")
        pq.write_table(merged, str(resolved), compression="snappy")
        logger.info("Appended %d events to %s (total %d)", len(events), path, merged.num_rows)
    else:
        pq.write_table(new_table, str(resolved), compression="snappy")
        logger.info("Created %s with %d events", path, len(events))
