# K-VENDOR-RUD-001 -- Rudder Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | Normation (Rudder)                          |
| License      | GPL-3.0 (server), Apache-2.0 (agent)        |
| Integration  | HTTP REST API only (GPL server boundary)    |
| Consumers    | KAI-DEPLOY, KAI-RISK, CoreSec               |

## Overview

Rudder is a continuous configuration management and compliance platform.
Kubric uses Rudder to enforce security hardening policies and track
configuration drift across managed infrastructure. The Rudder server
is GPL-3.0 licensed and accessed via REST API only.

## Kubric Integration Points

- **KAI-DEPLOY** validates that target nodes meet security baselines
  before approving change management deployments, querying Rudder's
  compliance API.
- **KAI-RISK** ingests node compliance percentages for posture-based
  risk scoring in FAIR models.
- **CoreSec** cross-references Rudder node inventory with endpoint
  telemetry for asset context enrichment.
- **Watchdog** manages Rudder server container lifecycle.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| RUD-002        | Techniques         |
| RUD-003        | License Boundary   |
