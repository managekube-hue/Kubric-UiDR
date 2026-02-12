# TMP (Technology Management) Practice Map

Maps Kubric to ITIL technology practices.

## Technology Stack Justifications

| Technology | ITIL Practice | Why |
|---|---|---|
| **Kubernetes** | Service Orchestration | Declarative state management, self-healing |
| **Terraform** | Infrastructure as Code | Version control, repeatable deployments |
| **Ansible** | Configuration Management | Agentless automation, idempotent playbooks |
| **NATS** | Event Bus | High-performance, low-latency messaging |
| **ClickHouse** | Data Lakehouse | Real-time analytics on massive datasets |
| **PostgreSQL** | Transactional Store | ACID compliance, RLS for access control |
| **Prometheus** | Monitoring | Time-series metrics, alerting rules |
| **Vault** | Secrets Management | Encryption at rest and in transit |
| **Rust + eBPF** | Performance Agents | Kernel-level monitoring with minimal overhead |
| **Python + CrewAI** | AI-Driven Analysis | LLM-based triage and decision support |

## Technology Deprecation

- **Legacy Protocol Support**: Agents support NATS protocol versions N-1 and N for backward compatibility
- **Version Sunset**: Major version support for 24 months from release
- **Migration Paths**: Documented playbooks for feature migration and deprecation announcements

---

Generated: 2026-02-12
