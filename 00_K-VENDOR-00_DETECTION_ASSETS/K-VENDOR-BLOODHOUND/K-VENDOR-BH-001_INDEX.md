# K-VENDOR-BH-001 -- BloodHound Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | SpecterOps                                  |
| License      | Apache-2.0 (BloodHound CE)                  |
| Integration  | REST API + Cypher query via Neo4j/API       |
| Consumers    | KAI-HUNTER, KAI-RISK, KAI-SIMULATE          |

## Overview

BloodHound Community Edition maps Active Directory and Azure AD attack
paths using graph analysis. Kubric ingests BloodHound data via its REST
API and submits Cypher queries to identify privilege escalation paths
and misconfigurations.

## Kubric Integration Points

- **KAI-HUNTER** runs pre-built Cypher queries against BloodHound
  to discover high-risk attack paths during proactive hunts.
- **KAI-RISK** consumes attack path counts and Tier-Zero exposure
  metrics for FAIR-based quantitative risk scoring.
- **KAI-SIMULATE** uses BloodHound paths to model lateral movement
  scenarios during attack simulation exercises.
- **Watchdog** manages BloodHound CE container and data ingestion.

## Document Map

| Doc ID         | Title               |
|----------------|---------------------|
| BH-002         | Windows AD Cypher   |
| BH-003         | Azure AD Cypher     |
