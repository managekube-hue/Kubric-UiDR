# Zammad Legacy Code (Archived)

**Archived:** 2025-01-XX  
**Reason:** Replaced by ERPNext as complete ITSM solution

## Why Archived

Zammad was initially considered for a "dual ITSM strategy":
- ERPNext: Customer-facing portal
- Zammad: Internal agent workflows

**Problem:** This created unnecessary complexity:
- Two ticketing systems to maintain
- Two databases (Postgres + Elasticsearch for Zammad)
- Two UIs for agents to learn
- Sync complexity between systems
- Confusing deployment story

## Solution

**ERPNext is the complete ITSM platform:**
- Customer portal (self-service)
- Agent interface (issue management)
- Billing integration
- Asset tracking
- Contract management
- Knowledge base

**n8n provides multi-channel support:**
- Email-to-ticket workflows
- Slack/Teams notifications
- Escalation automation
- SLA breach alerts

## Files Archived

- `06_K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_zammad_bridge.go`
- `06_K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go`
- `internal/psa/zammad.go` (Go client)
- `kai/psa/zammad.py` (Python client)

## If You Need Zammad

If you have a specific use case requiring Zammad:
1. Restore these files from archive
2. Add Zammad to `docker-compose.yml`:
   ```yaml
   zammad:
     image: zammad/zammad-docker-compose:latest
     ports:
       - "3000:3000"
   ```
3. Update environment variables
4. Document why ERPNext alone is insufficient

## Current Stack

```
ERPNext (port 8000)
  ├─ Customer portal
  ├─ Agent interface
  ├─ Billing
  └─ Assets/Contracts

n8n (port 5678)
  ├─ Email → ERPNext
  ├─ Slack/Teams notifications
  └─ Workflow automation

KAI
  ├─ Auto-create issues
  ├─ Generate invoices
  └─ Track SLAs
```

**Result:** Simpler deployment, single source of truth, no sync complexity.
