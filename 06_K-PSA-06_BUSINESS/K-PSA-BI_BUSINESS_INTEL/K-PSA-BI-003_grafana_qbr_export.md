# K-PSA-BI-003 -- Grafana QBR Dashboard Export

**Role:** Grafana dashboard definitions for QBR metrics, plus automated PDF/PNG snapshot export for inclusion in Quarterly Business Review documents.

---

## 1. Architecture

```
┌──────────────┐  Queries    ┌──────────────┐  HTTP API    ┌──────────────┐
│  ClickHouse  │────────────►│  Grafana      │◄────────────►│  Go Service  │
│  PostgreSQL  │  datasources│  Dashboards   │  Render API  │  (PSA/BI)    │
│              │             │              │  /render/d/  │              │
│              │             │  QBR Panels   │              │  Export PDFs │
└──────────────┘             └──────────────┘              └──────────────┘
```

---

## 2. Grafana Dashboard JSON Model

```json
{
  "dashboard": {
    "id": null,
    "uid": "kubric-qbr-overview",
    "title": "Kubric QBR Overview",
    "tags": ["qbr", "business-review", "kubric"],
    "timezone": "utc",
    "editable": false,
    "templating": {
      "list": [
        {
          "name": "tenant_id",
          "type": "query",
          "datasource": "PostgreSQL",
          "query": "SELECT id AS __value, name AS __text FROM tenants WHERE active = true ORDER BY name",
          "refresh": 1
        },
        {
          "name": "quarter",
          "type": "custom",
          "options": [
            {"text": "2025-Q1", "value": "2025-Q1"},
            {"text": "2025-Q2", "value": "2025-Q2"},
            {"text": "2025-Q3", "value": "2025-Q3"},
            {"text": "2025-Q4", "value": "2025-Q4"}
          ],
          "current": {"text": "2025-Q1", "value": "2025-Q1"}
        }
      ]
    },
    "panels": [
      {
        "id": 1,
        "title": "Overall Security Score",
        "type": "gauge",
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT avg(score) AS value FROM kubric.tenant_scores WHERE tenant_id = '${tenant_id}' AND month >= toDate(toStartOfQuarter(now()))",
            "format": "table"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "min": 0,
            "max": 100,
            "thresholds": {
              "steps": [
                {"value": 0, "color": "red"},
                {"value": 50, "color": "orange"},
                {"value": 70, "color": "yellow"},
                {"value": 90, "color": "green"}
              ]
            }
          }
        }
      },
      {
        "id": 2,
        "title": "Monthly Alert Volume",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 6, "y": 0},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT toStartOfMonth(event_time) AS time, count() AS alerts, countIf(severity_id >= 4) AS critical FROM kubric.security_events WHERE tenant_id = '${tenant_id}' GROUP BY time ORDER BY time",
            "format": "time_series"
          }
        ]
      },
      {
        "id": 3,
        "title": "MTTD / MTTR Trend",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 6, "x": 18, "y": 0},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT toStartOfMonth(event_time) AS time, avg(time_to_detect_ms)/60000 AS mttd_min, avg(time_to_respond_ms)/60000 AS mttr_min FROM kubric.security_events WHERE tenant_id = '${tenant_id}' AND time_to_detect_ms > 0 GROUP BY time ORDER BY time",
            "format": "time_series"
          }
        ]
      },
      {
        "id": 4,
        "title": "Vulnerability Burn-down",
        "type": "barchart",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 8},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT toStartOfMonth(event_time) AS time, countIf(activity = 'discovered') AS discovered, countIf(activity = 'remediated') AS remediated FROM kubric.vulnerability_findings WHERE tenant_id = '${tenant_id}' GROUP BY time ORDER BY time",
            "format": "time_series"
          }
        ]
      },
      {
        "id": 5,
        "title": "Profitability Trend",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 8},
        "datasource": "PostgreSQL",
        "targets": [
          {
            "rawSql": "SELECT month AS time, revenue, cost, margin_pct FROM mv_monthly_profitability WHERE tenant_id = '${tenant_id}' ORDER BY month",
            "format": "time_series"
          }
        ]
      },
      {
        "id": 6,
        "title": "Endpoint Compliance",
        "type": "piechart",
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 16},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT compliance_status AS metric, count() AS value FROM kubric.endpoint_inventory WHERE tenant_id = '${tenant_id}' GROUP BY compliance_status",
            "format": "table"
          }
        ]
      },
      {
        "id": 7,
        "title": "Backup Success Rate",
        "type": "stat",
        "gridPos": {"h": 4, "w": 6, "x": 6, "y": 16},
        "datasource": "ClickHouse",
        "targets": [
          {
            "rawSql": "SELECT countIf(status='success') * 100.0 / count() AS value FROM kubric.backup_jobs WHERE tenant_id = '${tenant_id}' AND start_time >= toStartOfQuarter(now())",
            "format": "table"
          }
        ],
        "fieldConfig": {
          "defaults": {
            "unit": "percent",
            "thresholds": {
              "steps": [
                {"value": 0, "color": "red"},
                {"value": 95, "color": "yellow"},
                {"value": 99, "color": "green"}
              ]
            }
          }
        }
      }
    ]
  }
}
```

---

## 3. Go Export Service

```go
// internal/psa/qbr/grafana_export.go
package qbr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const grafanaBaseURL = "http://grafana.internal:3000"

// GrafanaExporter renders Grafana dashboards to PNG/PDF.
type GrafanaExporter struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	outputDir  string
}

func NewGrafanaExporter(apiKey, outputDir string) *GrafanaExporter {
	return &GrafanaExporter{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		baseURL:    grafanaBaseURL,
		apiKey:     apiKey,
		outputDir:  outputDir,
	}
}

// ExportDashboardPNG renders a dashboard to a PNG image.
func (ge *GrafanaExporter) ExportDashboardPNG(
	ctx context.Context,
	dashboardUID string,
	tenantID string,
	quarter string,
	width, height int,
) (string, error) {
	params := url.Values{
		"var-tenant_id": {tenantID},
		"var-quarter":   {quarter},
		"width":         {fmt.Sprint(width)},
		"height":        {fmt.Sprint(height)},
		"theme":         {"light"},
	}

	renderURL := fmt.Sprintf("%s/render/d/%s?%s", ge.baseURL, dashboardUID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", renderURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+ge.apiKey)

	resp, err := ge.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("grafana render: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("grafana render %d: %s", resp.StatusCode, string(body))
	}

	filename := fmt.Sprintf("qbr-%s-%s-%s.png", tenantID[:8], quarter,
		time.Now().Format("20060102"))
	outputPath := filepath.Join(ge.outputDir, filename)

	f, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	return outputPath, nil
}

// ExportPanelPNG renders a specific panel to PNG.
func (ge *GrafanaExporter) ExportPanelPNG(
	ctx context.Context,
	dashboardUID string,
	panelID int,
	tenantID string,
	width, height int,
) (string, error) {
	params := url.Values{
		"var-tenant_id": {tenantID},
		"panelId":       {fmt.Sprint(panelID)},
		"width":         {fmt.Sprint(width)},
		"height":        {fmt.Sprint(height)},
		"theme":         {"light"},
	}

	renderURL := fmt.Sprintf("%s/render/d-solo/%s?%s",
		ge.baseURL, dashboardUID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", renderURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+ge.apiKey)

	resp, err := ge.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	filename := fmt.Sprintf("panel-%d-%s-%s.png",
		panelID, tenantID[:8], time.Now().Format("20060102"))
	outputPath := filepath.Join(ge.outputDir, filename)

	f, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	io.Copy(f, resp.Body)
	return outputPath, nil
}

// ExportQBRPanels exports all QBR panels for a tenant.
func (ge *GrafanaExporter) ExportQBRPanels(
	ctx context.Context,
	tenantID string,
	quarter string,
) ([]string, error) {
	dashboardUID := "kubric-qbr-overview"
	panels := []struct {
		ID     int
		Width  int
		Height int
	}{
		{1, 600, 400},   // Security Score gauge
		{2, 1200, 400},  // Alert Volume
		{3, 600, 400},   // MTTD/MTTR
		{4, 1200, 400},  // Vuln Burn-down
		{5, 1200, 400},  // Profitability
		{6, 600, 400},   // Endpoint Compliance
		{7, 600, 200},   // Backup Success
	}

	var paths []string
	for _, p := range panels {
		path, err := ge.ExportPanelPNG(ctx, dashboardUID, p.ID, tenantID, p.Width, p.Height)
		if err != nil {
			continue
		}
		paths = append(paths, path)
	}

	return paths, nil
}
```

---

## 4. Grafana Provisioning

```yaml
# config/grafana/provisioning/dashboards/kubric.yaml
apiVersion: 1
providers:
  - name: Kubric QBR
    orgId: 1
    folder: QBR
    type: file
    disableDeletion: true
    updateIntervalSeconds: 60
    options:
      path: /var/lib/grafana/dashboards/qbr
      foldersFromFilesStructure: false
```

```yaml
# config/grafana/provisioning/datasources/kubric.yaml
apiVersion: 1
datasources:
  - name: ClickHouse
    type: grafana-clickhouse-datasource
    access: proxy
    url: http://clickhouse.internal:8123
    jsonData:
      defaultDatabase: kubric
      protocol: native
      port: 9000
    isDefault: true

  - name: PostgreSQL
    type: postgres
    access: proxy
    url: postgres.internal:5432
    jsonData:
      database: kubric
      sslmode: require
    secureJsonData:
      password: ${POSTGRES_PASSWORD}
```

---

## 5. Automated Export Schedule

```go
// Run QBR export on the 5th of each quarter's first month
func scheduleQBRExport(exporter *GrafanaExporter, tenants []TenantInfo) {
    ticker := time.NewTicker(24 * time.Hour)
    for t := range ticker.C {
        // Only on the 5th day of Jan, Apr, Jul, Oct
        month := t.Month()
        if t.Day() != 5 || (month != 1 && month != 4 && month != 7 && month != 10) {
            continue
        }

        quarter := fmt.Sprintf("%d-Q%d", t.Year(), (int(month)-1)/3)
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)

        for _, tenant := range tenants {
            paths, _ := exporter.ExportQBRPanels(ctx, tenant.ID, quarter)
            // paths are collected and included in the QBR PDF
            _ = paths
        }
        cancel()
    }
}
```
