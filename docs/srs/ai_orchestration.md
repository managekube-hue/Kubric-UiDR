# AI Orchestration & Automation Specification

## K-KAI-03: Orchestration Module

### Architecture Overview

The KAI module provides AI-driven orchestration using a crew-based approach:

```
Event Stream (NATS) 
     │
     ├─→ Triage Agent (CrewAI + Llama 3.1)
     │   └─→ Analyzes incidents, rates severity
     │
     ├─→ Housekeeper Agent (Ansible-Runner)
     │   └─→ Executes remediation playbooks
     │
     ├─→ Billing Clerk Agent (ClickHouse queries)
     │   └─→ Aggregates usage, calculates costs
     │
     └─→ Comm Agent (Vapi/Deepgram/Twilio)
         └─→ Sends notifications, escalations
```

### Triage Agent

**Framework**: CrewAI with Llama 3.1 LLM

**Responsibilities**:
- Analyze incoming security alerts
- Extract IOCs (Indicators of Compromise)
- Cross-reference with threat intelligence
- Assign severity (P1/P2/P3/P4)
- Generate runbook recommendations

**Input**: Raw security event from ClickHouse

**Output**: Triage result with:
- `severity` (1-4, 1=Critical)
- `iocs` (list of extracted indicators)
- `recommendations` (suggested remediation steps)
- `confidence` (0.0-1.0)

**Performance**: <5 seconds per incident (p95)

### Housekeeper Agent

**Framework**: Ansible-Runner + Composio

**Responsibilities**:
- Execute approved remediation playbooks
- Apply firewall rules
- Isolate compromised hosts
- Apply patches
- Restart services

**Playbooks** (Ansible-driven):
- `patch_cve.yml`: Apply security patches with safe reboot
- `isolate_host.yml`: Drain and cordon K8s node
- `restart_service.yml`: Restart service with health checks
- `rollback.yml`: Rollback deployment revision
- `deploy_agent.yml`: Deploy new agent version

**Integration**: Executes based on Triage Agent recommendations

### Billing Clerk Agent

**Framework**: LangChain + Pan… no, pure Go/ClickHouse

**Responsibilities**:
- Aggregate metered usage events
- Calculate costs per customer
- Generate invoice data
- Track SLA compliance

**Data Source**: ClickHouse queries on heartbeat stream

**Output**:
- Daily usage aggregates
- Invoice line items
- SLA breach notifications

### Comm Agent

**Frameworks**: Vapi (voice), Deepgram (transcription), Twilio (SMS)

**Responsibilities**:
- Send escalation notifications
- Place hands-free voice calls for P1 incidents
- Send SMS alerts to on-call
- Post to incident channel (Slack/Teams)

**Triggers**:
- P1 incidents → immediate voice call
- P2 incidents → SMS + Slack
- P3 incidents → Slack only

### Workflow Orchestration (n8n)

**n8n Workflows** provide visual automation:

1. **Security Alert Triage**: Alert → Triage Agent → Storage
2. **Host Drift Remediation**: Drift Detection → Housekeeper → Ansible
3. **Billing Reconciliation**: Heartbeat Events → Clerk Agent → Invoice

### Temporal.io Workflow Engine

For long-running, stateful orchestrations:

**State Machine Example** (Incident Lifecycle):
```
Created 
  ↓
Triaged ← (Triage Agent)
  ↓
Remediation ← (Housekeeper Agent)
  ↓
Resolved ← (Manual approval)
  ↓
Closed
```

**Features**:
- Automatic retries with exponential backoff
- Timeout handling
- Signal-based approval workflows
- Activity timeout detection

### CISO Assistant (RAG)

Framework**: LangChain + ClickHouse Vector Search

**Purpose**: Proof of concept for RAG-based compliance advisor.

**Knowledge Base**:
- NIST 800-53 controls
- OSCAL compliance mappings
- Internal policy documents
- Vulnerability research

**Queries**:
- User: "How do we control unauthorized software?"
- Assistant: Retrieves NIST AC-2 + company policy + relevant CVEs

---

Generated: 2026-02-12
