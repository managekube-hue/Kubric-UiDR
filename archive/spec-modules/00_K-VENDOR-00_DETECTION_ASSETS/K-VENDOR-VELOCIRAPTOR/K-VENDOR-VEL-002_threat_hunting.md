# K-VENDOR-VEL-002 -- Threat Hunting VQL Artifacts

| Field       | Value                                        |
|-------------|----------------------------------------------|
| Category    | Proactive Hunt Queries                       |
| Format      | VQL (Velociraptor Query Language)             |
| Consumer    | KAI-HUNTER via Velociraptor REST API          |

## Purpose

Pre-built VQL queries that KAI-HUNTER dispatches through the Velociraptor
API when a proactive hunt is triggered. These run server-side; Kubric
never executes VQL directly.

## Key Hunt Categories

| Hunt Type              | VQL Artifact                            |
|------------------------|-----------------------------------------|
| Process anomaly        | `Windows.Detection.ProcessTree`         |
| Lateral movement       | `Windows.Detection.LateralMovement`     |
| Persistence mechanisms | `Windows.Persistence.PermanentWMI`      |
| Credential access      | `Windows.Detection.LSASS`              |
| Living-off-the-land    | `Windows.Detection.BinaryRename`        |
| Linux rootkit          | `Linux.Detection.HiddenModules`         |

## Integration Flow

1. KAI-HUNTER receives elevated risk event on NATS.
2. Builds hunt request with tenant-scoped VQL artifact name.
3. POSTs to `https://vel-{tenant}/api/v1/CreateHunt`.
4. Polls `api/v1/GetHuntResults` until completion.
5. Publishes structured findings to `kubric.kai.hunter.findings`.

## Data Handling

- Hunt results are stored in the tenant-isolated Velociraptor filestore.
- Kubric only retrieves JSON-serialized result rows via the API.
- Raw NTFS / memory artifacts remain on the Velociraptor server.
