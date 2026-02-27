"""
K-KAI Comm: Twilio SMS
Sends SMS notifications for high and critical security alerts
via the Twilio REST API (v2010).
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Optional
import httpx

logger = logging.getLogger(__name__)
TWILIO_ACCOUNT_SID = os.getenv("TWILIO_ACCOUNT_SID", "")
TWILIO_AUTH_TOKEN  = os.getenv("TWILIO_AUTH_TOKEN",  "")
TWILIO_FROM_NUMBER = os.getenv("TWILIO_FROM_NUMBER", "")
TWILIO_API_BASE    = "https://api.twilio.com/2010-04-01"

@dataclass
class SmsResult:
    message_sid: str
    to:          str
    status:      str
    body:        str
    sent_at:     str

async def send_sms(to_number: str, body: str) -> bool:
    """
    Send an SMS via Twilio. Returns True if accepted (status queued/sent).
    The body is truncated to 160 characters to fit a single SMS segment.
    """
    if not TWILIO_ACCOUNT_SID or not TWILIO_AUTH_TOKEN or not TWILIO_FROM_NUMBER:
        logger.warning("Twilio credentials not configured; skipping SMS to %s", to_number)
        return False

    body = body[:160]
    url  = f"{TWILIO_API_BASE}/Accounts/{TWILIO_ACCOUNT_SID}/Messages.json"

    async with httpx.AsyncClient(timeout=15) as client:
        resp = await client.post(
            url,
            auth=(TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN),
            data={"To": to_number, "From": TWILIO_FROM_NUMBER, "Body": body},
        )
        data = resp.json()
        if resp.status_code not in (200, 201):
            logger.error("Twilio SMS failed: %s %s", resp.status_code, data.get("message"))
            return False

        sid    = data.get("sid", "")
        status = data.get("status", "")
        logger.info("SMS sent: sid=%s to=%s status=%s", sid, to_number, status)
        return status in ("queued", "sent", "delivered", "accepted")

async def get_message_status(message_sid: str) -> Optional[SmsResult]:
    if not TWILIO_ACCOUNT_SID:
        return None
    url = f"{TWILIO_API_BASE}/Accounts/{TWILIO_ACCOUNT_SID}/Messages/{message_sid}.json"
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.get(url, auth=(TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN))
        if resp.status_code != 200:
            return None
        d = resp.json()
        return SmsResult(
            message_sid=d.get("sid", message_sid),
            to=d.get("to", ""),
            status=d.get("status", ""),
            body=d.get("body", ""),
            sent_at=d.get("date_sent") or datetime.now(timezone.utc).isoformat(),
        )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    print(json.dumps({
        "configured":    bool(TWILIO_ACCOUNT_SID),
        "from_number":   TWILIO_FROM_NUMBER[:4] + "***" if TWILIO_FROM_NUMBER else "",
        "account_sid":   TWILIO_ACCOUNT_SID[:8] + "***" if TWILIO_ACCOUNT_SID else "",
    }))
