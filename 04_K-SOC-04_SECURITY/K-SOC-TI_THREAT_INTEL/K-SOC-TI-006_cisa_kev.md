# K-SOC-TI-006 -- CISA Known Exploited Vulnerabilities (KEV) Feed

**Source:** `https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json`  
**Update Cadence:** Polled every 6 hours  
**Role:** Cross-reference VDR vulnerability findings with actively exploited CVEs for priority escalation.

---

## 1. Go Feed Poller

```go
// internal/threatintel/cisa_kev.go
package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
)

const (
	cisaKEVURL    = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	kevPollInterval = 6 * time.Hour
)

// KEVCatalog is the CISA KEV JSON structure.
type KEVCatalog struct {
	Title           string         `json:"title"`
	CatalogVersion  string         `json:"catalogVersion"`
	DateReleased    string         `json:"dateReleased"`
	Count           int            `json:"count"`
	Vulnerabilities []KEVEntry     `json:"vulnerabilities"`
}

// KEVEntry is a single KEV vulnerability.
type KEVEntry struct {
	CveID                    string `json:"cveID"`
	VendorProject            string `json:"vendorProject"`
	Product                  string `json:"product"`
	VulnerabilityName        string `json:"vulnerabilityName"`
	DateAdded                string `json:"dateAdded"`
	ShortDescription         string `json:"shortDescription"`
	RequiredAction           string `json:"requiredAction"`
	DueDate                  string `json:"dueDate"`
	KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
	Notes                    string `json:"notes"`
}

// KEVPoller polls the CISA KEV feed on a schedule.
type KEVPoller struct {
	httpClient *http.Client
	nc         *nats.Conn
	tenantID   string
	lastCount  int
	kevMap     map[string]KEVEntry // keyed by CVE ID
}

func NewKEVPoller(nc *nats.Conn, tenantID string) *KEVPoller {
	return &KEVPoller{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		nc:         nc,
		tenantID:   tenantID,
		kevMap:     make(map[string]KEVEntry),
	}
}

// FetchCatalog downloads the current KEV catalog.
func (kp *KEVPoller) FetchCatalog(ctx context.Context) (*KEVCatalog, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", cisaKEVURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Kubric-SecurityPlatform/1.0")

	resp, err := kp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kev fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kev api status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var catalog KEVCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("parse kev: %w", err)
	}

	return &catalog, nil
}

// Poll starts the polling loop.
func (kp *KEVPoller) Poll(ctx context.Context) error {
	// Initial fetch
	if err := kp.pollOnce(ctx); err != nil {
		return fmt.Errorf("initial kev poll: %w", err)
	}

	ticker := time.NewTicker(kevPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := kp.pollOnce(ctx); err != nil {
				// Log but don't exit — retry on next tick
				fmt.Fprintf(os.Stderr, "kev poll error: %v\n", err)
			}
		}
	}
}

func (kp *KEVPoller) pollOnce(ctx context.Context) error {
	catalog, err := kp.FetchCatalog(ctx)
	if err != nil {
		return err
	}

	// Detect new entries since last poll
	newEntries := make([]KEVEntry, 0)
	for _, entry := range catalog.Vulnerabilities {
		if _, exists := kp.kevMap[entry.CveID]; !exists {
			newEntries = append(newEntries, entry)
			kp.kevMap[entry.CveID] = entry
		}
	}

	// Publish new KEV entries to NATS
	for _, entry := range newEntries {
		if err := kp.publishKEV(entry); err != nil {
			continue
		}
	}

	kp.lastCount = catalog.Count
	return nil
}

func (kp *KEVPoller) publishKEV(entry KEVEntry) error {
	severity := 4 // High
	if entry.KnownRansomwareCampaignUse == "Known" {
		severity = 5 // Critical for ransomware-associated CVEs
	}

	event := map[string]interface{}{
		"class_uid":    2002,     // OCSF VulnerabilityFinding
		"activity_id":  1,       // Create
		"category_uid": 2,       // Findings
		"severity_id":  severity,
		"time":         time.Now().UTC().Format(time.RFC3339),
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("CISA KEV: %s - %s",
				entry.CveID, entry.VulnerabilityName),
			"uid":   fmt.Sprintf("kev-%s", entry.CveID),
			"types": []string{"Known Exploited Vulnerability"},
			"analytic": map[string]string{
				"name": "CISA KEV Feed",
				"type": "Threat Intelligence",
			},
		},
		"vulnerabilities": []map[string]interface{}{
			{
				"cve": map[string]string{
					"uid": entry.CveID,
				},
				"vendor_name": entry.VendorProject,
				"affected_packages": []map[string]string{
					{"name": entry.Product},
				},
				"desc":            entry.ShortDescription,
				"remediation_desc": entry.RequiredAction,
			},
		},
		"metadata": map[string]interface{}{
			"product": map[string]string{
				"name":        "CISA KEV",
				"vendor_name": "CISA",
			},
			"tenant_uid": kp.tenantID,
		},
		"unmapped": map[string]interface{}{
			"date_added":     entry.DateAdded,
			"due_date":       entry.DueDate,
			"ransomware_use": entry.KnownRansomwareCampaignUse,
			"notes":          entry.Notes,
		},
	}

	data, _ := json.Marshal(event)
	return kp.nc.Publish(
		fmt.Sprintf("kubric.ti.ioc.%s", kp.tenantID), data,
	)
}

// IsKEV checks if a CVE is in the current KEV catalog.
func (kp *KEVPoller) IsKEV(cveID string) (KEVEntry, bool) {
	entry, ok := kp.kevMap[cveID]
	return entry, ok
}
```

---

## 2. VDR Cross-Reference

```go
// internal/threatintel/kev_vdr.go
package threatintel

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
)

// KEVCrossRef cross-references VDR findings with KEV catalog.
type KEVCrossRef struct {
	kevPoller *KEVPoller
	db        *sql.DB // ClickHouse
	nc        *nats.Conn
	tenantID  string
}

func NewKEVCrossRef(kp *KEVPoller, db *sql.DB, nc *nats.Conn, tenantID string) *KEVCrossRef {
	return &KEVCrossRef{
		kevPoller: kp,
		db:        db,
		nc:        nc,
		tenantID:  tenantID,
	}
}

// EscalateKEVFindings finds VDR vulnerabilities that are in the KEV catalog
// and escalates their priority.
func (kcr *KEVCrossRef) EscalateKEVFindings(ctx context.Context) error {
	// Query ClickHouse for unresolved vulnerability findings
	rows, err := kcr.db.QueryContext(ctx, `
		SELECT
			cve_id,
			hostname,
			product,
			severity,
			discovered_at
		FROM kubric.vulnerability_findings
		WHERE tenant_id = $1
		  AND status = 'open'
		  AND cve_id != ''
		ORDER BY discovered_at DESC
	`, kcr.tenantID)
	if err != nil {
		return fmt.Errorf("query vdr findings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cveID, hostname, product, severity string
		var discoveredAt time.Time
		if err := rows.Scan(&cveID, &hostname, &product, &severity, &discoveredAt); err != nil {
			continue
		}

		kevEntry, isKEV := kcr.kevPoller.IsKEV(cveID)
		if !isKEV {
			continue
		}

		// Escalate: publish high-priority alert
		alert := map[string]interface{}{
			"class_uid":    2002,
			"activity_id":  2, // Update
			"category_uid": 2,
			"severity_id":  5, // Critical — it's KEV + in our environment
			"time":         time.Now().UTC().Format(time.RFC3339),
			"finding_info": map[string]interface{}{
				"title": fmt.Sprintf(
					"KEV ESCALATION: %s on %s — known exploited in the wild",
					cveID, hostname),
				"uid":   fmt.Sprintf("kev-escalation-%s-%s", cveID, hostname),
				"types": []string{"Known Exploited Vulnerability", "Priority Escalation"},
			},
			"vulnerabilities": []map[string]interface{}{
				{
					"cve":             map[string]string{"uid": cveID},
					"vendor_name":     kevEntry.VendorProject,
					"remediation_desc": kevEntry.RequiredAction,
				},
			},
			"metadata": map[string]interface{}{
				"product":    map[string]string{"name": "Kubric VDR", "vendor_name": "Kubric"},
				"tenant_uid": kcr.tenantID,
			},
			"unmapped": map[string]interface{}{
				"original_severity": severity,
				"escalated_because": "CISA KEV",
				"kev_due_date":      kevEntry.DueDate,
				"ransomware_use":    kevEntry.KnownRansomwareCampaignUse,
				"hostname":          hostname,
			},
		}

		data, _ := json.Marshal(alert)
		_ = kcr.nc.Publish(
			fmt.Sprintf("kubric.ti.ioc.%s", kcr.tenantID), data,
		)
	}

	return nil
}
```

---

## 3. ClickHouse Storage

```sql
-- ClickHouse table for KEV catalog data
CREATE TABLE IF NOT EXISTS kubric.cisa_kev (
    cve_id            String,
    vendor_project    String,
    product           String,
    vulnerability_name String,
    date_added        Date,
    short_description String,
    required_action   String,
    due_date          Date,
    ransomware_use    LowCardinality(String),
    notes             String,
    ingested_at       DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(ingested_at)
ORDER BY (cve_id)
TTL ingested_at + INTERVAL 365 DAY;

-- View: match KEV against VDR findings
CREATE VIEW IF NOT EXISTS kubric.kev_matched_vulns AS
SELECT
    v.cve_id,
    v.tenant_id,
    v.hostname,
    v.product,
    v.severity AS original_severity,
    'critical' AS escalated_severity,
    k.date_added AS kev_date_added,
    k.due_date AS kev_due_date,
    k.ransomware_use,
    k.required_action
FROM kubric.vulnerability_findings v
INNER JOIN kubric.cisa_kev k ON v.cve_id = k.cve_id
WHERE v.status = 'open';
```

---

## 4. Operational Notes

| Item | Value |
|------|-------|
| Feed URL | `https://www.cisa.gov/.../known_exploited_vulnerabilities.json` |
| Poll interval | Every 6 hours |
| Current catalog size | ~1,200 CVEs (growing) |
| Escalation rule | Any VDR finding matching KEV → severity bumped to Critical |
| Ransomware flag | KEVs with `knownRansomwareCampaignUse: "Known"` get highest priority |
| NATS subject | `kubric.ti.ioc.{tenant_id}` |
