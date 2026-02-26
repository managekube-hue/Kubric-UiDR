# K-VENDOR-SIG-004 -- SaaS Sigma Rules

| Field | Value |
|-------|-------|
| **Rule Path** | `vendor/sigma/rules/cloud/` (SaaS-specific sub-rules) |
| **Log Sources** | Microsoft 365 UAL, Google Workspace, Okta System Log, GitHub Audit |
| **Ingress** | Kubric SaaS log connectors -> OCSF normalization -> CoreSec |

## Scope

SaaS Sigma rules target identity, collaboration, and DevOps platforms that
operate outside traditional infrastructure boundaries. Kubric ingests SaaS
audit logs via API connectors, normalizes them to OCSF `ProcessEvent` format,
and feeds them through CoreSec's `SigmaEngine` for real-time detection.

## Microsoft 365 / Entra ID

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| Mailbox Delegation | Add-MailboxPermission FullAccess/SendAs by non-admin | T1098.002 (Email Forwarding Rule) |
| eDiscovery Abuse | New-ComplianceSearch + Preview/Export by non-legal | T1114.002 (Remote Email Collection) |
| Conditional Access | Policy deletion or modification to weaken MFA | T1556 (Modify Authentication Process) |
| OAuth App Consent | Admin consent grant to unverified publisher | T1550.001 (Application Access Token) |
| SharePoint Sharing | Anonymous link creation on sensitive library | T1567 (Exfiltration Over Web Service) |

## Google Workspace

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| Drive Exfiltration | Bulk download, sharing to external domain | T1567.002 (Exfiltration to Cloud Storage) |
| Admin Role Grant | Super Admin role assigned outside break-glass | T1098 (Account Manipulation) |
| App Script Trigger | Authorized app installed with Drive scope | T1059.009 (Cloud API) |
| 2SV Bypass | 2-Step Verification disabled by user/admin | T1556.006 (MFA Disable) |

## Okta

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| Credential Stuffing | Rapid user.session.start failures from diverse IPs | T1110.004 (Credential Stuffing) |
| MFA Factor Reset | MFA factor reset followed by immediate sign-in | T1556 (Modify Authentication Process) |
| Policy Bypass | Zone/policy modification to widen trusted network | T1562 (Impair Defenses) |
| Admin API Token | API token created with okta.admins scope | T1528 (Steal Application Access Token) |

## GitHub Audit

| Rule Family | Detection Target | MITRE ATT&CK |
|-------------|-----------------|---------------|
| Repo Visibility | Private to public visibility change | T1567 (Exfiltration Over Web Service) |
| Deploy Key / PAT | New deploy key or classic PAT with repo scope | T1098.001 (Additional Cloud Credentials) |
| Webhook Creation | Webhook pointing to external non-org domain | T1041 (Exfiltration Over C2 Channel) |
| Branch Protection | Required-review or status-check rule disabled | T1195.002 (Compromise Software Supply Chain) |

## Field Mapping

SaaS events share the same `extract_field()` mapping used by all Sigma rules:

| ProcessEvent Field | SaaS Mapping |
|--------------------|--------------|
| `executable` | Operation / event type (e.g., `Add-MailboxPermission`) |
| `cmdline` | Serialized parameters / target resource |
| `user` | Actor UPN / email / service principal |
| `tenant_id` | Kubric tenant ID |
