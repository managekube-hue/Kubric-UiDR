"""
K-KAI Investigator: MISP Galaxy Query
Queries a MISP instance for threat actor galaxy clusters related to
observed indicators, enriching incident context for investigation.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional
import httpx

logger = logging.getLogger(__name__)

MISP_URL        = os.getenv("MISP_URL",   "https://misp.internal")
MISP_KEY        = os.getenv("MISP_KEY",   "")
MISP_VERIFY_SSL = os.getenv("MISP_VERIFY_SSL", "false").lower() == "true"

@dataclass
class GalaxyCluster:
    uuid:        str
    galaxy_type: str
    value:       str
    description: str
    synonyms:    List[str] = field(default_factory=list)
    mitre_ids:   List[str] = field(default_factory=list)

    @classmethod
    def from_api(cls, data: Dict[str, Any]) -> "GalaxyCluster":
        meta = data.get("GalaxyCluster", {}).get("meta", {})
        return cls(
            uuid        = data.get("GalaxyCluster", {}).get("uuid", ""),
            galaxy_type = data.get("GalaxyCluster", {}).get("type", ""),
            value       = data.get("GalaxyCluster", {}).get("value", ""),
            description = data.get("GalaxyCluster", {}).get("description", ""),
            synonyms    = meta.get("synonyms", []),
            mitre_ids   = meta.get("external_id", []),
        )

class MispGalaxyClient:
    def __init__(self, base_url: str = MISP_URL, api_key: str = MISP_KEY,
                 verify_ssl: bool = MISP_VERIFY_SSL):
        self.base_url  = base_url.rstrip("/")
        self.headers   = {
            "Authorization": api_key,
            "Accept":        "application/json",
            "Content-Type":  "application/json",
        }
        self.verify_ssl = verify_ssl

    async def search_galaxy(self, value: str, galaxy_type: str = "threat-actor") -> List[GalaxyCluster]:
        """Search MISP galaxy clusters by value (actor name, IP, domain, hash)."""
        url     = f"{self.base_url}/galaxy_clusters/restSearch"
        payload = {"value": value, "searchall": 1, "type": galaxy_type}
        async with httpx.AsyncClient(verify=self.verify_ssl, timeout=30) as client:
            resp = await client.post(url, headers=self.headers, json=payload)
            resp.raise_for_status()
            raw = resp.json()
        clusters = raw.get("response", {}).get("GalaxyCluster", [])
        return [GalaxyCluster.from_api({"GalaxyCluster": c}) for c in clusters]

    async def search_by_tag(self, mitre_technique_id: str) -> List[GalaxyCluster]:
        """Fetch clusters tagged with a specific MITRE ATT&CK technique."""
        url     = f"{self.base_url}/galaxy_clusters/restSearch"
        payload = {"tag": f'misp-galaxy:mitre-attack-pattern="{mitre_technique_id}"'}
        async with httpx.AsyncClient(verify=self.verify_ssl, timeout=30) as client:
            resp = await client.post(url, headers=self.headers, json=payload)
            resp.raise_for_status()
            raw  = resp.json()
        clusters = raw.get("response", {}).get("GalaxyCluster", [])
        return [GalaxyCluster.from_api({"GalaxyCluster": c}) for c in clusters]

    async def enrich_incident(self, incident: Dict[str, Any]) -> Dict[str, Any]:
        """
        Given an incident dict with optional keys: threat_actor, src_ip,
        domain, file_hash, mitre_techniques — returns enriched intelligence.
        """
        enrichment: Dict[str, Any] = {"clusters": [], "actor_aliases": [], "mitre_coverage": []}

        # Threat actor name match
        if actor := incident.get("threat_actor"):
            clusters = await self.search_galaxy(actor, "threat-actor")
            for c in clusters:
                enrichment["clusters"].append({
                    "uuid": c.uuid, "value": c.value,
                    "description": c.description[:200],
                    "synonyms": c.synonyms[:5],
                })
                enrichment["actor_aliases"].extend(c.synonyms)

        # MITRE technique coverage
        for tech in incident.get("mitre_techniques", []):
            tech_clusters = await self.search_by_tag(tech)
            enrichment["mitre_coverage"].append({
                "technique": tech,
                "actor_clusters": [c.value for c in tech_clusters],
            })

        return enrichment

# ── CLI entrypoint ────────────────────────────────────────────────
async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    client = MispGalaxyClient()
    incident = {
        "threat_actor":      "APT29",
        "mitre_techniques":  ["T1059", "T1021"],
    }
    result = await client.enrich_incident(incident)
    print(json.dumps(result, indent=2))

if __name__ == "__main__":
    asyncio.run(main())
