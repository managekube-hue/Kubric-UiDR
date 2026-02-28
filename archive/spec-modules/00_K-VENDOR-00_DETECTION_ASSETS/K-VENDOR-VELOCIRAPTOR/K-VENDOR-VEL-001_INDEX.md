# K-VENDOR-VEL-001 -- Velociraptor Index

| Field        | Value                                     |
|--------------|-------------------------------------------|
| Vendor       | Velociraptor (Rapid7)                     |
| License      | AGPL-3.0                                  |
| Integration  | HTTP/subprocess only (AGPL boundary)      |
| Consumers    | CoreSec agent, KAI-HUNTER, KAI-ANALYST    |

## Overview

Velociraptor is an endpoint visibility and forensic collection tool.
Kubric deploys Velociraptor servers per-tenant and communicates via the
gRPC/REST API exclusively. No Velociraptor library code is linked into
Kubric binaries.

## Kubric Integration Points

- **CoreSec** calls the Velociraptor REST API to launch artifact hunts
  and retrieve collected forensic data for endpoint alerts.
- **KAI-HUNTER** triggers server-side VQL hunts via HTTP when elevated
  risk scores arrive on `kubric.kai.foresight.risk`.
- **KAI-ANALYST** pulls completed hunt results for deep-dive analysis.
- **Watchdog** manages Velociraptor container lifecycle and version pinning.

## Document Map

| Doc ID         | Title                  |
|----------------|------------------------|
| VEL-002        | Threat Hunting VQL     |
| VEL-003        | Forensic Artifacts     |
| VEL-004        | License Boundary       |
