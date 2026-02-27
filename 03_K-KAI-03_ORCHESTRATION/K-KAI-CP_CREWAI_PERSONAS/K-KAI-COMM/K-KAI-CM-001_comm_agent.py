"""
K-KAI Comm Agent: Notification Orchestrator
Routes security notifications, billing alerts, and health warnings
to the appropriate channels (NATS, email, SMS, voice) based on priority
and tenant communication preferences.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from typing import Any, Dict, List, Optional
import httpx

logger = logging.getLogger(__name__)

KAI_API_URL  = os.getenv("KAI_API_URL",  "http://localhost:9000")
NOC_API_URL  = os.getenv("NOC_API_URL",  "http://localhost:8080")
SMTP_HOST    = os.getenv("SMTP_HOST",    "localhost")
SMTP_PORT    = int(os.getenv("SMTP_PORT", "587"))

class NotifChannel(str, Enum):
    EMAIL  = "email"
    SMS    = "sms"
    VOICE  = "voice"
    NATS   = "nats"
    SLACK  = "slack"
    TEAMS  = "teams"

class NotifPriority(int, Enum):
    CRITICAL = 5
    HIGH     = 4
    MEDIUM   = 3
    LOW      = 2
    INFO     = 1

@dataclass
class Notification:
    notif_id:   str
    tenant_id:  str
    priority:   NotifPriority
    category:   str           # "security_alert" | "billing" | "health" | "compliance"
    title:      str
    body:       str
    recipient:  str           # email, phone, or user_id
    channels:   List[NotifChannel]
    metadata:   Dict[str, Any] = field(default_factory=dict)
    created_at: str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())
    sent_via:   List[str] = field(default_factory=list)
    failed_via: List[str] = field(default_factory=list)

def _channel_for_priority(priority: NotifPriority) -> List[NotifChannel]:
    if priority == NotifPriority.CRITICAL:
        return [NotifChannel.VOICE, NotifChannel.SMS, NotifChannel.EMAIL, NotifChannel.NATS]
    if priority == NotifPriority.HIGH:
        return [NotifChannel.SMS, NotifChannel.EMAIL, NotifChannel.NATS]
    if priority == NotifPriority.MEDIUM:
        return [NotifChannel.EMAIL, NotifChannel.NATS]
    return [NotifChannel.NATS]

class CommAgent:
    def __init__(self) -> None:
        self._nc = None
        self._email_conf = {
            "host": SMTP_HOST, "port": SMTP_PORT,
            "from": os.getenv("SMTP_FROM", "noreply@kubric.io"),
            "user": os.getenv("SMTP_USER", ""),
            "pass": os.getenv("SMTP_PASS", ""),
        }

    async def _send_email(self, notif: Notification) -> bool:
        import aiosmtplib
        from email.mime.text import MIMEText
        msg        = MIMEText(notif.body, "plain")
        msg["Subject"] = notif.title
        msg["From"]    = self._email_conf["from"]
        msg["To"]      = notif.recipient
        try:
            smtp = aiosmtplib.SMTP(hostname=self._email_conf["host"],
                                   port=self._email_conf["port"], use_tls=False)
            await smtp.connect()
            if self._email_conf["user"]:
                await smtp.login(self._email_conf["user"], self._email_conf["pass"])
            await smtp.send_message(msg)
            await smtp.quit()
            return True
        except Exception as exc:
            logger.error("Email send failed: %s", exc)
            return False

    async def _send_sms(self, notif: Notification) -> bool:
        """Dispatch to Twilio SMS module."""
        import importlib.util, sys, os
        twilio_path = os.path.join(os.path.dirname(__file__), "K-KAI-CM-003_twilio_sms.py")
        spec = importlib.util.spec_from_file_location("twilio_sms", twilio_path)
        if spec and spec.loader:
            mod = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(mod)
            return await mod.send_sms(notif.recipient, notif.body[:160])
        return False

    async def _send_voice(self, notif: Notification) -> bool:
        """Dispatch to VAPI voice module."""
        import importlib.util, os
        vapi_path = os.path.join(os.path.dirname(__file__), "K-KAI-CM-002_vapi_phone.py")
        spec = importlib.util.spec_from_file_location("vapi_phone", vapi_path)
        if spec and spec.loader:
            mod = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(mod)
            return await mod.make_call(notif.recipient, notif.title + ". " + notif.body[:200])
        return False

    async def _publish_nats(self, notif: Notification) -> bool:
        import nats
        try:
            if self._nc is None:
                self._nc = await nats.connect(os.getenv("NATS_URL", "nats://localhost:4222"))
            subject = f"kai.notifications.{notif.tenant_id}.{notif.category}"
            payload = json.dumps({
                "notif_id": notif.notif_id, "priority": notif.priority.value,
                "title": notif.title, "body": notif.body, "ts": notif.created_at,
            }).encode()
            await self._nc.publish(subject, payload)
            return True
        except Exception as exc:
            logger.error("NATS publish failed: %s", exc)
            return False

    async def dispatch(self, notif: Notification) -> Dict[str, Any]:
        channels = notif.channels or _channel_for_priority(notif.priority)
        for ch in channels:
            ok = False
            try:
                if ch == NotifChannel.EMAIL:
                    ok = await self._send_email(notif)
                elif ch == NotifChannel.SMS:
                    ok = await self._send_sms(notif)
                elif ch == NotifChannel.VOICE:
                    ok = await self._send_voice(notif)
                elif ch == NotifChannel.NATS:
                    ok = await self._publish_nats(notif)
            except Exception as exc:
                logger.error("Channel %s failed: %s", ch, exc)
                ok = False

            if ok:
                notif.sent_via.append(ch.value)
            else:
                notif.failed_via.append(ch.value)

        return {
            "notif_id":  notif.notif_id,
            "sent_via":  notif.sent_via,
            "failed_via": notif.failed_via,
            "success":   len(notif.sent_via) > 0,
        }

if __name__ == "__main__":
    import uuid
    logging.basicConfig(level=logging.INFO)
    agent = CommAgent()
    notif = Notification(
        notif_id  = str(uuid.uuid4()),
        tenant_id = "demo-tenant",
        priority  = NotifPriority.HIGH,
        category  = "security_alert",
        title     = "High Severity Alert: Lateral Movement Detected",
        body      = "Detected lateral movement from 10.0.0.1 to 10.0.0.50 via SMB.",
        recipient = "admin@example.com",
        channels  = [NotifChannel.NATS],
    )
    print(json.dumps(asyncio.run(agent.dispatch(notif)), indent=2))
