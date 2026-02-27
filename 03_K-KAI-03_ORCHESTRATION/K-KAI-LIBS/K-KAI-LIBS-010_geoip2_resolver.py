"""
K-KAI-LIBS-010: GeoIP2 IP geolocation resolver using MaxMind GeoLite2-City.
Singleton with LRU cache (maxsize=10000).
Also checks a local tor-exit-nodes.txt file for Tor exit detection.
"""

import logging
import os
from functools import lru_cache
from pathlib import Path
from typing import Any

logger = logging.getLogger("kai.libs.geoip2")

_DEFAULT_DB_PATH = os.environ.get("GEOIP2_DB_PATH", "/etc/geoip/GeoLite2-City.mmdb")
_TOR_LIST_PATH = os.environ.get(
    "TOR_EXIT_NODES_PATH", "/etc/geoip/tor-exit-nodes.txt"
)

try:
    import geoip2.database
    import geoip2.errors
    _GEOIP2_AVAILABLE = True
except ImportError:
    _GEOIP2_AVAILABLE = False
    logger.warning("geoip2 not installed – GeoIPResolver will return empty results")

try:
    import maxminddb  # for ASN lookups when using geoip2
    _MAXMINDDB_AVAILABLE = True
except ImportError:
    _MAXMINDDB_AVAILABLE = False


# ---------------------------------------------------------------------------
# Tor exit node set (loaded once at import time)
# ---------------------------------------------------------------------------
def _load_tor_exits(path: str) -> set[str]:
    """Load Tor exit node IPs from a plain text file (one IP per line)."""
    p = Path(path)
    if not p.exists():
        logger.debug("Tor exit-node list not found at %s", path)
        return set()
    ips: set[str] = set()
    with open(str(p)) as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                ips.add(line)
    logger.info("Loaded %d Tor exit nodes from %s", len(ips), path)
    return ips


_TOR_EXITS: set[str] = _load_tor_exits(_TOR_LIST_PATH)


# ---------------------------------------------------------------------------
# GeoIPResolver
# ---------------------------------------------------------------------------
class GeoIPResolver:
    """
    Singleton GeoIP2 resolver with LRU cache.
    Falls back gracefully when the MMDB file is absent.
    """

    _instance: "GeoIPResolver | None" = None

    def __new__(cls) -> "GeoIPResolver":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self) -> None:
        if hasattr(self, "_initialized"):
            return
        self._initialized = True
        self._reader: Any = None
        self._db_path = _DEFAULT_DB_PATH
        self._open_reader()

    def _open_reader(self) -> None:
        if not _GEOIP2_AVAILABLE:
            return
        if Path(self._db_path).exists():
            try:
                self._reader = geoip2.database.Reader(self._db_path)
                logger.info("GeoIPResolver: opened %s", self._db_path)
            except Exception as exc:
                logger.error("GeoIPResolver: failed to open DB: %s", exc)
        else:
            logger.warning("GeoIPResolver: DB not found at %s", self._db_path)

    # ------------------------------------------------------------------
    @lru_cache(maxsize=10_000)
    def resolve(self, ip: str) -> dict:
        """
        Resolve an IP address to geo + ASN metadata.
        Returns dict with: country_iso, country_name, city, latitude,
        longitude, asn, org.
        """
        empty = {
            "country_iso": None,
            "country_name": None,
            "city": None,
            "latitude": None,
            "longitude": None,
            "asn": None,
            "org": None,
        }

        if not self._reader:
            return empty

        try:
            rec = self._reader.city(ip)
            return {
                "country_iso": rec.country.iso_code,
                "country_name": rec.country.name,
                "city": rec.city.name,
                "latitude": rec.location.latitude,
                "longitude": rec.location.longitude,
                "asn": getattr(rec.traits, "autonomous_system_number", None),
                "org": getattr(rec.traits, "autonomous_system_organization", None),
            }
        except geoip2.errors.AddressNotFoundError:
            return empty
        except Exception as exc:
            logger.debug("GeoIPResolver.resolve(%s) error: %s", ip, exc)
            return empty

    # ------------------------------------------------------------------
    def resolve_bulk(self, ips: list[str]) -> dict[str, dict]:
        """
        Resolve multiple IPs concurrently (thread pool for CPU-bound lookup).
        Returns dict mapping ip -> resolve() result.
        """
        from concurrent.futures import ThreadPoolExecutor
        results: dict[str, dict] = {}
        with ThreadPoolExecutor(max_workers=16, thread_name_prefix="geoip") as pool:
            futures = {ip: pool.submit(self.resolve, ip) for ip in ips}
            for ip, fut in futures.items():
                try:
                    results[ip] = fut.result(timeout=5)
                except Exception as exc:
                    logger.debug("resolve_bulk: error for %s: %s", ip, exc)
                    results[ip] = {}
        return results

    # ------------------------------------------------------------------
    def is_tor_exit(self, ip: str) -> bool:
        """Return True if *ip* is a known Tor exit node."""
        return ip in _TOR_EXITS

    # ------------------------------------------------------------------
    def close(self) -> None:
        """Close the MaxMind database reader."""
        if self._reader:
            self._reader.close()
            self._reader = None
            logger.info("GeoIPResolver: reader closed")


# ---------------------------------------------------------------------------
# Module-level singleton accessor
# ---------------------------------------------------------------------------
_resolver: GeoIPResolver | None = None


def get_resolver() -> GeoIPResolver:
    """Return the module-level GeoIPResolver singleton."""
    global _resolver
    if _resolver is None:
        _resolver = GeoIPResolver()
    return _resolver


def resolve_ip(ip: str) -> dict:
    """Convenience function: resolve a single IP."""
    return get_resolver().resolve(ip)


def resolve_ips(ips: list[str]) -> dict[str, dict]:
    """Convenience function: bulk resolve IPs."""
    return get_resolver().resolve_bulk(ips)


def is_tor_exit(ip: str) -> bool:
    """Convenience function: check if IP is a known Tor exit node."""
    return get_resolver().is_tor_exit(ip)
