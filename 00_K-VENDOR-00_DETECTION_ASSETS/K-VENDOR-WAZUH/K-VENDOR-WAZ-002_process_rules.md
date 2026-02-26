# K-VENDOR-WAZ-002 -- Wazuh Process Monitoring Rules

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Process creation and command-line detection     |
| Format      | Wazuh XML rule definitions                     |
| Consumer    | CoreSec (via Wazuh API), KAI-TRIAGE            |

## Purpose

Wazuh rules that detect suspicious process creation, command-line
activity, and parent-child process anomalies. CoreSec reads matching
alerts from the Wazuh REST API.

## Key Rule Groups

| Rule Group             | Rule ID Range | Detection Target               |
|------------------------|---------------|---------------------------------|
| Sysmon Process Create  | 61600-61699   | Sysmon EventID 1 analysis      |
| PowerShell logging     | 91800-91899   | Obfuscated/encoded commands    |
| Linux auditd process   | 80700-80799   | execve audit trail             |
| Suspicious binaries    | 92000-92099   | LOLBins (certutil, mshta, etc) |
| Process injection      | 92100-92199   | Sysmon EventID 8/10 hooks      |

## Example Detections

- Encoded PowerShell (`-enc` / `-encodedcommand`)
- certutil download cradle
- Parent-child mismatch (e.g., `svchost` spawning `cmd.exe`)
- Reverse shell patterns in bash/sh command lines
- Process hollowing indicators via Sysmon

## Integration Notes

- Wazuh rules are GPL-2.0; Kubric does not embed or modify them.
- CoreSec polls `GET /alerts?q=rule.groups:process` on the Wazuh API.
- Alert JSON is forwarded to NATS for KAI-TRIAGE correlation.
