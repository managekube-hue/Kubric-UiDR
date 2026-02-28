# K-PSA-BILL-005 -- PDF Invoice Generator

**Role:** Generate branded PDF invoices from billing data. Supports multi-line items, tax calculation, and white-label branding per tenant.

---

## 1. Architecture

```
┌──────────────┐  Billing data  ┌──────────────┐  PDF bytes   ┌──────────────┐
│  PostgreSQL  │───────────────►│  Go Service  │─────────────►│  MinIO       │
│  invoices    │                │  (PSA/BILL)  │              │  Object      │
│  line_items  │                │              │              │  Storage     │
│              │                │  go-pdf      │              │              │
└──────────────┘                │  template    │              │  Archive     │
                                │  render      │              │  + Download  │
                                └──────┬───────┘              └──────────────┘
                                       │ email (optional)
                                       ▼
                                ┌──────────────┐
                                │  SMTP / SES  │
                                └──────────────┘
```

---

## 2. Data Model

```go
// internal/psa/billing/invoice.go
package billing

import (
	"time"
)

// Invoice represents a billable invoice.
type Invoice struct {
	ID              string       `json:"id"`
	TenantID        string       `json:"tenant_id"`
	InvoiceNumber   string       `json:"invoice_number"` // INV-2025-0042
	Status          string       `json:"status"`         // draft, sent, paid, overdue
	IssueDate       time.Time    `json:"issue_date"`
	DueDate         time.Time    `json:"due_date"`
	Currency        string       `json:"currency"` // USD
	LineItems       []LineItem   `json:"line_items"`
	Subtotal        float64      `json:"subtotal"`
	TaxRate         float64      `json:"tax_rate"`
	TaxAmount       float64      `json:"tax_amount"`
	Discount        float64      `json:"discount"`
	Total           float64      `json:"total"`
	Notes           string       `json:"notes"`
	PaymentTerms    string       `json:"payment_terms"` // Net 30
	BillTo          BillToInfo   `json:"bill_to"`
	Branding        BrandConfig  `json:"branding"` // White-label
}

type LineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"` // SOC, NOC, vCISO, License, Project
}

type BillToInfo struct {
	CompanyName string `json:"company_name"`
	ContactName string `json:"contact_name"`
	Email       string `json:"email"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
}

type BrandConfig struct {
	CompanyName    string `json:"company_name"`
	LogoPath       string `json:"logo_path"` // MinIO path
	PrimaryColor   string `json:"primary_color"`   // hex
	AccentColor    string `json:"accent_color"`
	Website        string `json:"website"`
	SupportEmail   string `json:"support_email"`
	Address        string `json:"address"`
	TaxID          string `json:"tax_id"` // EIN or VAT
}
```

---

## 3. PDF Generator

```go
// internal/psa/billing/pdf_generator.go
package billing

import (
	"bytes"
	"fmt"
	"image/color"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// PDFGenerator creates branded PDF invoices.
type PDFGenerator struct {
	defaultBrand BrandConfig
}

func NewPDFGenerator(defaultBrand BrandConfig) *PDFGenerator {
	return &PDFGenerator{defaultBrand: defaultBrand}
}

// Generate creates a PDF invoice and returns the bytes.
func (pg *PDFGenerator) Generate(inv *Invoice) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	brand := inv.Branding
	if brand.CompanyName == "" {
		brand = pg.defaultBrand
	}

	pg.renderHeader(pdf, brand, inv)
	pg.renderBillTo(pdf, inv)
	pg.renderLineItems(pdf, inv)
	pg.renderTotals(pdf, inv)
	pg.renderFooter(pdf, brand, inv)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

func hexToRGB(hex string) (int, int, int) {
	var r, g, b int
	if len(hex) == 7 && hex[0] == '#' {
		fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	}
	return r, g, b
}

func (pg *PDFGenerator) renderHeader(pdf *gofpdf.Fpdf, brand BrandConfig, inv *Invoice) {
	r, g, b := hexToRGB(brand.PrimaryColor)

	// Company name (or logo if available)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetTextColor(r, g, b)
	pdf.CellFormat(100, 15, brand.CompanyName, "", 0, "L", false, 0, "")

	// Invoice label
	pdf.SetFont("Helvetica", "B", 28)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 15, "INVOICE", "", 1, "R", false, 0, "")

	pdf.Ln(5)

	// Company address
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(100, 4, brand.Address, "", 0, "L", false, 0, "")

	// Invoice details right side
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 4, fmt.Sprintf("Invoice #: %s", inv.InvoiceNumber), "", 1, "R", false, 0, "")
	pdf.CellFormat(100, 4, brand.Website, "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("Date: %s", inv.IssueDate.Format("January 2, 2006")), "", 1, "R", false, 0, "")
	pdf.CellFormat(100, 4, brand.SupportEmail, "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 4, fmt.Sprintf("Due: %s", inv.DueDate.Format("January 2, 2006")), "", 1, "R", false, 0, "")

	if brand.TaxID != "" {
		pdf.CellFormat(100, 4, fmt.Sprintf("Tax ID: %s", brand.TaxID), "", 0, "L", false, 0, "")
	}

	pdf.Ln(8)

	// Divider line
	pdf.SetDrawColor(r, g, b)
	pdf.SetLineWidth(0.8)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)
}

func (pg *PDFGenerator) renderBillTo(pdf *gofpdf.Fpdf, inv *Invoice) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 5, "BILL TO", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(0, 5, inv.BillTo.CompanyName, "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 4, inv.BillTo.ContactName, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 4, inv.BillTo.Address1, "", 1, "L", false, 0, "")
	if inv.BillTo.Address2 != "" {
		pdf.CellFormat(0, 4, inv.BillTo.Address2, "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(0, 4, fmt.Sprintf("%s, %s %s",
		inv.BillTo.City, inv.BillTo.State, inv.BillTo.PostalCode), "", 1, "L", false, 0, "")

	pdf.Ln(10)
}

func (pg *PDFGenerator) renderLineItems(pdf *gofpdf.Fpdf, inv *Invoice) {
	r, g, b := hexToRGB(inv.Branding.PrimaryColor)
	if inv.Branding.CompanyName == "" {
		r, g, b = hexToRGB(pg.defaultBrand.PrimaryColor)
	}

	// Table header
	pdf.SetFillColor(r, g, b)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 10)

	pdf.CellFormat(90, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(25, 8, "Category", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Unit Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(30, 8, "Amount", "1", 1, "R", true, 0, "")

	// Table rows
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(30, 30, 30)

	for i, item := range inv.LineItems {
		fillColor := color.RGBA{255, 255, 255, 255}
		if i%2 == 1 {
			fillColor = color.RGBA{245, 245, 245, 255}
		}
		pdf.SetFillColor(int(fillColor.R), int(fillColor.G), int(fillColor.B))

		pdf.CellFormat(90, 7, item.Description, "LR", 0, "L", true, 0, "")
		pdf.CellFormat(25, 7, item.Category, "LR", 0, "C", true, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprintf("%.1f", item.Quantity), "LR", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, fmt.Sprintf("$%.2f", item.UnitPrice), "LR", 0, "R", true, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("$%.2f", item.Amount), "LR", 1, "R", true, 0, "")
	}

	// Bottom border
	pdf.CellFormat(190, 0, "", "T", 1, "", false, 0, "")
	pdf.Ln(5)
}

func (pg *PDFGenerator) renderTotals(pdf *gofpdf.Fpdf, inv *Invoice) {
	x := 130.0

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(30, 30, 30)

	pdf.SetX(x)
	pdf.CellFormat(30, 6, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, fmt.Sprintf("$%.2f", inv.Subtotal), "", 1, "R", false, 0, "")

	if inv.Discount > 0 {
		pdf.SetX(x)
		pdf.CellFormat(30, 6, "Discount:", "", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("-$%.2f", inv.Discount), "", 1, "R", false, 0, "")
	}

	if inv.TaxRate > 0 {
		pdf.SetX(x)
		pdf.CellFormat(30, 6, fmt.Sprintf("Tax (%.1f%%):", inv.TaxRate), "", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("$%.2f", inv.TaxAmount), "", 1, "R", false, 0, "")
	}

	pdf.Ln(2)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetX(x)
	pdf.CellFormat(30, 8, "TOTAL:", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("$%.2f %s", inv.Total, inv.Currency), "", 1, "R", false, 0, "")
}

func (pg *PDFGenerator) renderFooter(pdf *gofpdf.Fpdf, brand BrandConfig, inv *Invoice) {
	pdf.Ln(15)

	if inv.PaymentTerms != "" {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 5, "Payment Terms", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 4, inv.PaymentTerms, "", 1, "L", false, 0, "")
		pdf.Ln(3)
	}

	if inv.Notes != "" {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 5, "Notes", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 4, inv.Notes, "", "L", false)
	}

	// Footer at bottom
	pdf.SetY(-25)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 4, fmt.Sprintf("Generated by %s on %s",
		brand.CompanyName, time.Now().Format("2006-01-02")), "", 1, "C", false, 0, "")
}
```

---

## 4. MinIO Storage Integration

```go
// internal/psa/billing/invoice_store.go
package billing

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

const invoiceBucket = "kubric-invoices"

// InvoiceStore manages PDF invoice storage in MinIO.
type InvoiceStore struct {
	minio *minio.Client
}

func NewInvoiceStore(mc *minio.Client) *InvoiceStore {
	return &InvoiceStore{minio: mc}
}

// Store saves a PDF invoice to MinIO and returns the object key.
func (is *InvoiceStore) Store(
	ctx context.Context,
	tenantID, invoiceNumber string,
	pdfBytes []byte,
) (string, error) {
	key := fmt.Sprintf("%s/%s/%s.pdf",
		tenantID,
		time.Now().Format("2006/01"),
		invoiceNumber,
	)

	_, err := is.minio.PutObject(ctx, invoiceBucket, key,
		bytes.NewReader(pdfBytes), int64(len(pdfBytes)),
		minio.PutObjectOptions{
			ContentType: "application/pdf",
			UserMetadata: map[string]string{
				"tenant-id":      tenantID,
				"invoice-number": invoiceNumber,
				"generated-at":   time.Now().UTC().Format(time.RFC3339),
			},
		})

	if err != nil {
		return "", fmt.Errorf("minio put: %w", err)
	}
	return key, nil
}

// GetPresignedURL generates a temporary download URL (valid 24 hours).
func (is *InvoiceStore) GetPresignedURL(
	ctx context.Context,
	key string,
) (string, error) {
	url, err := is.minio.PresignedGetObject(ctx, invoiceBucket, key,
		24*time.Hour, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
```

---

## 5. Batch Invoice Generation

```go
// Generate invoices for all tenants at month end.
func generateMonthlyInvoices(
	ctx context.Context,
	db *sql.DB,
	gen *PDFGenerator,
	store *InvoiceStore,
) error {
	rows, err := db.QueryContext(ctx, `
		SELECT i.id, i.tenant_id, i.invoice_number, i.issue_date, i.due_date,
		       i.subtotal, i.tax_rate, i.tax_amount, i.discount, i.total, i.currency,
		       i.notes, i.payment_terms, i.status,
		       t.name, b.company_name, b.contact_name, b.email,
		       b.address_1, b.city, b.state, b.postal_code
		FROM invoices i
		JOIN tenants t ON t.id = i.tenant_id
		JOIN billing_contacts b ON b.tenant_id = i.tenant_id AND b.is_primary = true
		WHERE i.status = 'draft'
		  AND i.issue_date = CURRENT_DATE
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var inv Invoice
		// ... scan rows into inv ...

		pdfBytes, err := gen.Generate(&inv)
		if err != nil {
			continue
		}

		_, err = store.Store(ctx, inv.TenantID, inv.InvoiceNumber, pdfBytes)
		if err != nil {
			continue
		}

		// Update invoice status to 'sent'
		db.ExecContext(ctx, `
			UPDATE invoices SET status = 'sent', sent_at = NOW()
			WHERE id = $1
		`, inv.ID)
	}
	return nil
}
```
