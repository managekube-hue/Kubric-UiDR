"""
KAI FastAPI application.

Phase 1 stub — health and readiness endpoints only.
Layer 2 will add:
  - POST /v1/triage       (alert enrichment via CrewAI)
  - POST /v1/score        (risk scoring)
  - GET  /v1/insights     (KAI-SENTINEL health scores)
  - POST /v1/remediate    (KAI-KEEPER remediation plan)
"""

import orjson
from fastapi import FastAPI
from fastapi.responses import JSONResponse

from kai.config import settings


class ORJSONResponse(JSONResponse):
    media_type = "application/json"

    def render(self, content: object) -> bytes:
        return orjson.dumps(content)


app = FastAPI(
    title="Kubric KAI",
    version="0.1.0",
    default_response_class=ORJSONResponse,
)


@app.get("/healthz")
async def health() -> dict:
    return {"status": "ok", "tenant_id": settings.tenant_id}


@app.get("/readyz")
async def ready() -> dict:
    # Layer 2: check NATS connectivity, ClickHouse ping, model availability
    return {"status": "ok"}
