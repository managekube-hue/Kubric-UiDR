"""
K-KAI-LIBS-001: Polars dataframe utility wrappers for security event analytics.
High-performance columnar operations on OCSF-structured security events.
"""

import logging
from pathlib import Path
from typing import Any

import polars as pl

logger = logging.getLogger("kai.libs.polars")


# ---------------------------------------------------------------------------
# I/O
# ---------------------------------------------------------------------------
def read_ndjson_events(path: str) -> pl.DataFrame:
    """
    Read a newline-delimited JSON file of OCSF security events into a
    Polars DataFrame.  Unknown columns are inferred from the data.
    """
    resolved = Path(path).resolve()
    if not resolved.exists():
        raise FileNotFoundError(f"NDJSON file not found: {resolved}")
    df = pl.read_ndjson(str(resolved))
    logger.info("read_ndjson_events: %d rows, %d cols from %s", df.height, df.width, path)
    return df


# ---------------------------------------------------------------------------
# Filtering
# ---------------------------------------------------------------------------
def filter_by_severity(df: pl.DataFrame, min_severity: int) -> pl.DataFrame:
    """
    Return rows where severity_id >= min_severity.
    The OCSF severity_id field is used (1=Informational … 6=Fatal).
    Falls back to 'severity' column as int if severity_id is absent.
    """
    if "severity_id" in df.columns:
        col = "severity_id"
    elif "severity" in df.columns:
        col = "severity"
    else:
        logger.warning("filter_by_severity: no severity column found, returning all rows")
        return df

    result = df.filter(pl.col(col).cast(pl.Int32, strict=False) >= min_severity)
    logger.debug("filter_by_severity min=%d: %d -> %d rows", min_severity, df.height, result.height)
    return result


# ---------------------------------------------------------------------------
# Aggregation
# ---------------------------------------------------------------------------
def aggregate_by_tenant(df: pl.DataFrame) -> pl.DataFrame:
    """
    Group events by tenant_id and return a summary DataFrame with:
      - tenant_id
      - count           (total events)
      - avg_severity    (mean severity_id)
      - max_ts          (latest time_dt or time)
    """
    tenant_col = "tenant_id" if "tenant_id" in df.columns else None
    sev_col = next((c for c in ("severity_id", "severity") if c in df.columns), None)
    ts_col = next((c for c in ("time_dt", "time", "start_time") if c in df.columns), None)

    if tenant_col is None:
        raise ValueError("DataFrame must contain a 'tenant_id' column for aggregation")

    aggs: list[pl.Expr] = [pl.len().alias("count")]

    if sev_col:
        aggs.append(pl.col(sev_col).cast(pl.Float64, strict=False).mean().alias("avg_severity"))
    else:
        aggs.append(pl.lit(None).cast(pl.Float64).alias("avg_severity"))

    if ts_col:
        aggs.append(pl.col(ts_col).max().alias("max_ts"))
    else:
        aggs.append(pl.lit(None).alias("max_ts"))

    result = df.group_by(tenant_col).agg(aggs)
    logger.debug("aggregate_by_tenant: %d tenant groups", result.height)
    return result


# ---------------------------------------------------------------------------
# Join helpers
# ---------------------------------------------------------------------------
def join_with_context(
    events_df: pl.DataFrame,
    context_df: pl.DataFrame,
    on: str = "src_ip",
    how: str = "left",
) -> pl.DataFrame:
    """
    Join security events with a context table (e.g., asset data,
    threat intel) on the given key column.
    Duplicate column suffixes are handled with _right suffix.
    """
    if on not in events_df.columns:
        raise ValueError(f"join_with_context: events_df missing column '{on}'")
    if on not in context_df.columns:
        raise ValueError(f"join_with_context: context_df missing column '{on}'")

    result = events_df.join(context_df, on=on, how=how, suffix="_ctx")
    logger.debug(
        "join_with_context on='%s' how='%s': %d events + %d ctx -> %d rows",
        on,
        how,
        events_df.height,
        context_df.height,
        result.height,
    )
    return result


# ---------------------------------------------------------------------------
# Top talkers
# ---------------------------------------------------------------------------
def top_talkers(df: pl.DataFrame, n: int = 10) -> pl.DataFrame:
    """
    Return the top N source IPs by event count.
    Expects a 'src_ip' column (or falls back to 'src_endpoint.ip').
    Result columns: src_ip, event_count (descending).
    """
    if "src_ip" in df.columns:
        ip_col = "src_ip"
    elif "src_endpoint" in df.columns:
        # unnest if needed
        logger.warning("top_talkers: 'src_endpoint' column found; attempting struct access")
        df = df.with_columns(pl.col("src_endpoint").struct.field("ip").alias("src_ip"))
        ip_col = "src_ip"
    else:
        raise ValueError("top_talkers: DataFrame missing 'src_ip' column")

    result = (
        df.group_by(ip_col)
        .agg(pl.len().alias("event_count"))
        .sort("event_count", descending=True)
        .head(n)
    )
    logger.debug("top_talkers n=%d: top IP is %s (%d events)", n,
                 result[ip_col][0] if result.height > 0 else "N/A",
                 result["event_count"][0] if result.height > 0 else 0)
    return result


# ---------------------------------------------------------------------------
# Schema helpers
# ---------------------------------------------------------------------------
def ensure_columns(df: pl.DataFrame, required: list[str]) -> pl.DataFrame:
    """Add missing columns as null so downstream operations don't fail."""
    for col in required:
        if col not in df.columns:
            df = df.with_columns(pl.lit(None).alias(col))
    return df
