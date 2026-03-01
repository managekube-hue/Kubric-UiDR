# PSA - ITSM Integration

**Module:** K-PSA-06  
**Purpose:** Professional Services Automation - Ticketing and billing

---

## Overview

Kubric uses **ERPNext** as the complete ITSM platform for:
- Customer portal (self-service)
- Issue tracking
- Billing and invoicing
- Asset management
- Contract management
- Knowledge base

---

## ERPNext Integration

### Access
- **URL:** `http://erpnext:8000`
- **Admin:** Administrator
- **Customer Portal:** `http://erpnext:8000/portal`

### API Endpoints
```
Base URL: http://erpnext:8000/api
```

### Create Issue
```bash
curl -X POST http://erpnext:8000/api/resource/Issue \
  -H "Authorization: token <api_key>:<api_secret>" \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "Network connectivity issue",
    "description": "Customer reports intermittent connectivity",
    "customer": "CUST-001",
    "priority": "High",
    "issue_type": "Incident"
  }'
```

### Get Customer Issues
```bash
curl http://erpnext:8000/api/resource/Issue?filters=[["customer","=","CUST-001"]] \
  -H "Authorization: token <api_key>:<api_secret>"
```

---

## KAI Integration

### Auto-Create Issues from Alerts
```python
# KAI COMM persona
from kai.psa.erpnext import ERPNextClient

client = ERPNextClient(
    base_url="http://erpnext:8000",
    api_key=os.getenv("ERPNEXT_API_KEY"),
    api_secret=os.getenv("ERPNEXT_API_SECRET")
)

# Create issue from security alert
issue = await client.create_issue(
    subject=f"Security Alert: {alert.title}",
    description=alert.description,
    customer=alert.tenant_id,
    priority="Critical",
    issue_type="Security Incident"
)
```

### Billing Integration
```python
# KAI BILL persona - Generate invoice
invoice = await client.create_sales_invoice(
    customer="ACME-CORP",
    items=[{
        "item_code": "EDR-MONTHLY",
        "qty": 500,  # 500 endpoints
        "rate": 5.00  # $5/endpoint
    }]
)
```

---

## n8n Workflow Automation

### Email to Ticket
```
Email → n8n → ERPNext Issue
```

### Slack Notifications
```
ERPNext Issue Created → n8n → Slack Channel
```

### SLA Breach Alerts
```
ERPNext SLA Breach → n8n → VAPI Voice Call
```

---

## Customer Portal Features

### Self-Service
- Create and track issues
- View service contracts
- Access knowledge base
- Download reports
- View billing history

### Asset Management
- View deployed agents
- Check agent health
- Review security posture
- Download compliance reports

---

## Billing

### Usage Metering
```
NATS Event → ClickHouse → Aggregation → ERPNext Invoice
```

### Invoice Generation
- Monthly automated invoicing
- Usage-based pricing (per endpoint)
- Contract-based pricing (fixed)
- Custom pricing tiers

### Payment Integration
- Stripe integration
- Automatic payment processing
- Invoice email notifications

---

## Configuration

### Environment Variables
```bash
ERPNEXT_BASE_URL=http://erpnext:8000
ERPNEXT_API_KEY=<generate_in_erpnext>
ERPNEXT_API_SECRET=<generate_in_erpnext>
ERPNEXT_DB_PASSWORD=<secure_password>
ERPNEXT_ADMIN_PASSWORD=<admin_password>
```

### Generate API Keys
1. Login to ERPNext as Administrator
2. Go to Settings → API
3. Click "Generate Keys"
4. Copy API Key and API Secret
5. Add to `.env` file

---

## Related Documentation

- [Customer Portal](PORTAL.md)
- [ERPNext Integration Guide](../docs/ERPNEXT-INTEGRATION.md)
- [n8n Workflows](../architecture/ARCHITECTURE.md#n8n)
