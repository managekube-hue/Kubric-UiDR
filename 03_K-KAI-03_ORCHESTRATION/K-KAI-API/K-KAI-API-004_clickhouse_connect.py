"""
K-KAI-API-004: ClickHouse connection client using clickhouse-connect library.
Module-level singleton with lazy initialization.
Parses CLICKHOUSE_URL (format: http://user:pass@host:8123/db).
"""

import json
import logging
import os
from typing import Any
from urllib.parse import urlparse

import clickhouse_connect
from clickhouse_connect.driver.client import Client

logger = logging.getLogger("kai.clickhouse")

# ---------------------------------------------------------------------------
# Module-level singleton
# ---------------------------------------------------------------------------
_client: "ClickHouseClient | None" = None


def _parse_clickhouse_url(url: str) -> dict:
    """Parse http://user:pass@host:8123/db into connect kwargs."""
    p = urlparse(url)
    return {
        "host": p.hostname or "localhost",
        "port": p.port or 8123,
        "username": p.username or "default",
        "password": p.password or "",
        "database": p.path.lstrip("/") or "default",
    }


# ---------------------------------------------------------------------------
# ClickHouseClient
# ---------------------------------------------------------------------------
class ClickHouseClient:
    """Wraps clickhouse-connect for KAI security event analytics."""

    def __init__(self, url: str | None = None) -> None:
        self._url = url or os.environ.get(
            "CLICKHOUSE_URL", "http://default:@localhost:8123/kai"
        )
        self._ch: Client | None = None

    # ------------------------------------------------------------------
    # Connection / lazy init
    # ------------------------------------------------------------------
    def _get(self) -> Client:
        if self._ch is None:
            kwargs = _parse_clickhouse_url(self._url)
            self._ch = clickhouse_connect.get_client(**kwargs)
            logger.info(
                "ClickHouse connected to %s:%s/%s",
                kwargs["host"],
                kwargs["port"],
                kwargs["database"],
            )
        return self._ch

    # ------------------------------------------------------------------
    # API
    # ------------------------------------------------------------------
    def query(
        self, sql: str, params: dict | None = None
    ) -> list[dict]:
        """Execute a SELECT and return rows as list of dicts."""
        ch = self._get()
        result = ch.query(sql, parameters=params or {})
        columns = result.column_names
        rows: list[dict] = []
        for row in result.result_rows:
            record: dict[str, Any] = {}
            for col, val in zip(columns, row):
                # Auto-parse JSON/String columns that look like JSON
                if isinstance(val, str) and val.startswith(("{", "[")):
                    try:
                        val = json.loads(val)
                    except json.JSONDecodeError:
                        pass
                record[col] = val
            rows.append(record)
        logger.debug("ClickHouse query returned %d rows", len(rows))
        return rows

    def insert(
        self,
        table: str,
        data: list[dict],
        column_names: list[str],
    ) -> None:
        """
        Bulk insert rows into *table*.
        column_names must match the keys present in each dict of *data*.
        """
        if not data:
            logger.debug("insert: no data to insert into %s", table)
            return
        ch = self._get()
        rows = [[row.get(col) for col in column_names] for row in data]
        ch.insert(table, rows, column_names=column_names)
        logger.info("Inserted %d rows into %s", len(rows), table)

    def command(self, sql: str) -> None:
        """Execute a DDL or non-SELECT statement (CREATE TABLE, etc.)."""
        ch = self._get()
        ch.command(sql)
        logger.debug("ClickHouse command executed: %s", sql[:120])

    def close(self) -> None:
        """Close the underlying HTTP client."""
        if self._ch:
            self._ch.close()
            self._ch = None
            logger.info("ClickHouse client closed")


# ---------------------------------------------------------------------------
# Singleton accessor
# ---------------------------------------------------------------------------
def get_clickhouse_client() -> ClickHouseClient:
    """Return the module-level singleton, creating it on first call."""
    global _client
    if _client is None:
        _client = ClickHouseClient()
    return _client


# ---------------------------------------------------------------------------
# Convenience wrappers
# ---------------------------------------------------------------------------
def ch_query(sql: str, params: dict | None = None) -> list[dict]:
    return get_clickhouse_client().query(sql, params)


def ch_insert(
    table: str, data: list[dict], column_names: list[str]
) -> None:
    get_clickhouse_client().insert(table, data, column_names)


def ch_command(sql: str) -> None:
    get_clickhouse_client().command(sql)
