# K-VENDOR-SHF-001 -- Shuffle Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | Shuffle AS                                  |
| License      | AGPL-3.0                                    |
| Integration  | HTTP REST API only (AGPL boundary)          |
| Consumers    | KAI-KEEPER, KAI-COMM, KAI-DEPLOY            |

## Overview

Shuffle is an open-source SOAR (Security Orchestration, Automation and
Response) platform. Kubric uses Shuffle to execute multi-step playbooks
that orchestrate containment, notification, and remediation workflows.
All interaction is through Shuffle's REST API.

## Kubric Integration Points

- **KAI-KEEPER** triggers Shuffle workflows for automated remediation
  sequences (e.g., isolate host, block IP, notify SOC).
- **KAI-COMM** invokes Shuffle notification workflows to route alerts
  through email, Slack, PagerDuty, and Teams channels.
- **KAI-DEPLOY** uses Shuffle playbooks for change management approval
  and post-deployment validation sequences.
- **Watchdog** manages Shuffle container and app image lifecycle.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| SHF-002        | SOAR Workflows     |
| SHF-003        | License Boundary   |
