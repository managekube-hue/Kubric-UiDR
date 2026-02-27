"""
JMESPath query utilities for AWS/API response processing.
Module: K-DEV-TOOLS-008

Install: pip install jmespath
Usage:
    python K-DEV-TOOLS-008_jmespath_querier.py
"""
from __future__ import annotations

import json
import sys
from typing import Any

try:
    import jmespath
    from jmespath.exceptions import JMESPathError
except ImportError:
    print("ERROR: pip install jmespath", file=sys.stderr)
    sys.exit(1)

# ---------------------------------------------------------------------------
# Pre-built JMESPath expressions for Kubric API responses
# ---------------------------------------------------------------------------
TENANT_IDS          = "tenants[*].id"
TENANT_NAMES        = "tenants[*].name"
CRITICAL_ALERTS     = "alerts[?severity=='critical']"
OPEN_ALERTS         = "alerts[?status=='open']"
ALERT_IDS           = "alerts[*].id"
ALERT_SEVERITIES    = "alerts[*].severity"
CVE_IDS             = "vulnerabilities[*].cve_id"
HIGH_CVSS_VULNS     = "vulnerabilities[?cvss_v3 >= `7.0`]"
CRITICAL_CVSS_VULNS = "vulnerabilities[?cvss_v3 >= `9.0`]"
ACTIVE_AGENTS       = "agents[?status=='active']"
FAILED_PATCH_JOBS   = "patch_jobs[?status=='failed']"


class JMESPathQuerier:
    """Utility class for JMESPath queries over Kubric API and AWS responses."""

    def query(self, data: Any, expression: str) -> Any:
        """
        Execute a JMESPath `expression` against `data`.
        Returns None if the expression matches nothing.
        Raises JMESPathError on invalid expressions.
        """
        return jmespath.search(expression, data)

    def filter_by_field(
        self,
        data: list[dict[str, Any]],
        field: str,
        value: str | int | float | bool,
    ) -> list[dict[str, Any]]:
        """
        Filter a list of dicts where item[field] == value.
        Wraps jmespath filter syntax: [?field=='value'] or [?field==`value`]
        """
        if isinstance(value, str):
            expr = f"[?{field}=='{value}']"
        elif isinstance(value, bool):
            expr = f"[?{field}==`{'true' if value else 'false'}`]"
        else:
            expr = f"[?{field}==`{value}`]"
        result = jmespath.search(expr, data)
        return result if result is not None else []

    def extract_nested(
        self,
        response: dict[str, Any],
        path: str,
    ) -> list[Any]:
        """
        Extract a list at `path` from a (potentially nested) API response.
        Returns an empty list when the path does not exist.
        """
        result = jmespath.search(path, response)
        if result is None:
            return []
        if isinstance(result, list):
            return result
        return [result]

    def flatten_aws_response(
        self,
        data: dict[str, Any],
        items_key: str = "Items",
    ) -> list[dict[str, Any]]:
        """
        Normalize a typical AWS SDK paginated response into a flat list.
        Handles:
          - DynamoDB: {"Items": [...], "Count": N, "LastEvaluatedKey": {...}}
          - EC2/describe-*: {"Reservations": [{"Instances": [...]}]}
          - Generic: root-level list at `items_key`
        """
        # DynamoDB style
        if "Items" in data:
            return data["Items"]

        # EC2 Reservations style
        if "Reservations" in data:
            instances: list[dict[str, Any]] = []
            for r in data["Reservations"]:
                instances.extend(r.get("Instances", []))
            return instances

        # Generic: look for the first top-level list value
        result = data.get(items_key)
        if isinstance(result, list):
            return result

        # Last resort: first list value found
        for v in data.values():
            if isinstance(v, list):
                return v

        return []

    def search_ec2_instances_by_tag(
        self,
        ec2_response: dict[str, Any],
        tag_key: str,
        tag_value: str,
    ) -> list[dict[str, Any]]:
        """
        Extract EC2 instances from a describe-instances response where
        the instance has a tag matching tag_key=tag_value.
        """
        all_instances = self.flatten_aws_response(ec2_response)
        expr = f"[?Tags[?Key=='{tag_key}' && Value=='{tag_value}']]"
        result = jmespath.search(expr, all_instances)
        return result if result is not None else []

    def multi_select(
        self,
        data: Any,
        fields: list[str],
    ) -> list[dict[str, Any]]:
        """
        Project `fields` from each item in a list using JMESPath multi-select.
        Example: multi_select(alerts, ["id", "severity", "status"])
        => [{"id": ..., "severity": ..., "status": ...}, ...]
        """
        fields_expr = ", ".join(f"{f}: {f}" for f in fields)
        expr = f"[*].{{{fields_expr}}}"
        result = jmespath.search(expr, data)
        return result if result is not None else []

    def sort_by_field(
        self,
        data: list[dict[str, Any]],
        field: str,
        reverse: bool = False,
    ) -> list[dict[str, Any]]:
        """
        Sort a list of dicts by `field` using JMESPath sort_by.
        Falls back to Python sort if JMESPath sort_by is unavailable.
        """
        expr = f"sort_by(@, &{field})"
        try:
            result = jmespath.search(expr, data)
            if result is None:
                return data
            return list(reversed(result)) if reverse else result
        except JMESPathError:
            return sorted(data, key=lambda x: x.get(field, ""), reverse=reverse)

    # ------------------------------------------------------------------
    # Named helpers for common Kubric patterns
    # ------------------------------------------------------------------

    def critical_alerts(self, payload: dict[str, Any]) -> list[dict[str, Any]]:
        return self.extract_nested(payload, CRITICAL_ALERTS)

    def open_alerts(self, payload: dict[str, Any]) -> list[dict[str, Any]]:
        return self.extract_nested(payload, OPEN_ALERTS)

    def high_cvss_vulns(self, payload: dict[str, Any]) -> list[dict[str, Any]]:
        return self.extract_nested(payload, HIGH_CVSS_VULNS)

    def failed_patch_jobs(self, payload: dict[str, Any]) -> list[dict[str, Any]]:
        return self.extract_nested(payload, FAILED_PATCH_JOBS)

    def tenant_ids(self, payload: dict[str, Any]) -> list[str]:
        result = self.query(payload, TENANT_IDS)
        return result if result is not None else []


# ---------------------------------------------------------------------------
# Module-level convenience instance
# ---------------------------------------------------------------------------
querier = JMESPathQuerier()


# ---------------------------------------------------------------------------
# CLI demo
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    sample_payload = {
        "tenants": [
            {"id": "t1", "name": "Acme Corp",  "plan": "enterprise"},
            {"id": "t2", "name": "Globex Inc", "plan": "professional"},
        ],
        "alerts": [
            {"id": "a1", "severity": "critical", "status": "open",        "cvss": 9.8},
            {"id": "a2", "severity": "high",     "status": "in_progress", "cvss": 7.5},
            {"id": "a3", "severity": "critical", "status": "open",        "cvss": 9.1},
            {"id": "a4", "severity": "medium",   "status": "open",        "cvss": 5.4},
        ],
        "vulnerabilities": [
            {"cve_id": "CVE-2024-1234", "cvss_v3": 9.8, "epss_score": 0.92},
            {"cve_id": "CVE-2023-5678", "cvss_v3": 7.5, "epss_score": 0.11},
            {"cve_id": "CVE-2022-9999", "cvss_v3": 5.0, "epss_score": 0.03},
        ],
        "patch_jobs": [
            {"id": "pj1", "status": "failed",    "asset_id": "asset-01"},
            {"id": "pj2", "status": "completed", "asset_id": "asset-02"},
            {"id": "pj3", "status": "failed",    "asset_id": "asset-03"},
        ],
    }

    q = JMESPathQuerier()

    print("=== Tenant IDs ===")
    print(q.tenant_ids(sample_payload))

    print("\n=== Critical alerts ===")
    print(json.dumps(q.critical_alerts(sample_payload), indent=2))

    print("\n=== High CVSS vulns (>= 7.0) ===")
    print(json.dumps(q.high_cvss_vulns(sample_payload), indent=2))

    print("\n=== Failed patch jobs ===")
    print(json.dumps(q.failed_patch_jobs(sample_payload), indent=2))

    print("\n=== Alerts filtered by status='open' ===")
    print(json.dumps(q.filter_by_field(sample_payload["alerts"], "status", "open"), indent=2))

    print("\n=== Multi-select alerts (id, severity) ===")
    print(json.dumps(q.multi_select(sample_payload["alerts"], ["id", "severity"]), indent=2))

    print("\n=== Alerts sorted by cvss desc ===")
    print(json.dumps(q.sort_by_field(sample_payload["alerts"], "cvss", reverse=True), indent=2))

    # AWS flatten demo
    aws_ec2_response = {
        "Reservations": [
            {"Instances": [
                {"InstanceId": "i-001", "Tags": [{"Key": "env", "Value": "prod"}]},
                {"InstanceId": "i-002", "Tags": [{"Key": "env", "Value": "dev"}]},
            ]},
            {"Instances": [
                {"InstanceId": "i-003", "Tags": [{"Key": "env", "Value": "prod"}]},
            ]},
        ]
    }
    print("\n=== Flatten AWS EC2 response ===")
    print(json.dumps(q.flatten_aws_response(aws_ec2_response), indent=2))

    print("\n=== EC2 instances tagged env=prod ===")
    print(json.dumps(q.search_ec2_instances_by_tag(aws_ec2_response, "env", "prod"), indent=2))
