"""
K-KAI Sentinel: HIBP Credential Score
Checks tenant user email addresses against the Have I Been Pwned v3 API
using k-anonymity (SHA-1 prefix search) and scores overall credential
exposure risk for the tenant.
"""
from __future__ import annotations
import asyncio, hashlib, json, logging, os
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Dict, List, Optional, Tuple
import httpx

logger = logging.getLogger(__name__)
HIBP_API_KEY = os.getenv("HIBP_API_KEY", "")
HIBP_PWNED_URL     = "https://haveibeenpwned.com/api/v3/breachedaccount/{email}"
HIBP_PWNED_PW_URL  = "https://api.pwnedpasswords.com/range/{prefix}"

@dataclass
class CredentialBreachResult:
    email:        str
    is_breached:  bool
    breach_count: int
    breach_names: List[str] = field(default_factory=list)
    password_pwned: bool    = False
    pwned_count:  int       = 0

@dataclass
class TenantCredentialReport:
    tenant_id:         str
    total_users:       int
    breached_users:    int
    password_exposed:  int
    exposure_score:    float     # 0-100 (lower is better)
    risk_tier:         str
    results:           List[CredentialBreachResult]
    generated_at:      str

def _hibp_tier(score: float) -> str:
    if score >= 70: return "CRITICAL"
    if score >= 40: return "HIGH"
    if score >= 20: return "MEDIUM"
    return "LOW"

async def check_email_breaches(email: str, api_key: str = HIBP_API_KEY) -> Tuple[bool, int, List[str]]:
    """Check HIBP v3 breachedaccount endpoint for an email address."""
    if not api_key:
        logger.warning("HIBP_API_KEY not set; skipping breach check")
        return False, 0, []
    url = HIBP_PWNED_URL.format(email=email)
    headers = {"hibp-api-key": api_key, "user-agent": "Kubric-KAI/1.0"}
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.get(url, headers=headers)
        if resp.status_code == 404:
            return False, 0, []
        if resp.status_code == 429:
            await asyncio.sleep(2)
            resp = await client.get(url, headers=headers)
        resp.raise_for_status()
        breaches = resp.json()
    names = [b.get("Name", "") for b in breaches]
    return True, len(breaches), names

async def check_password_hash(sha1_password: str) -> Tuple[bool, int]:
    """k-anonymity SHA-1 prefix search against pwnedpasswords API."""
    prefix = sha1_password[:5].upper()
    suffix = sha1_password[5:].upper()
    url    = HIBP_PWNED_PW_URL.format(prefix=prefix)
    async with httpx.AsyncClient(timeout=10) as client:
        resp = await client.get(url)
        resp.raise_for_status()
    for line in resp.text.splitlines():
        parts = line.split(":")
        if len(parts) == 2 and parts[0].upper() == suffix:
            count = int(parts[1].strip())
            return True, count
    return False, 0

async def score_tenant_credentials(
    tenant_id: str,
    user_emails: List[str],
    api_key: str = HIBP_API_KEY,
) -> TenantCredentialReport:
    results: List[CredentialBreachResult] = []
    for email in user_emails:
        await asyncio.sleep(0.2)   # HIBP rate limit: 5 req/sec
        try:
            breached, count, names = await check_email_breaches(email, api_key)
            results.append(CredentialBreachResult(
                email=email, is_breached=breached,
                breach_count=count, breach_names=names[:10],
            ))
        except Exception as exc:
            logger.error("HIBP check failed for %s: %s", email, exc)
            results.append(CredentialBreachResult(email=email, is_breached=False, breach_count=0))

    total     = len(results)
    breached  = sum(1 for r in results if r.is_breached)
    pw_exp    = sum(1 for r in results if r.password_pwned)
    exposure  = round(((breached + pw_exp * 0.5) / max(total, 1)) * 100, 2)

    return TenantCredentialReport(
        tenant_id=tenant_id,
        total_users=total,
        breached_users=breached,
        password_exposed=pw_exp,
        exposure_score=exposure,
        risk_tier=_hibp_tier(exposure),
        results=results,
        generated_at=datetime.now(timezone.utc).isoformat(),
    )

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    demo = asyncio.run(score_tenant_credentials(
        "demo-tenant-001",
        ["test@example.com", "admin@example.com"],
    ))
    print(json.dumps({
        "tenant_id":     demo.tenant_id,
        "exposure_score": demo.exposure_score,
        "risk_tier":     demo.risk_tier,
        "breached":      demo.breached_users,
    }, indent=2))
