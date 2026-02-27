"""
K-KAI Housekeeper: Rollback Manager
Manages automated and manual rollback of configuration changes and patches.
Stores rollback snapshots in PostgreSQL and coordinates agent-side restoration.
"""
from __future__ import annotations
import asyncio, json, logging, os, uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
import asyncpg
import httpx

logger = logging.getLogger(__name__)
PG_DSN      = os.getenv("PG_DSN",      "postgresql://kubric:kubric@localhost:5432/kubric")
NOC_API_URL = os.getenv("NOC_API_URL", "http://localhost:8080")

@dataclass
class RollbackSnapshot:
    snapshot_id: str
    tenant_id:   str
    agent_id:    str
    trigger:     str     # "patch" | "config_change" | "manual"
    config_blob: Dict[str, Any]
    created_at:  str

@dataclass
class RollbackResult:
    snapshot_id: str
    agent_id:    str
    success:     bool
    message:     str
    rolled_back_at: str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

class RollbackManager:
    def __init__(self, pg_dsn: str = PG_DSN, noc_url: str = NOC_API_URL):
        self.pg_dsn  = pg_dsn
        self.noc_url = noc_url.rstrip("/")
        self._pool: Optional[asyncpg.Pool] = None

    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=8)

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()

    async def create_snapshot(
        self,
        tenant_id: str,
        agent_id:  str,
        trigger:   str,
        config:    Dict[str, Any],
    ) -> RollbackSnapshot:
        snap_id = str(uuid.uuid4())
        now     = datetime.now(timezone.utc)
        assert self._pool
        await self._pool.execute(
            """INSERT INTO kai_rollback_snapshots
               (id, tenant_id, agent_id, trigger, config_blob, created_at)
               VALUES ($1, $2, $3, $4, $5, $6)""",
            snap_id, tenant_id, agent_id, trigger, json.dumps(config), now,
        )
        logger.info("Snapshot created: %s for agent %s", snap_id, agent_id)
        return RollbackSnapshot(
            snapshot_id=snap_id,
            tenant_id=tenant_id,
            agent_id=agent_id,
            trigger=trigger,
            config_blob=config,
            created_at=now.isoformat(),
        )

    async def get_latest_snapshot(
        self, tenant_id: str, agent_id: str
    ) -> Optional[RollbackSnapshot]:
        assert self._pool
        row = await self._pool.fetchrow(
            """SELECT * FROM kai_rollback_snapshots
               WHERE tenant_id=$1 AND agent_id=$2
               ORDER BY created_at DESC LIMIT 1""",
            tenant_id, agent_id,
        )
        if not row:
            return None
        return RollbackSnapshot(
            snapshot_id=row["id"],
            tenant_id=row["tenant_id"],
            agent_id=row["agent_id"],
            trigger=row["trigger"],
            config_blob=json.loads(row["config_blob"]),
            created_at=row["created_at"].isoformat(),
        )

    async def execute_rollback(self, snapshot: RollbackSnapshot) -> RollbackResult:
        """Push rollback config to agent via NOC provisioning API."""
        url = f"{self.noc_url}/api/agents/{snapshot.agent_id}/config/apply"
        try:
            async with httpx.AsyncClient(timeout=30) as client:
                resp = await client.post(url, json={
                    "config":      snapshot.config_blob,
                    "rollback_id": snapshot.snapshot_id,
                    "reason":      f"Rollback to snapshot {snapshot.snapshot_id}",
                })
                resp.raise_for_status()
            logger.info("Rollback executed: agent=%s snap=%s",
                        snapshot.agent_id, snapshot.snapshot_id)
            return RollbackResult(
                snapshot_id=snapshot.snapshot_id,
                agent_id=snapshot.agent_id,
                success=True,
                message="Rollback applied successfully",
            )
        except Exception as exc:
            logger.error("Rollback failed: %s", exc)
            return RollbackResult(
                snapshot_id=snapshot.snapshot_id,
                agent_id=snapshot.agent_id,
                success=False,
                message=str(exc),
            )

    async def auto_rollback_on_failure(
        self, tenant_id: str, agent_id: str, failure_reason: str
    ) -> RollbackResult:
        snap = await self.get_latest_snapshot(tenant_id, agent_id)
        if not snap:
            return RollbackResult(
                snapshot_id="none", agent_id=agent_id,
                success=False, message="No snapshot available for rollback",
            )
        logger.warning("Auto-rollback triggered for agent %s: %s", agent_id, failure_reason)
        return await self.execute_rollback(snap)

async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    mgr = RollbackManager()
    await mgr.connect()
    try:
        snap = await mgr.create_snapshot(
            tenant_id="demo-tenant",
            agent_id="agent-001",
            trigger="patch",
            config={"version": "1.2.3", "settings": {"log_level": "info"}},
        )
        print(json.dumps({"snapshot_id": snap.snapshot_id}, indent=2))
    finally:
        await mgr.close()

if __name__ == "__main__":
    asyncio.run(main())
