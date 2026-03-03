#!/usr/bin/env python3
"""
Kubric Cortex Responder — Block IP via iptables.

License: AGPL 3.0 (executed as subprocess, not linked)
Invocation: python3 block_ip.py < job.json
Input  (stdin): Cortex job JSON with dataType=ip
Output (stdout): Cortex result JSON
"""

import json
import subprocess
import sys


def run():
    job = json.load(sys.stdin)
    data = job.get("data", "")
    if not data:
        print(json.dumps({"success": False, "message": "No IP address in job data"}))
        sys.exit(0)

    try:
        result = subprocess.run(
            ["iptables", "-I", "INPUT", "-s", data, "-j", "DROP"],
            capture_output=True, text=True, timeout=10
        )
        if result.returncode == 0:
            print(json.dumps({
                "success": True,
                "message": f"Blocked IP {data} via iptables INPUT DROP",
                "full": {"ip": data, "rule": "INPUT DROP"}
            }))
        else:
            print(json.dumps({
                "success": False,
                "message": f"iptables failed: {result.stderr.strip()}"
            }))
    except Exception as e:
        print(json.dumps({"success": False, "message": str(e)}))


if __name__ == "__main__":
    run()
