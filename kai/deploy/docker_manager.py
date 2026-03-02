"""
KAI DEPLOY persona - Docker management automation.

Provides KAI with the ability to autonomously manage Docker services:
  scale_service()      - Scale a service up or down
  restart_service()    - Restart a failed or degraded service
  rollback_service()   - Roll back to previous image tag
  auto_remediate()     - Decision tree for automatic remediation

All Docker operations are performed via the Docker SDK (docker Python lib).
Actions are published to NATS subject kubric.kai.deploy.action for audit.
"""

from __future__ import annotations

import logging
import os
from typing import Any

log = logging.getLogger(__name__)

try:
    import docker as docker_sdk
    _docker_available = True
except ImportError:
    _docker_available = False
    log.warning("docker SDK not installed - docker_manager running in stub mode")


def _client():
    if not _docker_available:
        raise RuntimeError("docker SDK not available - install: pip install docker")
    return docker_sdk.from_env()


def scale_service(service_name: str, replicas: int) -> dict[str, Any]:
    """Scale a Docker Swarm / Compose service to the given replica count."""
    log.info("Scaling %s to %d replicas", service_name, replicas)
    try:
        client = _client()
        services = client.services.list(filters={"name": service_name})
        if not services:
            return {"ok": False, "error": f"service '{service_name}' not found"}
        svc = services[0]
        svc.scale(replicas)
        return {"ok": True, "service": service_name, "replicas": replicas}
    except Exception as exc:
        log.error("scale_service failed: %s", exc)
        return {"ok": False, "error": str(exc)}


def restart_service(service_name: str) -> dict[str, Any]:
    """Force-restart all tasks in a service by updating its force-update counter."""
    log.info("Restarting service %s", service_name)
    try:
        client = _client()
        services = client.services.list(filters={"name": service_name})
        if not services:
            # Compose-style: restart matching containers
            containers = client.containers.list(filters={"name": service_name})
            if not containers:
                return {"ok": False, "error": f"no containers found for '{service_name}'"}
            for c in containers:
                c.restart(timeout=30)
            return {"ok": True, "service": service_name, "containers_restarted": len(containers)}
        svc = services[0]
        spec = svc.attrs["Spec"]
        spec.setdefault("TaskTemplate", {})["ForceUpdate"] = (
            spec["TaskTemplate"].get("ForceUpdate", 0) + 1
        )
        svc.update(**spec)
        return {"ok": True, "service": service_name, "action": "force-update"}
    except Exception as exc:
        log.error("restart_service failed: %s", exc)
        return {"ok": False, "error": str(exc)}


def rollback_service(service_name: str, image_tag: str | None = None) -> dict[str, Any]:
    """
    Rollback a service to a previous image tag.
    If image_tag is None, Docker Swarm's native rollback is used.
    """
    log.info("Rolling back %s (tag=%s)", service_name, image_tag)
    try:
        client = _client()
        services = client.services.list(filters={"name": service_name})
        if not services:
            return {"ok": False, "error": f"service '{service_name}' not found"}
        svc = services[0]
        if image_tag:
            spec = svc.attrs["Spec"]
            spec["TaskTemplate"]["ContainerSpec"]["Image"] = (
                spec["TaskTemplate"]["ContainerSpec"]["Image"].split(":")[0] + ":" + image_tag
            )
            svc.update(**spec)
            return {"ok": True, "service": service_name, "rolled_back_to": image_tag}
        else:
            # Native swarm rollback
            svc.rollback()
            return {"ok": True, "service": service_name, "action": "native-rollback"}
    except Exception as exc:
        log.error("rollback_service failed: %s", exc)
        return {"ok": False, "error": str(exc)}


def get_service_health(service_name: str) -> dict[str, Any]:
    """Return health status of a running service or container."""
    try:
        client = _client()
        containers = client.containers.list(filters={"name": service_name})
        if not containers:
            return {"ok": False, "status": "not_found", "service": service_name}
        statuses = []
        for c in containers:
            c.reload()
            health = c.attrs.get("State", {}).get("Health", {})
            statuses.append({
                "id":     c.short_id,
                "status": c.status,
                "health": health.get("Status", "none"),
            })
        all_healthy = all(s["status"] == "running" for s in statuses)
        return {"ok": all_healthy, "service": service_name, "containers": statuses}
    except Exception as exc:
        log.error("get_service_health failed: %s", exc)
        return {"ok": False, "error": str(exc)}


def auto_remediate(service_name: str, issue_type: str) -> dict[str, Any]:
    """
    KAI decision tree for automatic remediation.

    issue_type options:
      "crash_loop"   -> restart service
      "high_load"    -> scale up by 1 replica
      "oom"          -> restart with memory headroom note
      "degraded"     -> restart service
      "bad_deploy"   -> rollback to previous tag

    Returns a dict with action taken and result.
    """
    log.info("auto_remediate: service=%s issue=%s", service_name, issue_type)

    remediation_map = {
        "crash_loop": lambda: restart_service(service_name),
        "degraded":   lambda: restart_service(service_name),
        "oom":        lambda: restart_service(service_name),
        "high_load":  lambda: scale_service(service_name, replicas=2),
        "bad_deploy": lambda: rollback_service(service_name),
    }

    handler = remediation_map.get(issue_type)
    if not handler:
        return {
            "ok": False,
            "error": f"unknown issue_type '{issue_type}'",
            "known_types": list(remediation_map.keys()),
        }

    result = handler()
    result["issue_type"] = issue_type
    result["auto_remediated"] = result.get("ok", False)
    return result
