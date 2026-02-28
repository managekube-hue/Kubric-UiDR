# ITSM Simplification: ERPNext Only

**Date:** 2025-01-XX  
**Decision:** Remove Zammad, use ERPNext as complete ITSM solution  
**Rationale:** Deployment simplicity, single source of truth, customer experience

---

## Problem: Dual ITSM Complexity

### Original "Strategy"
- **ERPNext** (port 8000): Customer portal, billing, assets, contracts
- **Zammad** (port 3000): Internal agent workflows, quick triage

### Real-World Deployment Issues

1. **Customer Confusion**
   - "Which URL do I use to log in?"
   - "Why are there two ticketing systems?"
   - "Where do I see my billing?"

2. **Agent Confusion**
   - "Do I create tickets in ERPNext or Zammad?"
   - "Which system has the latest status?"
   - "How do I sync between them?"

3. **Deployment Complexity**
   - Zammad requires: Postgres + Elasticsearch + Redis
   - ERPNext requires: MariaDB
   - Total: 5 additional services for "dual ITSM"

4. **Integration Overhead**
   - KAI needs to sync tickets between systems
   - Billing only in ERPNext, tickets in both
   - Customer sees partial data in each system

5. **Documentation Burden**
   - "Use ERPNext for customer-facing tickets"
   - "Use Zammad for internal tickets"
   - "Sync happens via KAI COMM persona"
   - **Result:** Nobody understands the system

---

## Solution: ERPNext Complete ITSM

### Single Platform Benefits

✅ **Customer Portal** - Self-service ticket creation, knowledge base  
✅ **Agent Interface** - Full issue management, SLA tracking  
✅ **Billing Integration** - Automatic invoicing from usage  
✅ **Asset Management** - Link tickets to customer infrastructure  
✅ **Contract Management** - Service agreements, SLAs  
✅ **Modern UI** - React-based, mobile-friendly  
✅ **Full ERP** - Can expand to inventory, HR, procurement

### Multi-Channel via n8n

Instead of Zammad for "multi-channel support", use **n8n workflows**:

```
Email → n8n → ERPNext Issue
Slack → n8n → ERPNext Issue
Teams → n8n → ERPNext Issue
Phone → Vapi → n8n → ERPNext Issue
```

**Benefits:**
- No separate ticketing system
- Flexible workflow automation
- Single source of truth (ERPNext)
- Already in the stack (port 5678)

---

## Architecture Comparison

### Before (Dual ITSM)

```
Customer → ERPNext (port 8000) → Billing ✓
                                → Tickets ✓
                                → Assets ✓

Agent → Zammad (port 3000) → Tickets ✓
                            → Billing ✗ (must check ERPNext)
                            → Assets ✗ (must check ERPNext)

KAI → Sync between ERPNext ↔ Zammad
    → Create tickets in both systems
    → Update status in both systems
    → Confusion everywhere
```

**Services:** ERPNext + erpnext-db + Zammad + zammad-db + zammad-es + zammad-redis = **6 services**

### After (ERPNext Only)

```
Customer → ERPNext (port 8000) → Billing ✓
                                → Tickets ✓
                                → Assets ✓
                                → Portal ✓

Agent → ERPNext (port 8000) → Tickets ✓
                             → Billing ✓
                             → Assets ✓
                             → Customer context ✓

KAI → ERPNext only
    → Single API client
    → No sync complexity
    → Clear data flow

n8n → Email/Slack/Teams → ERPNext
    → Workflow automation
    → Multi-channel support
```

**Services:** ERPNext + erpnext-db = **2 services**

---

## Deployment Impact

### Before
```bash
docker compose up -d
# Wait for 15+ services to start
# Configure ERPNext
# Configure Zammad
# Configure KAI to sync between them
# Explain to customer which URL to use
# Train agents on two systems
```

### After
```bash
docker compose up -d
# Wait for 10 services to start
# Configure ERPNext
# Done
```

**Reduction:** 5 fewer services, 1 system to learn, 1 source of truth

---

## Migration Path

### For New Deployments
- Use ERPNext only (already in docker-compose.yml)
- Configure n8n workflows for multi-channel
- Train agents on ERPNext interface

### For Existing Zammad Users
1. Export tickets from Zammad
2. Import into ERPNext via REST API
3. Update KAI to use ERPNext client only
4. Remove Zammad from docker-compose.yml
5. Archive Zammad client code (done)

---

## Code Changes

### Archived
- `06_K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_zammad_bridge.go`
- `06_K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go`
- `internal/psa/zammad.go` (Go client)
- `kai/psa/zammad.py` (Python client)

**Location:** `archive/zammad-legacy/` with README explaining decision

### Active
- `internal/psa/erpnext.go` (Go client)
- `kai/psa/erpnext.py` (Python client)
- `docker-compose.yml` (ERPNext services)
- `docs/ERPNEXT-INTEGRATION.md` (configuration guide)

---

## KAI Integration

### Before (Dual ITSM)
```python
# KAI COMM persona
from kai.psa.zammad import ZammadClient
from kai.psa.erpnext import ERPNextClient

# Create ticket in Zammad for internal
zammad = ZammadClient(...)
internal_ticket = await zammad.create_ticket(...)

# Create issue in ERPNext for customer
erpnext = ERPNextClient(...)
customer_issue = await erpnext.create_issue(...)

# Sync status between systems
await sync_ticket_status(internal_ticket, customer_issue)
```

### After (ERPNext Only)
```python
# KAI COMM persona
from kai.psa.erpnext import ERPNextClient

# Create issue in ERPNext (visible to customer and agents)
erpnext = ERPNextClient(...)
issue = await erpnext.create_issue(
    subject="Critical: Database down",
    description="Customer ACME Corp database unreachable",
    customer="ACME-CORP",
    priority="Critical",
    issue_type="Incident"
)
# Done. No sync needed.
```

**Result:** 50% less code, zero sync complexity, single source of truth

---

## Customer Experience

### Before
**Customer:** "I got an email about a ticket, but when I log into the portal, I don't see it."  
**Support:** "That's an internal ticket in Zammad. You need to check ERPNext for customer tickets."  
**Customer:** "Why do you have two systems?"  
**Support:** "It's our dual ITSM strategy..."  
**Customer:** *confused*

### After
**Customer:** "I got an email about a ticket."  
**Support:** "Log into ERPNext at https://portal.kubric.security"  
**Customer:** *sees ticket, billing, assets, contracts, knowledge base*  
**Customer:** "This is great!"

---

## Decision Summary

**Remove Zammad because:**
1. ERPNext already provides complete ITSM (customer + agent)
2. n8n already provides multi-channel automation
3. Dual systems confuse customers and agents
4. Deployment complexity hurts adoption
5. Sync complexity creates bugs
6. Single source of truth is better architecture

**Keep ERPNext because:**
1. Customer portal (self-service)
2. Agent interface (full ITSM)
3. Billing integration (automatic invoicing)
4. Asset management (infrastructure tracking)
5. Contract management (SLAs)
6. Modern UI (React)
7. Full ERP (future expansion)

**Result:** Simpler deployment, better UX, clearer architecture, easier to explain.

---

**Status:** ✅ Zammad archived, ERPNext is complete ITSM solution  
**Commit:** [pending]  
**Files Changed:** 8 (4 archived, 2 docs updated, 1 README created, 1 ARCHITECTURE.md updated)
