"""
K-NOC-CM-SALT-002 — SaltStack State Apply Wrapper.
Applies SaltStack states via salt-api with retry logic and job history persistence.
"""

from __future__ import annotations

import asyncio
import os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

import asyncpg
import httpx
import structlog

logger = structlog.get_logger(__name__)

SALT_API_URL = os.getenv("SALT_API_URL", "http://localhost:8080")
SALT_API_USER = os.getenv("SALT_API_USER", "salt-api")
SALT_API_PASSWORD = os.getenv("SALT_API_PASSWORD", "")
DB_DSN = os.getenv("DATABASE_URL", "postgresql://kubric:kubric@localhost/kubric")
MAX_RETRIES = 3
RETRY_BACKOFF_SECONDS = 5


@dataclass
class SaltResult:
    jid: str
    minions_targeted: list[str]
    minions_success: list[str]
    minions_failed: list[str]
    results: dict[str, Any] = field(default_factory=dict)


class SaltStateApply:
    """Async wrapper around the salt-api REST interface."""

    def __init__(self, token: str, db_pool: asyncpg.Pool) -> None:
        self._token = token
        self._db_pool = db_pool
        self._http = httpx.AsyncClient(base_url=SALT_API_URL, timeout=120.0)

    @classmethod
    async def create(cls) -> "SaltStateApply":
        """Factory: login to salt-api, acquire DB pool, return configured instance."""
        db_pool = await asyncpg.create_pool(DB_DSN, min_size=1, max_size=5)
        token = await cls._login()
        instance = cls(token=token, db_pool=db_pool)
        return instance

    @staticmethod
    async def _login() -> str:
        """Authenticate against salt-api and return the session token."""
        async with httpx.AsyncClient(base_url=SALT_API_URL, timeout=30.0) as http:
            resp = await http.post(
                "/login",
                json={
                    "username": SALT_API_USER,
                    "password": SALT_API_PASSWORD,
                    "eauth": "pam",
                },
            )
            resp.raise_for_status()
            data = resp.json()
        token = data["return"][0]["token"]
        logger.info("salt-api login ok", user=SALT_API_USER)
        return token

    def _auth_headers(self) -> dict[str, str]:
        return {"X-Auth-Token": self._token}

    async def _post_with_retry(self, payload: dict[str, Any]) -> dict[str, Any]:
        """POST to salt-api root endpoint with up to MAX_RETRIES retries."""
        for attempt in range(1, MAX_RETRIES + 1):
            try:
                resp = await self._http.post(
                    "/",
                    json=payload,
                    headers=self._auth_headers(),
                )
                resp.raise_for_status()
                return resp.json()
            except (httpx.ConnectError, httpx.RemoteProtocolError) as exc:
                if attempt == MAX_RETRIES:
                    raise
                logger.warning(
                    "salt-api transient error, retrying",
                    attempt=attempt,
                    error=str(exc),
                )
                await asyncio.sleep(RETRY_BACKOFF_SECONDS)
        raise RuntimeError("unreachable")  # pragma: no cover

    async def apply_state(
        self,
        target: str,
        state: str,
        pillar: dict[str, Any] | None = None,
    ) -> SaltResult:
        """Apply a Salt state to a target minion."""
        payload: dict[str, Any] = {
            "client": "local",
            "tgt": target,
            "fun": "state.apply",
            "arg": [state],
        }
        if pillar:
            payload["kwarg"] = {"pillar": pillar}

        data = await self._post_with_retry(payload)
        result = self._parse_result(data)
        await self._record_job(result, target, state)
        return result

    async def highstate(self, target: str) -> SaltResult:
        """Run a highstate on the given target."""
        payload = {
            "client": "local",
            "tgt": target,
            "fun": "state.highstate",
        }
        data = await self._post_with_retry(payload)
        result = self._parse_result(data)
        await self._record_job(result, target, "highstate")
        return result

    async def run_cmd(self, target: str, cmd: str) -> dict[str, Any]:
        """Run an arbitrary shell command on the target minion."""
        payload = {
            "client": "local",
            "tgt": target,
            "fun": "cmd.run",
            "arg": [cmd],
        }
        data = await self._post_with_retry(payload)
        return data.get("return", [{}])[0]

    async def get_job_result(self, jid: str) -> dict[str, Any]:
        """Retrieve the results of a completed job by JID."""
        resp = await self._http.get(
            f"/jobs/{jid}",
            headers=self._auth_headers(),
        )
        resp.raise_for_status()
        return resp.json().get("return", [{}])[0]

    def _parse_result(self, data: dict[str, Any]) -> SaltResult:
        raw = data.get("return", [{}])[0]
        jid = data.get("jid", "")
        success: list[str] = []
        failed: list[str] = []

        for minion, minion_result in raw.items():
            if isinstance(minion_result, dict):
                # Each value in a state.apply result is a state execution result.
                all_ok = all(
                    v.get("result", True) is not False
                    for v in minion_result.values()
                    if isinstance(v, dict)
                )
                if all_ok:
                    success.append(minion)
                else:
                    failed.append(minion)
            else:
                success.append(minion)

        return SaltResult(
            jid=jid,
            minions_targeted=list(raw.keys()),
            minions_success=success,
            minions_failed=failed,
            results=raw,
        )

    async def _record_job(self, result: SaltResult, target: str, state: str) -> None:
        """Persist the job outcome into the salt_job_history table."""
        try:
            async with self._db_pool.acquire() as conn:
                await conn.execute(
                    """
                    INSERT INTO salt_job_history
                        (jid, target, state, success, output, applied_at)
                    VALUES ($1, $2, $3, $4, $5, $6)
                    ON CONFLICT (jid) DO UPDATE
                        SET output = EXCLUDED.output,
                            success = EXCLUDED.success
                    """,
                    result.jid,
                    target,
                    state,
                    len(result.minions_failed) == 0,
                    str(result.results),
                    datetime.now(timezone.utc),
                )
        except Exception as exc:  # noqa: BLE001
            logger.error("failed to record salt job", jid=result.jid, error=str(exc))

    async def close(self) -> None:
        """Release resources."""
        await self._http.aclose()
        await self._db_pool.close()
