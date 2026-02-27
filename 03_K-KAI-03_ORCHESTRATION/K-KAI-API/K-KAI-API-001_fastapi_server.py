"""
K-KAI-API-001: FastAPI Application Server for KAI Orchestration Layer
Mounts routers for /agents, /incidents, /health, /metrics.
Connects to NATS JetStream and asyncpg on startup via lifespan.
Validates X-Tenant-ID header and logs all requests via middleware.
"""

import os
import time
import logging
from contextlib import asynccontextmanager
from typing import Any

import orjson
from fastapi import FastAPI, Request, Response, HTTPException, APIRouter, Depends
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
import nats
import asyncpg

logger = logging.getLogger("kai.api")
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)

_startup_time: float = time.time()

# ---------------------------------------------------------------------------
# Shared application state
# ---------------------------------------------------------------------------
class _AppState:
    nats_client: Any = None
    js: Any = None
    pg_pool: asyncpg.Pool | None = None

state = _AppState()

# ---------------------------------------------------------------------------
# Lifespan context manager
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):  # noqa: ARG001
    logger.info("KAI API starting up")

    nats_url = os.environ.get("NATS_URL", "nats://localhost:4222")
    try:
        state.nats_client = await nats.connect(
            nats_url,
            reconnect_time_wait=2,
            max_reconnect_attempts=-1,
        )
        state.js = state.nats_client.jetstream()
        logger.info("Connected to NATS at %s", nats_url)
    except Exception as exc:
        logger.error("NATS connection failed: %s", exc)

    db_url = os.environ.get("DATABASE_URL", "postgresql://kai:kai@localhost:5432/kai")
    try:
        state.pg_pool = await asyncpg.create_pool(db_url, min_size=2, max_size=10)
        logger.info("asyncpg pool ready")
    except Exception as exc:
        logger.error("asyncpg pool failed: %s", exc)

    yield  # application runs here

    logger.info("KAI API shutting down")
    if state.pg_pool:
        await state.pg_pool.close()
    if state.nats_client and not state.nats_client.is_closed:
        await state.nats_client.drain()

# ---------------------------------------------------------------------------
# FastAPI application
# ---------------------------------------------------------------------------
app = FastAPI(
    title="KAI Orchestration API",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

# ---------------------------------------------------------------------------
# Middleware: X-Tenant-ID validation + structured request logging
# ---------------------------------------------------------------------------
_NO_TENANT_PATHS = frozenset({"/health", "/metrics", "/docs", "/openapi.json", "/redoc"})

@app.middleware("http")
async def tenant_and_logging_middleware(request: Request, call_next):
    path = request.url.path
    tenant_id = request.headers.get("X-Tenant-ID", "")

    if path not in _NO_TENANT_PATHS and not tenant_id:
        return JSONResponse(
            status_code=400,
            content={"detail": "Missing required header: X-Tenant-ID"},
        )

    t0 = time.perf_counter()
    response: Response = await call_next(request)
    duration_ms = (time.perf_counter() - t0) * 1000

    logger.info(
        "method=%s path=%s tenant=%s status=%d duration_ms=%.1f",
        request.method,
        path,
        tenant_id or "-",
        response.status_code,
        duration_ms,
    )
    response.headers["X-Response-Time-Ms"] = f"{duration_ms:.1f}"
    return response

# ---------------------------------------------------------------------------
# Dependency: extract tenant
# ---------------------------------------------------------------------------
def _tenant(request: Request) -> str:
    return request.headers.get("X-Tenant-ID", "unknown")

# ---------------------------------------------------------------------------
# Persona registry
# ---------------------------------------------------------------------------
PERSONAS = ["TRIAGE", "ANALYST", "HUNTER", "KEEPER", "RISK", "INVEST", "SENTINEL", "FORESIGHT"]
_agent_status: dict[str, str] = {p: "idle" for p in PERSONAS}

# ---------------------------------------------------------------------------
# /health & /metrics
# ---------------------------------------------------------------------------
health_router = APIRouter(tags=["infrastructure"])

@health_router.get("/health")
async def health_check():
    return {
        "status": "ok",
        "uptime_s": round(time.time() - _startup_time, 2),
        "nats_connected": bool(
            state.nats_client and state.nats_client.is_connected
        ),
        "pg_pool_size": state.pg_pool.get_size() if state.pg_pool else 0,
    }

@health_router.get("/metrics")
async def metrics():
    return {
        "uptime_s": round(time.time() - _startup_time, 2),
        "agent_statuses": dict(_agent_status),
        "pg_pool_idle": state.pg_pool.get_idle_size() if state.pg_pool else 0,
    }

# ---------------------------------------------------------------------------
# /agents
# ---------------------------------------------------------------------------
agents_router = APIRouter(prefix="/agents", tags=["agents"])

@agents_router.get("")
async def list_agents(tenant: str = Depends(_tenant)):
    return [
        {"persona": p, "status": _agent_status[p], "tenant_id": tenant}
        for p in PERSONAS
    ]

@agents_router.post("/{persona}/trigger")
async def trigger_agent(
    persona: str,
    request: Request,
    tenant: str = Depends(_tenant),
):
    key = persona.upper()
    if key not in PERSONAS:
        raise HTTPException(status_code=404, detail=f"Unknown persona: {persona}")

    body: dict = {}
    try:
        raw = await request.body()
        if raw:
            body = orjson.loads(raw)
    except Exception:
        pass

    _agent_status[key] = "triggered"
    subject = f"kubric.{tenant}.agent.{key}.trigger"
    payload = orjson.dumps({"persona": key, "tenant_id": tenant, "params": body})

    if state.js:
        try:
            await state.js.publish(subject, payload)
        except Exception as exc:
            logger.warning("NATS publish failed for %s: %s", subject, exc)

    return {"status": "triggered", "persona": key, "tenant_id": tenant}

# ---------------------------------------------------------------------------
# /incidents
# ---------------------------------------------------------------------------
incidents_router = APIRouter(prefix="/incidents", tags=["incidents"])

@incidents_router.get("")
async def list_incidents(tenant: str = Depends(_tenant)):
    if not state.pg_pool:
        raise HTTPException(status_code=503, detail="Database unavailable")
    rows = await state.pg_pool.fetch(
        "SELECT * FROM incidents WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT 100",
        tenant,
    )
    return [dict(r) for r in rows]

@incidents_router.get("/{incident_id}")
async def get_incident(incident_id: str, tenant: str = Depends(_tenant)):
    if not state.pg_pool:
        raise HTTPException(status_code=503, detail="Database unavailable")
    row = await state.pg_pool.fetchrow(
        "SELECT * FROM incidents WHERE id=$1 AND tenant_id=$2",
        incident_id,
        tenant,
    )
    if not row:
        raise HTTPException(status_code=404, detail="Incident not found")
    return dict(row)

# ---------------------------------------------------------------------------
# Mount routers
# ---------------------------------------------------------------------------
app.include_router(health_router)
app.include_router(agents_router)
app.include_router(incidents_router)
