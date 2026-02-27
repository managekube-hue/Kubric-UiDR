"""
K-KAI-TR-001: CrewAI Triage Agent
Subscribes to kubric.*.detection.alert NATS subjects, reasons about severity
using LLaMA3/Anthropic, and publishes triage decisions to
kubric.{tenant}.triage.decision.
"""

import asyncio
import json
import logging
import os
import sys
from datetime import datetime, timezone
from typing import Any

import nats
from nats.aio.client import Client as NATSClient
from crewai import Agent

# Sibling module imports – resolved at runtime from the same package directory
sys.path.insert(0, os.path.dirname(__file__))
from K_KAI_TR_002_llama3_reasoning import LlamaReasoner  # noqa: E402
from K_KAI_TR_003_ocsf_analyzer import OCSFAnalyzer  # noqa: E402
from K_KAI_TR_004_kiss_calculator import KISSCalculator  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
logger = logging.getLogger("K-KAI-TR-001")

NATS_URL: str = os.getenv("NATS_URL", "nats://localhost:4222")


class TriageAgent:
    """
    CrewAI-backed triage agent.

    Lifecycle:
        agent = TriageAgent()
        asyncio.run(agent.start())   # blocks until stop() is called
    """

    def __init__(self) -> None:
        self.nc: NATSClient | None = None
        self.reasoner: LlamaReasoner = LlamaReasoner()
        self.analyzer: OCSFAnalyzer = OCSFAnalyzer()
        self.calculator: KISSCalculator = KISSCalculator()
        self._running: bool = False
        self._subscription: Any = None

        # CrewAI persona – guides the prompt framing sent to the LLM
        self._crew_agent = Agent(
            role="Security Triage Analyst",
            goal=(
                "Accurately assess the severity and urgency of every incoming "
                "security detection alert and recommend the correct response action."
            ),
            backstory=(
                "You are a battle-hardened SOC analyst with 10+ years triaging "
                "security events for critical infrastructure. You distinguish real "
                "threats from false positives with calm precision."
            ),
            verbose=False,
            allow_delegation=False,
        )

    # ------------------------------------------------------------------
    # Public interface
    # ------------------------------------------------------------------

    async def start(self) -> None:
        """Connect to NATS JetStream and begin consuming detection alerts."""
        logger.info("TriageAgent connecting to NATS at %s", NATS_URL)
        self.nc = await nats.connect(
            NATS_URL,
            reconnect_time_wait=2,
            max_reconnect_attempts=-1,
        )
        self._running = True

        self._subscription = await self.nc.subscribe(
            "kubric.*.detection.alert",
            cb=self._process_alert,
            queue="triage-workers",
        )
        logger.info("TriageAgent subscribed to kubric.*.detection.alert")

        try:
            while self._running:
                await asyncio.sleep(1)
        finally:
            await self.stop()

    async def stop(self) -> None:
        """Unsubscribe and close NATS connection."""
        self._running = False
        if self._subscription:
            try:
                await self._subscription.unsubscribe()
            except Exception:  # noqa: BLE001
                pass
        if self.nc and not self.nc.is_closed:
            await self.nc.drain()
            await self.nc.close()
        logger.info("TriageAgent stopped.")

    # ------------------------------------------------------------------
    # Internal NATS callback
    # ------------------------------------------------------------------

    async def _process_alert(self, msg: Any) -> None:
        """Deserialise the incoming NATS message, score it, and publish decision."""
        try:
            raw: dict = json.loads(msg.data.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError) as exc:
            logger.error("Failed to decode alert message: %s", exc)
            return

        # Extract tenant from subject  kubric.<tenant>.detection.alert
        parts = msg.subject.split(".")
        tenant_id: str = parts[1] if len(parts) >= 3 else "default"

        try:
            loop = asyncio.get_event_loop()
            decision: dict = await loop.run_in_executor(None, self._score_alert, raw)
        except Exception as exc:  # noqa: BLE001
            logger.exception("Error scoring alert for tenant %s: %s", tenant_id, exc)
            return

        decision["tenant_id"] = tenant_id
        decision["source_subject"] = msg.subject
        decision["processed_at"] = datetime.now(timezone.utc).isoformat()

        out_subject: str = f"kubric.{tenant_id}.triage.decision"
        payload: bytes = json.dumps(decision, default=str).encode("utf-8")
        await self.nc.publish(out_subject, payload)

        logger.info(
            "Triage decision published → %s | score=%s grade=%s action=%s",
            out_subject,
            decision.get("score"),
            decision.get("grade"),
            decision.get("recommended_action"),
        )

    # ------------------------------------------------------------------
    # Core scoring logic (runs in thread executor to avoid blocking)
    # ------------------------------------------------------------------

    def _score_alert(self, alert_data: dict) -> dict:
        """
        Full triage pipeline for a single alert.

        Steps:
          1. Normalise to unified OCSF schema.
          2. Extract IoCs.
          3. Classify OCSF severity (1-6).
          4. Compute KISS composite risk score.
          5. Generate AI reasoning narrative via LlamaReasoner.

        Returns an OCSF Detection Finding (class_uid 2004)-aligned dict.
        """
        normalized: dict = self.analyzer.normalize(alert_data)
        indicators: list[dict] = self.analyzer.extract_indicators(normalized)
        ocsf_severity: int = self.analyzer.classify_severity(normalized)

        cvss: float = float(normalized.get("cvss_base_score") or 5.0)
        epss: float = float(normalized.get("epss") or 0.1)
        criticality: int = int(normalized.get("asset_criticality") or 3)
        exploited: bool = bool(normalized.get("actively_exploited", False))

        kiss_result: dict = self.calculator.score(cvss, epss, criticality, exploited)

        # Build a concise prompt for the AI persona
        ioc_preview: str = json.dumps(indicators[:5], default=str) if indicators else "[]"
        prompt: str = (
            f"Triage this security alert.\n"
            f"Class: {normalized.get('class_name', 'Unknown')}\n"
            f"Severity rank: {ocsf_severity}/6\n"
            f"CVSS base: {cvss}\n"
            f"EPSS probability: {epss}\n"
            f"Asset criticality: {criticality}/5\n"
            f"Active exploitation: {exploited}\n"
            f"Top IoCs: {ioc_preview}\n"
            f"Summary: {str(normalized.get('message', ''))[:400]}\n\n"
            f"Is this a true positive, false positive, or needs investigation? "
            f"What immediate action should the analyst take?"
        )
        reasoning: str = self.reasoner.reason(prompt, context=normalized)

        return {
            "class_uid": 2004,
            "class_name": "Detection Finding",
            "activity_id": 1,
            "severity_id": ocsf_severity,
            "score": kiss_result["score"],
            "grade": kiss_result["grade"],
            "recommended_action": kiss_result["recommended_action"],
            "ai_reasoning": reasoning,
            "indicators": indicators,
            "normalized_event": normalized,
        }


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    agent = TriageAgent()
    try:
        asyncio.run(agent.start())
    except KeyboardInterrupt:
        logger.info("Received interrupt, shutting down.")
