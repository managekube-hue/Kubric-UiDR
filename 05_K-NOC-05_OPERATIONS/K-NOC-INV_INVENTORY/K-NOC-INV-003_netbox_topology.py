"""
K-NOC-INV-003 — NetBox Network Topology Sync.
Syncs network topology from NetBox and builds an asset graph using networkx.
"""

from __future__ import annotations

import asyncio
import os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any

import asyncpg
import networkx as nx
import pynetbox
import structlog

logger = structlog.get_logger(__name__)

NETBOX_URL = os.getenv("NETBOX_URL", "http://netbox.local")
NETBOX_TOKEN = os.getenv("NETBOX_TOKEN", "")
DB_DSN = os.getenv("DATABASE_URL", "postgresql://kubric:kubric@localhost/kubric")
NETBOX_SYNC_INTERVAL = int(os.getenv("NETBOX_SYNC_INTERVAL", "3600"))


@dataclass
class Device:
    id: int
    name: str
    device_type: str
    rack: str
    site: str
    primary_ip: str
    status: str
    tags: list[str] = field(default_factory=list)


@dataclass
class Interface:
    id: int
    device_id: int
    device_name: str
    name: str
    mac_address: str
    enabled: bool


@dataclass
class IPAddress:
    id: int
    address: str
    interface_id: int | None
    status: str


class NetBoxTopology:
    """Syncs NetBox topology and builds a network graph for path queries."""

    def __init__(self) -> None:
        self.nb = pynetbox.api(NETBOX_URL, token=NETBOX_TOKEN)
        self._graph: nx.MultiDiGraph = nx.MultiDiGraph()

    def sync_devices(self) -> list[Device]:
        """Fetch all devices from NetBox DCIM."""
        devices = []
        for d in self.nb.dcim.devices.all():
            devices.append(Device(
                id=d.id,
                name=str(d.name),
                device_type=str(d.device_type) if d.device_type else "",
                rack=str(d.rack) if d.rack else "",
                site=str(d.site) if d.site else "",
                primary_ip=str(d.primary_ip) if d.primary_ip else "",
                status=str(d.status) if d.status else "",
                tags=[str(t) for t in (d.tags or [])],
            ))
        logger.info("synced devices", count=len(devices))
        return devices

    def sync_interfaces(self) -> list[Interface]:
        """Fetch all DCIM interfaces from NetBox."""
        interfaces = []
        for iface in self.nb.dcim.interfaces.all():
            interfaces.append(Interface(
                id=iface.id,
                device_id=iface.device.id if iface.device else 0,
                device_name=str(iface.device) if iface.device else "",
                name=str(iface.name),
                mac_address=str(iface.mac_address) if iface.mac_address else "",
                enabled=bool(iface.enabled),
            ))
        logger.info("synced interfaces", count=len(interfaces))
        return interfaces

    def sync_ip_addresses(self) -> list[IPAddress]:
        """Fetch all IPAM IP addresses from NetBox."""
        addresses = []
        for ip in self.nb.ipam.ip_addresses.all():
            iface_id = None
            if ip.assigned_object_id:
                iface_id = ip.assigned_object_id
            addresses.append(IPAddress(
                id=ip.id,
                address=str(ip.address),
                interface_id=iface_id,
                status=str(ip.status) if ip.status else "",
            ))
        logger.info("synced ip_addresses", count=len(addresses))
        return addresses

    def build_topology_graph(self) -> nx.MultiDiGraph:
        """Build a directed multigraph connecting devices via their interfaces/cables."""
        devices = self.sync_devices()
        self._graph = nx.MultiDiGraph()

        for dev in devices:
            self._graph.add_node(dev.name, **{
                "id": dev.id,
                "site": dev.site,
                "primary_ip": dev.primary_ip,
                "status": dev.status,
            })

        # Build edges from cable connections.
        for cable in self.nb.dcim.cables.all():
            try:
                a = cable.a_terminations
                b = cable.b_terminations
                if not a or not b:
                    continue
                dev_a = str(a[0].object.device) if hasattr(a[0].object, "device") else None
                dev_b = str(b[0].object.device) if hasattr(b[0].object, "device") else None
                if dev_a and dev_b and dev_a in self._graph and dev_b in self._graph:
                    self._graph.add_edge(dev_a, dev_b, cable_id=cable.id)
                    self._graph.add_edge(dev_b, dev_a, cable_id=cable.id)
            except Exception as exc:  # noqa: BLE001
                logger.warning("skip cable edge", cable_id=cable.id, error=str(exc))

        logger.info("topology graph built", nodes=self._graph.number_of_nodes(),
                    edges=self._graph.number_of_edges())
        return self._graph

    async def save_to_db(self, db_pool: asyncpg.Pool) -> None:
        """Upsert devices and links into the database."""
        devices = self.sync_devices()
        async with db_pool.acquire() as conn:
            for dev in devices:
                await conn.execute(
                    """
                    INSERT INTO netbox_devices
                        (netbox_id, name, device_type, rack, site, primary_ip, status, tags, synced_at)
                    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
                    ON CONFLICT (netbox_id) DO UPDATE
                        SET name=EXCLUDED.name,
                            device_type=EXCLUDED.device_type,
                            primary_ip=EXCLUDED.primary_ip,
                            status=EXCLUDED.status,
                            synced_at=EXCLUDED.synced_at
                    """,
                    dev.id, dev.name, dev.device_type, dev.rack,
                    dev.site, dev.primary_ip, dev.status,
                    dev.tags, datetime.now(timezone.utc),
                )

            for u, v, data in self._graph.edges(data=True):
                await conn.execute(
                    """
                    INSERT INTO netbox_links (device_a, device_b, cable_id, synced_at)
                    VALUES ($1,$2,$3,$4)
                    ON CONFLICT DO NOTHING
                    """,
                    u, v, data.get("cable_id"), datetime.now(timezone.utc),
                )
        logger.info("saved topology to db")

    def find_path(self, src_device: str, dst_device: str) -> list[str]:
        """Return the shortest hop list between two devices in the topology graph."""
        try:
            path: list[Any] = nx.shortest_path(self._graph, source=src_device, target=dst_device)
            return path
        except (nx.NetworkXNoPath, nx.NodeNotFound) as exc:
            logger.warning("no path found", src=src_device, dst=dst_device, error=str(exc))
            return []


async def run_sync_loop() -> None:
    """Continuously sync NetBox topology at the configured interval."""
    db_pool = await asyncpg.create_pool(DB_DSN, min_size=1, max_size=5)
    topology = NetBoxTopology()
    while True:
        try:
            topology.build_topology_graph()
            await topology.save_to_db(db_pool)
        except Exception as exc:  # noqa: BLE001
            logger.error("netbox sync error", error=str(exc))
        await asyncio.sleep(NETBOX_SYNC_INTERVAL)


if __name__ == "__main__":
    asyncio.run(run_sync_loop())
