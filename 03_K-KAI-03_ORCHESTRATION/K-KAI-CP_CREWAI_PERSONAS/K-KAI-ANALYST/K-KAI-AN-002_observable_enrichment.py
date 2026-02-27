"""
K-KAI-AN-002: Observable Enrichment Pipeline
Enriches IPs via AbuseIPDB, domains via VirusTotal, file hashes via
MalwareBazaar. In-memory TTL cache (1-hour) per observable.
"""

import hashlib
import logging
import os
import time
from typing import Any

import httpx

logger = logging.getLogger("K-KAI-AN-002")

ABUSEIPDB_KEY: str = os.getenv("ABUSEIPDB_KEY", "")
VT_API_KEY: str = os.getenv("VT_API_KEY", "")

HTTP_TIMEOUT: float = 20.0
CACHE_TTL_SECONDS: int = 3600  # 1 hour

# ---------------------------------------------------------------------------
# Simple in-memory TTL cache
# ---------------------------------------------------------------------------

class _TTLCache:
    def __init__(self, ttl: int) -> None:
        self._ttl = ttl
        self._store: dict[str, tuple[float, Any]] = {}

    def get(self, key: str) -> Any | None:
        entry = self._store.get(key)
        if entry is None:
            return None
        ts, value = entry
        if time.monotonic() - ts > self._ttl:
            del self._store[key]
            return None
        return value

    def set(self, key: str, value: Any) -> None:
        self._store[key] = (time.monotonic(), value)

    def invalidate_expired(self) -> None:
        now = time.monotonic()
        expired = [k for k, (ts, _) in self._store.items() if now - ts > self._ttl]
        for k in expired:
            del self._store[k]


class ObservableEnricher:
    """
    Enriches security observables with threat-intelligence data.

    Supported types: ip, domain, hash (md5/sha1/sha256), url.
    """

    def __init__(self) -> None:
        self._cache = _TTLCache(CACHE_TTL_SECONDS)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def enrich(self, observable: str, obs_type: str) -> dict:
        """
        Enrich a single observable.

        Returns a dict with: observable, type, sources (list of source result
        dicts), verdict, enriched_at (epoch seconds).
        """
        cache_key = f"{obs_type}:{observable.lower()}"
        cached = self._cache.get(cache_key)
        if cached is not None:
            logger.debug("Cache hit for %s %s", obs_type, observable)
            return cached

        result = self._dispatch(observable, obs_type.lower())
        self._cache.set(cache_key, result)
        self._cache.invalidate_expired()
        return result

    def enrich_bulk(self, observables: list[dict]) -> list[dict]:
        """
        Enrich a list of observables.

        Each item must be a dict with keys 'value' and 'type'.
        Returns a list of enrichment result dicts in the same order.
        """
        results: list[dict] = []
        for item in observables:
            try:
                result = self.enrich(item["value"], item["type"])
            except Exception as exc:  # noqa: BLE001
                logger.warning("Enrichment failed for %s: %s", item.get("value"), exc)
                result = {
                    "observable": item.get("value"),
                    "type": item.get("type"),
                    "error": str(exc),
                    "verdict": "error",
                }
            results.append(result)
        return results

    # ------------------------------------------------------------------
    # Dispatch
    # ------------------------------------------------------------------

    def _dispatch(self, observable: str, obs_type: str) -> dict:
        if obs_type == "ip":
            return self._enrich_ip(observable)
        if obs_type in ("domain", "hostname"):
            return self._enrich_domain(observable)
        if obs_type in ("md5", "sha1", "sha256", "hash"):
            return self._enrich_hash(observable)
        if obs_type == "url":
            return self._enrich_url(observable)
        return {
            "observable": observable,
            "type": obs_type,
            "verdict": "unknown",
            "sources": [],
            "enriched_at": time.time(),
        }

    # ------------------------------------------------------------------
    # AbuseIPDB  (IP enrichment)
    # ------------------------------------------------------------------

    def _enrich_ip(self, ip: str) -> dict:
        sources: list[dict] = []
        verdict = "unknown"

        if ABUSEIPDB_KEY:
            try:
                with httpx.Client(timeout=HTTP_TIMEOUT) as c:
                    resp = c.get(
                        "https://api.abuseipdb.com/api/v2/check",
                        params={"ipAddress": ip, "maxAgeInDays": 90},
                        headers={"Key": ABUSEIPDB_KEY, "Accept": "application/json"},
                    )
                    resp.raise_for_status()
                    data = resp.json().get("data", {})
                    abuse_score: int = int(data.get("abuseConfidenceScore", 0))
                    sources.append({
                        "source": "AbuseIPDB",
                        "abuse_score": abuse_score,
                        "total_reports": data.get("totalReports"),
                        "usage_type": data.get("usageType"),
                        "isp": data.get("isp"),
                        "country": data.get("countryCode"),
                    })
                    verdict = "malicious" if abuse_score > 80 else (
                        "suspicious" if abuse_score > 30 else "clean"
                    )
            except Exception as exc:  # noqa: BLE001
                logger.warning("AbuseIPDB error for %s: %s", ip, exc)
                sources.append({"source": "AbuseIPDB", "error": str(exc)})
        else:
            logger.debug("ABUSEIPDB_KEY not set, skipping IP enrichment.")

        return {
            "observable": ip,
            "type": "ip",
            "verdict": verdict,
            "sources": sources,
            "enriched_at": time.time(),
        }

    # ------------------------------------------------------------------
    # VirusTotal  (domain / URL enrichment)
    # ------------------------------------------------------------------

    def _enrich_domain(self, domain: str) -> dict:
        return self._vt_lookup(
            f"https://www.virustotal.com/api/v3/domains/{domain}", domain, "domain"
        )

    def _enrich_url(self, url: str) -> dict:
        url_id = hashlib.sha256(url.encode()).hexdigest()
        return self._vt_lookup(
            f"https://www.virustotal.com/api/v3/urls/{url_id}", url, "url"
        )

    def _vt_lookup(self, vt_url: str, observable: str, obs_type: str) -> dict:
        sources: list[dict] = []
        verdict = "unknown"

        if VT_API_KEY:
            try:
                with httpx.Client(timeout=HTTP_TIMEOUT) as c:
                    resp = c.get(vt_url, headers={"x-apikey": VT_API_KEY})
                    resp.raise_for_status()
                    stats = (
                        resp.json()
                        .get("data", {})
                        .get("attributes", {})
                        .get("last_analysis_stats", {})
                    )
                    malicious: int = int(stats.get("malicious", 0))
                    suspicious: int = int(stats.get("suspicious", 0))
                    sources.append({
                        "source": "VirusTotal",
                        "malicious_engines": malicious,
                        "suspicious_engines": suspicious,
                        "harmless_engines": stats.get("harmless"),
                    })
                    verdict = "malicious" if malicious > 3 else (
                        "suspicious" if (malicious > 0 or suspicious > 3) else "clean"
                    )
            except Exception as exc:  # noqa: BLE001
                logger.warning("VirusTotal error for %s: %s", observable, exc)
                sources.append({"source": "VirusTotal", "error": str(exc)})
        else:
            logger.debug("VT_API_KEY not set, skipping VirusTotal enrichment.")

        return {
            "observable": observable,
            "type": obs_type,
            "verdict": verdict,
            "sources": sources,
            "enriched_at": time.time(),
        }

    # ------------------------------------------------------------------
    # MalwareBazaar  (hash enrichment, no key required)
    # ------------------------------------------------------------------

    def _enrich_hash(self, file_hash: str) -> dict:
        sources: list[dict] = []
        verdict = "unknown"

        try:
            with httpx.Client(timeout=HTTP_TIMEOUT) as c:
                resp = c.post(
                    "https://mb-api.abuse.ch/api/v1/",
                    data={"query": "get_info", "hash": file_hash},
                )
                resp.raise_for_status()
                data = resp.json()
                query_status = data.get("query_status")
                if query_status == "ok":
                    sample = data.get("data", [{}])[0]
                    verdict = "malicious"
                    sources.append({
                        "source": "MalwareBazaar",
                        "found": True,
                        "file_name": sample.get("file_name"),
                        "file_type": sample.get("file_type_mime"),
                        "tags": sample.get("tags", []),
                        "first_seen": sample.get("first_seen"),
                        "signature": sample.get("signature"),
                    })
                else:
                    verdict = "clean"
                    sources.append({"source": "MalwareBazaar", "found": False, "status": query_status})
        except Exception as exc:  # noqa: BLE001
            logger.warning("MalwareBazaar error for %s: %s", file_hash, exc)
            sources.append({"source": "MalwareBazaar", "error": str(exc)})

        return {
            "observable": file_hash,
            "type": "hash",
            "verdict": verdict,
            "sources": sources,
            "enriched_at": time.time(),
        }
