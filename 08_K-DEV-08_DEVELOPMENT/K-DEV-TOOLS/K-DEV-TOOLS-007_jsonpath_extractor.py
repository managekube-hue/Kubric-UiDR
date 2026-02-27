"""
JSONPath extraction utilities for Kubric event and API payload processing.
Module: K-DEV-TOOLS-007

Install: pip install jsonpath-ng
"""
from __future__ import annotations

from typing import Any

from jsonpath_ng import jsonpath  # noqa: F401 — re-exported for callers
from jsonpath_ng import parse as jparse

# ---------------------------------------------------------------------------
# Pre-built JSONPath expressions used across the Kubric platform
# ---------------------------------------------------------------------------
ALERT_IDS          = "$.alerts[*].id"
ALERT_SEVERITIES   = "$.alerts[*].severity"
CVE_LIST           = "$.vulnerabilities[*].cve_id"
TENANT_IDS         = "$.tenants[*].id"
ASSET_HOSTNAMES    = "$.assets[*].hostname"
AGENT_IDS          = "$.agents[*].id"
PATCH_JOB_STATUS   = "$.patch_jobs[*].status"
DECISION_PERSONA   = "$.decisions[*].agent_persona"
OCSF_EVENT_TIMES   = "$..event_time"
NESTED_TENANT_ID   = "$..tenant_id"


class JSONPathExtractor:
    """Utility class for JSONPath extraction from Kubric payloads."""

    def __init__(self) -> None:
        # cache compiled expressions
        self._cache: dict[str, Any] = {}

    def _compiled(self, expr: str) -> Any:
        if expr not in self._cache:
            self._cache[expr] = jparse(expr)
        return self._cache[expr]

    def extract(self, data: dict[str, Any] | list[Any], expr: str) -> list[Any]:
        """
        Parse `expr` and return all matching values from `data`.
        Returns an empty list if nothing matches.
        """
        matches = self._compiled(expr).find(data)
        return [m.value for m in matches]

    def extract_first(
        self,
        data: dict[str, Any] | list[Any],
        expr: str,
    ) -> Any | None:
        """Return the first match or None."""
        results = self.extract(data, expr)
        return results[0] if results else None

    def extract_alerts_by_severity(
        self,
        alerts: list[dict[str, Any]],
        severity: str,
    ) -> list[dict[str, Any]]:
        """
        Return only alert dicts whose severity matches `severity` (case-insensitive).
        Uses JSONPath filter expression: $[?(@.severity == 'severity')].
        """
        expr = f"$[?(@.severity == '{severity.lower()}')]"
        return self.extract(alerts, expr)

    def flatten_nested(
        self,
        data: dict[str, Any] | list[Any],
        path: str,
    ) -> list[Any]:
        """
        Extract all values at `path` (including deeply nested), returning a
        flat list. Useful for flattening NATS message batches.
        Example: path = "$..event_time"
        """
        return self.extract(data, path)

    def extract_by_field_value(
        self,
        data: list[dict[str, Any]],
        field: str,
        value: Any,
    ) -> list[dict[str, Any]]:
        """
        Return items from `data` where item[field] == value.
        Equivalent to: $[?(@.<field> == '<value>')]
        """
        if isinstance(value, str):
            expr = f"$[?(@.{field} == '{value}')]"
        else:
            expr = f"$[?(@.{field} == {value})]"
        return self.extract(data, expr)

    def extract_keys(
        self,
        data: dict[str, Any] | list[Any],
        parent_path: str,
        key: str,
    ) -> list[Any]:
        """
        Extract a single key from each object in a collection.
        parent_path: e.g. "$.alerts[*]"
        key: e.g. "id"
        """
        return self.extract(data, f"{parent_path}.{key}")

    # ------------------------------------------------------------------
    # Named helpers mirroring the pre-built expressions
    # ------------------------------------------------------------------

    def alert_ids(self, payload: dict[str, Any]) -> list[str]:
        return self.extract(payload, ALERT_IDS)

    def cve_list(self, payload: dict[str, Any]) -> list[str]:
        return self.extract(payload, CVE_LIST)

    def tenant_ids(self, payload: dict[str, Any]) -> list[str]:
        return self.extract(payload, TENANT_IDS)

    def asset_hostnames(self, payload: dict[str, Any]) -> list[str]:
        return self.extract(payload, ASSET_HOSTNAMES)

    def critical_alerts(self, payload: dict[str, Any]) -> list[dict[str, Any]]:
        return self.extract_alerts_by_severity(payload.get("alerts", []), "critical")


# ---------------------------------------------------------------------------
# Module-level convenience instance
# ---------------------------------------------------------------------------
extractor = JSONPathExtractor()


# ---------------------------------------------------------------------------
# CLI demo
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import json

    sample = {
        "alerts": [
            {"id": "a1", "severity": "critical", "title": "Ransomware"},
            {"id": "a2", "severity": "high",     "title": "Lateral move"},
            {"id": "a3", "severity": "critical", "title": "Data exfil"},
            {"id": "a4", "severity": "medium",   "title": "Brute force"},
        ],
        "vulnerabilities": [
            {"cve_id": "CVE-2024-1234", "cvss_v3": 9.8},
            {"cve_id": "CVE-2023-5678", "cvss_v3": 7.5},
        ],
        "tenant": {
            "id": "tenant-abc",
            "assets": [
                {"hostname": "prod-01", "tenant_id": "tenant-abc"},
                {"hostname": "prod-02", "tenant_id": "tenant-abc"},
            ],
        },
    }

    x = JSONPathExtractor()

    print("Alert IDs:", x.alert_ids(sample))
    print("CVE list:", x.cve_list(sample))
    print("Critical alerts:", json.dumps(x.critical_alerts(sample), indent=2))
    print("All tenant_ids (nested):", x.flatten_nested(sample, NESTED_TENANT_ID))
    print("Alerts by severity=high:", x.extract_alerts_by_severity(sample["alerts"], "high"))
