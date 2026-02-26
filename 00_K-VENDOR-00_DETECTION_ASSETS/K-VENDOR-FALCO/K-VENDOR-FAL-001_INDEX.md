# K-VENDOR-FAL-001 -- Falco Index

| Field        | Value                                       |
|--------------|---------------------------------------------|
| Vendor       | The Falco Project / Sysdig (CNCF)           |
| License      | Apache-2.0                                  |
| Integration  | Direct integration permitted (Apache)       |
| Consumers    | PerfTrace agent, KAI-TRIAGE                 |

## Overview

Falco is a cloud-native runtime security tool that detects anomalous
activity in containers, hosts, and Kubernetes clusters using kernel-level
system call inspection. Licensed under Apache-2.0, Falco rule files and
libraries may be directly integrated into Kubric components.

## Kubric Integration Points

- **PerfTrace** consumes Falco alert output (JSON over stdout/gRPC)
  for container runtime anomaly detection in the Kubric K8s cluster.
- **KAI-TRIAGE** correlates Falco alerts with endpoint and network
  telemetry to produce composite severity scores.
- **Watchdog** deploys Falco as a DaemonSet and manages rule updates.

## Document Map

| Doc ID         | Title              |
|----------------|--------------------|
| FAL-002        | Falco Rules        |
| FAL-003        | K8s Rules          |
