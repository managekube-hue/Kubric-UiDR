"""
KAI-COMM — Alert escalation and notification routing.

Subscribes to:  kubric.{tenant}.comm.alert.*  (from KAI-TRIAGE enriched results)
Channels:       n8n ITSM webhook (primary), Vapi voice (CRITICAL), Twilio SMS (HIGH)

Agent persona: Security Communications Officer
CrewAI integration: make_comm_crew() → Crew.kickoff() → JSON escalation decision
The forward_to_n8n @tool is called by the CrewAI agent automatically when
severity meets the CRITICAL/HIGH threshold.
"""

from __future__ import annotations

import os
import time
from typing import Any

import httpx
import orjson
import structlog

from kai.core.crew import make_comm_crew, parse_crew_output
from kai.core.nats_client import nats_client

log = structlog.get_logger(__name__)

_VAPI_API_KEY  = os.getenv("KUBRIC_VAPI_API_KEY", "")
_TWILIO_SID    = os.getenv("KUBRIC_TWILIO_SID", "")
_TWILIO_TOKEN  = os.getenv("KUBRIC_TWILIO_TOKEN", "")
_TWILIO_FROM   = os.getenv("KUBRIC_TWILIO_FROM", "")

_VOICE_THRESHOLD = {"CRITICAL"}
_SMS_THRESHOLD   = {"CRITICAL", "HIGH"}


class CommAgent:
    """KAI-COMM — routes enriched alerts to the correct notification channel."""

    async def handle(self, *, subject: str, event: dict[str, Any]) -> dict[str, Any]:
        severity  = str(event.get("severity", "LOW")).upper()
        tenant_id = _extract_tenant(subject)

        # ── CrewAI crew decides escalation + calls forward_to_n8n @tool ──────
        incident_json = orjson.dumps(event).decode()
        crew   = make_comm_crew(incident_json, tenant_id)
        output = crew.kickoff()
        comm_decision = parse_crew_output(output)

        dispatched: list[str] = []
        if comm_decision.get("notification_sent"):
            dispatched.append("n8n_itsm")

        # Direct channel escalations (Vapi / Twilio) — bypass CrewAI for speed
        if severity in _VOICE_THRESHOLD and _VAPI_API_KEY:
            ok = await _send_vapi_voice(event)
            if ok:
                dispatched.append("voice")

        if severity in _SMS_THRESHOLD and _TWILIO_SID:
            ok = await _send_twilio_sms(event)
            if ok:
                dispatched.append("sms")

        result: dict[str, Any] = {
            "tenant_id":         tenant_id,
            "alert_id":          event.get("triage_id", event.get("event_id", "")),
            "severity":          severity,
            "channels_notified": dispatched,
            "comm_decision":     comm_decision,
            "timestamp":         int(time.time() * 1000),
        }

        await nats_client.publish("kubric.kai.comm.dispatched", orjson.dumps(result))
        log.info("comm.dispatched", tenant_id=tenant_id, severity=severity, channels=dispatched)
        return result


# ─── channel implementations ─────────────────────────────────────────────────

async def _send_vapi_voice(event: dict[str, Any]) -> bool:
    phone_number = event.get("contact_phone") or os.getenv("KUBRIC_ESCALATION_PHONE", "")
    if not phone_number:
        log.warning("comm.vapi_no_phone")
        return False
    payload = {
        "type": "outbound",
        "to": phone_number,
        "assistant": {
            "firstMessage": (
                f"This is Kubric AI. A critical security alert has been detected: "
                f"{event.get('summary', 'Unknown alert')}. "
                "Please log into the Kubric portal immediately."
            )
        },
    }
    headers = {"Authorization": f"Bearer {_VAPI_API_KEY}", "Content-Type": "application/json"}
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.post("https://api.vapi.ai/call", headers=headers, json=payload)
            resp.raise_for_status()
            log.info("comm.vapi_call_initiated", to=phone_number)
            return True
    except Exception as exc:
        log.warning("comm.vapi_failed", error=str(exc))
        return False


async def _send_twilio_sms(event: dict[str, Any]) -> bool:
    to_number = event.get("contact_phone") or os.getenv("KUBRIC_ESCALATION_PHONE", "")
    if not to_number:
        return False
    body = (
        f"[Kubric ALERT] {event.get('severity', 'UNKNOWN')} — "
        f"{event.get('summary', 'Security event detected')}. Check portal."
    )
    try:
        async with httpx.AsyncClient(
            timeout=10.0, auth=(_TWILIO_SID, _TWILIO_TOKEN)
        ) as client:
            resp = await client.post(
                f"https://api.twilio.com/2010-04-01/Accounts/{_TWILIO_SID}/Messages.json",
                data={"To": to_number, "From": _TWILIO_FROM, "Body": body},
            )
            resp.raise_for_status()
            log.info("comm.sms_sent", to=to_number)
            return True
    except Exception as exc:
        log.warning("comm.twilio_failed", error=str(exc))
        return False


def _extract_tenant(subject: str) -> str:
    parts = subject.split(".")
    return parts[1] if len(parts) >= 2 else "default"
