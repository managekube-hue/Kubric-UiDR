"""
K-SOC-TI-004 — PhishTank phishing URL checker.

Downloads the PhishTank online-valid dataset, maintains an in-memory set of
phishing URLs, and provides O(1) URL / domain lookups.  Auto-reloads every
6 hours in a background task.

Env vars:
    PHISH_TANK_APP_KEY  optional API key (enhances CSV URL access)
    PHISH_DATA_DIR      directory for persisting the download timestamp
                        (default: /tmp/phishtank)
"""

from __future__ import annotations

import asyncio
import csv
import gzip
import io
import logging
import os
import pickle
import time
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

import httpx
import structlog

log = structlog.get_logger(__name__)

PHISH_TANK_APP_KEY = os.getenv("PHISH_TANK_APP_KEY", "")
PHISH_DATA_DIR = Path(os.getenv("PHISH_DATA_DIR", "/tmp/phishtank"))
RELOAD_INTERVAL = 6 * 3600  # 6 hours in seconds

# PhishTank provides the CSV without auth; with app_key the URL is the same
# but rate limits are higher.
PHISH_CSV_URL = "http://data.phishtank.com/data/online-valid.csv.gz"


class PhishTankClient:
    """
    In-memory PhishTank database with automatic background refresh.

    Thread-safety: asyncio-safe — the set is replaced atomically.
    """

    def __init__(self) -> None:
        self._urls: frozenset[str] = frozenset()
        self._last_downloaded: float = 0.0
        self._lock = asyncio.Lock()
        PHISH_DATA_DIR.mkdir(parents=True, exist_ok=True)
        self._state_file = PHISH_DATA_DIR / "state.pkl"

    # ------------------------------------------------------------------
    # Download & parse
    # ------------------------------------------------------------------

    async def _download_csv(self) -> list[str]:
        """Download and decompress the PhishTank CSV. Returns list of phishing URLs."""
        params: dict[str, str] = {}
        if PHISH_TANK_APP_KEY:
            params["app_key"] = PHISH_TANK_APP_KEY

        async with httpx.AsyncClient(timeout=httpx.Timeout(120.0)) as client:
            log.info("phishtank.downloading", url=PHISH_CSV_URL)
            resp = await client.get(PHISH_CSV_URL, params=params, follow_redirects=True)
            resp.raise_for_status()
            raw = resp.content

        with gzip.open(io.BytesIO(raw), "rt", encoding="utf-8", errors="replace") as f:
            reader = csv.DictReader(f)
            urls = []
            for row in reader:
                url = row.get("url", "").strip()
                if url:
                    urls.append(url)

        log.info("phishtank.loaded", count=len(urls))
        return urls

    async def refresh(self) -> None:
        """Download and reload the phishing URL database."""
        async with self._lock:
            urls = await self._download_csv()
            self._urls = frozenset(urls)
            self._last_downloaded = time.time()
            # Persist download timestamp so we survive restarts.
            try:
                self._state_file.write_bytes(
                    pickle.dumps({"ts": self._last_downloaded, "count": len(self._urls)})
                )
            except OSError:
                pass

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    async def check_url(self, url: str) -> bool:
        """Return True if the exact URL is in the PhishTank database (O(1))."""
        return url in self._urls

    async def check_domain(self, domain: str) -> list[str]:
        """Return all phishing URLs in the database that belong to domain."""
        domain_lower = domain.lower()
        return [
            url for url in self._urls
            if _extract_domain(url) == domain_lower
        ]

    # ------------------------------------------------------------------
    # Background auto-reload
    # ------------------------------------------------------------------

    async def run_forever(self) -> None:
        """Load once immediately, then reload every 6 hours."""
        # Restore state if available.
        if self._state_file.exists():
            try:
                state = pickle.loads(self._state_file.read_bytes())
                self._last_downloaded = state.get("ts", 0.0)
            except Exception:  # noqa: BLE001
                pass

        while True:
            try:
                await self.refresh()
            except Exception as exc:  # noqa: BLE001
                log.error("phishtank.refresh_error", error=str(exc))
            await asyncio.sleep(RELOAD_INTERVAL)


# ---------------------------------------------------------------------------
# Helper
# ---------------------------------------------------------------------------

def _extract_domain(url: str) -> str:
    """Extract lowercase domain from a URL string."""
    try:
        parsed = urlparse(url)
        return (parsed.netloc or parsed.path).split(":")[0].lower()
    except Exception:  # noqa: BLE001
        return ""


# ---------------------------------------------------------------------------
# Entrypoint — starts the refresh loop
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.processors.JSONRenderer(),
        ]
    )

    async def _main() -> None:
        client = PhishTankClient()
        await client.run_forever()

    asyncio.run(_main())
