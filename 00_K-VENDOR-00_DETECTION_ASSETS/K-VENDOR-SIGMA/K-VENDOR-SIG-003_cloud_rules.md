# K-VENDOR-SIG-003 -- Cloud Sigma Rules

| Field | Value |
|-------|-------|
| **Rule Path** | `vendor/sigma/rules/cloud/` |
| **Sub-directories** | `aws/`, `azure/`, `gcp/` |
| **Log Sources** | CloudTrail, Azure Activity/Sign-in, GCP Audit Log |

## Scope

Cloud Sigma rules detect misconfigurations, privilege escalation, and attacker
activity across AWS, Azure, and GCP control planes. These rules are evaluated
by CoreSec's `SigmaEngine` when cloud audit events are normalized into
`ProcessEvent`-compatible structures by the Kubric ingest pipeline. The ingest
layer maps cloud-specific fields (e.g., `eventName`, `operationName`) into the
`executable` and `cmdline` fields consumed by `extract_field()`.

## AWS CloudTrail Rules

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| IAM Persistence | CreateUser, AttachUserPolicy, CreateAccessKey | T1098 (Account Manipulation) |
| S3 Exfiltration | PutBucketPolicy public-read, GetObject bulk | T1537 (Transfer to Cloud Account) |
| GuardDuty Tampering | DeleteDetector, UpdateDetector disabled | T1562.001 (Disable Security Tools) |
| EC2 Abuse | RunInstances crypto-mining AMI, StopLogging | T1496 (Resource Hijacking) |
| Credential Access | GetSecretValue anomalous, AssumeRole cross-acct | T1528 (Steal App Access Token) |

## Azure Activity / Entra ID Rules

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| Entra ID Risky Sign-in | Impossible travel, TOR exit IP, unfamiliar location | T1078 (Valid Accounts) |
| Privileged Role Assignment | Global Admin added, PIM elevation outside policy | T1098.001 (Additional Cloud Credentials) |
| Key Vault Access | Secret/Key read by non-approved principal | T1552.005 (Cloud Instance Metadata) |
| NSG Modification | Inbound allow 0.0.0.0/0 on management ports | T1562.007 (Disable Cloud Logs) |

## GCP Audit Log Rules

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| IAM Binding | setIamPolicy granting owner/editor to external | T1098 (Account Manipulation) |
| Firewall | Ingress allow 0.0.0.0/0 SSH/RDP | T1133 (External Remote Services) |
| Logging Sink Deletion | DeleteSink, UpdateSink disabled | T1562.008 (Disable Cloud Logs) |
| GKE Cluster Exposure | setMasterAuthorizedNetworks 0.0.0.0/0 | T1610 (Deploy Container) |

## CoreSec Integration

Cloud rules follow the same `SigmaEngine::evaluate()` path as endpoint rules.
The cloud ingest adapter (upstream of CoreSec) maps cloud event JSON into an
OCSF `ProcessEvent` with the following conventions:

| ProcessEvent Field | Cloud Mapping |
|--------------------|---------------|
| `executable` | API action name (e.g., `iam.CreateUser`) |
| `cmdline` | Serialized request parameters |
| `user` | Caller principal / ARN / UPN |
| `tenant_id` | Kubric tenant owning the cloud account |

Matched rules produce `SigmaMatch` structs whose `tags` array carries the ATT&CK
technique IDs for downstream correlation by the Kai analyst agents.
