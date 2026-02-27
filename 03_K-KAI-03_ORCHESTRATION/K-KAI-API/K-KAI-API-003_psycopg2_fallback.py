"""
K-KAI-API-003: psycopg2 synchronous fallback PostgreSQL client.
Used by batch jobs that cannot use async. Provides a SimpleConnectionPool
(minconn=1, maxconn=5) and a context-manager-friendly PsycoPG2Client class
with RealDictCursor for dict results.
"""

import logging
import os
from contextlib import contextmanager
from typing import Any, Generator

import psycopg2
import psycopg2.extras
import psycopg2.pool

logger = logging.getLogger("kai.psycopg2")

# ---------------------------------------------------------------------------
# Module-level connection pool (lazy)
# ---------------------------------------------------------------------------
_pool: psycopg2.pool.SimpleConnectionPool | None = None

_DEFAULT_DSN = os.environ.get(
    "DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai"
)


def _get_pool() -> psycopg2.pool.SimpleConnectionPool:
    global _pool
    if _pool is None:
        _pool = psycopg2.pool.SimpleConnectionPool(
            minconn=1,
            maxconn=5,
            dsn=_DEFAULT_DSN,
        )
        logger.info("psycopg2 SimpleConnectionPool created (min=1, max=5)")
    return _pool


@contextmanager
def get_connection() -> Generator[psycopg2.extensions.connection, None, None]:
    """Yield a connection from the pool and return it when done."""
    pool = _get_pool()
    conn = pool.getconn()
    try:
        yield conn
    finally:
        pool.putconn(conn)


# ---------------------------------------------------------------------------
# PsycoPG2Client
# ---------------------------------------------------------------------------
class PsycoPG2Client:
    """
    Synchronous psycopg2 client with context manager support.
    Uses RealDictCursor so all results are plain Python dicts.
    """

    def __init__(self, dsn: str | None = None) -> None:
        self._dsn = dsn or _DEFAULT_DSN
        self._conn: psycopg2.extensions.connection | None = None
        self._pool: psycopg2.pool.SimpleConnectionPool | None = None

    # ------------------------------------------------------------------
    # Context manager
    # ------------------------------------------------------------------
    def __enter__(self) -> "PsycoPG2Client":
        pool = _get_pool()
        self._conn = pool.getconn()
        logger.debug("PsycoPG2Client: connection acquired from pool")
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> bool:
        if self._conn:
            if exc_type:
                self._conn.rollback()
                logger.warning("PsycoPG2Client: transaction rolled back due to %s", exc_type)
            else:
                self._conn.commit()
            _get_pool().putconn(self._conn)
            self._conn = None
        return False  # do not suppress exceptions

    # ------------------------------------------------------------------
    # Internal helper
    # ------------------------------------------------------------------
    def _cursor(self) -> psycopg2.extras.RealDictCursor:
        if self._conn is None:
            raise RuntimeError(
                "PsycoPG2Client must be used as a context manager "
                "(with PsycoPG2Client() as client:)."
            )
        return self._conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------
    def execute(self, query: str, params: tuple | list | None = None) -> int:
        """
        Execute a query that does not return rows.
        Returns the rowcount.
        """
        with self._cursor() as cur:
            cur.execute(query, params)
            return cur.rowcount

    def fetchall(
        self, query: str, params: tuple | list | None = None
    ) -> list[dict]:
        """Return all matching rows as a list of dicts."""
        with self._cursor() as cur:
            cur.execute(query, params)
            rows = cur.fetchall()
        return [dict(r) for r in rows]

    def fetchone(
        self, query: str, params: tuple | list | None = None
    ) -> dict | None:
        """Return the first matching row as a dict, or None."""
        with self._cursor() as cur:
            cur.execute(query, params)
            row = cur.fetchone()
        return dict(row) if row else None

    def executemany(
        self, query: str, param_list: list[tuple | list]
    ) -> None:
        """Execute the same query for each parameter set."""
        with self._cursor() as cur:
            cur.executemany(query, param_list)


# ---------------------------------------------------------------------------
# Standalone helper (no context manager needed for simple one-shot queries)
# ---------------------------------------------------------------------------
def query_one(sql: str, params: tuple | None = None) -> dict | None:
    """Execute sql and return first row as dict (autocommit). Convenience wrapper."""
    with get_connection() as conn:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(sql, params)
            row = cur.fetchone()
        conn.commit()
    return dict(row) if row else None


def query_all(sql: str, params: tuple | None = None) -> list[dict]:
    """Execute sql and return all rows as list of dicts (autocommit)."""
    with get_connection() as conn:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        conn.commit()
    return [dict(r) for r in rows]
