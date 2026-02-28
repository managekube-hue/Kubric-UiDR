# K-VENDOR-OSP-001 -- OpenSCAP Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | OpenSCAP (NIST / Red Hat)                   |
| License      | LGPL-2.1 (libraries), public domain (SCAP)  |
| Integration  | Subprocess execution (oscap CLI)            |
| Consumers    | CoreSec agent, KAI-RISK, KAI-DEPLOY         |

## Overview

OpenSCAP is a SCAP (Security Content Automation Protocol) compliance
scanner. Kubric invokes the `oscap` CLI as a subprocess to evaluate
hosts against CIS Benchmarks and DISA STIG content. SCAP data streams
(XCCDF/OVAL) are public-domain content distributed separately.

## Kubric Integration Points

- **CoreSec** runs `oscap xccdf eval` as a subprocess on managed
  endpoints and uploads ARF (Asset Reporting Format) results.
- **KAI-RISK** ingests compliance scores to compute posture-based
  risk adjustments in FAIR models.
- **KAI-DEPLOY** validates hardening baselines before approving
  change management deployments.
- **Watchdog** manages OpenSCAP binary versions and content updates.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| OSP-002        | CIS Benchmarks     |
| OSP-003        | STIG Content       |
