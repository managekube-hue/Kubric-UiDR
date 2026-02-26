# K-VENDOR-OSP-002 -- CIS Benchmark SCAP Content

| Field       | Value                                         |
|-------------|-----------------------------------------------|
| Category    | CIS hardening compliance checks                |
| Format      | XCCDF + OVAL data streams                     |
| Consumer    | CoreSec agent (subprocess), KAI-RISK           |

## Purpose

CIS Benchmark SCAP content defines configuration checks for operating
systems and applications. CoreSec executes these via `oscap xccdf eval`
and reports pass/fail results per check.

## Supported Benchmarks

| Benchmark                    | Profile                     |
|------------------------------|-----------------------------|
| CIS Ubuntu 22.04 LTS        | Level 1 Server / Level 2    |
| CIS RHEL 9                  | Level 1 Server / Level 2    |
| CIS Windows Server 2022     | Level 1 Member Server       |
| CIS Amazon Linux 2023       | Level 1                     |
| CIS Kubernetes (CIS-K8s)    | Node / Master profiles      |

## Integration Flow

1. Watchdog distributes SCAP content bundles to managed endpoints.
2. CoreSec invokes `oscap xccdf eval --profile {profile} --results-arf {output}`.
3. ARF XML results are parsed; pass/fail counts are extracted.
4. Compliance percentage is published to `kubric.coresec.compliance`.
5. KAI-RISK consumes scores to adjust vulnerability-based risk factors.

## Scoring

- Each tenant dashboard displays CIS compliance percentage per host.
- Hosts below the threshold (configurable, default 80%) trigger
  a remediation recommendation from KAI-KEEPER.
