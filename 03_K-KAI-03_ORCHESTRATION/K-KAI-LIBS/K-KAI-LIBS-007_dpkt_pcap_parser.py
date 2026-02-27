"""
K-KAI-LIBS-007: dpkt PCAP file parser for offline network forensics.
parse_pcap() extracts per-packet metadata.
PCAPAnomalyDetector.scan() flags port scans, large payloads, unusual protocols.
Returns OCSF class 4003 (Network Activity) formatted events.
"""

import datetime
import logging
import socket
import struct
from collections import defaultdict
from pathlib import Path
from typing import Any

import dpkt

logger = logging.getLogger("kai.libs.dpkt")

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def _ip_to_str(addr: bytes) -> str:
    try:
        if len(addr) == 4:
            return socket.inet_ntoa(addr)
        return socket.inet_ntop(socket.AF_INET6, addr)
    except Exception:
        return addr.hex()


def _proto_name(proto: int) -> str:
    return {1: "ICMP", 6: "TCP", 17: "UDP", 58: "ICMPv6"}.get(proto, str(proto))


# ---------------------------------------------------------------------------
# parse_pcap
# ---------------------------------------------------------------------------
def parse_pcap(path: str) -> list[dict]:
    """
    Parse a PCAP file using dpkt.
    Returns a list of dicts with per-packet metadata:
      timestamp, src_ip, dst_ip, src_port, dst_port, proto, size, payload_hex
    """
    resolved = Path(path)
    if not resolved.exists():
        raise FileNotFoundError(f"PCAP not found: {resolved}")

    packets: list[dict] = []
    with open(str(resolved), "rb") as f:
        try:
            reader = dpkt.pcap.Reader(f)
        except ValueError:
            try:
                f.seek(0)
                reader = dpkt.pcapng.Reader(f)
            except Exception as exc:
                raise ValueError(f"Cannot read PCAP/PCAPng: {exc}") from exc

        for ts, raw in reader:
            try:
                eth = dpkt.ethernet.Ethernet(raw)
            except Exception:
                continue

            ip = None
            if isinstance(eth.data, dpkt.ip.IP):
                ip = eth.data
            elif isinstance(eth.data, dpkt.ip6.IP6):
                ip = eth.data
            else:
                continue

            src_ip = _ip_to_str(ip.src)
            dst_ip = _ip_to_str(ip.dst)
            proto = ip.p
            proto_name = _proto_name(proto)
            size = len(raw)
            payload_hex = ""
            src_port = dst_port = 0

            transport = ip.data
            if isinstance(transport, (dpkt.tcp.TCP, dpkt.udp.UDP)):
                src_port = transport.sport
                dst_port = transport.dport
                payload_hex = bytes(transport.data)[:64].hex()
            elif isinstance(transport, dpkt.icmp.ICMP):
                payload_hex = bytes(ip.data)[:32].hex()

            packets.append({
                "timestamp": datetime.datetime.utcfromtimestamp(ts).isoformat() + "Z",
                "src_ip": src_ip,
                "dst_ip": dst_ip,
                "src_port": src_port,
                "dst_port": dst_port,
                "proto": proto_name,
                "size": size,
                "payload_hex": payload_hex,
            })

    logger.info("parse_pcap: %d packets from %s", len(packets), path)
    return packets


# ---------------------------------------------------------------------------
# PCAPAnomalyDetector
# ---------------------------------------------------------------------------
class PCAPAnomalyDetector:
    """
    Scans a PCAP file for anomalies and returns OCSF class 4003 events.
    Detection rules:
      - Port scan: >20 unique dst_ports from same src in 60 s window
      - Large payload: packet size > 65000 bytes
      - Unusual protocol: not TCP (6), UDP (17), or ICMP (1)
    """

    PORT_SCAN_THRESHOLD = 20
    LARGE_PAYLOAD_BYTES = 65000
    ALLOWED_PROTOS = {"TCP", "UDP", "ICMP", "ICMPv6"}

    def scan(self, pcap_path: str) -> list[dict]:
        """Return a list of OCSF class 4003 anomaly events."""
        packets = parse_pcap(pcap_path)
        findings: list[dict] = []

        # Index: src_ip -> list of (ts_epoch, dst_port)
        src_ports: dict[str, list[tuple[float, int]]] = defaultdict(list)
        # src_ip -> set of dst_ports in current 60s window
        src_window_ports: dict[str, set] = defaultdict(set)

        for pkt in packets:
            ts_str = pkt["timestamp"]
            try:
                ts = datetime.datetime.fromisoformat(ts_str.rstrip("Z")).timestamp()
            except Exception:
                ts = 0.0

            # Large payload check
            if pkt["size"] > self.LARGE_PAYLOAD_BYTES:
                findings.append(self._ocsf_event(pkt, "large_payload",
                    f"Packet size {pkt['size']} exceeds {self.LARGE_PAYLOAD_BYTES} bytes", 4))

            # Unusual protocol check
            if pkt["proto"] not in self.ALLOWED_PROTOS:
                findings.append(self._ocsf_event(pkt, "unusual_protocol",
                    f"Unusual protocol {pkt['proto']}", 3))

            # Port scan tracking (60 s sliding window)
            src_ip = pkt["src_ip"]
            dst_port = pkt["dst_port"]
            if dst_port > 0:
                src_ports[src_ip].append((ts, dst_port))
                # Prune entries older than 60 s
                src_ports[src_ip] = [
                    (t, p) for (t, p) in src_ports[src_ip] if ts - t <= 60
                ]
                unique_ports = {p for (_, p) in src_ports[src_ip]}
                if len(unique_ports) > self.PORT_SCAN_THRESHOLD:
                    if src_ip not in src_window_ports or unique_ports != src_window_ports[src_ip]:
                        src_window_ports[src_ip] = unique_ports
                        findings.append(self._ocsf_event(pkt, "port_scan",
                            f"Source {src_ip} scanned {len(unique_ports)} unique ports in 60s", 5))

        logger.info("PCAPAnomalyDetector.scan: %d anomalies in %s", len(findings), pcap_path)
        return findings

    # ------------------------------------------------------------------
    def _ocsf_event(
        self,
        pkt: dict,
        anomaly_type: str,
        message: str,
        severity_id: int,
    ) -> dict:
        return {
            "class_uid": 4003,
            "class_name": "Network Activity",
            "time_dt": pkt["timestamp"],
            "src_ip": pkt["src_ip"],
            "dst_ip": pkt["dst_ip"],
            "src_port": pkt["src_port"],
            "dst_port": pkt["dst_port"],
            "proto": pkt["proto"],
            "size": pkt["size"],
            "anomaly_type": anomaly_type,
            "message": message,
            "severity_id": severity_id,
            "severity": {1: "Informational", 2: "Low", 3: "Medium", 4: "High", 5: "Critical"}.get(severity_id, "Unknown"),
        }
