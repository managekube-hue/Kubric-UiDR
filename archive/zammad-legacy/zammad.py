"""
Zammad PSA integration — creates and updates tickets via REST API.

Credentials read from KUBRIC_ZAMMAD_URL and KUBRIC_ZAMMAD_TOKEN env vars (injected by Vault in production).
Idempotent: checks event_id → ticket_id mapping before creating duplicates.
Uses in-memory cache (Redis optional) for event dedup.
"""

from __future__ import annotations

from typing import Any

import httpx
import structlog

from kai.config import settings

log = structlog.get_logger(__name__)

# In-memory event dedup — maps event_id → ticket_id
_event_ticket_map: dict[str, int] = {}


class ZammadClient:
    """HTTP client for Zammad REST API."""

    def __init__(
        self,
        url: str | None = None,
        api_token: str | None = None,
    ) -> None:
        # Prefer explicit args, then settings (Vault-injected), then empty
        self.url = (url or settings.zammad_url).rstrip("/")
        self.api_token = api_token or settings.zammad_token
        self._headers = {
            "Authorization": f"Token token={self.api_token}",
            "Content-Type": "application/json",
        }

    def __repr__(self) -> str:
        return f"ZammadClient(url={self.url!r})"

    async def create_ticket(
        self,
        tenant_id: str,
        title: str,
        body: str,
        priority: str = "2 normal",
        event_id: str | None = None,
    ) -> int:
        """Create a Zammad ticket. Returns the ticket ID.

        Idempotent: if event_id was already used, returns the
        existing ticket ID without creating a duplicate.
        """
        # Dedup check
        if event_id and event_id in _event_ticket_map:
            existing = _event_ticket_map[event_id]
            log.info(
                "zammad.ticket_exists",
                event_id=event_id,
                ticket_id=existing,
            )
            return existing

        priority_id = _priority_to_id(priority)

        payload: dict[str, Any] = {
            "title": title,
            "group": "Kubric Security",
            "customer_id": "guess:support@kubric.security",
            "article": {
                "subject": title,
                "body": body,
                "type": "note",
                "internal": False,
            },
            "priority_id": priority_id,
            "tags": f"kubric,tenant:{tenant_id}",
        }

        async with httpx.AsyncClient(timeout=15.0) as client:
            resp = await client.post(
                f"{self.url}/api/v1/tickets",
                headers=self._headers,
                json=payload,
            )
            resp.raise_for_status()
            data = resp.json()
            ticket_id: int = data["id"]

        # Cache for idempotency
        if event_id:
            _event_ticket_map[event_id] = ticket_id

        log.info(
            "zammad.ticket_created",
            ticket_id=ticket_id,
            tenant_id=tenant_id,
            title=title,
        )
        return ticket_id

    async def update_ticket(self, ticket_id: int, status: str) -> bool:
        """Update a Zammad ticket status. Returns True on success."""
        state_id = _status_to_state_id(status)

        async with httpx.AsyncClient(timeout=15.0) as client:
            resp = await client.put(
                f"{self.url}/api/v1/tickets/{ticket_id}",
                headers=self._headers,
                json={"state_id": state_id},
            )
            resp.raise_for_status()

        log.info("zammad.ticket_updated", ticket_id=ticket_id, status=status)
        return True


def _priority_to_id(priority: str) -> int:
    """Map priority string to Zammad priority_id."""
    mapping = {
        "1 low": 1,
        "2 normal": 2,
        "3 high": 3,
        "critical": 3,
        "high": 3,
        "medium": 2,
        "low": 1,
    }
    return mapping.get(priority.lower(), 2)


def _status_to_state_id(status: str) -> int:
    """Map status string to Zammad state_id."""
    mapping = {
        "open": 1,
        "pending": 3,
        "closed": 4,
        "resolved": 4,
    }
    return mapping.get(status.lower(), 1)


def clear_event_cache() -> None:
    """Clear the in-memory event dedup cache. Used in tests."""
    _event_ticket_map.clear()
