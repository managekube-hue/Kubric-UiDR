# K-VENDOR-BH-002 -- Windows AD Cypher Queries

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Active Directory attack path analysis          |
| Format      | Cypher queries via BloodHound REST API         |
| Consumer    | KAI-HUNTER, KAI-RISK                          |

## Purpose

Pre-built Cypher queries that KAI-HUNTER executes against BloodHound to
detect dangerous Active Directory privilege escalation paths, Kerberos
misconfigurations, and Tier-Zero exposure.

## Key Cypher Queries

| Query Purpose                     | Returns                          |
|-----------------------------------|----------------------------------|
| Shortest path to Domain Admin     | Attack path node chains          |
| Kerberoastable service accounts   | SPNs with weak encryption        |
| AS-REP roastable users            | Accounts without pre-auth        |
| Unconstrained delegation hosts    | Machines with TGT forwarding     |
| Users with DCSync rights          | Non-DA accounts with Repl perms  |
| GPO abuse paths                   | GPO-linked escalation chains     |
| Tier-Zero asset enumeration       | DA, EA, Schema Admin members     |

## Integration Flow

1. SharpHound collector uploads AD graph data to BloodHound CE.
2. KAI-HUNTER POSTs Cypher queries to `https://bh-{tenant}/api/v2/graphs/cypher`.
3. Parses returned path objects into structured findings.
4. Publishes AD risk findings to `kubric.kai.hunter.findings`.

## Scoring

- Path length and number of distinct paths to Tier-Zero assets feed
  into KAI-RISK's FAIR loss-event-frequency calculation.
- Kerberoastable and AS-REP roastable account counts are tracked as
  KPIs on the KAI-HOUSE SOC dashboard.
