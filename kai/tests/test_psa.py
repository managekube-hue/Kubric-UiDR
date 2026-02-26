"""
PSA integration tests — verifies Zammad ticket idempotency.

Uses a mock HTTP server to simulate the Zammad REST API.
"""

from __future__ import annotations

import asyncio
import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread

import pytest

from kai.psa.zammad import ZammadClient, clear_event_cache


_ticket_counter = 0
_created_tickets: list[dict] = []


class MockZammadHandler(BaseHTTPRequestHandler):
    """Minimal Zammad API mock."""

    def do_POST(self) -> None:
        global _ticket_counter
        if self.path == "/api/v1/tickets":
            length = int(self.headers.get("Content-Length", 0))
            body = json.loads(self.rfile.read(length)) if length else {}
            _ticket_counter += 1
            _created_tickets.append(body)
            response = {"id": _ticket_counter, "title": body.get("title", "")}
            self.send_response(201)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()

    def do_PUT(self) -> None:
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(b'{"ok": true}')

    def log_message(self, *_args: object) -> None:
        pass  # suppress logs during tests


@pytest.fixture(scope="module")
def mock_server():
    """Start a mock Zammad HTTP server for the test module."""
    global _ticket_counter, _created_tickets
    _ticket_counter = 0
    _created_tickets = []

    server = HTTPServer(("127.0.0.1", 0), MockZammadHandler)
    port = server.server_address[1]
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    yield f"http://127.0.0.1:{port}"
    server.shutdown()


@pytest.fixture(autouse=True)
def _clear_cache():
    """Clear event dedup cache before each test."""
    clear_event_cache()
    yield
    clear_event_cache()


def test_create_ticket(mock_server: str) -> None:
    """Verify a ticket can be created via Zammad API."""
    client = ZammadClient(url=mock_server, api_token="test-token")
    ticket_id = asyncio.get_event_loop().run_until_complete(
        client.create_ticket(
            tenant_id="tenant-1",
            title="Critical alert detected",
            body="A critical vulnerability was found.",
            priority="critical",
        )
    )
    assert ticket_id > 0


def test_idempotency(mock_server: str) -> None:
    """Calling create_ticket twice with the same event_id creates only 1 ticket."""
    global _ticket_counter
    initial_count = _ticket_counter

    client = ZammadClient(url=mock_server, api_token="test-token")

    # First call — creates a ticket
    ticket_id_1 = asyncio.get_event_loop().run_until_complete(
        client.create_ticket(
            tenant_id="tenant-2",
            title="Agent offline",
            body="Agent coresec-01 not reporting.",
            priority="high",
            event_id="evt-12345",
        )
    )
    assert ticket_id_1 > 0
    assert _ticket_counter == initial_count + 1

    # Second call with SAME event_id — should NOT create a new ticket
    ticket_id_2 = asyncio.get_event_loop().run_until_complete(
        client.create_ticket(
            tenant_id="tenant-2",
            title="Agent offline",
            body="Agent coresec-01 not reporting.",
            priority="high",
            event_id="evt-12345",
        )
    )
    assert ticket_id_2 == ticket_id_1
    assert _ticket_counter == initial_count + 1  # count unchanged


def test_different_events_create_separate_tickets(mock_server: str) -> None:
    """Different event_ids must create separate tickets."""
    global _ticket_counter
    initial_count = _ticket_counter

    client = ZammadClient(url=mock_server, api_token="test-token")

    tid1 = asyncio.get_event_loop().run_until_complete(
        client.create_ticket(
            tenant_id="tenant-3",
            title="Alert 1",
            body="First alert",
            event_id="evt-aaa",
        )
    )
    tid2 = asyncio.get_event_loop().run_until_complete(
        client.create_ticket(
            tenant_id="tenant-3",
            title="Alert 2",
            body="Second alert",
            event_id="evt-bbb",
        )
    )
    assert tid1 != tid2
    assert _ticket_counter == initial_count + 2


def test_update_ticket(mock_server: str) -> None:
    """Verify ticket status can be updated."""
    client = ZammadClient(url=mock_server, api_token="test-token")
    result = asyncio.get_event_loop().run_until_complete(
        client.update_ticket(ticket_id=1, status="closed")
    )
    assert result is True
