# K-VENDOR-WAZ-001 -- Wazuh Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | Wazuh Inc.                                  |
| License      | GPL-2.0 (manager/agent)                     |
| Integration  | HTTP REST API + syslog forwarding only      |
| Consumers    | CoreSec agent, KAI-TRIAGE, KAI-KEEPER       |

## Overview

Wazuh provides host-based intrusion detection (HIDS), log analysis,
file integrity monitoring, and SCA (Security Configuration Assessment).
Kubric communicates with the Wazuh Manager exclusively through its REST
API and syslog output. No Wazuh source code is linked into Kubric.

## Kubric Integration Points

- **CoreSec** reads Wazuh alert JSON from the REST API to correlate
  with endpoint telemetry and YARA/Sigma detections.
- **KAI-TRIAGE** pulls Wazuh alerts for enrichment and severity scoring.
- **KAI-KEEPER** triggers Wazuh active response modules via the API
  for containment actions (block IP, kill process).
- **Watchdog** manages Wazuh manager/agent deployment and rule updates.

## Document Map

| Doc ID         | Title                  |
|----------------|------------------------|
| WAZ-002        | Process Rules          |
| WAZ-003        | AD Rules               |
| WAZ-004        | SCA Rules              |
| WAZ-005        | License Boundary       |
