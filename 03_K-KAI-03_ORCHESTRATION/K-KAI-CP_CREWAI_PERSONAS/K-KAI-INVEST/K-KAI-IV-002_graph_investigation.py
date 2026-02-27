"""
K-KAI Investigator: Graph Investigation
Builds a bi-directional entity relationship graph from OCSF events
(IPs, domains, hashes, users, processes) and extracts lateral movement
paths, pivot chains, and centrality scores for investigation triage.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Tuple
import networkx as nx
import asyncpg

logger = logging.getLogger(__name__)

PG_DSN = os.getenv("PG_DSN", "postgresql://kubric:kubric@localhost:5432/kubric")

NODE_TYPES = {"ip", "domain", "hash", "user", "process", "host"}

@dataclass
class EntityNode:
    node_id:   str
    node_type: str
    label:     str
    attrs:     Dict[str, Any] = field(default_factory=dict)

@dataclass
class EntityEdge:
    src:        str
    dst:        str
    relation:   str          # e.g. "connected_to", "ran_as", "downloaded", "resolved"
    weight:     float = 1.0
    timestamp:  Optional[str] = None
    evidence:   List[str] = field(default_factory=list)

class ThreatGraph:
    def __init__(self) -> None:
        self._g: nx.MultiDiGraph = nx.MultiDiGraph()

    # ── graph building ────────────────────────────────────────────
    def add_node(self, node: EntityNode) -> None:
        self._g.add_node(node.node_id, type=node.node_type,
                         label=node.label, **node.attrs)

    def add_edge(self, edge: EntityEdge) -> None:
        self._g.add_edge(edge.src, edge.dst,
                         relation=edge.relation, weight=edge.weight,
                         timestamp=edge.timestamp, evidence=edge.evidence)

    def from_events(self, events: List[Dict[str, Any]]) -> None:
        """Ingest a list of OCSF-style dicts and build the graph."""
        for ev in events:
            src_ip   = ev.get("src_ip")
            dst_ip   = ev.get("dst_ip")
            domain   = ev.get("domain")
            user     = ev.get("user", {}).get("name") if isinstance(ev.get("user"), dict) else ev.get("user")
            process  = ev.get("process", {}).get("name") if isinstance(ev.get("process"), dict) else ev.get("process")
            file_hash = ev.get("file", {}).get("hash") if isinstance(ev.get("file"), dict) else ev.get("file_hash")

            ts = ev.get("time") or datetime.now(timezone.utc).isoformat()

            for ip in filter(None, [src_ip, dst_ip]):
                self.add_node(EntityNode(ip, "ip", ip))

            if domain:
                self.add_node(EntityNode(domain, "domain", domain))

            if user:
                self.add_node(EntityNode(user, "user", user))

            if process:
                self.add_node(EntityNode(process, "process", process))

            if file_hash:
                self.add_node(EntityNode(file_hash, "hash", file_hash[:16]))

            # Build edges
            if src_ip and dst_ip:
                self.add_edge(EntityEdge(src_ip, dst_ip, "connected_to", timestamp=ts))
            if src_ip and domain:
                self.add_edge(EntityEdge(src_ip, domain, "resolved", timestamp=ts))
            if user and process:
                self.add_edge(EntityEdge(user, process, "ran_as", timestamp=ts))
            if process and file_hash:
                self.add_edge(EntityEdge(process, file_hash, "loaded", timestamp=ts))

    # ── analysis ──────────────────────────────────────────────────
    def top_centrality(self, n: int = 10) -> List[Tuple[str, float]]:
        try:
            scores = nx.betweenness_centrality(self._g.to_undirected(), normalized=True)
        except Exception:
            scores = nx.degree_centrality(self._g)
        return sorted(scores.items(), key=lambda x: x[1], reverse=True)[:n]

    def lateral_movement_paths(self, src: str, max_hops: int = 5) -> List[List[str]]:
        """Find all simple paths from src that cross >1 host/IP node."""
        paths: List[List[str]] = []
        targets = [n for n, d in self._g.nodes(data=True)
                   if d.get("type") in ("ip", "host") and n != src]
        for tgt in targets[:50]:  # cap to avoid combinatorial explosion
            try:
                for path in nx.all_simple_paths(self._g, src, tgt, cutoff=max_hops):
                    if len(path) >= 3:
                        paths.append(path)
            except (nx.NetworkXNoPath, nx.NodeNotFound):
                continue
        return paths[:20]

    def get_pivot_chains(self) -> List[Dict[str, Any]]:
        """Return IP nodes that act as relay nodes (in-degree >= 2, out-degree >= 2)."""
        pivots = []
        for node, data in self._g.nodes(data=True):
            if data.get("type") != "ip":
                continue
            in_d  = self._g.in_degree(node)
            out_d = self._g.out_degree(node)
            if in_d >= 2 and out_d >= 2:
                pivots.append({
                    "node":       node,
                    "in_degree":  in_d,
                    "out_degree": out_d,
                    "neighbors":  list(self._g.successors(node))[:10],
                })
        return sorted(pivots, key=lambda x: x["in_degree"] + x["out_degree"], reverse=True)

    def summary(self) -> Dict[str, Any]:
        return {
            "nodes":          self._g.number_of_nodes(),
            "edges":          self._g.number_of_edges(),
            "top_centrality": self.top_centrality(5),
            "pivot_chains":   self.get_pivot_chains()[:5],
        }

# ── DB loader ─────────────────────────────────────────────────────
async def load_incident_events(incident_id: str, pg_dsn: str = PG_DSN) -> List[Dict[str, Any]]:
    pool = await asyncpg.create_pool(pg_dsn, min_size=2, max_size=4)
    try:
        sql  = "SELECT raw FROM kai_incident_events WHERE incident_id = $1 LIMIT 2000"
        rows = await pool.fetch(sql, incident_id)
        return [json.loads(r["raw"]) for r in rows]
    finally:
        await pool.close()

async def investigate(incident_id: str) -> Dict[str, Any]:
    events = await load_incident_events(incident_id)
    graph  = ThreatGraph()
    graph.from_events(events)
    summary = graph.summary()
    if events:
        first_src = events[0].get("src_ip", "")
        if first_src:
            summary["lateral_paths"] = graph.lateral_movement_paths(first_src)
    return summary

# ── entrypoint ────────────────────────────────────────────────────
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    # demo with synthetic events
    demo_events = [
        {"src_ip": "10.0.0.1", "dst_ip": "10.0.0.2", "user": "admin", "process": "cmd.exe"},
        {"src_ip": "10.0.0.2", "dst_ip": "10.0.0.5", "domain": "evil.ru"},
        {"src_ip": "10.0.0.5", "dst_ip": "10.0.0.10"},
    ]
    g = ThreatGraph()
    g.from_events(demo_events)
    print(json.dumps(g.summary(), indent=2, default=str))
