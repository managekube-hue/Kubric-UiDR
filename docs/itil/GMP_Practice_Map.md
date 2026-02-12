# GMP (ITIL Guiding Principles) Practice Map

Maps Kubric to ITIL 4 Guiding Principles.

## 1. Focus on Value

**How Kubric Delivers**:
- ✅ Metrics-driven automation reduces MTTR
- ✅ Billing Clerk tracks cost per incident
- ✅ SLA tracking ensures service delivery
- ✅ Incident prioritization (P1-P4) based on business impact

**Implementation**:
- Dashboard: `Tremor` for executive reporting
- Data: ClickHouse incident metrics with cost correlation

---

## 2. Start Where You Are

**How Kubric Supports**:
- ✅ Agent registration works with existing infrastructure
- ✅ NATS integrates with current event streams
- ✅ Ansible playbooks can execute against legacy systems
- ✅ Terraform modules support brownfield environments

---

## 3. Progress Iteratively

**How Kubric Enables**:
- ✅ Workflow versioning in n8n and Temporal
- ✅ Agent update mechanism (Zstd delta patching)
- ✅ Playbook versioning in Git
- ✅ Metrics-driven feedback loops for continuous improvement

---

## 4. Collaborate and Promote Visibility

**How Kubric Facilitates**:
- ✅ Comm Agent sends notifications to all stakeholders
- ✅ Web Portal provides unified incident dashboard
- ✅ Audit logs ensure transparency and accountability
- ✅ Post-incident review data in ClickHouse

---

## 5. Think and Work Holistically

**How Kubric Integrates**:
- ✅ Multi-module architecture spans security, ops, and business
- ✅ NATS event bus connects all components
- ✅ Temporal workflows orchestrate cross-module actions
- ✅ CISO Assistant provides holistic compliance view

---

## 6. Keep It Simple and Practical

**How Kubric Maintains Simplicity**:
- ✅ Kubectl applies entire stack via `kustomization.yaml`
- ✅ Terraform single-command provisioning
- ✅ Pre-built Ansible playbooks reduce complexity
- ✅ OpenAPI spec provides simple integration contract

---

## 7. Optimize and Automate

**How Kubric Optimizes**:
- ✅ eBPF agents minimize CPU overhead
- ✅ Housekeeper Agent automates remediation
- ✅ Billing Clerk automates invoice generation
- ✅ NATS JetStream optimizes event throughput

---

Generated: 2026-02-12
