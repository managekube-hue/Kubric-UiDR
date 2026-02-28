# K-VENDOR-FAL-002 -- Falco System Rules

| Field       | Value                                       |
|-------------|---------------------------------------------|
| Category    | Host and container runtime detection         |
| Format      | Falco YAML rule definitions                 |
| Consumer    | PerfTrace agent, KAI-TRIAGE                 |

## Purpose

Falco rules that detect suspicious system call activity on hosts and
inside containers. PerfTrace reads Falco's JSON alert stream and
forwards matching events to NATS for downstream processing.

## Key Rule Categories

| Rule Category            | Example Rule                              |
|--------------------------|-------------------------------------------|
| Shell in container       | `Terminal shell in container`             |
| Sensitive file read      | `Read sensitive file untrusted`           |
| Binary modification      | `Write below binary dir`                  |
| Privilege escalation     | `Change thread namespace`                 |
| Reverse shell            | `Redirect STDOUT/STDIN to network`        |
| Credential access        | `Read /etc/shadow`                        |
| Cryptomining             | `Detect crypto miners using stratum`      |

## Rule Severity Mapping

| Falco Priority | Kubric Severity |
|----------------|-----------------|
| EMERGENCY      | Critical (P1)   |
| ALERT          | High (P2)       |
| WARNING        | Medium (P3)     |
| NOTICE         | Low (P4)        |

## Integration Notes

- Falco rules are Apache-2.0 licensed and may be bundled in Kubric.
- Custom Kubric rules are appended via `rules.d/` ConfigMap overlay.
- Rule updates are pulled by Watchdog from `falcosecurity/rules`.
