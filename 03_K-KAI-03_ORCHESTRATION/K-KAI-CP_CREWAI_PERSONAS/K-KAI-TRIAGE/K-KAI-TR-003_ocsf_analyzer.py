"""
K-KAI-TR-003: OCSF Event Analyzer
Normalises OCSF class events into a unified triage schema and extracts
indicators of compromise.

Supported OCSF classes:
  4001 - File System Activity (FIM)
  4003 - Network Activity
  4007 - Process Activity
  2004 - Detection Finding
"""

import logging
import re
from typing import Any

logger = logging.getLogger("K-KAI-TR-003")

# ---------------------------------------------------------------------------
# OCSF severity_id → integer mapping (1 = Informational … 6 = Fatal)
# ---------------------------------------------------------------------------
_SEVERITY_MAP: dict[int, int] = {
    0: 1,  # Unknown → Informational
    1: 1,  # Informational
    2: 2,  # Low
    3: 3,  # Medium
    4: 4,  # High
    5: 5,  # Critical
    6: 6,  # Fatal
}

# Regex patterns for IoC extraction
_IP_RE = re.compile(
    r"\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b"
)
_DOMAIN_RE = re.compile(
    r"\b(?:[a-zA-Z0-9\-]+\.)+(?:com|net|org|io|info|biz|co|ru|cn|de|uk|gov|edu|mil)\b"
)
_MD5_RE = re.compile(r"\b[0-9a-fA-F]{32}\b")
_SHA1_RE = re.compile(r"\b[0-9a-fA-F]{40}\b")
_SHA256_RE = re.compile(r"\b[0-9a-fA-F]{64}\b")
_URL_RE = re.compile(r"https?://[^\s\"'<>]+")
_CVE_RE = re.compile(r"\bCVE-\d{4}-\d{4,}\b", re.IGNORECASE)


class OCSFAnalyzer:
    """Normalises and analyses OCSF-formatted events."""

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def normalize(self, raw_event: dict) -> dict:
        """
        Map an OCSF event of any supported class into the unified triage schema.

        Always returns a dict; unknown class_uid produces a best-effort mapping.
        """
        class_uid: int = int(raw_event.get("class_uid", 0))
        dispatch = {
            4001: self._normalize_fim,
            4003: self._normalize_network,
            4007: self._normalize_process,
            2004: self._normalize_detection,
        }
        handler = dispatch.get(class_uid, self._normalize_generic)
        return handler(raw_event)

    def extract_indicators(self, event: dict) -> list[dict]:
        """
        Walk all string values in the normalised event and extract IoCs.

        Returns a deduplicated list of dicts with keys: type, value.
        """
        text: str = self._flatten_to_text(event)
        seen: set[str] = set()
        results: list[dict] = []

        def _add(ioc_type: str, value: str) -> None:
            key = f"{ioc_type}:{value.lower()}"
            if key not in seen:
                seen.add(key)
                results.append({"type": ioc_type, "value": value})

        for match in _SHA256_RE.finditer(text):
            _add("sha256", match.group(0).lower())
        for match in _SHA1_RE.finditer(text):
            _add("sha1", match.group(0).lower())
        for match in _MD5_RE.finditer(text):
            _add("md5", match.group(0).lower())
        for match in _IP_RE.finditer(text):
            ip = match.group(0)
            if not ip.startswith(("10.", "192.168.", "172.")):
                _add("ip", ip)
        for match in _URL_RE.finditer(text):
            _add("url", match.group(0))
        for match in _DOMAIN_RE.finditer(text):
            _add("domain", match.group(0).lower())
        for match in _CVE_RE.finditer(text):
            _add("cve", match.group(0).upper())

        return results

    def classify_severity(self, event: dict) -> int:
        """
        Return an OCSF severity integer (1–6) for the normalised event.

        Strategy (highest wins):
          1. Use normalised severity_id if already present.
          2. Derive from CVSS base score.
          3. Derive from keyword heuristics in the message field.
        """
        # 1. Explicit severity from normalised event
        sid = event.get("severity_id")
        if sid is not None:
            return _SEVERITY_MAP.get(int(sid), 3)

        # 2. CVSS-derived
        cvss = event.get("cvss_base_score")
        if cvss is not None:
            return self._cvss_to_severity(float(cvss))

        # 3. Message heuristics
        message: str = str(event.get("message", "")).lower()
        if any(w in message for w in ("ransomware", "exfiltration", "rootkit", "critical")):
            return 5
        if any(w in message for w in ("malware", "exploit", "backdoor", "high")):
            return 4
        if any(w in message for w in ("suspicious", "anomaly", "medium")):
            return 3
        if any(w in message for w in ("informational", "low", "benign")):
            return 2

        return 3  # default: medium

    # ------------------------------------------------------------------
    # Class-specific normalisers
    # ------------------------------------------------------------------

    def _normalize_fim(self, ev: dict) -> dict:
        base = self._base(ev)
        base.update({
            "class_name": "File System Activity",
            "file_path": self._deep(ev, "file", "path"),
            "file_hash": self._deep(ev, "file", "hashes", 0, "value"),
            "activity": self._deep(ev, "activity_name"),
            "actor_process": self._deep(ev, "actor", "process", "name"),
        })
        return base

    def _normalize_network(self, ev: dict) -> dict:
        base = self._base(ev)
        conn = ev.get("connection_info", {}) or {}
        base.update({
            "class_name": "Network Activity",
            "src_ip": self._deep(ev, "src_endpoint", "ip") or conn.get("src_ip"),
            "dst_ip": self._deep(ev, "dst_endpoint", "ip") or conn.get("dst_ip"),
            "dst_port": self._deep(ev, "dst_endpoint", "port"),
            "protocol": conn.get("protocol_name"),
            "direction": conn.get("direction"),
        })
        return base

    def _normalize_process(self, ev: dict) -> dict:
        base = self._base(ev)
        proc = ev.get("process", {}) or {}
        base.update({
            "class_name": "Process Activity",
            "process_name": proc.get("name"),
            "process_pid": proc.get("pid"),
            "cmd_line": proc.get("cmd_line"),
            "parent_process": self._deep(ev, "parent_process", "name"),
            "actor_user": self._deep(ev, "actor", "user", "name"),
        })
        return base

    def _normalize_detection(self, ev: dict) -> dict:
        base = self._base(ev)
        finding = ev.get("finding", {}) or {}
        base.update({
            "class_name": "Detection Finding",
            "rule_name": finding.get("title") or ev.get("rule_name"),
            "rule_uid": finding.get("uid"),
            "confidence": finding.get("confidence_id"),
            "cvss_base_score": self._extract_cvss(ev),
            "epss": ev.get("epss"),
            "actively_exploited": bool(ev.get("actively_exploited", False)),
        })
        return base

    def _normalize_generic(self, ev: dict) -> dict:
        base = self._base(ev)
        base["class_name"] = ev.get("class_name", "Unknown")
        return base

    # ------------------------------------------------------------------
    # Helpers
    # ------------------------------------------------------------------

    def _base(self, ev: dict) -> dict:
        return {
            "class_uid": ev.get("class_uid", 0),
            "time": ev.get("time") or ev.get("timestamp"),
            "message": ev.get("message") or ev.get("description", ""),
            "severity_id": ev.get("severity_id"),
            "status_id": ev.get("status_id"),
            "asset_criticality": ev.get("asset_criticality", 3),
            "tenant_id": ev.get("tenant_id"),
            "device": ev.get("device", {}),
            "_raw": ev,
        }

    @staticmethod
    def _deep(obj: Any, *keys: Any) -> Any:
        """Safe nested dict/list accessor. Returns None if path missing."""
        cur = obj
        for k in keys:
            if cur is None:
                return None
            try:
                cur = cur[k] if isinstance(k, (int, str)) else None
            except (KeyError, IndexError, TypeError):
                return None
        return cur

    @staticmethod
    def _cvss_to_severity(cvss: float) -> int:
        if cvss >= 9.0:
            return 5  # Critical
        if cvss >= 7.0:
            return 4  # High
        if cvss >= 4.0:
            return 3  # Medium
        if cvss >= 0.1:
            return 2  # Low
        return 1

    @staticmethod
    def _extract_cvss(ev: dict) -> float | None:
        """Try several common CVSS field locations."""
        for path in [
            ["cvss", "base_score"],
            ["vulnerability", "cvss", "base_score"],
            ["cvss_base_score"],
        ]:
            val = ev
            for key in path:
                if isinstance(val, dict):
                    val = val.get(key)
                else:
                    val = None
                    break
            if val is not None:
                try:
                    return float(val)
                except (TypeError, ValueError):
                    pass
        return None

    @staticmethod
    def _flatten_to_text(obj: Any, _depth: int = 0) -> str:
        """Recursively flatten a dict/list to a single string for regex scanning."""
        if _depth > 8:
            return ""
        if isinstance(obj, str):
            return obj + " "
        if isinstance(obj, (int, float)):
            return str(obj) + " "
        if isinstance(obj, dict):
            return " ".join(
                OCSFAnalyzer._flatten_to_text(v, _depth + 1) for v in obj.values()
            )
        if isinstance(obj, list):
            return " ".join(
                OCSFAnalyzer._flatten_to_text(item, _depth + 1) for item in obj
            )
        return ""
