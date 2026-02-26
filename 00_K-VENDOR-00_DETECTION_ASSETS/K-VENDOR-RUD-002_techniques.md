# K-VENDOR-RUD-002 -- Rudder Techniques

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | Configuration enforcement and drift detection  |
| Format      | Rudder Technique definitions (YAML/JSON)       |
| Consumer    | KAI-DEPLOY, KAI-RISK                           |

## Purpose

Rudder techniques define desired-state configuration policies that
are continuously enforced on managed nodes. Kubric uses these to
maintain security hardening baselines and detect configuration drift.

## Key Techniques Used by Kubric

| Technique                    | Enforcement Target                  |
|------------------------------|-------------------------------------|
| SSH hardening                | sshd_config parameters              |
| Firewall baseline            | iptables/nftables rule sets         |
| NTP synchronization          | chrony/ntpd configuration           |
| Log forwarding               | rsyslog/syslog-ng to SIEM pipeline  |
| User account policy          | Password complexity, sudo rules     |
| Package compliance           | Required/prohibited package lists    |
| File permissions             | Critical file ownership and modes   |
| Service management           | Enabled/disabled service whitelist   |

## Compliance States

| State         | Meaning                                    |
|---------------|--------------------------------------------|
| Compliant     | Node matches desired state                 |
| Non-compliant | Drift detected, Rudder will auto-remediate |
| Error         | Technique could not be applied             |
| N/A           | Technique does not apply to this node      |

## Integration Flow

1. Rudder agent evaluates techniques on each node every 5 minutes.
2. Compliance reports are sent to the Rudder server.
3. KAI-DEPLOY queries `GET /api/latest/compliance/nodes` via REST.
4. Non-compliant nodes block deployment approval until remediated.
5. KAI-RISK factors compliance percentages into posture scoring.
