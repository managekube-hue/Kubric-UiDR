"""
K-KAI Workflow: Flower Monitor
Polls the Celery Flower REST API to monitor task queues, detect
stalled workers, and publish queue depth metrics to NATS.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Dict, List, Optional
import httpx
import nats

logger = logging.getLogger(__name__)
FLOWER_URL = os.getenv("FLOWER_URL",  "http://localhost:5555")
NATS_URL   = os.getenv("NATS_URL",   "nats://localhost:4222")
POLL_SECS  = int(os.getenv("FLOWER_POLL_SECS", "30"))
STALE_SECS = int(os.getenv("FLOWER_STALE_SECS", "300"))  # worker considered stale after 5min

@dataclass
class WorkerStatus:
    name:         str
    status:       str        # online / offline / stale
    active_tasks: int
    processed:    int
    failed:        int
    queues:       List[str] = field(default_factory=list)
    last_heartbeat: Optional[str] = None

@dataclass
class FlowerMetrics:
    workers:        List[WorkerStatus]
    total_active:   int
    total_processed: int
    total_failed:   int
    queues:         Dict[str, int]      # queue_name -> pending tasks
    ts:             str

class FlowerMonitor:
    def __init__(self, flower_url: str = FLOWER_URL, nats_url: str = NATS_URL):
        self.flower_url = flower_url.rstrip("/")
        self.nats_url   = nats_url
        self._nc        = None

    async def _fetch_workers(self) -> Dict:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(f"{self.flower_url}/api/workers")
            resp.raise_for_status()
            return resp.json()

    async def _fetch_queues(self) -> List[Dict]:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(f"{self.flower_url}/api/queues/length")
            resp.raise_for_status()
            return resp.json().get("active_queues", [])

    def _parse_workers(self, raw: Dict) -> List[WorkerStatus]:
        workers = []
        for name, info in raw.items():
            active = len(info.get("active", {}) or {})
            ws = WorkerStatus(
                name=name,
                status="online" if info.get("status") else "offline",
                active_tasks=active,
                processed=info.get("processed", 0),
                failed=info.get("failed", 0) or info.get("stats", {}).get("total", {}).get("failure", 0),
                queues=[q.get("name", "") for q in (info.get("queues") or [])],
                last_heartbeat=info.get("heartbeat"),
            )
            workers.append(ws)
        return workers

    async def collect(self) -> FlowerMetrics:
        raw_workers = await self._fetch_workers()
        raw_queues  = await self._fetch_queues()
        workers     = self._parse_workers(raw_workers)
        queue_map   = {q["name"]: q.get("messages", 0) for q in raw_queues}
        return FlowerMetrics(
            workers=workers,
            total_active=sum(w.active_tasks for w in workers),
            total_processed=sum(w.processed for w in workers),
            total_failed=sum(w.failed for w in workers),
            queues=queue_map,
            ts=datetime.now(timezone.utc).isoformat(),
        )

    async def publish(self, metrics: FlowerMetrics) -> None:
        if not self._nc:
            self._nc = await nats.connect(self.nats_url)
        payload = json.dumps({
            "total_active":    metrics.total_active,
            "total_processed": metrics.total_processed,
            "total_failed":    metrics.total_failed,
            "queues":          metrics.queues,
            "worker_count":    len(metrics.workers),
            "ts":              metrics.ts,
        }).encode()
        await self._nc.publish("kai.metrics.celery", payload)

    async def run_forever(self) -> None:
        logger.info("FlowerMonitor started polling %s every %ds", self.flower_url, POLL_SECS)
        while True:
            try:
                metrics = await self.collect()
                await self.publish(metrics)
                # Warn on stalled workers
                for w in metrics.workers:
                    if w.status == "offline":
                        logger.warning("Worker %s is OFFLINE", w.name)
            except Exception as exc:
                logger.error("Flower poll error: %s", exc)
            await asyncio.sleep(POLL_SECS)

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    asyncio.run(FlowerMonitor().run_forever())
