"""
K-KAI-API-002: asyncpg PostgreSQL client with connection pool management.
Module-level singleton pool with automatic reconnection (exponential backoff,
max 5 retries, 30 s max wait).
"""

import asyncio
import logging
import os
from typing import Any

import asyncpg

logger = logging.getLogger("kai.asyncpg")

# ---------------------------------------------------------------------------
# Module-level singleton
# ---------------------------------------------------------------------------
_pool: asyncpg.Pool | None = None


# ---------------------------------------------------------------------------
# AsyncPGClient
# ---------------------------------------------------------------------------
class AsyncPGClient:
    """Thin wrapper around an asyncpg connection pool."""

    def __init__(self) -> None:
        self._pool: asyncpg.Pool | None = None

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------
    async def connect(
        self,
        dsn: str | None = None,
        min_size: int = 2,
        max_size: int = 10,
    ) -> None:
        """Open a connection pool with exponential backoff retries."""
        dsn = dsn or os.environ.get(
            "DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai"
        )
        max_retries = 5
        base_delay = 1.0
        max_delay = 30.0

        for attempt in range(1, max_retries + 1):
            try:
                self._pool = await asyncpg.create_pool(
                    dsn, min_size=min_size, max_size=max_size
                )
                logger.info("asyncpg pool created (attempt %d)", attempt)
                # Expose as module singleton
                global _pool
                _pool = self._pool
                return
            except Exception as exc:
                delay = min(base_delay * (2 ** (attempt - 1)), max_delay)
                logger.warning(
                    "asyncpg connect attempt %d/%d failed: %s — retrying in %.1fs",
                    attempt,
                    max_retries,
                    exc,
                    delay,
                )
                if attempt == max_retries:
                    logger.error("asyncpg: all %d connect attempts failed", max_retries)
                    raise
                await asyncio.sleep(delay)

    async def disconnect(self) -> None:
        """Close the connection pool gracefully."""
        if self._pool:
            await self._pool.close()
            self._pool = None
            global _pool
            _pool = None
            logger.info("asyncpg pool closed")

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------
    def _get_pool(self) -> asyncpg.Pool:
        if self._pool is None:
            raise RuntimeError("AsyncPGClient is not connected. Call connect() first.")
        return self._pool

    # ------------------------------------------------------------------
    # Query methods
    # ------------------------------------------------------------------
    async def execute(self, query: str, *args: Any) -> str:
        """Execute a query that does not return rows (INSERT/UPDATE/DELETE/DDL)."""
        pool = self._get_pool()
        async with pool.acquire() as conn:
            result = await conn.execute(query, *args)
        logger.debug("execute: %s -> %s", query[:80], result)
        return result

    async def fetch(self, query: str, *args: Any) -> list[dict]:
        """Return all matching rows as a list of dicts."""
        pool = self._get_pool()
        async with pool.acquire() as conn:
            rows = await conn.fetch(query, *args)
        return [dict(r) for r in rows]

    async def fetchrow(self, query: str, *args: Any) -> dict | None:
        """Return the first matching row as a dict, or None."""
        pool = self._get_pool()
        async with pool.acquire() as conn:
            row = await conn.fetchrow(query, *args)
        return dict(row) if row else None

    async def fetchval(self, query: str, *args: Any) -> Any:
        """Return the first column of the first matching row."""
        pool = self._get_pool()
        async with pool.acquire() as conn:
            value = await conn.fetchval(query, *args)
        return value

    async def executemany(self, query: str, args_list: list[tuple]) -> None:
        """Execute the same query for each set of arguments in args_list."""
        pool = self._get_pool()
        async with pool.acquire() as conn:
            await conn.executemany(query, args_list)


# ---------------------------------------------------------------------------
# Module-level singleton instance
# ---------------------------------------------------------------------------
_client = AsyncPGClient()


async def get_client() -> AsyncPGClient:
    """Return the module-level singleton, connecting if needed."""
    global _client
    if _client._pool is None:
        await _client.connect()
    return _client


# ---------------------------------------------------------------------------
# Convenience module-level functions
# ---------------------------------------------------------------------------
async def execute(query: str, *args: Any) -> str:
    c = await get_client()
    return await c.execute(query, *args)


async def fetch(query: str, *args: Any) -> list[dict]:
    c = await get_client()
    return await c.fetch(query, *args)


async def fetchrow(query: str, *args: Any) -> dict | None:
    c = await get_client()
    return await c.fetchrow(query, *args)


async def fetchval(query: str, *args: Any) -> Any:
    c = await get_client()
    return await c.fetchval(query, *args)
