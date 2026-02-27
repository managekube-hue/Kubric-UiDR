"""
K-KAI Deploy: Fleet Rollout Manager
Manages staged fleet-wide deployments with configurable wave sizes,
health check gates, and automatic pause/resume based on error rate.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
import asyncpg
import httpx

logger = logging.getLogger(__name__)
PG_DSN      = os.getenv("PG_DSN",      "postgresql://kubric:kubric@localhost:5432/kubric")
NOC_API_URL = os.getenv("NOC_API_URL", "http://localhost:8080")

WAVE_SIZE         = int(os.getenv("FLEET_WAVE_SIZE",      "100"))
WAVE_DELAY_SECS   = int(os.getenv("FLEET_WAVE_DELAY",     "60"))
MAX_FAIL_RATE_PCT = float(os.getenv("FLEET_MAX_FAIL_PCT", "5.0"))

@dataclass
class FleetRolloutPlan:
    plan_id:       str
    tenant_id:     str
    package:       str
    version:       str
    total_agents:  int
    waves:         List[List[str]]
    wave_delay_s:  int = WAVE_DELAY_SECS
    max_fail_rate: float = MAX_FAIL_RATE_PCT
    status:        str = "pending"

@dataclass
class WaveResult:
    wave_n:    int
    agents:    int
    ok:        int
    failed:    int
    skipped:   int
    error_rate: float
    paused:    bool = False
    ts:        str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

class FleetRolloutManager:
    def __init__(self, pg_dsn: str = PG_DSN, noc_url: str = NOC_API_URL):
        self.pg_dsn  = pg_dsn
        self.noc_url = noc_url.rstrip("/")
        self._pool: Optional[asyncpg.Pool] = None

    async def connect(self) -> None:
        self._pool = await asyncpg.create_pool(self.pg_dsn, min_size=2, max_size=8)

    async def close(self) -> None:
        if self._pool:
            await self._pool.close()

    def build_waves(self, agent_ids: List[str], wave_size: int = WAVE_SIZE) -> List[List[str]]:
        return [agent_ids[i:i+wave_size] for i in range(0, len(agent_ids), wave_size)]

    async def _deploy_agent(self, agent_id: str, package: str, version: str) -> bool:
        url = f"{self.noc_url}/api/agents/{agent_id}/deploy"
        try:
            async with httpx.AsyncClient(timeout=30) as client:
                resp = await client.post(url, json={"package": package, "version": version})
                return resp.status_code in (200, 201, 202)
        except Exception as exc:
            logger.debug("Deploy error for %s: %s", agent_id, exc)
            return False

    async def _check_wave_health(self, agent_ids: List[str]) -> float:
        """Query NOC for online/healthy status; return failure rate 0-1."""
        try:
            async with httpx.AsyncClient(timeout=15) as client:
                resp = await client.post(
                    f"{self.noc_url}/api/agents/health/batch",
                    json={"agent_ids": agent_ids},
                )
                if resp.status_code == 200:
                    data       = resp.json()
                    unhealthy  = sum(1 for a in data.get("agents", []) if not a.get("healthy"))
                    return unhealthy / max(len(agent_ids), 1)
        except Exception:
            pass
        return 0.0

    async def execute_plan(self, plan: FleetRolloutPlan) -> Dict[str, Any]:
        plan.status      = "running"
        wave_results: List[WaveResult] = []
        total_ok         = 0
        total_failed     = 0

        for i, wave in enumerate(plan.waves):
            logger.info("Wave %d/%d: deploying to %d agents", i+1, len(plan.waves), len(wave))
            tasks  = [self._deploy_agent(a, plan.package, plan.version) for a in wave]
            flags  = await asyncio.gather(*tasks)
            ok     = sum(flags)
            failed = len(flags) - ok
            err_rate = (failed / max(len(wave), 1)) * 100

            # Health gate
            post_fail = await self._check_wave_health(wave)
            if post_fail * 100 > plan.max_fail_rate or err_rate > plan.max_fail_rate:
                logger.warning("Wave %d exceeded failure rate %.1f%% -- pausing", i+1, err_rate)
                wave_results.append(WaveResult(
                    wave_n=i+1, agents=len(wave), ok=ok,
                    failed=failed, skipped=0, error_rate=err_rate, paused=True,
                ))
                plan.status = "paused"
                break

            total_ok     += ok
            total_failed += failed
            wave_results.append(WaveResult(
                wave_n=i+1, agents=len(wave), ok=ok,
                failed=failed, skipped=0, error_rate=err_rate,
            ))
            if i < len(plan.waves) - 1:
                await asyncio.sleep(plan.wave_delay_s)

        if plan.status != "paused":
            plan.status = "completed" if total_failed == 0 else "partial"

        return {
            "plan_id":      plan.plan_id,
            "status":       plan.status,
            "total_agents": plan.total_agents,
            "total_ok":     total_ok,
            "total_failed": total_failed,
            "waves":        len(wave_results),
            "wave_details": [vars(w) for w in wave_results],
        }

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    print(json.dumps({
        "wave_size":       WAVE_SIZE,
        "wave_delay_secs": WAVE_DELAY_SECS,
        "max_fail_rate":   MAX_FAIL_RATE_PCT,
    }))
