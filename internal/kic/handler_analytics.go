// handler_analytics.go — KIC HTTP handlers for DuckDB analytical queries.
package kic

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/managekube-hue/Kubric-UiDR/internal/analytics"
)

type analyticsHandler struct {
	engine *analytics.Engine
}

func newAnalyticsHandler(e *analytics.Engine) *analyticsHandler {
	return &analyticsHandler{engine: e}
}

// ingestEvent adds an event to the analytics store.
//
//	POST /analytics/events
//	Body: {"id":"...","tenant_id":"...","source":"noc","severity":"high","category":"auth","summary":"...","raw_json":"..."}
func (h *analyticsHandler) ingestEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID       string `json:"id"`
		TenantID string `json:"tenant_id"`
		Source   string `json:"source"`
		Severity string `json:"severity"`
		Category string `json:"category"`
		Summary  string `json:"summary"`
		RawJSON  string `json:"raw_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.engine.IngestEvent(r.Context(), body.ID, body.TenantID, body.Source,
		body.Severity, body.Category, body.Summary, body.RawJSON); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ingested"})
}

// eventSummary returns hourly event counts.
//
//	GET /analytics/events/summary?tenant_id=...&since=2024-01-01T00:00:00Z
func (h *analyticsHandler) eventSummary(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	since := time.Now().Add(-24 * time.Hour)
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	results, err := h.engine.EventSummaryByHour(r.Context(), tenantID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// ingestComplianceSnapshot records a compliance point-in-time measurement.
//
//	POST /analytics/compliance/snapshot
func (h *analyticsHandler) ingestComplianceSnapshot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID    string `json:"tenant_id"`
		FrameworkID string `json:"framework_id"`
		PassCount   int    `json:"pass_count"`
		FailCount   int    `json:"fail_count"`
		TotalChecks int    `json:"total_checks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.engine.IngestComplianceSnapshot(r.Context(), body.TenantID,
		body.FrameworkID, body.PassCount, body.FailCount, body.TotalChecks); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "snapshot recorded"})
}

// complianceTrend returns daily pass-rate trend.
//
//	GET /analytics/compliance/trend?tenant_id=...&framework_id=...&since=...
func (h *analyticsHandler) complianceTrend(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	fwID := r.URL.Query().Get("framework_id")
	if tenantID == "" || fwID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and framework_id required")
		return
	}
	since := time.Now().Add(-30 * 24 * time.Hour) // default 30 days
	if s := r.URL.Query().Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}
	results, err := h.engine.ComplianceTrendDaily(r.Context(), tenantID, fwID, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// ingestMetric records a timestamped metric sample.
//
//	POST /analytics/metrics
func (h *analyticsHandler) ingestMetric(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID   string  `json:"tenant_id"`
		MetricName string  `json:"metric_name"`
		Value      float64 `json:"value"`
		Labels     string  `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := h.engine.IngestMetric(r.Context(), body.TenantID, body.MetricName, body.Value, body.Labels); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "metric recorded"})
}
