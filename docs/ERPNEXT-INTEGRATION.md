# ERPNext Integration Configuration

## Overview

ERPNext is the complete ITSM solution for Kubric, providing:
- **Customer Portal** - Self-service for customers
- **ITSM** - Issue tracking and SLA management
- **Billing** - Integrated invoicing
- **Asset Management** - Track customer assets
- **Contract Management** - Service agreements
- **Knowledge Base** - Help articles

## Environment Variables

```bash
# ERPNext API
ERPNEXT_BASE_URL=http://erpnext:8000
ERPNEXT_API_KEY=<api_key>
ERPNEXT_API_SECRET=<api_secret>
```

## Usage

### Python (KAI)

```python
from kai.psa.erpnext import ERPNextClient

client = ERPNextClient(
    base_url=os.getenv("ERPNEXT_BASE_URL"),
    api_key=os.getenv("ERPNEXT_API_KEY"),
    api_secret=os.getenv("ERPNEXT_API_SECRET")
)

# Create customer issue
issue = await client.create_issue(
    subject="Network connectivity issue",
    description="Customer reports intermittent connectivity",
    customer="CUST-001",
    priority="High",
    issue_type="Incident"
)

# Get customer portal access
contracts = await client.get_customer_contracts("CUST-001")
assets = await client.get_customer_assets("CUST-001")
```

### Go (K-SVC Billing)

```go
import "github.com/managekube-hue/Kubric-UiDR/internal/psa/erpnext"

client := erpnext.NewClient(
    os.Getenv("ERPNEXT_BASE_URL"),
    os.Getenv("ERPNEXT_API_KEY"),
    os.Getenv("ERPNEXT_API_SECRET"),
)

// Create invoice
invoice, err := client.CreateSalesInvoice(ctx, &erpnext.SalesInvoice{
    Customer: "CUST-001",
    Items: []erpnext.SalesInvoiceItem{
        {
            ItemCode: "EDR-MONTHLY",
            Qty: 100,
            Rate: 5.00,
        },
    },
})
```

## Architecture

```
ERPNext (Port 8000)
  ├─ Customer Portal
  │   ├─ Self-service issue creation
  │   ├─ View contracts & assets
  │   ├─ Knowledge base access
  │   └─ Billing history
  │
  ├─ Agent Interface
  │   ├─ Issue management
  │   ├─ SLA tracking
  │   └─ Customer communication
  │
  └─ KAI Integration
      ├─ Auto-create issues from alerts
      ├─ Update issue status
      ├─ Generate invoices
      └─ Track SLAs
```

## Benefits

1. **Single ITSM Platform** - One system for all ticketing needs
2. **Customer Self-Service** - Portal for customers to manage their own tickets
3. **Integrated Billing** - Automatic invoice generation from usage
4. **Asset Tracking** - Link issues to specific customer assets
5. **Modern UI** - React-based interface
6. **Full ERP** - Can expand to inventory, HR, procurement
7. **n8n Integration** - Workflow automation for multi-channel support (email, Slack, Teams)

## Workflow Automation

Use n8n (port 5678) for:
- Email-to-ticket conversion
- Slack/Teams notifications
- Escalation workflows
- SLA breach alerts
- Customer notifications
