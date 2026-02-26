# K-VENDOR-VEL-003 -- Forensic Artifact Collection

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Forensic evidence acquisition                 |
| Format      | VQL artifact definitions (YAML + VQL)         |
| Consumer    | KAI-ANALYST, KAI-INVEST via REST API          |

## Purpose

Velociraptor artifact definitions used by Kubric to collect forensic
evidence from endpoints during incident response. KAI-ANALYST and
KAI-INVEST trigger collections via the Velociraptor REST API.

## Core Forensic Artifacts

| Artifact                              | Evidence Type           |
|---------------------------------------|-------------------------|
| `Windows.KapeFiles.Targets`           | Triage image (MFT, EVT) |
| `Windows.Timeline.MFT`               | NTFS timeline           |
| `Windows.EventLogs.Evtx`             | Windows event logs      |
| `Windows.Registry.NTUser`            | User registry hives     |
| `Linux.Forensics.Timeline`           | Filesystem timeline     |
| `Generic.Forensic.LocalHashes.Glob`  | File hash survey        |
| `Windows.Memory.Acquisition`         | Full memory capture     |

## Integration Flow

1. KAI-INVEST receives incident escalation on NATS.
2. Requests artifact collection via Velociraptor `CollectArtifact` API.
3. Monitors collection status until upload completes.
4. Downloads collected ZIP/JSON from the Velociraptor filestore API.
5. Parses results and publishes evidence chain to `kubric.kai.invest.evidence`.

## Evidence Chain

- All collection requests are logged with operator ID and timestamp.
- SHA-256 hashes of collected bundles are recorded for chain of custody.
- Tenant isolation is enforced by Velociraptor's org/multi-tenancy model.
