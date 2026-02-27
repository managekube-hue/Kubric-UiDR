"""
K-KAI-API-009: python-socketio real-time WebSocket server for KAI dashboard.
Namespaces: /dashboard (tenant-specific rooms), /alerts (real-time alert feed).
Integrated with FastAPI via socketio.ASGIApp.
"""

import logging
from typing import Any

import socketio
from fastapi import FastAPI

logger = logging.getLogger("kai.socketio")


# ---------------------------------------------------------------------------
# Socket.IO async server
# ---------------------------------------------------------------------------
sio = socketio.AsyncServer(
    async_mode="asgi",
    cors_allowed_origins="*",        # tighten per environment
    logger=False,
    engineio_logger=False,
)


# ---------------------------------------------------------------------------
# /dashboard namespace
# ---------------------------------------------------------------------------
@sio.event(namespace="/dashboard")
async def connect(sid: str, environ: dict, auth: dict | None = None):
    tenant_id = (auth or {}).get("tenant_id") or "unknown"
    logger.info("[/dashboard] connect sid=%s tenant=%s", sid, tenant_id)
    # Store tenant association in session
    await sio.save_session(sid, {"tenant_id": tenant_id}, namespace="/dashboard")
    await sio.emit(
        "connected",
        {"message": f"Connected to dashboard as tenant {tenant_id}"},
        room=sid,
        namespace="/dashboard",
    )


@sio.event(namespace="/dashboard")
async def disconnect(sid: str):
    session = await sio.get_session(sid, namespace="/dashboard")
    tenant_id = session.get("tenant_id", "unknown") if session else "unknown"
    logger.info("[/dashboard] disconnect sid=%s tenant=%s", sid, tenant_id)


@sio.event(namespace="/dashboard")
async def join_tenant_room(sid: str, data: dict):
    """Client calls this to subscribe to a specific tenant room."""
    tenant_id = data.get("tenant_id", "")
    if not tenant_id:
        await sio.emit(
            "error",
            {"message": "tenant_id required"},
            room=sid,
            namespace="/dashboard",
        )
        return
    room = f"tenant:{tenant_id}"
    sio.enter_room(sid, room, namespace="/dashboard")
    await sio.save_session(sid, {"tenant_id": tenant_id}, namespace="/dashboard")
    logger.info("[/dashboard] sid=%s joined room %s", sid, room)
    await sio.emit(
        "room_joined",
        {"room": room, "tenant_id": tenant_id},
        room=sid,
        namespace="/dashboard",
    )


# ---------------------------------------------------------------------------
# /alerts namespace
# ---------------------------------------------------------------------------
@sio.event(namespace="/alerts")
async def connect(sid: str, environ: dict, auth: dict | None = None):
    tenant_id = (auth or {}).get("tenant_id") or "unknown"
    logger.info("[/alerts] connect sid=%s tenant=%s", sid, tenant_id)
    await sio.save_session(sid, {"tenant_id": tenant_id}, namespace="/alerts")


@sio.event(namespace="/alerts")
async def disconnect(sid: str):
    logger.info("[/alerts] disconnect sid=%s", sid)


@sio.event(namespace="/alerts")
async def subscribe_alerts(sid: str, data: dict):
    tenant_id = data.get("tenant_id", "")
    if not tenant_id:
        return
    room = f"alerts:{tenant_id}"
    sio.enter_room(sid, room, namespace="/alerts")
    await sio.save_session(sid, {"tenant_id": tenant_id}, namespace="/alerts")
    logger.info("[/alerts] sid=%s subscribed to %s", sid, room)


# ---------------------------------------------------------------------------
# Broadcast helpers (called from KAI persona agents)
# ---------------------------------------------------------------------------
async def broadcast_alert(tenant_id: str, alert: dict) -> None:
    """Emit a real-time alert to all dashboard and alert subscribers for a tenant."""
    room = f"tenant:{tenant_id}"
    alert_room = f"alerts:{tenant_id}"

    await sio.emit("alert", alert, room=room, namespace="/dashboard")
    await sio.emit("alert", alert, room=alert_room, namespace="/alerts")

    logger.debug(
        "broadcast_alert tenant=%s severity=%s",
        tenant_id,
        alert.get("severity", "?"),
    )


async def broadcast_health(tenant_id: str, score: int) -> None:
    """Emit an updated health/risk score to the tenant dashboard room."""
    room = f"tenant:{tenant_id}"
    await sio.emit(
        "health_score",
        {"tenant_id": tenant_id, "score": score},
        room=room,
        namespace="/dashboard",
    )
    logger.debug("broadcast_health tenant=%s score=%d", tenant_id, score)


async def broadcast_incident(tenant_id: str, incident: dict) -> None:
    """Emit a new or updated incident to the tenant dashboard and alerts rooms."""
    room = f"tenant:{tenant_id}"
    alert_room = f"alerts:{tenant_id}"

    await sio.emit("incident", incident, room=room, namespace="/dashboard")
    await sio.emit("incident", incident, room=alert_room, namespace="/alerts")

    logger.debug(
        "broadcast_incident tenant=%s incident_id=%s",
        tenant_id,
        incident.get("id", "?"),
    )


# ---------------------------------------------------------------------------
# FastAPI integration
# ---------------------------------------------------------------------------
def mount_socketio(app: FastAPI, path: str = "/ws") -> socketio.ASGIApp:
    """
    Wrap FastAPI with the Socket.IO ASGI app so both HTTP and WS
    are served from the same process.

    Usage in main.py::

        from K_KAI_API_009_socketio_realtime import mount_socketio
        combined_app = mount_socketio(app)
        # Run combined_app with uvicorn instead of app
    """
    combined = socketio.ASGIApp(sio, other_asgi_app=app, socketio_path=path)
    logger.info("Socket.IO mounted at path=%s", path)
    return combined
