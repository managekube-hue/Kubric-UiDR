# K-VENDOR-THV-001 -- TheHive Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | TheHive Project / StrangeBee                |
| License      | AGPL-3.0                                    |
| Integration  | HTTP REST API only (AGPL boundary)          |
| Consumers    | KAI-TRIAGE, KAI-KEEPER, KAI-COMM            |

## Overview

TheHive is a security incident response platform for case management,
alert tracking, and collaboration. Kubric uses TheHive as the central
case management backend, accessed exclusively through its REST API.
No TheHive source code is imported into Kubric.

## Kubric Integration Points

- **KAI-TRIAGE** creates alerts in TheHive when enriched events
  exceed severity thresholds.
- **KAI-KEEPER** promotes alerts to cases and attaches remediation
  task lists to case objects via the API.
- **KAI-COMM** reads case status and escalation data to route
  notifications to SOC analysts and tenant contacts.
- **Watchdog** manages TheHive container lifecycle and data backups.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| THV-002        | Case Schema        |
| THV-003        | Alert Schema       |
| THV-004        | License Boundary   |
