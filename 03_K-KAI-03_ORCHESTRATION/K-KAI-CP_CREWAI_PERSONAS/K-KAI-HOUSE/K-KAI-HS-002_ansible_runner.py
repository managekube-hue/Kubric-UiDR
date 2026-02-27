"""
K-KAI Housekeeper: Ansible Runner
Executes Ansible playbooks for configuration remediation and
patch deployment, wrapping the ansible-runner Python API.
"""
from __future__ import annotations
import json, logging, os, tempfile, uuid
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)

ANSIBLE_BASE_DIR = Path(os.getenv("ANSIBLE_BASE_DIR", "/opt/kubric/ansible"))
VAULT_PASS_FILE  = os.getenv("ANSIBLE_VAULT_PASS_FILE", "/run/secrets/ansible_vault_pass")

@dataclass
class PlaybookResult:
    run_id:       str
    playbook:     str
    host_pattern: str
    status:       str      # "successful" | "failed" | "timeout" | "error"
    rc:           int
    stdout:       str
    stderr:       str
    duration_s:   float
    hosts_ok:     int = 0
    hosts_failed: int = 0
    hosts_changed: int = 0

class AnsibleRunner:
    def __init__(self, base_dir: Path = ANSIBLE_BASE_DIR):
        self.base_dir = base_dir

    def _build_inventory(self, hosts: List[str], ssh_user: str = "kubric") -> str:
        lines = ["[targets]"]
        for h in hosts:
            lines.append(f"{h} ansible_user={ssh_user} ansible_ssh_private_key_file=/run/secrets/ssh_key")
        return "\n".join(lines)

    def run_playbook(
        self,
        playbook_name: str,
        hosts: List[str],
        extra_vars: Optional[Dict[str, Any]] = None,
        timeout: int = 300,
    ) -> PlaybookResult:
        """
        Execute an Ansible playbook via ansible-runner.
        Falls back to subprocess call if ansible-runner package not available.
        """
        import time
        start   = time.monotonic()
        run_id  = str(uuid.uuid4())[:8]

        playbook_path = self.base_dir / "playbooks" / playbook_name
        if not playbook_path.exists():
            return PlaybookResult(
                run_id=run_id, playbook=playbook_name,
                host_pattern=",".join(hosts),
                status="error", rc=2,
                stdout="", stderr=f"Playbook not found: {playbook_path}",
                duration_s=0,
            )

        try:
            import ansible_runner  # type: ignore
        except ImportError:
            # Fallback: subprocess call
            return self._subprocess_run(run_id, playbook_path, hosts, extra_vars, timeout, start)

        with tempfile.TemporaryDirectory() as tmpdir:
            inv_content = self._build_inventory(hosts)
            inv_path    = Path(tmpdir) / "inventory"
            inv_path.write_text(inv_content)

            evar_path = Path(tmpdir) / "extravars.json"
            evar_path.write_text(json.dumps(extra_vars or {}))

            runner = ansible_runner.run(
                playbook   = str(playbook_path),
                inventory  = str(inv_path),
                extravars  = extra_vars or {},
                timeout    = timeout,
                quiet      = True,
            )
            elapsed = time.monotonic() - start
            stats   = runner.stats or {}
            ok      = sum(v.get("ok", 0) for v in stats.values())
            failed  = sum(v.get("failures", 0) + v.get("unreachable", 0) for v in stats.values())
            changed = sum(v.get("changed", 0) for v in stats.values())

            return PlaybookResult(
                run_id=run_id, playbook=playbook_name,
                host_pattern=",".join(hosts),
                status=runner.status, rc=runner.rc or 0,
                stdout=runner.stdout.read() if runner.stdout else "",
                stderr=runner.stderr.read() if runner.stderr else "",
                duration_s=round(elapsed, 2),
                hosts_ok=ok, hosts_failed=failed, hosts_changed=changed,
            )

    def _subprocess_run(
        self,
        run_id: str,
        playbook_path: Path,
        hosts: List[str],
        extra_vars: Optional[Dict],
        timeout: int,
        start: float,
    ) -> PlaybookResult:
        import subprocess, time
        cmd = [
            "ansible-playbook",
            str(playbook_path),
            "-i", ",".join(hosts) + ",",
            "--timeout", str(timeout),
        ]
        if extra_vars:
            cmd.extend(["--extra-vars", json.dumps(extra_vars)])
        if Path(VAULT_PASS_FILE).exists():
            cmd.extend(["--vault-password-file", VAULT_PASS_FILE])

        try:
            proc = subprocess.run(
                cmd, capture_output=True, text=True, timeout=timeout
            )
            elapsed = time.monotonic() - start
            return PlaybookResult(
                run_id=run_id, playbook=playbook_path.name,
                host_pattern=",".join(hosts),
                status="successful" if proc.returncode == 0 else "failed",
                rc=proc.returncode, stdout=proc.stdout[-4000:],
                stderr=proc.stderr[-2000:],
                duration_s=round(elapsed, 2),
            )
        except subprocess.TimeoutExpired:
            return PlaybookResult(
                run_id=run_id, playbook=playbook_path.name,
                host_pattern=",".join(hosts),
                status="timeout", rc=124, stdout="", stderr="Timed out",
                duration_s=float(timeout),
            )

    def list_playbooks(self) -> List[str]:
        pb_dir = self.base_dir / "playbooks"
        if not pb_dir.exists():
            return []
        return [p.name for p in pb_dir.glob("*.yml")]

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    runner = AnsibleRunner()
    print("Available playbooks:", runner.list_playbooks())
