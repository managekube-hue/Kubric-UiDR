"""
K-KAI-LIBS-009: Live PCAP capture using Scapy / pypcap.
LiveCapture publishes OCSF class 4003 events to NATS and dumps
a PCAP ring buffer on graceful stop.
"""

import asyncio
import datetime
import logging
import os
import threading
import time
from collections import deque
from pathlib import Path
from typing import Any

logger = logging.getLogger("kai.libs.pcap_capture")

try:
    from scapy.all import sniff, IP, TCP, UDP, ICMP, wrpcap, conf as scapy_conf
    _SCAPY_AVAILABLE = True
except ImportError:
    _SCAPY_AVAILABLE = False
    logger.warning("scapy not installed – LiveCapture will raise on use")

_PCAP_OUTPUT_DIR = os.environ.get("PCAP_OUTPUT_DIR", "/tmp/kubric_pcap")
_RING_BUFFER_SIZE = 10_000


def _ip_to_str(pkt) -> tuple[str, str]:
    """Return (src_ip, dst_ip) from a scapy packet."""
    if IP in pkt:
        return pkt[IP].src, pkt[IP].dst
    return "0.0.0.0", "0.0.0.0"


def _ports(pkt) -> tuple[int, int]:
    """Return (src_port, dst_port) from a scapy packet."""
    if TCP in pkt:
        return pkt[TCP].sport, pkt[TCP].dport
    if UDP in pkt:
        return pkt[UDP].sport, pkt[UDP].dport
    return 0, 0


def _proto(pkt) -> str:
    if TCP in pkt:
        return "TCP"
    if UDP in pkt:
        return "UDP"
    if ICMP in pkt:
        return "ICMP"
    if IP in pkt:
        return str(pkt[IP].proto)
    return "UNKNOWN"


def _pkt_to_ocsf(pkt, tenant_id: str) -> dict:
    """Convert a scapy packet to an OCSF class 4003 dict."""
    src_ip, dst_ip = _ip_to_str(pkt)
    src_port, dst_port = _ports(pkt)
    return {
        "class_uid": 4003,
        "class_name": "Network Activity",
        "time_dt": datetime.datetime.utcnow().isoformat() + "Z",
        "tenant_id": tenant_id,
        "src_ip": src_ip,
        "dst_ip": dst_ip,
        "src_port": src_port,
        "dst_port": dst_port,
        "proto": _proto(pkt),
        "size": len(pkt),
        "severity_id": 1,
        "severity": "Informational",
    }


# ---------------------------------------------------------------------------
# LiveCapture
# ---------------------------------------------------------------------------
class LiveCapture:
    """
    Live packet capture using Scapy.
    Captures packets asynchronously in a background thread.
    Publishes each packet as OCSF class 4003 to NATS.
    Maintains a ring buffer of up to 10,000 raw scapy packets.
    On stop(), dumps the ring buffer to a PCAP file.
    """

    def __init__(self, tenant_id: str) -> None:
        if not _SCAPY_AVAILABLE:
            raise RuntimeError("scapy is not installed")
        self._tenant_id = tenant_id
        self._ring: deque = deque(maxlen=_RING_BUFFER_SIZE)
        self._running = False
        self._thread: threading.Thread | None = None
        self._stats: dict[str, int] = {"captured": 0, "published": 0, "errors": 0}
        self._interface: str = ""
        self._bpf_filter: str = ""
        scapy_conf.verb = 0

    # ------------------------------------------------------------------
    def start(
        self,
        interface: str,
        bpf_filter: str = "port not 22",
    ) -> None:
        """
        Start live capture on *interface* with optional BPF *filter*.
        Runs in a background daemon thread so it does not block the caller.
        """
        if self._running:
            logger.warning("LiveCapture already running on %s", self._interface)
            return
        self._interface = interface
        self._bpf_filter = bpf_filter
        self._running = True
        self._thread = threading.Thread(
            target=self._capture_thread,
            daemon=True,
            name=f"kai-pcap-{interface}",
        )
        self._thread.start()
        logger.info(
            "LiveCapture started interface=%s filter=%r tenant=%s",
            interface,
            bpf_filter,
            self._tenant_id,
        )

    def stop(self) -> None:
        """
        Stop capture, dump ring buffer to PCAP file, and clean up.
        """
        if not self._running:
            return
        self._running = False
        # Scapy sniff doesn't have a direct stop; we use a short timeout loop
        if self._thread:
            self._thread.join(timeout=5)

        self._dump_pcap()
        logger.info(
            "LiveCapture stopped. stats=%s", self._stats
        )

    # ------------------------------------------------------------------
    def get_stats(self) -> dict:
        return {
            "interface": self._interface,
            "tenant_id": self._tenant_id,
            "running": self._running,
            "ring_buffer_size": len(self._ring),
            **self._stats,
        }

    # ------------------------------------------------------------------
    # Internal
    # ------------------------------------------------------------------
    def _capture_thread(self) -> None:
        """Background thread: sniffs packets and routes each to _on_packet."""
        sniff(
            iface=self._interface,
            filter=self._bpf_filter,
            prn=self._on_packet,
            store=False,
            stop_filter=lambda _: not self._running,
        )

    def _on_packet(self, pkt) -> None:
        """Called by scapy for each captured packet."""
        self._ring.append(pkt)
        self._stats["captured"] += 1

        ocsf_event = _pkt_to_ocsf(pkt, self._tenant_id)

        # Publish to NATS (fire-and-forget from sync thread via asyncio)
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                asyncio.run_coroutine_threadsafe(
                    self._publish(ocsf_event), loop
                )
        except Exception as exc:
            self._stats["errors"] += 1
            logger.debug("LiveCapture publish error: %s", exc)

    async def _publish(self, event: dict) -> None:
        """Publish OCSF event to NATS."""
        try:
            from K_KAI_API_006_nats_py_client import get_nats_client
            client = get_nats_client()
            if client.is_connected:
                subject = f"kubric.{self._tenant_id}.network.flow"
                await client.publish(subject, event)
                self._stats["published"] += 1
        except Exception as exc:
            self._stats["errors"] += 1
            logger.debug("LiveCapture NATS publish failed: %s", exc)

    def _dump_pcap(self) -> None:
        """Write ring buffer contents to a PCAP file."""
        packets = list(self._ring)
        if not packets:
            return
        out_dir = Path(_PCAP_OUTPUT_DIR)
        out_dir.mkdir(parents=True, exist_ok=True)
        ts = datetime.datetime.utcnow().strftime("%Y%m%dT%H%M%SZ")
        filename = out_dir / f"{self._tenant_id}_{self._interface}_{ts}.pcap"
        try:
            wrpcap(str(filename), packets)
            logger.info(
                "LiveCapture: wrote %d packets to %s", len(packets), filename
            )
        except Exception as exc:
            logger.error("LiveCapture PCAP dump failed: %s", exc)
