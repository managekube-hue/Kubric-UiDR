"""
K-KAI-KP-002: Cortex Responder Subprocess Runner
Triggers Cortex responders (isolation, block, takedown) via the Cortex REST API,
with async job polling.
"""

import logging
import os
import time
from typing import Any

import httpx

logger = logging.getLogger("K-KAI-KP-002")

CORTEX_URL: str = os.getenv("CORTEX_URL", "http://cortex:9001")
CORTEX_API_KEY: str = os.getenv("CORTEX_API_KEY", "")

HTTP_TIMEOUT: float = 30.0
POLL_INTERVAL: float = 3.0
MAX_POLL_ATTEMPTS: int = 40   # 40 × 3 s = 2 minutes

# Well-known Cortex responder names
_RESPONDER_ISOLATE: str = "Cortex-AgentIsolation_v1"
_RESPONDER_BLOCK_IP: str = "Firewall-BlockIP_v1"


class CortexResponder:
    """
    Drives Cortex responders for automated response actions.

    All methods block until the Cortex job completes or times out.
    """

    def __init__(self) -> None:
        if not CORTEX_API_KEY:
            logger.warning("CORTEX_API_KEY is not set; Cortex responder calls may fail.")
        self._headers: dict[str, str] = {
            "Authorization": f"Bearer {CORTEX_API_KEY}",
            "Content-Type": "application/json",
        }

    # ------------------------------------------------------------------
    # High-level response actions
    # ------------------------------------------------------------------

    def isolate_host(self, tenant_id: str, ip: str) -> dict:
        """
        Trigger a host isolation responder for the given IP.

        Args:
            tenant_id: Kubric tenant identifier (stored in job metadata).
            ip:        IP address of the host to isolate.

        Returns:
            Cortex job result dict.
        """
        logger.info("Initiating host isolation for IP %s (tenant %s)", ip, tenant_id)
        data = {
            "_type": "case_artifact",
            "dataType": "ip",
            "data": ip,
            "message": f"Kubric-UiDR: isolate host {ip} (tenant {tenant_id})",
            "tlp": 3,
            "pap": 2,
        }
        return self.run_responder(_RESPONDER_ISOLATE, data)

    def block_ip(self, tenant_id: str, ip: str, reason: str) -> dict:
        """
        Trigger a firewall block-IP responder.

        Args:
            tenant_id: Kubric tenant identifier.
            ip:        IP address to block.
            reason:    Human-readable reason (e.g. "C2 communication detected").

        Returns:
            Cortex job result dict.
        """
        logger.info("Initiating IP block for %s (tenant %s): %s", ip, tenant_id, reason)
        data = {
            "_type": "case_artifact",
            "dataType": "ip",
            "data": ip,
            "message": f"Kubric-UiDR: block IP {ip} – {reason} (tenant {tenant_id})",
            "tlp": 3,
            "pap": 2,
        }
        return self.run_responder(_RESPONDER_BLOCK_IP, data)

    def run_responder(self, responder_name: str, data: dict) -> dict:
        """
        Submit a Cortex responder job and poll until completion.

        Args:
            responder_name: Cortex responder worker name.
            data:           Observable / artifact data dict.

        Returns:
            dict with: job_id, responder, status, report, operations.
        """
        job_id = self._submit_responder_job(responder_name, data)
        logger.info("Cortex responder job %s submitted for '%s'", job_id, responder_name)
        return self._poll_responder_job(job_id, responder_name)

    # ------------------------------------------------------------------
    # Internal Cortex API calls
    # ------------------------------------------------------------------

    def _submit_responder_job(self, responder_name: str, data: dict) -> str:
        """
        POST to /api/responder/{responder_name}/run.

        Returns the Cortex job ID.
        """
        url = f"{CORTEX_URL}/api/responder/{responder_name}/run"
        payload: dict[str, Any] = {
            "data": data,
            "tlp": data.get("tlp", 2),
            "pap": data.get("pap", 2),
            "message": data.get("message", ""),
        }
        with httpx.Client(timeout=HTTP_TIMEOUT) as client:
            resp = client.post(url, json=payload, headers=self._headers)
            resp.raise_for_status()
            result = resp.json()
            job_id: str = result.get("id") or result.get("_id") or result["id"]
            return job_id

    def _poll_responder_job(self, job_id: str, responder_name: str) -> dict:
        """
        Poll GET /api/job/{job_id}/waitreport until terminal state.
        """
        url = f"{CORTEX_URL}/api/job/{job_id}/waitreport"

        for attempt in range(MAX_POLL_ATTEMPTS):
            try:
                with httpx.Client(timeout=HTTP_TIMEOUT) as c:
                    resp = c.get(url, headers=self._headers)
                    resp.raise_for_status()
                    data = resp.json()

                status: str = data.get("status", "").lower()
                if status == "success":
                    report = data.get("report", {})
                    operations = report.get("operations", [])
                    logger.info(
                        "Responder job %s (%s) succeeded with %d operations.",
                        job_id,
                        responder_name,
                        len(operations),
                    )
                    return {
                        "job_id": job_id,
                        "responder": responder_name,
                        "status": "success",
                        "report": report,
                        "operations": operations,
                    }
                if status in ("failure", "deleted"):
                    error_msg = data.get("errorMessage", "Unknown error")
                    logger.error(
                        "Responder job %s (%s) failed: %s", job_id, responder_name, error_msg
                    )
                    return {
                        "job_id": job_id,
                        "responder": responder_name,
                        "status": "failure",
                        "error": error_msg,
                        "report": {},
                        "operations": [],
                    }

                logger.debug(
                    "Responder job %s status=%s (attempt %d/%d)",
                    job_id, status, attempt + 1, MAX_POLL_ATTEMPTS
                )
            except httpx.TimeoutException:
                logger.warning("Timeout polling responder job %s (attempt %d)", job_id, attempt + 1)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Error polling responder job %s: %s", job_id, exc)

            time.sleep(POLL_INTERVAL)

        logger.error("Responder job %s timed out after %d attempts.", job_id, MAX_POLL_ATTEMPTS)
        return {
            "job_id": job_id,
            "responder": responder_name,
            "status": "timeout",
            "report": {},
            "operations": [],
        }
