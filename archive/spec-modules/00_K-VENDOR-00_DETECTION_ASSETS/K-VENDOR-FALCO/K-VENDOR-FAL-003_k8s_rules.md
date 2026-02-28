# K-VENDOR-FAL-003 -- Falco Kubernetes Rules

| Field       | Value                                       |
|-------------|---------------------------------------------|
| Category    | Kubernetes audit log and workload detection  |
| Format      | Falco YAML rule definitions (k8s_audit)     |
| Consumer    | PerfTrace agent, KAI-TRIAGE                 |

## Purpose

Falco rules that consume Kubernetes audit log events to detect
suspicious cluster-level activity. These complement the syscall-based
rules in FAL-002 with API-server-level visibility.

## Key K8s Audit Rules

| Rule                                    | MITRE Mapping          |
|-----------------------------------------|------------------------|
| `K8s ServiceAccount created`            | Persistence            |
| `K8s Role/ClusterRole created`          | Privilege Escalation   |
| `Pod created in kube-system`            | Defense Evasion        |
| `Attach/Exec to privileged pod`         | Execution              |
| `ConfigMap with sensitive data`         | Credential Access      |
| `NodePort service created`              | Lateral Movement       |
| `Anonymous request allowed`             | Initial Access         |

## Integration Flow

1. Kubernetes API server sends audit events to Falco via webhook.
2. Falco evaluates k8s_audit rules against each event.
3. PerfTrace captures Falco k8s alerts from the gRPC output stream.
4. Alerts are published to `kubric.perftrace.k8s.alert` on NATS.
5. KAI-TRIAGE correlates with pod/node context for scoring.

## Deployment

- Falco is deployed as a DaemonSet managed by Watchdog.
- K8s audit policy is configured to forward relevant verbs
  (create, update, patch, delete) for security-sensitive resources.
- Apache-2.0 license permits direct rule bundling.
