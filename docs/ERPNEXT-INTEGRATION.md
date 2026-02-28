# ERPNext Integration Configuration

## Overview

ERPNext provides:
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

# Zammad (fallback for internal ticketing)
ZAMMAD_URL=http://zammad:3000
ZAMMAD_TOKEN=<token>
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
Customer Portal (ERPNext)
  ├─ Self-service issue creation
  ├─ View contracts & assets
  ├─ Knowledge base access
  └─ Billing history

KAI Integration
  ├─ Auto-create issues from alerts
  ├─ Update issue status
  ├─ Generate invoices
  └─ Track SLAs

Zammad (Internal)
  ├─ Agent-only tickets
  ├─ Quick triage
  └─ Multi-channel support
```

## Benefits

1. **Customer Self-Service** - Portal for customers to manage their own tickets
2. **Integrated Billing** - Automatic invoice generation from usage
3. **Asset Tracking** - Link issues to specific customer assets
4. **Modern UI** - React-based interface
5. **Full ERP** - Can expand to inventory, HR, etc.

## Migration from Zammad

Zammad remains as internal ticketing system. ERPNext handles customer-facing workflows.

**Dual ITSM Strategy:**
- ERPNext: Customer portal, billing, contracts
- Zammad: Internal agent workflows, quick triage
