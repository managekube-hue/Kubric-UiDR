# K-VENDOR-WAZ-004 -- Wazuh SCA (Security Configuration Assessment) Rules

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Host configuration compliance checks           |
| Format      | Wazuh SCA YAML policy definitions              |
| Consumer    | CoreSec (via Wazuh API), KAI-RISK              |

## Purpose

Wazuh SCA policies define configuration checks similar to CIS Benchmarks
but evaluated continuously by the Wazuh agent. Results are read from
the Wazuh REST API; no SCA engine code is imported into Kubric.

## Key SCA Policies

| Policy                        | Check Count | Target OS         |
|-------------------------------|-------------|-------------------|
| CIS Ubuntu 22.04 (via Wazuh) | ~250        | Ubuntu 22.04 LTS  |
| CIS RHEL 9 (via Wazuh)       | ~230        | RHEL 9 / Rocky 9  |
| CIS Windows Server 2022      | ~280        | Win Server 2022   |
| PCI-DSS v4 requirements      | ~45         | Multi-platform    |
| HIPAA safeguards              | ~35         | Multi-platform    |

## SCA Check Types

| Type        | Description                             |
|-------------|-----------------------------------------|
| File        | Permission, ownership, content checks   |
| Registry    | Windows registry key/value validation   |
| Process     | Required/prohibited running services     |
| Command     | Shell command output validation          |

## Integration Flow

1. Wazuh agent runs SCA scans on schedule (default every 12 hours).
2. CoreSec polls `GET /sca/{agent_id}` for latest scan results.
3. Compliance percentages are published to `kubric.coresec.compliance`.
4. KAI-RISK factors SCA scores into tenant posture risk model.
5. Failed checks below threshold trigger KAI-KEEPER remediation recs.
