"""
K-KAI Billing: Invoice Renderer
Renders tenant bills into structured JSON invoices suitable for
PDF generation, email delivery, and accounting system ingestion (ANSI X12 810).
"""
from __future__ import annotations
import json, logging, os, uuid
from dataclasses import dataclass, field, asdict
from datetime import datetime, date, timezone, timedelta
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)

@dataclass
class InvoiceAddress:
    name:       str
    company:    str
    street:     str
    city:       str
    state:      str
    zip_code:   str
    country:    str = "US"

@dataclass
class InvoiceLineItem:
    item_no:    int
    description: str
    quantity:   float
    unit:       str
    unit_price: float
    amount:     float

@dataclass
class Invoice:
    invoice_id:     str
    tenant_id:      str
    billing_month:  str
    issue_date:     str
    due_date:       str
    billing_from:   InvoiceAddress
    billing_to:     InvoiceAddress
    line_items:     List[InvoiceLineItem]
    subtotal:       float
    discount_pct:   float
    discount_amount: float
    tax_pct:        float
    tax_amount:     float
    total:          float
    currency:       str = "USD"
    payment_terms:  str = "Net 30"
    status:         str = "pending"
    notes:          str = ""

VENDOR_ADDRESS = InvoiceAddress(
    name    = "Kubric Platform",
    company = "Kubric Security Inc.",
    street  = "123 Cyber Way",
    city    = "San Francisco",
    state   = "CA",
    zip_code= "94105",
    country = "US",
)

def render_invoice(
    tenant_id:    str,
    billing_month: str,
    line_items:   List[Dict[str, Any]],
    subtotal:     float,
    discount_pct: float,
    discount_amt: float,
    tax_pct:      float,
    tax_amount:   float,
    total:        float,
    tenant_info:  Optional[Dict] = None,
) -> Invoice:
    invoice_id = f"INV-{billing_month.replace('-', '')}-{str(uuid.uuid4())[:8].upper()}"
    issue_date = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    due_date   = (datetime.now(timezone.utc) + timedelta(days=30)).strftime("%Y-%m-%d")

    ti = tenant_info or {}
    billing_to = InvoiceAddress(
        name    = ti.get("contact_name",    "Account Administrator"),
        company = ti.get("company_name",    f"Tenant {tenant_id}"),
        street  = ti.get("billing_street",  ""),
        city    = ti.get("billing_city",    ""),
        state   = ti.get("billing_state",   ""),
        zip_code= ti.get("billing_zip",     ""),
        country = ti.get("billing_country", "US"),
    )

    items = [
        InvoiceLineItem(
            item_no=i+1,
            description=li["description"],
            quantity=li["quantity"],
            unit=li["unit"],
            unit_price=li["unit_price"],
            amount=li["total"],
        ) for i, li in enumerate(line_items)
    ]

    return Invoice(
        invoice_id=invoice_id,
        tenant_id=tenant_id,
        billing_month=billing_month,
        issue_date=issue_date,
        due_date=due_date,
        billing_from=VENDOR_ADDRESS,
        billing_to=billing_to,
        line_items=items,
        subtotal=subtotal,
        discount_pct=discount_pct,
        discount_amount=discount_amt,
        tax_pct=tax_pct,
        tax_amount=tax_amount,
        total=total,
    )

def invoice_to_dict(inv: Invoice) -> Dict[str, Any]:
    d = asdict(inv)
    d["line_items"]    = [asdict(li) for li in inv.line_items]
    d["billing_from"]  = asdict(inv.billing_from)
    d["billing_to"]    = asdict(inv.billing_to)
    return d

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    demo_items = [
        {"description": "Platform fee",    "quantity": 1,    "unit": "month",      "unit_price": 99.00,  "total": 99.00},
        {"description": "Agent seats",     "quantity": 25,   "unit": "agents/mo",  "unit_price": 25.00,  "total": 625.00},
        {"description": "Events 5M",       "quantity": 5.0,  "unit": "million",    "unit_price": 10.00,  "total": 50.00},
    ]
    inv = render_invoice(
        tenant_id="demo-tenant",
        billing_month="2025-01",
        line_items=demo_items,
        subtotal=774.00, discount_pct=5.0, discount_amt=38.70,
        tax_pct=0.0, tax_amount=0.0, total=735.30,
    )
    print(json.dumps(invoice_to_dict(inv), indent=2))
