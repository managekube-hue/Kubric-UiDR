# K-VENDOR-COR-003 -- Cortex Responders

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Active response / containment actions          |
| Format      | Cortex Responder JSON jobs via REST API        |
| Consumer    | KAI-KEEPER                                     |

## Purpose

Cortex responders execute containment and remediation actions against
infrastructure. Kubric triggers responders through the Cortex REST API
as part of automated or analyst-approved remediation workflows.

## Key Responders Used by Kubric

| Responder               | Action                          | Target        |
|--------------------------|---------------------------------|---------------|
| Mailer_1_0               | Send notification email         | SOC / client  |
| Wazuh_1_0                | Trigger active response         | Wazuh agent   |
| Velociraptor_1_0         | Launch endpoint collection      | VR server     |
| DNS_Sinkhole             | Redirect domain to sinkhole     | DNS server    |
| Firewall_Block           | Add IP to block list            | FW API        |
| TheHive_CloseAlert       | Close resolved alert in TheHive | TheHive API   |

## Integration Flow

1. KAI-KEEPER generates a remediation plan (Temporal workflow).
2. For each containment step, POSTs to `https://cortex-{tenant}/api/responder/{id}/run`.
3. Monitors job status via `api/job/{id}/waitreport`.
4. Logs responder outcome and updates case status in TheHive.
5. Publishes remediation result to `kubric.kai.keeper.result`.

## Approval Gates

- High-impact responders (firewall block, DNS sinkhole) require analyst
  approval before execution unless the tenant enables full auto-response.
- All responder invocations are audit-logged with operator identity.
