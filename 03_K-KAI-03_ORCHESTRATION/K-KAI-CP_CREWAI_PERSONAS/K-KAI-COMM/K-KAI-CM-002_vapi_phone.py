"""
K-KAI Comm: VAPI Voice Caller
Triggers AI voice calls via the VAPI.ai API for critical security
notifications that require immediate human acknowledgement.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Dict, Optional
import httpx

logger = logging.getLogger(__name__)
VAPI_API_KEY      = os.getenv("VAPI_API_KEY",      "")
VAPI_PHONE_NUMBER = os.getenv("VAPI_PHONE_NUMBER", "")
VAPI_ASSISTANT_ID = os.getenv("VAPI_ASSISTANT_ID", "")
VAPI_BASE_URL     = "https://api.vapi.ai"

@dataclass
class VoiceCallResult:
    call_id:      str
    to_number:    str
    status:       str    # "queued" | "ringing" | "in-progress" | "completed" | "failed"
    duration_s:   float
    message_sent: str
    initiated_at: str

async def make_call(to_number: str, message: str) -> bool:
    """
    Initiate an outbound voice call via VAPI.
    Returns True if call was successfully queued.
    """
    if not VAPI_API_KEY or not VAPI_PHONE_NUMBER or not VAPI_ASSISTANT_ID:
        logger.warning("VAPI credentials not configured; skipping voice call to %s", to_number)
        return False

    payload: Dict[str, Any] = {
        "phoneNumberId": VAPI_PHONE_NUMBER,
        "assistantId":   VAPI_ASSISTANT_ID,
        "customer": {
            "number": to_number,
        },
        "assistantOverrides": {
            "firstMessage": f"Hello, this is Kubric AI Security. {message}. "
                            "Please press 1 to acknowledge or stay on the line.",
            "variableValues": {
                "security_message": message,
                "timestamp": datetime.now(timezone.utc).isoformat(),
            },
        },
    }

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(
            f"{VAPI_BASE_URL}/call/phone",
            headers={"Authorization": f"Bearer {VAPI_API_KEY}",
                     "Content-Type": "application/json"},
            json=payload,
        )
        if resp.status_code not in (200, 201):
            logger.error("VAPI call failed: %s %s", resp.status_code, resp.text[:500])
            return False

        call_data = resp.json()
        logger.info("VAPI call initiated: id=%s to=%s",
                    call_data.get("id"), to_number)
        return True

async def get_call_status(call_id: str) -> Optional[VoiceCallResult]:
    if not VAPI_API_KEY:
        return None
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.get(
            f"{VAPI_BASE_URL}/call/{call_id}",
            headers={"Authorization": f"Bearer {VAPI_API_KEY}"},
        )
        if resp.status_code != 200:
            return None
        d = resp.json()
        return VoiceCallResult(
            call_id=d.get("id", call_id),
            to_number=d.get("customer", {}).get("number", ""),
            status=d.get("status", "unknown"),
            duration_s=float(d.get("duration", 0)),
            message_sent=d.get("assistantOverrides", {}).get("firstMessage", ""),
            initiated_at=d.get("createdAt", datetime.now(timezone.utc).isoformat()),
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    print(json.dumps({
        "configured": bool(VAPI_API_KEY),
        "phone_number": VAPI_PHONE_NUMBER[:4] + "***" if VAPI_PHONE_NUMBER else "",
        "assistant_id": VAPI_ASSISTANT_ID,
    }))
