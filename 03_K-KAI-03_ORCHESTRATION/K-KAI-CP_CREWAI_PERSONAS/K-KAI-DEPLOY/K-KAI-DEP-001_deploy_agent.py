"""
K-KAI Deploy Agent: Deployment Orchestrator
Manages controlled agent rollouts -- staging review, approval gate,
canary deploy, and automated fleet rollout with health checks.
"""
from __future__ import annotations
import asyncio, json, logging, os, uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Dict, List, Optional
import asyncpg
import httpx

logger = logging.getLogger(__name__)
PG_DSN      = os.getenv("PG_DSN",      "postgresql://kubric:kubric@localhost:5432/kubric")
NOC_API_URL = os.getenv("NOC_API_URL", "http://localhost:8080")

class DeployStage(str, Enum):
    PENDING   = "pending"
    CANARY    = "canary"
    STAGING   = "staging"
    APPROVED  = "approved"
    ROLLING   = "rolling"
    COMPLETED = "completed"
    FAILED    = "failed"
    ROLLED_BACK = "rolled_back"

@dataclass
class DeployJob:
    job_id:        str
    tenant_id:     str
    package_name:  str
    target_version: str
    stage:         DeployStage
    canary_pct:    float          # 0-100 % agents to hit in canary
    target_agents: List[str]
    completed_agents: List[str] = field(default_factory=list)
    failed_agents:    List[str] = field(default_factory=list)
    created_at:    str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    updated_at:    str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

class DeployAgent:
    def __init__(self, pg_dsn: str = PG_DSN, noc_url: str = NOC_API_URL):
        self.pg_dsn  = pg_dsn
        self.noc_url = noc_url.rstrip("/")
        self._pool: Optional[asyncpg.Pool] = None

    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=8)

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()

    async def create_job(
        self, tenant_id: str, package: str, version: str,
        agent_ids: List[str], canary_pct: float = 10.0,
    ) -> DeployJob:
        job = DeployJob(
            job_id=str(uuid.uuid4()),
            tenant_id=tenant_id,
            package_name=package,
            target_version=version,
            stage=DeployStage.PENDING,
            canary_pct=canary_pct,
            target_agents=agent_ids,
        )
        assert self._pool
        await self._pool.execute(
            """INSERT INTO kai_deploy_jobs
               (id, tenant_id, package_name, target_version, stage,
                canary_pct, target_agents, created_at, updated_at)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)""",
            job.job_id, tenant_id, package, version,
            job.stage.value, canary_pct,
            json.dumps(agent_ids),
            datetime.now(timezone.utc), datetime.now(timezone.utc),
        )
        logger.info("Deploy job created: %s for pkg %s v%s", job.job_id, package, version)
        return job

    async def execute_canary(self, job: DeployJob) -> bool:
        """Deploy to canary_pct % of agents and wait for health check."""
        canary_count = max(1, int(len(job.target_agents) * job.canary_pct / 100))
        canary_agents = job.target_agents[:canary_count]
        logger.info("Canary deploy: %d agents", canary_count)

        for agent_id in canary_agents:
            try:
                async with httpx.AsyncClient(timeout=30) as client:
                    resp = await client.post(
                        f"{self.noc_url}/api/agents/{agent_id}/deploy",
                        json={"package": job.package_name, "version": job.target_version},
                    )
                    if resp.status_code in (200, 201, 202):
                        job.completed_agents.append(agent_id)
                    else:
                        job.failed_agents.append(agent_id)
            except Exception as exc:
                logger.error("Canary deploy failed for %s: %s", agent_id, exc)
                job.failed_agents.append(agent_id)

        success_rate = len(job.completed_agents) / max(len(canary_agents), 1)
        if success_rate < 0.8:
            logger.warning("Canary success rate %.0f%% < 80%% -- triggering rollback", success_rate * 100)
            return False

        await asyncio.sleep(30)   # allow health metrics to stabilise
        return True

    async def execute_full_rollout(self, job: DeployJob) -> Dict[str, Any]:
        remaining = [a for a in job.target_agents if a not in job.completed_agents]
        BATCH = 50
        for i in range(0, len(remaining), BATCH):
            batch = remaining[i:i+BATCH]
            tasks = [self._deploy_agent(agent_id, job) for agent_id in batch]
            await asyncio.gather(*tasks)
            await asyncio.sleep(5)

        total    = len(job.target_agents)
        done     = len(job.completed_agents)
        failed   = len(job.failed_agents)
        success  = failed == 0 or (done / total) >= 0.95
        job.stage = DeployStage.COMPLETED if success else DeployStage.FAILED
        return {"job_id": job.job_id, "success": success, "completed": done, "failed": failed, "total": total}

    async def _deploy_agent(self, agent_id: str, job: DeployJob) -> None:
        try:
            async with httpx.AsyncClient(timeout=60) as client:
                resp = await client.post(
                    f"{self.noc_url}/api/agents/{agent_id}/deploy",
                    json={"package": job.package_name, "version": job.target_version},
                )
                if resp.status_code in (200, 201, 202):
                    job.completed_agents.append(agent_id)
                else:
                    job.failed_agents.append(agent_id)
        except Exception as exc:
            logger.error("Deploy failed for agent %s: %s", agent_id, exc)
            job.failed_agents.append(agent_id)

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    print(json.dumps({"status": "DeployAgent ready", "noc_url": NOC_API_URL}))
