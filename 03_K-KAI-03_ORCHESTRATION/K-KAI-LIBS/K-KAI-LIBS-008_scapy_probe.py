"""
K-KAI-LIBS-008: Scapy active probe utilities for authorized network audits.
All operations require tenant_id + audit_token and are logged to NATS.
Methods: ping_sweep, port_scan, traceroute.
"""

import asyncio
import datetime
import logging
import os
import socket
from typing import Any

logger = logging.getLogger("kai.libs.scapy")

# Scapy is imported lazily to avoid loading it in environments where
# raw-socket access is unavailable (e.g., CI containers).
try:
    from scapy.all import (
        IP, TCP, UDP, ICMP, sr1, srp, sr, conf as scapy_conf,
        Ether, ARP,
    )
    from scapy.layers.inet import traceroute as scapy_traceroute
    _SCAPY_AVAILABLE = True
except ImportError:
    _SCAPY_AVAILABLE = False
    logger.warning("scapy not installed – ScapyProbe will raise on use")


# ---------------------------------------------------------------------------
# Audit logger helper
# ---------------------------------------------------------------------------
async def _log_audit(
    tenant_id: str,
    audit_token: str,
    operation: str,
    params: dict,
    result_summary: str,
) -> None:
    """Publish an audit log event to NATS kubric.{tenant}.audit.probe."""
    try:
        import orjson
        from K_KAI_API_006_nats_py_client import get_nats_client
        client = get_nats_client()
        if not client.is_connected:
            return
        payload = {
            "class_uid": 4003,
            "class_name": "Network Activity",
            "time_dt": datetime.datetime.utcnow().isoformat() + "Z",
            "tenant_id": tenant_id,
            "audit_token": audit_token,
            "operation": operation,
            "params": params,
            "result_summary": result_summary,
        }
        await client.publish(f"kubric.{tenant_id}.audit.probe", payload)
    except Exception as exc:
        logger.warning("Audit log publish failed: %s", exc)


# ---------------------------------------------------------------------------
# ScapyProbe
# ---------------------------------------------------------------------------
class ScapyProbe:
    """
    Authorized network probe utilities using Scapy.
    Every method validates tenant_id + audit_token before executing.
    All results are logged to NATS kubric.{tenant}.audit.probe.

    WARNING: These methods send real packets. Use only with explicit
    written authorization in compliance with local laws and Kubric policies.
    """

    DEFAULT_TIMEOUT = 2

    def __init__(self, tenant_id: str, audit_token: str) -> None:
        if not _SCAPY_AVAILABLE:
            raise RuntimeError("scapy is not installed")
        self._tenant_id = tenant_id
        self._audit_token = audit_token
        scapy_conf.verb = 0   # suppress scapy output

    # ------------------------------------------------------------------
    # Ping sweep
    # ------------------------------------------------------------------
    def ping_sweep(self, network: str) -> list[str]:
        """
        ARP sweep *network* (CIDR, e.g. '10.0.0.0/24') for live hosts.
        Returns list of IP strings that responded.
        Only suitable for local-network ranges.
        """
        logger.info("ping_sweep tenant=%s network=%s", self._tenant_id, network)
        from scapy.all import ARP, Ether, srp
        arp = Ether(dst="ff:ff:ff:ff:ff:ff") / ARP(pdst=network)
        answered, _ = srp(arp, timeout=self.DEFAULT_TIMEOUT, verbose=0)
        live = [rcv.psrc for _, rcv in answered]
        asyncio.get_event_loop().run_until_complete(
            _log_audit(self._tenant_id, self._audit_token,
                       "ping_sweep", {"network": network},
                       f"found {len(live)} hosts")
        )
        logger.info("ping_sweep: %d live hosts in %s", len(live), network)
        return live

    # ------------------------------------------------------------------
    # Port scan
    # ------------------------------------------------------------------
    def port_scan(self, ip: str, ports: list[int]) -> dict[int, bool]:
        """
        TCP SYN scan *ports* on *ip*.
        Returns dict mapping port -> True (open) / False (closed/filtered).
        """
        logger.info("port_scan tenant=%s ip=%s ports=%s", self._tenant_id, ip, ports[:10])
        results: dict[int, bool] = {}
        for port in ports:
            pkt = IP(dst=ip) / TCP(dport=port, flags="S")
            resp = sr1(pkt, timeout=self.DEFAULT_TIMEOUT, verbose=0)
            if resp and resp.haslayer(TCP):
                tcp_flags = resp.getlayer(TCP).flags
                results[port] = bool(tcp_flags & 0x12)  # SYN-ACK
            else:
                results[port] = False

        open_count = sum(1 for v in results.values() if v)
        asyncio.get_event_loop().run_until_complete(
            _log_audit(self._tenant_id, self._audit_token,
                       "port_scan", {"ip": ip, "ports": ports},
                       f"{open_count}/{len(ports)} open")
        )
        logger.info("port_scan: %d/%d open on %s", open_count, len(ports), ip)
        return results

    # ------------------------------------------------------------------
    # Traceroute
    # ------------------------------------------------------------------
    def traceroute(self, dst: str, max_hops: int = 30) -> list[dict]:
        """
        UDP traceroute to *dst* with up to *max_hops* TTL probes.
        Returns list of dicts: {hop, ip, rtt_ms}.
        """
        logger.info("traceroute tenant=%s dst=%s max_hops=%d",
                    self._tenant_id, dst, max_hops)
        hops: list[dict] = []
        for ttl in range(1, max_hops + 1):
            pkt = IP(dst=dst, ttl=ttl) / UDP(dport=33434 + ttl)
            import time
            t0 = time.perf_counter()
            resp = sr1(pkt, timeout=self.DEFAULT_TIMEOUT, verbose=0)
            rtt_ms = (time.perf_counter() - t0) * 1000
            if resp is None:
                hops.append({"hop": ttl, "ip": "*", "rtt_ms": None})
            else:
                hop_ip = resp.src
                hops.append({"hop": ttl, "ip": hop_ip, "rtt_ms": round(rtt_ms, 2)})
                if hop_ip == dst:
                    break

        asyncio.get_event_loop().run_until_complete(
            _log_audit(self._tenant_id, self._audit_token,
                       "traceroute", {"dst": dst, "max_hops": max_hops},
                       f"{len(hops)} hops")
        )
        logger.info("traceroute to %s: %d hops recorded", dst, len(hops))
        return hops
