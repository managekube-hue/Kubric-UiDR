"""
kai/intel/ti_feeds.py — Threat Intelligence Feed Ingestion Pipeline (L3-4)

Async feed pullers using httpx.  All API keys are read from environment
variables (Vault injects them at container startup in production).

Feeds implemented
-----------------
  1. NVD CVE Feed         GET https://services.nvd.nist.gov/rest/json/cves/2.0
                          → ClickHouse kubric.vuln_findings
  2. CISA KEV             GET https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json
                          → ClickHouse kubric.vuln_findings  (in_kev=1)
                          → NATS      kubric.kev.updated
  3. FIRST EPSS           GET https://epss.cyentia.com/epss_scores-current.csv.gz
                          → ClickHouse kubric.epss_scores   (parsed with polars)
  4. AlienVault OTX       GET https://otx.alienvault.com/api/v1/pulses/subscribed
                          → ClickHouse kubric.ti_indicators  (type=otx)
  5. AbuseIPDB            GET https://api.abuseipdb.com/api/v2/blacklist
                          → ClickHouse kubric.ti_indicators  (type=abuseipdb)
  6. IPSum                GET https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt
                          → ClickHouse kubric.ti_indicators  (type=ipsum)
  7. PyMISP               MISP REST API  /attributes/restSearch
                          → ClickHouse kubric.ti_indicators  (type=misp)
"""

from __future__ import annotations

import csv
import gzip
import io
import logging
import os
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

import httpx

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Config helpers — read from env; Vault injects at startup
# ---------------------------------------------------------------------------

def _env(key: str, default: str = "") -> str:
    return os.environ.get(key, default)


@dataclass
class _FeedConfig:
    clickhouse_url: str = field(
        default_factory=lambda: _env("KUBRIC_CLICKHOUSE_URL", "clickhouse://default:@clickhouse:9000/kubric")
    )
    nats_url: str = field(
        default_factory=lambda: _env("KUBRIC_NATS_URL", "nats://nats:4222")
    )
    nvd_api_key: str = field(
        default_factory=lambda: _env("KUBRIC_NVD_API_KEY")
    )
    otx_api_key: str = field(
        default_factory=lambda: _env("KUBRIC_OTX_API_KEY")
    )
    abuseipdb_api_key: str = field(
        default_factory=lambda: _env("KUBRIC_ABUSEIPDB_API_KEY")
    )
    misp_url: str = field(
        default_factory=lambda: _env("KUBRIC_MISP_URL")
    )
    misp_api_key: str = field(
        default_factory=lambda: _env("KUBRIC_MISP_API_KEY")
    )
    tenant_id: str = field(
        default_factory=lambda: _env("KUBRIC_TENANT_ID", "system")
    )


# ---------------------------------------------------------------------------
# ClickHouse bulk-insert helper
# ---------------------------------------------------------------------------

class _ClickHouseWriter:
    """Thin wrapper around clickhouse-connect for bulk row inserts."""

    def __init__(self, url: str) -> None:
        self._url = url
        self._client: Any = None

    def _connect(self) -> Any:
        if self._client is None:
            try:
                import clickhouse_connect  # type: ignore[import]
                # Parse clickhouse://user:pass@host:port/db
                from urllib.parse import urlparse
                p = urlparse(self._url)
                self._client = clickhouse_connect.get_client(
                    host=p.hostname or "clickhouse",
                    port=p.port or 9000,
                    username=p.username or "default",
                    password=p.password or "",
                    database=p.path.lstrip("/") or "kubric",
                )
            except Exception as exc:  # noqa: BLE001
                logger.warning("ClickHouse unavailable (%s) — rows discarded", exc)
        return self._client

    def insert(self, table: str, column_names: list[str], rows: list[list[Any]]) -> None:
        if not rows:
            return
        client = self._connect()
        if client is None:
            logger.debug("ClickHouse offline — skipping insert into %s (%d rows)", table, len(rows))
            return
        try:
            client.insert(table, rows, column_names=column_names)
            logger.info("ClickHouse insert %s: %d rows", table, len(rows))
        except Exception as exc:  # noqa: BLE001
            logger.error("ClickHouse insert %s failed: %s", table, exc)


# ---------------------------------------------------------------------------
# NATS publish helper (best-effort; not required for correctness)
# ---------------------------------------------------------------------------

def _nats_publish_sync(nats_url: str, subject: str, payload: bytes) -> None:
    """Fire-and-forget NATS publish using nats-py in a minimal sync wrapper."""
    try:
        import asyncio
        import nats  # type: ignore[import]

        async def _pub() -> None:
            nc = await nats.connect(nats_url)
            await nc.publish(subject, payload)
            await nc.drain()

        asyncio.run(_pub())
    except Exception as exc:  # noqa: BLE001
        logger.warning("NATS publish %s failed: %s", subject, exc)


# ---------------------------------------------------------------------------
# Base feed class
# ---------------------------------------------------------------------------

class _BaseFeed:
    name: str = "base"

    def __init__(self, cfg: _FeedConfig, ch: _ClickHouseWriter) -> None:
        self._cfg = cfg
        self._ch = ch

    # Subclasses implement this
    def pull(self) -> int:
        """Pull the feed and write to ClickHouse.  Returns number of records inserted."""
        raise NotImplementedError

    def _http_get(
        self,
        url: str,
        headers: dict[str, str] | None = None,
        params: dict[str, str] | None = None,
        timeout: float = 30.0,
    ) -> httpx.Response:
        with httpx.Client(timeout=timeout, follow_redirects=True) as client:
            r = client.get(url, headers=headers or {}, params=params or {})
            r.raise_for_status()
            return r


# ---------------------------------------------------------------------------
# Feed 1 — NVD CVE
# ---------------------------------------------------------------------------

class NVDFeed(_BaseFeed):
    """
    NIST NVD CVE 2.0 API — pulls up to 2 000 CVEs per call.
    Table: kubric.nvd_cve (cve_id, cvss_score, published_at, tenant_id, source)
    """

    name = "nvd"
    _URL = "https://services.nvd.nist.gov/rest/json/cves/2.0"
    _COLS = ["cve_id", "cvss_score", "published_at", "description", "source", "tenant_id", "fetched_at"]

    def pull(self) -> int:
        headers: dict[str, str] = {}
        if self._cfg.nvd_api_key:
            headers["apiKey"] = self._cfg.nvd_api_key

        params: dict[str, str] = {"resultsPerPage": "2000"}
        try:
            resp = self._http_get(self._URL, headers=headers, params=params, timeout=60.0)
            data = resp.json()
        except Exception as exc:  # noqa: BLE001
            logger.error("NVD fetch failed: %s", exc)
            return 0

        rows: list[list[Any]] = []
        now = datetime.now(timezone.utc)
        for item in data.get("vulnerabilities", []):
            cve = item.get("cve", {})
            cve_id: str = cve.get("id", "")
            published_str: str = cve.get("published", "")
            try:
                published_at = datetime.fromisoformat(published_str.rstrip("Z"))
            except ValueError:
                published_at = now

            # Extract highest CVSS score available
            cvss_score: float = 0.0
            metrics = cve.get("metrics", {})
            for metric_key in ("cvssMetricV31", "cvssMetricV30", "cvssMetricV2"):
                metric_list = metrics.get(metric_key, [])
                if metric_list:
                    cvss_score = float(
                        metric_list[0].get("cvssData", {}).get("baseScore", 0.0)
                    )
                    break

            # First English description
            description = ""
            for d in cve.get("descriptions", []):
                if d.get("lang") == "en":
                    description = d.get("value", "")[:2000]
                    break

            rows.append([cve_id, cvss_score, published_at, description, "nvd", self._cfg.tenant_id, now])

        self._ch.insert("kubric.nvd_cve", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# Feed 2 — CISA KEV
# ---------------------------------------------------------------------------

class CISAKEVFeed(_BaseFeed):
    """
    CISA Known Exploited Vulnerabilities catalog.
    Inserts into kubric.nvd_cve with in_kev=1.
    Also publishes kubric.kev.updated on NATS.
    """

    name = "cisa_kev"
    _URL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
    _COLS = ["cve_id", "cvss_score", "published_at", "description", "source", "tenant_id", "fetched_at", "in_kev"]

    def pull(self) -> int:
        try:
            resp = self._http_get(self._URL, timeout=30.0)
            data = resp.json()
        except Exception as exc:  # noqa: BLE001
            logger.error("CISA KEV fetch failed: %s", exc)
            return 0

        rows: list[list[Any]] = []
        now = datetime.now(timezone.utc)
        for vuln in data.get("vulnerabilities", []):
            cve_id = vuln.get("cveID", "")
            date_added_str = vuln.get("dateAdded", "")
            try:
                published_at = datetime.strptime(date_added_str, "%Y-%m-%d").replace(tzinfo=timezone.utc)
            except ValueError:
                published_at = now
            description = vuln.get("shortDescription", "")[:2000]
            rows.append([cve_id, 0.0, published_at, description, "cisa_kev", self._cfg.tenant_id, now, 1])

        self._ch.insert("kubric.nvd_cve", self._COLS, rows)

        if rows:
            import orjson  # type: ignore[import]
            payload = orjson.dumps({"count": len(rows), "source": "cisa_kev", "ts": now.isoformat()})
            _nats_publish_sync(self._cfg.nats_url, "kubric.kev.updated", payload)

        return len(rows)


# ---------------------------------------------------------------------------
# Feed 3 — FIRST EPSS
# ---------------------------------------------------------------------------

class EPSSFeed(_BaseFeed):
    """
    FIRST EPSS daily CSV (gzip).
    Table: kubric.epss_scores (cve_id, epss_score, percentile, fetched_at, tenant_id)
    Uses polars for efficient CSV parsing; falls back to csv module if polars unavailable.
    """

    name = "epss"
    _URL = "https://epss.cyentia.com/epss_scores-current.csv.gz"
    _COLS = ["cve_id", "epss_score", "percentile", "fetched_at", "tenant_id"]

    def pull(self) -> int:
        try:
            resp = self._http_get(self._URL, timeout=60.0)
        except Exception as exc:  # noqa: BLE001
            logger.error("EPSS fetch failed: %s", exc)
            return 0

        raw_bytes = gzip.decompress(resp.content)
        now = datetime.now(timezone.utc)
        rows: list[list[Any]] = []

        try:
            import polars as pl  # type: ignore[import]
            df = pl.read_csv(
                io.BytesIO(raw_bytes),
                skip_rows=1,  # skip "#model_version..." comment line
                has_header=True,
                schema_overrides={"cve": pl.Utf8, "epss": pl.Float64, "percentile": pl.Float64},
            )
            for row in df.iter_rows():
                rows.append([row[0], row[1], row[2], now, self._cfg.tenant_id])
        except ImportError:
            # Fallback: stdlib csv
            text = raw_bytes.decode("utf-8")
            reader = csv.reader(io.StringIO(text))
            for line in reader:
                if not line or line[0].startswith("#") or line[0] == "cve":
                    continue
                try:
                    rows.append([line[0], float(line[1]), float(line[2]), now, self._cfg.tenant_id])
                except (IndexError, ValueError):
                    continue

        self._ch.insert("kubric.epss_scores", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# Feed 4 — AlienVault OTX
# ---------------------------------------------------------------------------

class OTXFeed(_BaseFeed):
    """
    AlienVault OTX subscribed pulses.
    Table: kubric.ti_indicators (ioc_type, ioc_value, source, confidence, tenant_id, fetched_at)
    """

    name = "otx"
    _BASE = "https://otx.alienvault.com/api/v1/pulses/subscribed"
    _COLS = ["ioc_type", "ioc_value", "source", "confidence", "tenant_id", "fetched_at", "tags"]

    def pull(self) -> int:
        if not self._cfg.otx_api_key:
            logger.info("OTX: no API key configured — skipping")
            return 0

        headers = {"X-OTX-API-KEY": self._cfg.otx_api_key}
        rows: list[list[Any]] = []
        now = datetime.now(timezone.utc)
        url: str | None = self._BASE

        while url:
            try:
                resp = self._http_get(url, headers=headers, timeout=30.0)
                data = resp.json()
            except Exception as exc:  # noqa: BLE001
                logger.error("OTX fetch failed: %s", exc)
                break

            for pulse in data.get("results", []):
                tags = ",".join(pulse.get("tags", []))[:500]
                for ind in pulse.get("indicators", []):
                    ioc_type = ind.get("type", "")
                    ioc_value = str(ind.get("indicator", ""))[:512]
                    rows.append([ioc_type, ioc_value, "otx", 80, self._cfg.tenant_id, now, tags])

            url = data.get("next")  # OTX paginates with "next" URL
            if url and len(rows) > 50_000:
                logger.info("OTX: capped at 50k indicators per run")
                break

        self._ch.insert("kubric.ti_indicators", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# Feed 5 — AbuseIPDB
# ---------------------------------------------------------------------------

class AbuseIPDBFeed(_BaseFeed):
    """
    AbuseIPDB blacklist (confidence ≥55).
    Table: kubric.ti_indicators (ioc_type='ip', ...)
    """

    name = "abuseipdb"
    _URL = "https://api.abuseipdb.com/api/v2/blacklist"
    _COLS = ["ioc_type", "ioc_value", "source", "confidence", "tenant_id", "fetched_at", "tags"]

    def pull(self) -> int:
        if not self._cfg.abuseipdb_api_key:
            logger.info("AbuseIPDB: no API key configured — skipping")
            return 0

        headers = {
            "Key": self._cfg.abuseipdb_api_key,
            "Accept": "application/json",
        }
        params = {"confidenceMinimum": "55", "limit": "10000"}
        try:
            resp = self._http_get(self._URL, headers=headers, params=params, timeout=60.0)
            data = resp.json()
        except Exception as exc:  # noqa: BLE001
            logger.error("AbuseIPDB fetch failed: %s", exc)
            return 0

        now = datetime.now(timezone.utc)
        rows: list[list[Any]] = []
        for entry in data.get("data", []):
            ip = str(entry.get("ipAddress", ""))[:64]
            confidence = int(entry.get("abuseConfidenceScore", 0))
            rows.append(["ip", ip, "abuseipdb", confidence, self._cfg.tenant_id, now, ""])

        self._ch.insert("kubric.ti_indicators", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# Feed 6 — IPSum
# ---------------------------------------------------------------------------

class IPSumFeed(_BaseFeed):
    """
    IPSum community IP blocklist (stamparm/ipsum).
    Uses the level-3 block (detected by 3+ tools).
    Table: kubric.ti_indicators (ioc_type='ip', confidence=60, source='ipsum')
    """

    name = "ipsum"
    _URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/ipsum.txt"
    _COLS = ["ioc_type", "ioc_value", "source", "confidence", "tenant_id", "fetched_at", "tags"]

    def pull(self) -> int:
        try:
            resp = self._http_get(self._URL, timeout=30.0)
        except Exception as exc:  # noqa: BLE001
            logger.error("IPSum fetch failed: %s", exc)
            return 0

        now = datetime.now(timezone.utc)
        rows: list[list[Any]] = []
        for line in resp.text.splitlines():
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            parts = line.split()
            ip = parts[0]
            try:
                count = int(parts[1]) if len(parts) > 1 else 1
            except ValueError:
                count = 1
            if count >= 3:  # only include IPs seen by 3+ sources
                rows.append(["ip", ip, "ipsum", min(count * 10, 100), self._cfg.tenant_id, now, ""])

        self._ch.insert("kubric.ti_indicators", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# Feed 7 — PyMISP
# ---------------------------------------------------------------------------

class MISPFeed(_BaseFeed):
    """
    MISP attribute ingestion via PyMISP (or raw REST if pymisp not installed).
    Table: kubric.ti_indicators (ioc_type, ioc_value, source='misp', ...)
    """

    name = "misp"
    _COLS = ["ioc_type", "ioc_value", "source", "confidence", "tenant_id", "fetched_at", "tags"]

    # Max indicators to fetch per run (safety cap)
    _MAX_ROWS = 100_000
    _PAGE_SIZE = 5_000

    def pull(self) -> int:
        if not self._cfg.misp_url or not self._cfg.misp_api_key:
            logger.info("MISP: no URL/key configured — skipping")
            return 0

        rows: list[list[Any]] = []
        now = datetime.now(timezone.utc)

        try:
            from pymisp import PyMISP  # type: ignore[import]
            misp = PyMISP(self._cfg.misp_url, self._cfg.misp_api_key, ssl=False)
            page = 1
            while len(rows) < self._MAX_ROWS:
                result = misp.search(
                    controller="attributes",
                    limit=self._PAGE_SIZE,
                    page=page,
                    to_ids=1,
                )
                attributes = result.get("Attribute", []) if isinstance(result, dict) else []
                if not attributes:
                    break
                for attr in attributes:
                    ioc_type  = attr.get("type", "")
                    ioc_value = str(attr.get("value", ""))[:512]
                    tags = ",".join(
                        t.get("name", "") for t in attr.get("Tag", [])
                    )[:500]
                    rows.append([ioc_type, ioc_value, "misp", 70,
                                 self._cfg.tenant_id, now, tags])
                logger.debug("MISP page %d: %d attributes", page, len(attributes))
                if len(attributes) < self._PAGE_SIZE:
                    break  # last page
                page += 1

        except ImportError:
            # Fallback: raw MISP REST API (POST to /attributes/restSearch)
            headers = {
                "Authorization": self._cfg.misp_api_key,
                "Accept":        "application/json",
                "Content-Type":  "application/json",
            }
            url = f"{self._cfg.misp_url.rstrip('/')}/attributes/restSearch"
            page = 1
            while len(rows) < self._MAX_ROWS:
                body = {
                    "returnFormat": "json",
                    "limit":  self._PAGE_SIZE,
                    "page":   page,
                    "to_ids": 1,
                }
                try:
                    with httpx.Client(timeout=30.0, follow_redirects=True) as client:
                        resp = client.post(url, headers=headers, json=body)
                        resp.raise_for_status()
                        data = resp.json()
                    attributes = data.get("response", {}).get("Attribute", [])
                    if not attributes:
                        break
                    for attr in attributes:
                        ioc_type  = attr.get("type", "")
                        ioc_value = str(attr.get("value", ""))[:512]
                        rows.append([ioc_type, ioc_value, "misp", 70,
                                     self._cfg.tenant_id, now, ""])
                    if len(attributes) < self._PAGE_SIZE:
                        break
                    page += 1
                except Exception as exc:  # noqa: BLE001
                    logger.error("MISP REST fallback page %d failed: %s", page, exc)
                    break

        except Exception as exc:  # noqa: BLE001
            logger.error("MISP PyMISP pull failed: %s", exc)

        self._ch.insert("kubric.ti_indicators", self._COLS, rows)
        return len(rows)


# ---------------------------------------------------------------------------
# TIFeedManager — orchestrates all feeds
# ---------------------------------------------------------------------------

class TIFeedManager:
    """
    Manages all TI feeds.  Instantiate once and call pull_all() or
    individual pull_<name>() methods from a scheduler.

    Example
    -------
    >>> mgr = TIFeedManager()
    >>> mgr.pull_all()
    """

    def __init__(self, cfg: _FeedConfig | None = None) -> None:
        self._cfg = cfg or _FeedConfig()
        self._ch = _ClickHouseWriter(self._cfg.clickhouse_url)
        self._feeds: dict[str, _BaseFeed] = {}
        for cls in (NVDFeed, CISAKEVFeed, EPSSFeed, OTXFeed, AbuseIPDBFeed, IPSumFeed, MISPFeed):
            feed = cls(self._cfg, self._ch)
            self._feeds[feed.name] = feed

    def pull(self, name: str) -> int:
        """Pull a single feed by name.  Returns record count."""
        feed = self._feeds.get(name)
        if feed is None:
            raise ValueError(f"Unknown feed: {name!r}.  Available: {list(self._feeds)}")
        start = time.monotonic()
        logger.info("[TI] pulling %s...", name)
        try:
            count = feed.pull()
            elapsed = time.monotonic() - start
            logger.info("[TI] %s done: %d records in %.1fs", name, count, elapsed)
            return count
        except Exception as exc:  # noqa: BLE001
            logger.error("[TI] %s failed: %s", name, exc)
            return 0

    def pull_all(self) -> dict[str, int]:
        """Pull all feeds and return {feed_name: record_count}."""
        results: dict[str, int] = {}
        for name in self._feeds:
            results[name] = self.pull(name)
        return results

    # Convenience passthrough methods (used by scheduler by name)
    def pull_nvd(self) -> int:
        return self.pull("nvd")

    def pull_cisa_kev(self) -> int:
        return self.pull("cisa_kev")

    def pull_epss(self) -> int:
        return self.pull("epss")

    def pull_otx(self) -> int:
        return self.pull("otx")

    def pull_abuseipdb(self) -> int:
        return self.pull("abuseipdb")

    def pull_ipsum(self) -> int:
        return self.pull("ipsum")

    def pull_misp(self) -> int:
        return self.pull("misp")

    def __repr__(self) -> str:
        return f"TIFeedManager(feeds={list(self._feeds)}, tenant={self._cfg.tenant_id!r})"
