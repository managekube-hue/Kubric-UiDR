#!/usr/bin/env python3
"""
Kubric Cortex Responder -- Isolate host by blocking all traffic except NATS/management.

License: AGPL 3.0 (executed as subprocess, not linked into binary)
Protocol: Cortex Analyzer/Responder v2 -- reads JSON job from stdin, writes result to stdout
"""

import json
import subprocess
import sys


def run_cmd(cmd):
    return subprocess.run(cmd, capture_output=True, text=True, timeout=10)


def run():
    job = json.load(sys.stdin)
    params = job.get("parameters", {})
    mgmt_port = int(params.get("mgmt_port", 4222))   # NATS default
    ssh_port  = int(params.get("ssh_port",  22))

    rules = [
        # Keep established sessions alive
        ["iptables", "-I", "INPUT",  "1", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"],
        ["iptables", "-I", "OUTPUT", "1", "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"],
        # Allow NATS management channel outbound
        ["iptables", "-A", "OUTPUT", "-p", "tcp", "--dport", str(mgmt_port), "-j", "ACCEPT"],
        # Allow SSH inbound for emergency access
        ["iptables", "-A", "INPUT", "-p", "tcp", "--dport", str(ssh_port), "-j", "ACCEPT"],
        # Drop everything else
        ["iptables", "-A", "INPUT",  "-j", "DROP"],
        ["iptables", "-A", "OUTPUT", "-j", "DROP"],
        ["iptables", "-A", "FORWARD", "-j", "DROP"],
    ]

    errors = []
    applied = 0
    for cmd in rules:
        r = run_cmd(cmd)
        if r.returncode == 0:
            applied += 1
        else:
            errors.append(r.stderr.strip())

    if errors:
        print(json.dumps({
            "success": False,
            "message": "Partial isolation; errors: " + "; ".join(errors),
            "full": {"rules_applied": applied, "errors": errors}
        }))
    else:
        print(json.dumps({
            "success": True,
            "message": (
                f"Host isolated. Outbound: NATS:{mgmt_port} only. "
                f"Inbound: SSH:{ssh_port} + established only."
            ),
            "full": {"rules_applied": applied, "mgmt_port": mgmt_port, "ssh_port": ssh_port}
        }))


if __name__ == "__main__":
    run()
