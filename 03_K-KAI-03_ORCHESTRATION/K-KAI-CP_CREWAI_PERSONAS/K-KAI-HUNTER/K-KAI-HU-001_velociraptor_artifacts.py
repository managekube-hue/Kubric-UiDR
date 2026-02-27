"""
K-KAI-HU-001: Velociraptor Artifact Runner
Connects to the Velociraptor HTTP API, creates collection flows, polls for
results, and returns structured findings.
"""

import logging
import os
import time
from typing import Any

import httpx

logger = logging.getLogger("K-KAI-HU-001")

VELOCIRAPTOR_URL: str = os.getenv("VELOCIRAPTOR_URL", "https://velociraptor:8001")
VELOCIRAPTOR_API_KEY: str = os.getenv("VELOCIRAPTOR_API_KEY", "")

HTTP_TIMEOUT: float = 30.0
POLL_INTERVAL: float = 5.0
MAX_POLL_ATTEMPTS: int = 60  # 60 × 5 s = 5 minutes


class VelociraptorArtifactRunner:
    """
    Wraps the Velociraptor v1 REST API for artifact collection and hunting.

    Velociraptor API endpoints used:
      POST /api/v1/CreateCollectorDownload  → create a flow
      GET  /api/v1/GetFlowDetails          → poll flow status
      GET  /api/v1/GetTable                → retrieve result rows
      POST /api/v1/VFSListDirectory        → list client endpoints
    """

    def __init__(self) -> None:
        if not VELOCIRAPTOR_API_KEY:
            logger.warning("VELOCIRAPTOR_API_KEY not set; requests will be unauthenticated.")
        self._headers: dict[str, str] = {
            "Grpc-Metadata-Api-Key": VELOCIRAPTOR_API_KEY,
            "Content-Type": "application/json",
        }

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def hunt(self, client_id: str, artifact: str, parameters: dict) -> dict:
        """
        Create an artifact collection flow on a specific client.

        Args:
            client_id:  Velociraptor client ID (e.g. "C.abc123").
            artifact:   Fully qualified artifact name (e.g.
                        "Windows.System.Processes").
            parameters: Dict of artifact parameter overrides.

        Returns:
            dict with flow_id, client_id, artifact, status, results (list).
        """
        flow_id: str = self._create_flow(client_id, artifact, parameters)
        logger.info("Created flow %s for artifact %s on client %s", flow_id, artifact, client_id)

        # Poll until the flow completes
        for attempt in range(MAX_POLL_ATTEMPTS):
            details = self._get_flow_details(client_id, flow_id)
            state: str = details.get("state", "RUNNING")

            if state in ("FINISHED",):
                break
            if state in ("ERROR", "CANCELLED"):
                logger.error("Flow %s ended with state %s", flow_id, state)
                return {
                    "flow_id": flow_id,
                    "client_id": client_id,
                    "artifact": artifact,
                    "status": state,
                    "results": [],
                }

            logger.debug("Flow %s state=%s (attempt %d/%d)", flow_id, state, attempt + 1, MAX_POLL_ATTEMPTS)
            time.sleep(POLL_INTERVAL)
        else:
            logger.warning("Flow %s timed out waiting for completion.", flow_id)
            return {
                "flow_id": flow_id,
                "client_id": client_id,
                "artifact": artifact,
                "status": "TIMEOUT",
                "results": [],
            }

        results = self.get_results(flow_id)
        return {
            "flow_id": flow_id,
            "client_id": client_id,
            "artifact": artifact,
            "status": "FINISHED",
            "results": results,
        }

    def get_results(self, flow_id: str) -> list[dict]:
        """
        Retrieve all result rows for a completed flow.

        Returns:
            List of row dicts from the Velociraptor result table.
        """
        url = f"{VELOCIRAPTOR_URL}/api/v1/GetTable"
        params: dict[str, Any] = {
            "flow_id": flow_id,
            "artifact": "",  # let server infer from flow
            "rows": 1000,
            "start_row": 0,
        }
        rows: list[dict] = []
        try:
            with httpx.Client(timeout=HTTP_TIMEOUT, verify=False) as client:  # noqa: S501
                resp = client.get(url, params=params, headers=self._headers)
                resp.raise_for_status()
                data = resp.json()
                columns: list[str] = data.get("columns", [])
                for row_values in data.get("rows", []):
                    cell_list = row_values.get("cell", [])
                    rows.append(dict(zip(columns, cell_list)))
        except Exception as exc:  # noqa: BLE001
            logger.error("Failed to retrieve results for flow %s: %s", flow_id, exc)
        return rows

    def list_clients(self, tenant_id: str) -> list[dict]:
        """
        Return all registered Velociraptor clients for a given tenant.

        The tenant_id is used as a label filter (Velociraptor labelling
        convention: kubric_tenant=<tenant_id>).
        """
        url = f"{VELOCIRAPTOR_URL}/api/v1/SearchClients"
        payload = {
            "query": f"label:kubric_tenant={tenant_id}",
            "limit": 500,
            "start": 0,
        }
        clients: list[dict] = []
        try:
            with httpx.Client(timeout=HTTP_TIMEOUT, verify=False) as c:  # noqa: S501
                resp = c.post(url, json=payload, headers=self._headers)
                resp.raise_for_status()
                data = resp.json()
                for item in data.get("items", []):
                    clients.append({
                        "client_id": item.get("client_id"),
                        "hostname": item.get("os_info", {}).get("hostname"),
                        "os": item.get("os_info", {}).get("system"),
                        "last_seen": item.get("last_seen_at"),
                        "labels": item.get("labels", []),
                    })
        except Exception as exc:  # noqa: BLE001
            logger.error("Failed to list clients for tenant %s: %s", tenant_id, exc)
        return clients

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _create_flow(self, client_id: str, artifact: str, parameters: dict) -> str:
        """POST to Velociraptor API to start a collection flow."""
        url = f"{VELOCIRAPTOR_URL}/api/v1/CollectArtifact"
        payload: dict[str, Any] = {
            "client_id": client_id,
            "artifacts": [artifact],
            "parameters": {
                "env": [{"key": k, "value": str(v)} for k, v in parameters.items()]
            },
            "urgent": False,
        }
        with httpx.Client(timeout=HTTP_TIMEOUT, verify=False) as c:  # noqa: S501
            resp = c.post(url, json=payload, headers=self._headers)
            resp.raise_for_status()
            data = resp.json()
            flow_id: str = data.get("flow_id") or data["flow_id"]
            return flow_id

    def _get_flow_details(self, client_id: str, flow_id: str) -> dict:
        """GET flow state from Velociraptor."""
        url = f"{VELOCIRAPTOR_URL}/api/v1/GetFlowDetails"
        params = {"client_id": client_id, "flow_id": flow_id}
        with httpx.Client(timeout=HTTP_TIMEOUT, verify=False) as c:  # noqa: S501
            resp = c.get(url, params=params, headers=self._headers)
            resp.raise_for_status()
            return resp.json().get("context", {})
