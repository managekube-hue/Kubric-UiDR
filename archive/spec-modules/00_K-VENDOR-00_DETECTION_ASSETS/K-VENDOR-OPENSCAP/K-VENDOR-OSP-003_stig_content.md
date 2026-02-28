# K-VENDOR-OSP-003 -- DISA STIG SCAP Content

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | DISA STIG compliance checks (DoD baselines)    |
| Format      | XCCDF + OVAL data streams                     |
| Consumer    | CoreSec agent (subprocess), KAI-RISK           |

## Purpose

DISA STIG (Security Technical Implementation Guide) content provides
DoD-mandated security configuration baselines. Required for tenants in
government and defense verticals.

## Supported STIG Content

| STIG                            | Version Tracking          |
|---------------------------------|---------------------------|
| RHEL 9 STIG                    | Quarterly DISA releases   |
| Ubuntu 22.04 STIG              | Quarterly DISA releases   |
| Windows Server 2022 STIG       | Quarterly DISA releases   |
| Apache HTTP Server STIG        | Quarterly DISA releases   |
| PostgreSQL STIG                 | Quarterly DISA releases   |

## Severity Categories

| CAT    | DISA Severity | Kubric Mapping  |
|--------|---------------|-----------------|
| CAT I  | High          | Critical (P1)   |
| CAT II | Medium        | High (P2)       |
| CAT III| Low           | Medium (P3)     |

## Integration Flow

1. STIG content is pulled from DISA's public SCAP repository.
2. CoreSec runs `oscap xccdf eval --stig-viewer` for each profile.
3. Results are parsed into per-finding pass/fail/not-applicable.
4. CAT I open findings generate automatic alerts on NATS.
5. KAI-RISK weights CAT I findings heavily in posture scoring.

## Notes

- STIG content is public domain (U.S. Government work product).
- Tenants may select STIG or CIS profiles independently.
