"""
K-KAI Deploy: SaltStack Client
Executes SaltStack states and remote execution (cmd.run) across
minion pools for configuration management and package deployment.
"""
from __future__ import annotations
import json, logging, os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional
import httpx

logger = logging.getLogger(__name__)
SALT_API_URL  = os.getenv("SALT_API_URL",  "https://salt-master:8080")
SALT_USER     = os.getenv("SALT_USER",    "kubric")
SALT_PASSWORD = os.getenv("SALT_PASSWORD", "")
SALT_EAUTH    = os.getenv("SALT_EAUTH",   "pam")

@dataclass
class SaltResult:
    jid:      str
    target:   str
    fun:      str
    success:  bool
    minion_results: Dict[str, Any]
    failed_minions: List[str]
    ts: str = ""

    def __post_init__(self) -> None:
        self.ts = self.ts or datetime.now(timezone.utc).isoformat()

class SaltStackClient:
    def __init__(self, base_url: str = SALT_API_URL, user: str = SALT_USER,
                 password: str = SALT_PASSWORD, eauth: str = SALT_EAUTH):
        self.base_url  = base_url.rstrip("/")
        self.user      = user
        self.password  = password
        self.eauth     = eauth
        self._token: Optional[str] = None
        self._token_exp: Optional[float] = None

    async def _get_token(self) -> str:
        import time
        if self._token and self._token_exp and time.time() < self._token_exp - 60:
            return self._token
        async with httpx.AsyncClient(verify=False, timeout=15) as client:
            resp = await client.post(f"{self.base_url}/login", json={
                "username": self.user, "password": self.password, "eauth": self.eauth,
            })
            resp.raise_for_status()
            data = resp.json()
            self._token     = data["return"][0]["token"]
            self._token_exp = data["return"][0]["expire"]
            return self._token

    async def _call(self, payload: List[Dict]) -> Any:
        token = await self._get_token()
        async with httpx.AsyncClient(verify=False, timeout=60) as client:
            resp = await client.post(
                f"{self.base_url}/",
                headers={"X-Auth-Token": token, "Content-Type": "application/json"},
                json=payload,
            )
            resp.raise_for_status()
            return resp.json().get("return", [{}])[0]

    async def run_state(
        self, target: str, state: str, pillar: Optional[Dict] = None
    ) -> SaltResult:
        payload = [{"client": "local", "tgt": target, "fun": "state.apply",
                    "arg": [state], "kwarg": {"pillar": pillar or {}}}]
        result = await self._call(payload)
        minion_results: Dict[str, Any] = {}
        failed: List[str] = []
        for minion, state_result in result.items():
            if isinstance(state_result, dict):
                min_ok = all(v.get("result", False) for v in state_result.values()
                             if isinstance(v, dict))
                minion_results[minion] = state_result
                if not min_ok:
                    failed.append(minion)
            else:
                failed.append(minion)
        return SaltResult(
            jid=str(result.get("jid", "")),
            target=target, fun=f"state.apply {state}",
            success=len(failed) == 0,
            minion_results=minion_results,
            failed_minions=failed,
        )

    async def cmd_run(self, target: str, cmd: str) -> SaltResult:
        payload = [{"client": "local", "tgt": target, "fun": "cmd.run", "arg": [cmd]}]
        result  = await self._call(payload)
        failed  = [m for m, r in result.items() if not isinstance(r, str)]
        return SaltResult(
            jid="", target=target, fun=f"cmd.run {cmd}",
            success=len(failed) == 0,
            minion_results=result, failed_minions=failed,
        )

    async def list_minions(self) -> List[str]:
        payload = [{"client": "wheel", "fun": "key.list", "match": "accepted"}]
        result  = await self._call(payload)
        return list(result.get("data", {}).get("return", {}).get("accepted", []))

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    print(json.dumps({"salt_api": SALT_API_URL, "user": SALT_USER, "eauth": SALT_EAUTH}))
