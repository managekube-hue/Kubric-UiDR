"""
K-KAI-AN-001: Cortex Analyzer Chain
Submits observables to TheHive Cortex REST API, polls for job results,
and returns enriched findings.
"""

import logging
import os
import time
from typing import Any

import httpx

logger = logging.getLogger("K-KAI-AN-001")

CORTEX_URL: str = os.getenv("CORTEX_URL", "http://cortex:9001")
CORTEX_API_KEY: str = os.getenv("CORTEX_API_KEY", "")

# Polling configuration
POLL_INTERVAL_SECONDS: float = 3.0
MAX_POLL_ATTEMPTS: int = 40   # 40 × 3 s = 2 minutes max wait
HTTP_TIMEOUT: float = 30.0

# Analyzers to run per data type
_ANALYZER_MAP: dict[str, list[str]] = {
    "ip": ["AbuseIPDB_2_0", "Shodan_Host_1_0", "IPinfo_1_0"],
    "domain": ["Urlscan_io_Scan_0_1", "VirusTotal_GetReport_3_0", "MISP_2_1"],
    "hash": ["VirusTotal_GetReport_3_0", "MalwareBazaar_1_0", "Maltiverse_1_0"],
    "url": ["Urlscan_io_Scan_0_1", "VirusTotal_GetReport_3_0"],
}


class CortexAnalyzerChain:
    """
    Thin wrapper around the Cortex v2 REST API.

    Usage:
        chain = CortexAnalyzerChain()
        result = chain.analyze("8.8.8.8", "ip", "tenant_acme")
    """

    def __init__(self) -> None:
        if not CORTEX_API_KEY:
            logger.warning("CORTEX_API_KEY is not set; Cortex calls will be unauthenticated.")
        self._headers: dict[str, str] = {
            "Authorization": f"Bearer {CORTEX_API_KEY}",
            "Content-Type": "application/json",
        }

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def analyze(self, observable: str, datatype: str, tenant_id: str) -> dict:
        """
        Run all applicable analyzers for the observable, wait for results.

        Args:
            observable: The value to analyse (IP, domain, hash, URL).
            datatype:   One of 'ip', 'domain', 'hash', 'url'.
            tenant_id:  Kubric tenant identifier (stored in result metadata).

        Returns:
            dict with keys: observable, datatype, tenant_id, results (list of
            per-analyzer finding dicts), combined_verdict (str), combined_malicious (bool).
        """
        analyzers: list[str] = _ANALYZER_MAP.get(datatype.lower(), [])
        if not analyzers:
            logger.warning("No analyzers configured for datatype '%s'", datatype)
            return self._empty_result(observable, datatype, tenant_id)

        job_ids: list[str] = []
        for analyzer in analyzers:
            try:
                job_id = self._submit_job(observable, datatype, analyzer)
                job_ids.append(job_id)
                logger.debug("Submitted %s job %s for %s", analyzer, job_id, observable)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Failed to submit analyzer %s: %s", analyzer, exc)

        results: list[dict] = []
        for job_id in job_ids:
            try:
                result = self._poll_job(job_id)
                results.append(result)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Failed to retrieve job %s: %s", job_id, exc)

        combined_malicious = any(r.get("malicious", False) for r in results)
        combined_verdict = "malicious" if combined_malicious else (
            "suspicious" if any(r.get("suspicious", False) for r in results)
            else "clean"
        )

        return {
            "observable": observable,
            "datatype": datatype,
            "tenant_id": tenant_id,
            "results": results,
            "combined_verdict": combined_verdict,
            "combined_malicious": combined_malicious,
        }

    def _submit_job(self, observable: str, datatype: str, analyzer_id: str) -> str:
        """
        POST to /api/analyzer/{analyzer_id}/run and return the Cortex job ID.
        """
        payload: dict[str, Any] = {
            "data": observable,
            "dataType": datatype,
            "tlp": 2,
            "message": f"Kubric-UiDR automated analysis for {observable}",
        }
        url = f"{CORTEX_URL}/api/analyzer/{analyzer_id}/run"
        with httpx.Client(timeout=HTTP_TIMEOUT) as client:
            resp = client.post(url, json=payload, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()
            job_id: str = data.get("id") or data.get("_id") or data["id"]
            return job_id

    def _poll_job(self, job_id: str) -> dict:
        """
        Poll /api/job/{job_id}/waitreport until status is Success/Failure.
        Returns the report dict.
        """
        url = f"{CORTEX_URL}/api/job/{job_id}/waitreport"
        for attempt in range(MAX_POLL_ATTEMPTS):
            try:
                with httpx.Client(timeout=HTTP_TIMEOUT) as client:
                    resp = client.get(url, headers=self._headers)
                    resp.raise_for_status()
                    data = resp.json()

                status: str = data.get("status", "").lower()
                if status == "success":
                    report = data.get("report", {})
                    return self._parse_report(job_id, data.get("analyzerId", ""), report)
                if status in ("failure", "deleted"):
                    logger.warning("Cortex job %s ended with status: %s", job_id, status)
                    return {"job_id": job_id, "status": status, "error": data.get("errorMessage")}

            except httpx.TimeoutException:
                logger.debug("Poll timeout for job %s (attempt %d)", job_id, attempt + 1)
            except Exception as exc:  # noqa: BLE001
                logger.warning("Poll error for job %s: %s", job_id, exc)

            time.sleep(POLL_INTERVAL_SECONDS)

        raise TimeoutError(f"Cortex job {job_id} did not complete within the allowed time.")

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    @staticmethod
    def _parse_report(job_id: str, analyzer_id: str, report: dict) -> dict:
        """Extract structured verdict from an analyzer report."""
        level: str = report.get("summary", {}).get("taxonomies", [{}])[0].get("level", "info")
        return {
            "job_id": job_id,
            "analyzer_id": analyzer_id,
            "status": "success",
            "level": level,
            "malicious": level in ("malicious",),
            "suspicious": level in ("suspicious",),
            "full_report": report,
        }

    @staticmethod
    def _empty_result(observable: str, datatype: str, tenant_id: str) -> dict:
        return {
            "observable": observable,
            "datatype": datatype,
            "tenant_id": tenant_id,
            "results": [],
            "combined_verdict": "unknown",
            "combined_malicious": False,
        }
