package psa

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// LineItem is a single line on an invoice.
type LineItem struct {
	Description string
	Quantity    float64
	UnitPrice   float64
	Amount      float64
}

// InvoiceData holds all data needed to render an invoice.
// VendorName and VendorAddress are flat strings for easy embedding in templates.
type InvoiceData struct {
	InvoiceID     string
	TenantName    string
	TenantEmail   string
	BillingPeriod string
	LineItems     []LineItem
	SubtotalUSD   float64
	TaxRate       float64 // e.g. 0.08 for 8%
	TaxUSD        float64
	TotalUSD      float64
	DueDate       time.Time
	VendorName    string
	VendorAddress string
}

// PDFRenderer renders invoices to PDF (via wkhtmltopdf) or HTML.
type PDFRenderer struct {
	wkhtmlAvailable bool
	wkhtmlPath      string
}

// NewPDFRenderer creates a PDFRenderer, detecting wkhtmltopdf in PATH.
func NewPDFRenderer() *PDFRenderer {
	r := &PDFRenderer{}
	path, err := exec.LookPath("wkhtmltopdf")
	if err == nil {
		r.wkhtmlAvailable = true
		r.wkhtmlPath = path
	}
	return r
}

// RenderInvoice generates a PDF at outputPath.
// Falls back to writing HTML when wkhtmltopdf is unavailable.
func (r *PDFRenderer) RenderInvoice(invoice *InvoiceData, outputPath string) error {
	htmlContent, err := r.RenderHTML(invoice)
	if err != nil {
		return fmt.Errorf("render html: %w", err)
	}

	if !r.wkhtmlAvailable {
		htmlPath := outputPath
		if filepath.Ext(outputPath) != ".html" {
			htmlPath = outputPath + ".html"
		}
		return os.WriteFile(htmlPath, []byte(htmlContent), 0644)
	}

	tmpHTML, err := os.CreateTemp("", "kubric-invoice-*.html")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpHTML.Name())

	if _, err := tmpHTML.WriteString(htmlContent); err != nil {
		tmpHTML.Close()
		return fmt.Errorf("write temp html: %w", err)
	}
	tmpHTML.Close()

	var stderr bytes.Buffer
	cmd := exec.Command(r.wkhtmlPath,
		"--quiet",
		"--page-size", "A4",
		"--margin-top", "20mm",
		"--margin-bottom", "20mm",
		"--margin-left", "15mm",
		"--margin-right", "15mm",
		tmpHTML.Name(), outputPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wkhtmltopdf: %w: %s", err, stderr.String())
	}
	return nil
}

// RenderHTML returns a complete HTML invoice string rendered from the embedded template.
func (r *PDFRenderer) RenderHTML(invoice *InvoiceData) (string, error) {
	tmpl, err := template.New("invoice").Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("January 2, 2006")
		},
		"formatUSD": func(f float64) string {
			return fmt.Sprintf("$%.2f", f)
		},
		"pct": func(f float64) string {
			return fmt.Sprintf("%.0f%%", f*100)
		},
	}).Parse(r.GetTemplate())
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, invoice); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// GetTemplate returns the embedded HTML invoice template with Kubric Security Inc. branding.
func (r *PDFRenderer) GetTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<style>
  * { box-sizing: border-box; }
  body { font-family: 'Helvetica Neue', Arial, sans-serif; color: #1e293b; margin: 0; padding: 0; font-size: 13px; }
  .header { background: #0f172a; color: #fff; padding: 32px 48px; display: flex; justify-content: space-between; align-items: flex-start; }
  .brand h1 { margin: 0 0 4px; font-size: 20px; letter-spacing: 2px; font-weight: 700; color: #f8fafc; }
  .brand .tagline { margin: 0; font-size: 11px; color: #94a3b8; letter-spacing: 0.5px; }
  .brand .vendor-addr { margin-top: 10px; font-size: 11px; color: #94a3b8; line-height: 1.6; }
  .inv-meta { text-align: right; }
  .inv-meta h2 { margin: 0 0 6px; font-size: 32px; font-weight: 800; color: #38bdf8; letter-spacing: 2px; }
  .inv-meta .inv-num { font-size: 13px; color: #cbd5e1; margin: 2px 0; }
  .inv-meta .inv-detail { font-size: 12px; color: #94a3b8; margin: 2px 0; }
  .badge { display: inline-block; background: #fef9c3; color: #854d0e; padding: 3px 12px; border-radius: 20px; font-size: 10px; font-weight: 700; letter-spacing: 1px; margin-top: 8px; }
  .body { padding: 36px 48px; }
  .bill-section { display: flex; gap: 80px; margin-bottom: 36px; padding-bottom: 24px; border-bottom: 1px solid #e2e8f0; }
  .bill-block h4 { margin: 0 0 8px; font-size: 10px; text-transform: uppercase; color: #64748b; letter-spacing: 1.5px; }
  .bill-block p { margin: 2px 0; font-size: 13px; color: #1e293b; }
  .bill-block .name { font-weight: 600; font-size: 14px; }
  table { width: 100%; border-collapse: collapse; margin-bottom: 28px; }
  thead tr { background: #f1f5f9; border-bottom: 2px solid #cbd5e1; }
  th { text-align: left; padding: 10px 14px; font-size: 10px; text-transform: uppercase; color: #475569; letter-spacing: 1px; font-weight: 600; }
  th.num { text-align: right; }
  td { padding: 10px 14px; border-bottom: 1px solid #f1f5f9; vertical-align: top; }
  td.num { text-align: right; color: #334155; }
  tbody tr:hover { background: #f8fafc; }
  tbody tr:last-child td { border-bottom: 2px solid #cbd5e1; }
  .totals-wrap { display: flex; justify-content: flex-end; margin-bottom: 40px; }
  .totals { width: 320px; }
  .totals table { margin-bottom: 0; }
  .totals td { padding: 8px 14px; }
  .totals .subtotal-row td { color: #475569; }
  .totals .tax-row td { color: #475569; }
  .totals .total-row { background: #0f172a; }
  .totals .total-row td { color: #fff; font-weight: 700; font-size: 15px; border-bottom: none; }
  .notes { background: #f8fafc; border-left: 4px solid #38bdf8; padding: 14px 18px; margin-bottom: 28px; border-radius: 0 6px 6px 0; }
  .notes p { margin: 0; font-size: 12px; color: #475569; line-height: 1.6; }
  .footer { background: #f1f5f9; padding: 18px 48px; border-top: 1px solid #e2e8f0; display: flex; justify-content: space-between; }
  .footer p { margin: 0; font-size: 11px; color: #64748b; }
</style>
</head>
<body>

<div class="header">
  <div class="brand">
    <h1>KUBRIC SECURITY INC.</h1>
    <p class="tagline">Managed Detection &amp; Response Platform</p>
    <div class="vendor-addr">
      <p>{{.VendorName}}</p>
      <p>{{.VendorAddress}}</p>
    </div>
  </div>
  <div class="inv-meta">
    <h2>INVOICE</h2>
    <p class="inv-num"># {{.InvoiceID}}</p>
    <p class="inv-detail">Period: {{.BillingPeriod}}</p>
    <p class="inv-detail">Due: {{formatDate .DueDate}}</p>
    <p class="inv-detail">Billed to: {{.TenantEmail}}</p>
    <span class="badge">OUTSTANDING</span>
  </div>
</div>

<div class="body">
  <div class="bill-section">
    <div class="bill-block">
      <h4>Bill To</h4>
      <p class="name">{{.TenantName}}</p>
      <p>{{.TenantEmail}}</p>
    </div>
    <div class="bill-block">
      <h4>From</h4>
      <p class="name">{{.VendorName}}</p>
      <p>{{.VendorAddress}}</p>
    </div>
  </div>

  <table>
    <thead>
      <tr>
        <th>Description</th>
        <th class="num">Qty</th>
        <th class="num">Unit Price</th>
        <th class="num">Amount</th>
      </tr>
    </thead>
    <tbody>
      {{range .LineItems}}
      <tr>
        <td>{{.Description}}</td>
        <td class="num">{{.Quantity}}</td>
        <td class="num">{{formatUSD .UnitPrice}}</td>
        <td class="num">{{formatUSD .Amount}}</td>
      </tr>
      {{end}}
    </tbody>
  </table>

  <div class="totals-wrap">
    <div class="totals">
      <table>
        <tr class="subtotal-row">
          <td>Subtotal</td>
          <td class="num">{{formatUSD .SubtotalUSD}}</td>
        </tr>
        <tr class="tax-row">
          <td>Tax ({{pct .TaxRate}})</td>
          <td class="num">{{formatUSD .TaxUSD}}</td>
        </tr>
        <tr class="total-row">
          <td>Total Due</td>
          <td class="num">{{formatUSD .TotalUSD}}</td>
        </tr>
      </table>
    </div>
  </div>

  <div class="notes">
    <p>Payment terms: Net 30 days. Wire transfer or ACH accepted. Please include
    invoice number <strong>#{{.InvoiceID}}</strong> in your payment memo.</p>
  </div>
</div>

<div class="footer">
  <p>Kubric Security Inc. &mdash; Managed Detection &amp; Response</p>
  <p>Generated {{formatDate .DueDate}}</p>
</div>

</body>
</html>`
}
