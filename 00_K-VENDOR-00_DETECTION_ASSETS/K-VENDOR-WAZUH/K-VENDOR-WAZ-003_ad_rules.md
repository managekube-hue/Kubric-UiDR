# K-VENDOR-WAZ-003 -- Wazuh Active Directory Rules

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Windows AD / authentication event detection    |
| Format      | Wazuh XML rule definitions                     |
| Consumer    | CoreSec (via Wazuh API), KAI-TRIAGE            |

## Purpose

Wazuh rules that monitor Windows Security event logs for Active
Directory attacks, credential abuse, and authentication anomalies.

## Key Rule Groups

| Rule Group                | Rule ID Range | Detection Target                 |
|---------------------------|---------------|----------------------------------|
| Brute force (logon fails) | 60100-60199   | EventID 4625 threshold           |
| Kerberos anomalies        | 60200-60299   | EventID 4768/4769 ticket abuse   |
| Account manipulation      | 60300-60399   | EventID 4720/4726 user changes   |
| Group policy changes      | 60400-60499   | EventID 4739 domain policy mods  |
| Privilege use             | 60500-60599   | EventID 4672/4673 special logon  |
| DC replication (DCSync)   | 60600-60699   | EventID 4662 Repl-Secret access  |

## Example Detections

- Logon failure burst exceeding threshold (brute force / spray)
- Kerberos TGS request for SPN with RC4 encryption (Kerberoast)
- User added to Domain Admins or Enterprise Admins
- Directory Service replication from non-DC source (DCSync)
- Account lockout wave across multiple accounts

## Integration Notes

- AD event rules require Wazuh agent on domain controllers.
- Kubric reads alerts via Wazuh REST API (GPL boundary maintained).
- KAI-TRIAGE cross-references with BloodHound path data for context.
