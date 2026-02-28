# K-VENDOR-OSQ-001 -- osquery Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | osquery (Linux Foundation / Meta)           |
| License      | Apache-2.0 + GPL-2.0 (dual)                |
| Integration  | Thrift/HTTP API + direct table linking      |
| Consumers    | CoreSec agent, KAI-HUNTER, KAI-INVEST       |

## Overview

osquery exposes operating system state as SQL-queryable virtual tables.
Kubric uses osquery for scheduled endpoint visibility queries and
on-demand incident response data collection. The Apache-2.0 licensed
SDK components may be directly integrated; GPL-2.0 components are
accessed via subprocess/API boundaries only.

## Kubric Integration Points

- **CoreSec** schedules osquery packs via the osqueryd config and
  reads results from the JSON log output or Thrift API.
- **KAI-HUNTER** dispatches ad-hoc queries for hypothesis-driven
  threat hunts (e.g., checking for persistence mechanisms).
- **KAI-INVEST** collects forensic snapshots (process list, open
  files, network connections) during incident response.
- **Watchdog** manages osquery binary deployment and pack distribution.

## Document Map

| Doc ID         | Title                  |
|----------------|------------------------|
| OSQ-002        | Incident Response      |
| OSQ-003        | FIM Packs              |
