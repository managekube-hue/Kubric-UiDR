# K-VENDOR-WAZ-005 -- Wazuh License Boundary

| Field       | Value                                     |
|-------------|-------------------------------------------|
| License     | GPL-2.0 (Wazuh Manager and Agent)         |
| Boundary    | Process / network isolation required       |

## GPL-2.0 Obligations

Wazuh is licensed under GPL-2.0. Software that links to or incorporates
GPL code must itself be distributed under GPL. Kubric avoids this
obligation through strict process isolation.

## Kubric Compliance Strategy

- **No source-level integration.** Kubric does not import, compile, or
  link any Wazuh library or module into its binaries.
- **Process isolation.** Wazuh Manager runs as an independent container.
  Wazuh agents run as standalone processes on endpoints.
- **API boundary.** CoreSec and KAI agents interact exclusively through
  the Wazuh REST API (`GET /alerts`, `GET /sca`, `PUT /active-response`).
- **Syslog forwarding.** As an alternative path, Wazuh alert JSON is
  forwarded via syslog (UDP/TCP) to Kubric's log pipeline.
- **No rule modification.** Kubric appends custom rules via Wazuh's
  `local_rules.xml` mechanism, not by patching upstream GPL source.

## Permitted Operations

| Operation                        | Method          | Compliant |
|----------------------------------|-----------------|-----------|
| Read alerts via REST API         | HTTP GET        | Yes       |
| Trigger active response via API  | HTTP PUT        | Yes       |
| Forward alerts via syslog        | Syslog/TCP      | Yes       |
| Import Wazuh Python SDK          | Python import   | No        |
| Link libwazuh into CoreSec       | C/Rust linking  | No        |

## Audit Notes

- Document all API endpoints consumed by Kubric components.
- Verify no transitive GPL dependencies in `go.mod` / `Cargo.toml`.
