"""
K-KAI Risk: EPSS Scorer
Computes composite risk scores for vulnerabilities by combining
EPSS (exploitation probability) with CVSS base score and asset value.
"""
from __future__ import annotations
import asyncio, json, logging, os
from dataclasses import dataclass
from datetime import date
from typing import Dict, List, Optional
import httpx

logger = logging.getLogger(__name__)
EPSS_API = "https://api.first.org/data/v1/epss"

@dataclass
class VulnRiskScore:
    cve:          str
    cvss:         float        # 0-10
    epss:         float        # 0-1 exploitation probability
    epss_pct:     float        # 0-1 percentile
    asset_value:  float        # 1-5 business criticality
    composite:    float        # weighted composite 0-100
    risk_tier:    str          # CRITICAL / HIGH / MEDIUM / LOW / INFO

    @property
    def is_actionable(self) -> bool:
        return self.risk_tier in ("CRITICAL", "HIGH")

def _risk_tier(score: float) -> str:
    if score >= 80: return "CRITICAL"
    if score >= 60: return "HIGH"
    if score >= 40: return "MEDIUM"
    if score >= 20: return "LOW"
    return "INFO"

def compute_composite(cvss: float, epss: float, asset_value: float) -> float:
    """
    Composite = 0.40*cvss_norm + 0.40*epss_norm + 0.20*asset_norm
    All inputs expressed as 0-100 before weighting.
    """
    cvss_norm  = (cvss / 10.0) * 100
    epss_norm  = epss * 100
    asset_norm = ((asset_value - 1) / 4.0) * 100
    return round(0.40 * cvss_norm + 0.40 * epss_norm + 0.20 * asset_norm, 2)

async def fetch_epss(cves: List[str]) -> Dict[str, Dict[str, float]]:
    """Fetch EPSS scores for a list of CVEs. Returns dict[cve -> {epss, percentile}]."""
    result: Dict[str, Dict[str, float]] = {}
    async with httpx.AsyncClient(timeout=30) as client:
        for i in range(0, len(cves), 100):
            chunk  = cves[i:i+100]
            params = {"cve": ",".join(chunk)}
            resp   = await client.get(EPSS_API, params=params)
            resp.raise_for_status()
            for item in resp.json().get("data", []):
                result[item["cve"]] = {
                    "epss":       float(item.get("epss", 0)),
                    "percentile": float(item.get("percentile", 0)),
                }
    return result

async def score_vulnerabilities(
    vulns: List[Dict],            # [{"cve": "CVE-xxx", "cvss": 7.5, "asset_value": 3}]
) -> List[VulnRiskScore]:
    cves  = [v["cve"] for v in vulns]
    epss_data = await fetch_epss(cves)

    scored: List[VulnRiskScore] = []
    for v in vulns:
        cve        = v["cve"]
        cvss       = float(v.get("cvss", 5.0))
        asset_val  = float(v.get("asset_value", 3.0))
        ep         = epss_data.get(cve, {})
        epss       = ep.get("epss", 0.0)
        epss_pct   = ep.get("percentile", 0.0)
        composite  = compute_composite(cvss, epss, asset_val)
        scored.append(VulnRiskScore(
            cve=cve, cvss=cvss, epss=epss, epss_pct=epss_pct,
            asset_value=asset_val, composite=composite,
            risk_tier=_risk_tier(composite),
        ))
    return sorted(scored, key=lambda x: x.composite, reverse=True)

async def main() -> None:
    logging.basicConfig(level=logging.INFO)
    demo = [
        {"cve": "CVE-2023-44487", "cvss": 7.5, "asset_value": 4},
        {"cve": "CVE-2021-44228", "cvss": 10.0, "asset_value": 5},
        {"cve": "CVE-2022-0847",  "cvss": 7.8, "asset_value": 2},
    ]
    scores = await score_vulnerabilities(demo)
    for s in scores:
        print(json.dumps(vars(s), indent=2))

if __name__ == "__main__":
    asyncio.run(main())
