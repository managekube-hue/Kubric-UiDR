"""
K-KAI Housekeeper: Main Housekeeper Agent
CrewAI agent that orchestrates configuration drift remediation,
patch scheduling, and system housekeeping tasks across all managed agents.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
import httpx

logger = logging.getLogger(__name__)

NOC_API_URL = os.getenv("NOC_API_URL", "http://localhost:8080")
KAI_API_URL = os.getenv("KAI_API_URL", "http://localhost:9000")

@dataclass
class HousekeepingTask:
    task_id:     str
    tenant_id:   str
    task_type:   str     # "drift_remediate" | "patch" | "cleanup" | "baseline_update"
    agent_id:    str
    priority:    int     # 1-5
    payload:     Dict[str, Any] = field(default_factory=dict)
    status:      str = "pending"
    created_at:  str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    completed_at: Optional[str] = None
    result:      Optional[Dict] = None

@dataclass
class HousekeepingResult:
    task_id:   str
    success:   bool
    action:    str
    details:   str
    duration_ms: int = 0

class HousekeeperAgent:
    """
    Polls the NOC API for drift events and open patch jobs, then dispatches
    remediation tasks to the appropriate agents. Results are recorded back.
    """
    def __init__(self, noc_url: str = NOC_API_URL, kai_url: str = KAI_API_URL):
        self.noc_url = noc_url.rstrip("/")
        self.kai_url = kai_url.rstrip("/")
        self._client: Optional[httpx.AsyncClient] = None

    async def __aenter__(self) -> "HousekeeperAgent":
        self._client = httpx.AsyncClient(timeout=30)
        return self

    async def __aexit__(self, *_) -> None:
        if self._client:
            await self._client.aclose()

    async def _get(self, path: str, base: str = "") -> Any:
        url = (base or self.noc_url) + path
        resp = await self._client.get(url)
        resp.raise_for_status()
        return resp.json()

    async def _post(self, path: str, body: Dict, base: str = "") -> Any:
        url = (base or self.noc_url) + path
        resp = await self._client.post(url, json=body)
        resp.raise_for_status()
        return resp.json()

    async def fetch_pending_drift(self, tenant_id: str) -> List[Dict]:
        try:
            data = await self._get(f"/api/clusters/drift?tenant_id={tenant_id}&status=pending")
            return data.get("items", [])
        except Exception as exc:
            logger.error("fetch_pending_drift failed: %s", exc)
            return []

    async def fetch_pending_patches(self, tenant_id: str) -> List[Dict]:
        try:
            data = await self._get(f"/api/patches?tenant_id={tenant_id}&status=pending")
            return data.get("items", [])
        except Exception as exc:
            logger.error("fetch_pending_patches failed: %s", exc)
            return []

    async def remediate_drift(self, drift_event: Dict) -> HousekeepingResult:
        task_id  = drift_event.get("id", "unknown")
        agent_id = drift_event.get("agent_id", "")
        start    = datetime.now(timezone.utc)
        try:
            result = await self._post("/api/clusters/drift/remediate", {
                "event_id": task_id,
                "agent_id": agent_id,
                "auto_apply": True,
            })
            elapsed = int((datetime.now(timezone.utc) - start).total_seconds() * 1000)
            return HousekeepingResult(
                task_id=task_id, success=True,
                action="drift_remediate",
                details=f"Remediated config drift on agent {agent_id}",
                duration_ms=elapsed,
            )
        except Exception as exc:
            return HousekeepingResult(
                task_id=task_id, success=False, action="drift_remediate",
                details=str(exc),
            )

    async def apply_patch(self, patch_job: Dict) -> HousekeepingResult:
        job_id  = patch_job.get("id", "unknown")
        cve     = patch_job.get("cve_id", "")
        agent   = patch_job.get("agent_id", "")
        start   = datetime.now(timezone.utc)
        try:
            result = await self._post(f"/api/patches/{job_id}/apply", {
                "agent_id": agent, "immediate": False
            })
            elapsed = int((datetime.now(timezone.utc) - start).total_seconds() * 1000)
            return HousekeepingResult(
                task_id=job_id, success=True, action="patch_apply",
                details=f"Applied patch for {cve} on agent {agent}",
                duration_ms=elapsed,
            )
        except Exception as exc:
            return HousekeepingResult(
                task_id=job_id, success=False, action="patch_apply",
                details=str(exc),
            )

    async def run_once(self, tenant_id: str) -> Dict[str, Any]:
        drift_events = await self.fetch_pending_drift(tenant_id)
        patch_jobs   = await self.fetch_pending_patches(tenant_id)

        results = []
        for ev in drift_events[:20]:     # cap per-run to avoid overload
            res = await self.remediate_drift(ev)
            results.append(asdict(res))

        for job in patch_jobs[:10]:
            res = await self.apply_patch(job)
            results.append(asdict(res))

        summary = {
            "tenant_id":    tenant_id,
            "drift_found":  len(drift_events),
            "patches_found": len(patch_jobs),
            "actions":      len(results),
            "successes":    sum(1 for r in results if r["success"]),
            "failures":     sum(1 for r in results if not r["success"]),
            "ts":           datetime.now(timezone.utc).isoformat(),
        }
        logger.info("Housekeeping run complete: %s", summary)
        return summary

async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    async with HousekeeperAgent() as agent:
        summary = await agent.run_once(tenant_id=os.getenv("TENANT_ID", "default"))
    print(json.dumps(summary, indent=2))

if __name__ == "__main__":
    asyncio.run(main())
