"""
KAI FastAPI application — Layer 2.

Endpoints:
  GET  /healthz                    health probe
  GET  /readyz                     readiness probe (NATS + LLM checks)
  POST /v1/triage                  KAI-TRIAGE: enrich a raw OCSF event
  POST /v1/score/{tenant_id}       KAI-SENTINEL: compute KiSS health score
  GET  /v1/insights/{tenant_id}    KAI-SENTINEL: AI-generated security narrative
  POST /v1/remediate               KAI-KEEPER: submit remediation plan
  POST /v1/billing/run             KAI-CLERK: trigger billing aggregation
  POST /v1/n8n/callback            n8n bridge: forward incident payload to n8n webhook
  POST /v1/webhook/stripe          Stripe webhook: subscription + invoice events

Background tasks started at lifespan:
  - Vault secret injection (VAULT_ADDR must be set; silently skipped otherwise)
  - NATS subscriber (routes kubric.* events to agents)
  - KAI-FORESIGHT periodic loop
"""

from __future__ import annotations

import asyncio
import hashlib
import hmac
import os
import time
from contextlib import asynccontextmanager
from typing import Any

import httpx
import orjson
import structlog
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from kai.config import settings
from kai.core.nats_client import nats_client
from kai.core.subscriber import start_subscriber, stop_subscriber

log = structlog.get_logger(__name__)

# ─── startup / shutdown ───────────────────────────────────────────────────────

_subscriptions: list[object] = []
_background_tasks: list[asyncio.Task] = []  # type: ignore[type-arg]


@asynccontextmanager
async def lifespan(app: FastAPI):  # type: ignore[type-ignore]
    # ── startup ──
    # Vault secret injection — must happen before NATS/LLM init so downstream
    # components pick up the injected env vars.
    try:
        from kai.core.vault import inject_vault_secrets  # noqa: PLC0415
        inject_vault_secrets()
    except Exception as exc:
        log.warning("lifespan.vault_inject_failed", error=str(exc))

    await nats_client.connect()
    _subscriptions.extend(await start_subscriber())

    # KAI-FORESIGHT periodic loop (background)
    tenant_ids_env = os.getenv("KUBRIC_KNOWN_TENANTS", settings.tenant_id)
    known_tenants = [t.strip() for t in tenant_ids_env.split(",") if t.strip()]
    try:
        from kai.agents.foresight import run_foresight_loop  # noqa: PLC0415
        task = asyncio.create_task(run_foresight_loop(known_tenants))
        _background_tasks.append(task)
    except Exception as exc:
        log.warning("lifespan.foresight_start_failed", error=str(exc))

    log.info("kai.started", port=settings.api_port)
    yield

    # ── shutdown ──
    for task in _background_tasks:
        task.cancel()
    await stop_subscriber(_subscriptions)
    await nats_client.disconnect()
    log.info("kai.stopped")


# ─── app factory ─────────────────────────────────────────────────────────────

class ORJSONResponse(JSONResponse):
    media_type = "application/json"

    def render(self, content: object) -> bytes:
        return orjson.dumps(content)


app = FastAPI(
    title="Kubric KAI",
    version="0.2.0",
    description="Kubric AI orchestration layer — Layer 2",
    default_response_class=ORJSONResponse,
    lifespan=lifespan,
)


# ─── Pydantic request/response models ────────────────────────────────────────

class TriageRequest(BaseModel):
    event: dict[str, Any] = Field(..., description="Raw OCSF event as JSON object")
    tenant_id: str = Field(default="", description="Tenant ID (overrides env if provided)")


class TriageResponse(BaseModel):
    triage_id: str
    tenant_id: str
    severity: str
    mitre_techniques: list[str]
    summary: str
    recommended_action: str
    confidence: float
    model_used: str


class ScoreResponse(BaseModel):
    tenant_id: str
    computed_at: int
    kiss_score: float
    vuln_score: float
    compliance_score: float
    detection_score: float
    response_score: float


class InsightsResponse(BaseModel):
    tenant_id: str
    generated_at: int
    narrative: str
    top_risks: list[str]
    recommended_actions: list[str]


class RemediateRequest(BaseModel):
    finding_id: str
    tenant_id: str = ""
    finding: dict[str, Any] = Field(default_factory=dict)
    auto_apply: bool = False


class RemediateResponse(BaseModel):
    plan_id: str
    tenant_id: str
    remediation_type: str
    steps: list[str]
    ansible_playbook: str | None
    estimated_risk: str
    auto_safe: bool
    status: str


class BillingRunRequest(BaseModel):
    tenant_id: str
    stripe_customer_id: str
    billing_period: str = Field(..., pattern=r"^\d{4}-\d{2}$", description="YYYY-MM")


class N8nCallbackRequest(BaseModel):
    event_type: str = Field(..., description="Event type, e.g. 'triage.enriched'")
    tenant_id: str = Field(default="", description="Tenant ID")
    severity: str = Field(default="LOW", description="CRITICAL/HIGH/MEDIUM/LOW/INFO")
    summary: str = Field(default="", description="Human-readable incident summary")
    incident_id: str = Field(default="", description="Triage or finding ID")


# ─── probes ──────────────────────────────────────────────────────────────────

@app.get("/healthz")
async def health() -> dict[str, str]:
    return {"status": "ok", "tenant_id": settings.tenant_id}


@app.get("/readyz")
async def ready() -> dict[str, Any]:
    checks: dict[str, str] = {}

    # NATS
    checks["nats"] = "ok" if nats_client.connected else "degraded"

    # Ollama
    try:
        async with httpx.AsyncClient(timeout=2.0) as client:
            resp = await client.get(f"{settings.ollama_url}/api/tags")
            checks["ollama"] = "ok" if resp.status_code == 200 else "degraded"
    except Exception:
        checks["ollama"] = "degraded"

    overall = "ok" if all(v == "ok" for v in checks.values()) else "degraded"
    return {"status": overall, "checks": checks}


# ─── KAI-TRIAGE  ─────────────────────────────────────────────────────────────

@app.post("/v1/triage", response_model=TriageResponse)
async def triage(req: TriageRequest) -> Any:
    """
    Enrich a raw OCSF event using KAI-TRIAGE.
    Calls Ollama (local) → derives severity, MITRE techniques, summary.
    """
    from kai.agents.triage import TriageAgent  # noqa: PLC0415

    tenant_id = req.tenant_id or settings.tenant_id
    # Build a synthetic subject so TriageAgent can extract tenant_id
    subject = f"kubric.{tenant_id}.api.triage.v1"

    try:
        result = await TriageAgent().handle(subject=subject, event=req.event)
    except Exception as exc:
        log.error("api.triage_error", error=str(exc))
        raise HTTPException(status_code=500, detail=str(exc))

    return TriageResponse(**result)


# ─── KAI-SENTINEL ────────────────────────────────────────────────────────────

@app.post("/v1/score/{tenant_id}", response_model=ScoreResponse)
async def score(tenant_id: str) -> Any:
    """
    Compute the KiSS health score for a tenant.
    Reads from PostgreSQL (vdr_findings, kic_assessments) and ClickHouse.
    """
    from kai.agents.sentinel import SentinelAgent  # noqa: PLC0415

    try:
        result = await SentinelAgent().compute(tenant_id)
    except Exception as exc:
        log.error("api.score_error", tenant_id=tenant_id, error=str(exc))
        raise HTTPException(status_code=500, detail=str(exc))

    return ScoreResponse(**result)


# ─── KAI-SENTINEL insights ───────────────────────────────────────────────────

@app.get("/v1/insights/{tenant_id}", response_model=InsightsResponse)
async def insights(tenant_id: str) -> Any:
    """
    Generate an AI security narrative for a tenant.
    Aggregates recent triage results and asks Ollama to produce a QBR-style summary.
    """
    from kai.agents.sentinel import SentinelAgent  # noqa: PLC0415
    from kai.core.llm import complete_json  # noqa: PLC0415

    try:
        score_data = await SentinelAgent().compute(tenant_id)

        prompt = (
            f"Tenant security posture (last 24h):\n"
            f"  KiSS score: {score_data['kiss_score']}/100\n"
            f"  Vuln score: {score_data['vuln_score']}/100\n"
            f"  Compliance: {score_data['compliance_score']}/100\n"
            f"  Detection:  {score_data['detection_score']}/100\n"
            f"  Response:   {score_data['response_score']}/100\n\n"
            "Return a JSON object with keys: "
            '{"narrative": "2-3 sentence executive summary", '
            '"top_risks": ["risk 1", "risk 2", "risk 3"], '
            '"recommended_actions": ["action 1", "action 2"]}. '
            "JSON only, no preamble."
        )
        ai_result = await complete_json(prompt, max_tokens=600)

    except Exception as exc:
        log.error("api.insights_error", tenant_id=tenant_id, error=str(exc))
        raise HTTPException(status_code=500, detail=str(exc))

    return InsightsResponse(
        tenant_id=tenant_id,
        generated_at=int(time.time() * 1000),
        narrative=str(ai_result.get("narrative", "Insufficient data to generate narrative.")),
        top_risks=list(ai_result.get("top_risks", [])),
        recommended_actions=list(ai_result.get("recommended_actions", [])),
    )


# ─── KAI-KEEPER ──────────────────────────────────────────────────────────────

@app.post("/v1/remediate", response_model=RemediateResponse)
async def remediate(req: RemediateRequest) -> Any:
    """
    Generate a remediation plan for a finding.
    If auto_apply=True AND LLM deems it auto_safe, triggers Temporal workflow.
    """
    from kai.agents.keeper import KeeperAgent  # noqa: PLC0415

    tenant_id = req.tenant_id or settings.tenant_id
    subject = f"kubric.{tenant_id}.api.remediate.v1"

    event = {"id": req.finding_id, "event_id": req.finding_id, **req.finding}

    try:
        result = await KeeperAgent().handle(subject=subject, event=event)
    except Exception as exc:
        log.error("api.remediate_error", error=str(exc))
        raise HTTPException(status_code=500, detail=str(exc))

    return RemediateResponse(**result)


# ─── KAI-CLERK billing ───────────────────────────────────────────────────────

@app.post("/v1/billing/run")
async def billing_run(req: BillingRunRequest) -> dict[str, str]:
    """
    Trigger a billing aggregation run for a tenant.
    Submits a Temporal BillingWorkflow (or runs directly if Temporal unavailable).
    """
    from kai.workflows.billing import submit_billing  # noqa: PLC0415

    try:
        await submit_billing(
            tenant_id=req.tenant_id,
            stripe_customer_id=req.stripe_customer_id,
            billing_period=req.billing_period,
        )
    except Exception as exc:
        log.error("api.billing_error", error=str(exc))
        raise HTTPException(status_code=500, detail=str(exc))

    return {"status": "submitted", "tenant_id": req.tenant_id, "period": req.billing_period}


# ─── Stripe webhook ──────────────────────────────────────────────────────────

# Stripe events handled by this endpoint.
_STRIPE_HANDLED_EVENTS = {
    "customer.subscription.created",
    "customer.subscription.updated",
    "customer.subscription.deleted",
    "invoice.payment_succeeded",
    "invoice.payment_failed",
}

# Tolerance for Stripe timestamp replay-attack protection (5 minutes).
_STRIPE_TOLERANCE_SECS = 300


def _verify_stripe_signature(payload: bytes, sig_header: str, secret: str) -> dict[str, Any]:
    """
    Validate a Stripe webhook signature and return the parsed event dict.

    Uses Stripe's v1 scheme: HMAC-SHA256 over ``{timestamp}.{payload}``.
    Raises HTTPException 400 if the signature is invalid or stale.
    """
    if not secret:
        raise HTTPException(status_code=400, detail="Stripe webhook secret not configured")

    # Parse "t=<ts>,v1=<sig1>,v1=<sig2>..."
    parts: dict[str, list[str]] = {}
    for part in sig_header.split(","):
        k, _, v = part.partition("=")
        parts.setdefault(k.strip(), []).append(v.strip())

    timestamps = parts.get("t", [])
    v1_sigs    = parts.get("v1", [])

    if not timestamps or not v1_sigs:
        raise HTTPException(status_code=400, detail="Invalid Stripe-Signature header")

    try:
        ts = int(timestamps[0])
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid Stripe-Signature timestamp")

    # Replay-attack protection
    now = int(time.time())
    if abs(now - ts) > _STRIPE_TOLERANCE_SECS:
        raise HTTPException(status_code=400, detail="Stripe webhook timestamp out of tolerance")

    signed_payload = f"{ts}.".encode() + payload
    expected = hmac.new(
        secret.encode("utf-8"), signed_payload, hashlib.sha256
    ).hexdigest()

    if not any(hmac.compare_digest(expected, sig) for sig in v1_sigs):
        raise HTTPException(status_code=400, detail="Stripe signature mismatch")

    return orjson.loads(payload)


@app.post("/v1/webhook/stripe")
async def stripe_webhook(request: Request) -> dict[str, Any]:
    """
    Stripe webhook receiver.

    Validates HMAC-SHA256 Stripe-Signature, routes the event, and publishes
    a NATS lifecycle event so KAI-CLERK and downstream services can react.

    Expected env var: KUBRIC_STRIPE_WEBHOOK_SECRET
    """
    payload    = await request.body()
    sig_header = request.headers.get("stripe-signature", "")
    secret     = os.getenv("KUBRIC_STRIPE_WEBHOOK_SECRET", "")

    event = _verify_stripe_signature(payload, sig_header, secret)

    event_type  = event.get("type", "")
    event_id    = event.get("id", "")
    data_object = event.get("data", {}).get("object", {})

    if event_type not in _STRIPE_HANDLED_EVENTS:
        log.debug("stripe.ignored", event_type=event_type)
        return {"received": True, "handled": False, "type": event_type}

    log.info("stripe.event", event_type=event_type, event_id=event_id)

    # Derive tenant_id from Stripe metadata — fallback to customer_id
    tenant_id   = (
        data_object.get("metadata", {}).get("kubric_tenant_id")
        or data_object.get("customer", "unknown")
    )
    customer_id = data_object.get("customer", data_object.get("id", ""))

    # Publish lifecycle event to NATS
    nats_subject = f"kubric.{tenant_id}.billing.stripe.v1"
    nats_payload = orjson.dumps({
        "event_type":  event_type,
        "event_id":    event_id,
        "tenant_id":   tenant_id,
        "customer_id": customer_id,
        "status":      data_object.get("status", ""),
        "ts":          int(time.time() * 1000),
    })
    try:
        await nats_client.publish(nats_subject, nats_payload)
    except Exception as exc:
        log.warning("stripe.nats_publish_failed", error=str(exc))

    # For payment failures, also trigger a KAI-COMM notification
    if event_type == "invoice.payment_failed":
        try:
            from kai.agents.comm import CommAgent  # noqa: PLC0415
            await CommAgent().handle(
                subject=nats_subject,
                event={
                    "type": "billing.payment_failed",
                    "tenant_id": tenant_id,
                    "customer_id": customer_id,
                    "amount_due": data_object.get("amount_due", 0),
                },
            )
        except Exception as exc:
            log.warning("stripe.comm_notify_failed", error=str(exc))

    return {
        "received": True,
        "handled":  True,
        "type":     event_type,
        "tenant_id": tenant_id,
    }


# ─── n8n bridge ───────────────────────────────────────────────────────────────

@app.post("/v1/n8n/callback")
async def n8n_callback(req: N8nCallbackRequest) -> dict[str, Any]:
    """
    KAI → n8n bridge.
    Called by KAI-COMM when a CRITICAL/HIGH incident needs ITSM routing.
    Forwards the incident payload to the n8n incident webhook so that
    n8n can fan out to Slack, email, PagerDuty, etc.
    """
    n8n_url = f"{settings.n8n_base_url}/webhook/kubric-incident"
    payload = req.model_dump()
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.post(
                n8n_url,
                content=orjson.dumps(payload),
                headers={"Content-Type": "application/json"},
            )
            resp.raise_for_status()
    except Exception as exc:
        log.warning("api.n8n_callback_failed", n8n_url=n8n_url, error=str(exc))
        # Non-fatal: log and continue — n8n is best-effort
    log.info("api.n8n_callback_forwarded", tenant_id=req.tenant_id, severity=req.severity)
    return {"forwarded": True, "n8n_url": n8n_url, "tenant_id": req.tenant_id}
