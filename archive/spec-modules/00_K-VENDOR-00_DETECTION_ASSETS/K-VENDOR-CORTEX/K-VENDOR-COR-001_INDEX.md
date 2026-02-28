# K-VENDOR-COR-001 -- Cortex Index

| Field        | Value                                        |
|--------------|----------------------------------------------|
| Vendor       | TheHive Project / StrangeBee                 |
| License      | AGPL-3.0                                     |
| Integration  | HTTP REST API only (AGPL boundary)           |
| Consumers    | KAI-TRIAGE, KAI-ANALYST, KAI-KEEPER          |

## Overview

Cortex is an observable analysis and active response engine. It executes
analyzers (enrichment) and responders (containment actions) on demand.
Kubric calls Cortex exclusively through its REST API -- no Cortex source
code is imported or linked.

## Kubric Integration Points

- **KAI-TRIAGE** submits observables (IPs, hashes, domains) to Cortex
  analyzers for enrichment during alert scoring.
- **KAI-ANALYST** requests deep analysis jobs for forensic investigation.
- **KAI-KEEPER** triggers Cortex responders to execute containment
  actions (block IP, disable account) as part of remediation plans.
- **Watchdog** manages the Cortex container and analyzer/responder images.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| COR-002        | Analyzers          |
| COR-003        | Responders         |
| COR-004        | License Boundary   |
