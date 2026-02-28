# K-VENDOR-BH-003 -- Azure AD Cypher Queries

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Azure / Entra ID attack path analysis          |
| Format      | Cypher queries via BloodHound REST API         |
| Consumer    | KAI-HUNTER, KAI-RISK                          |

## Purpose

Cypher queries for analyzing Azure AD (Entra ID) attack paths ingested
by AzureHound into BloodHound CE. KAI-HUNTER uses these to detect
cloud-specific privilege escalation risks.

## Key Cypher Queries

| Query Purpose                          | Returns                        |
|----------------------------------------|--------------------------------|
| Paths to Global Admin                  | Azure role escalation chains   |
| App registrations with high privileges | Apps with Directory.ReadWrite  |
| Service principals with Key Vault access | SP to secret path           |
| Users who can reset Global Admin pwd   | Password reset abuse paths     |
| Managed Identity escalation paths      | MI to subscription owner       |
| Cross-tenant trust abuse               | B2B guest to admin paths       |

## Integration Flow

1. AzureHound collector ingests Entra ID, subscriptions, and RBAC data.
2. Data is uploaded to BloodHound CE via its ingestion API.
3. KAI-HUNTER submits Azure-specific Cypher to the BloodHound API.
4. Returned paths are normalized and published on NATS.

## Scoring

- Azure attack paths contribute to tenant cloud-risk scores
  computed by KAI-RISK.
- Global Admin path count and app registration risk are surfaced
  on the KAI-HOUSE dashboard under the cloud posture widget.
- Apache-2.0 license (BloodHound CE) permits direct query bundling.
