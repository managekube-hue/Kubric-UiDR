package vdr

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

var validSeverities = map[string]bool{
	"critical": true, "high": true, "medium": true, "low": true, "informational": true,
}
var validStatuses = map[string]bool{
	"open": true, "acknowledged": true, "resolved": true, "false-positive": true,
}
var validScanners = map[string]bool{
	"nuclei": true, "trivy": true, "grype": true, "manual": true,
}

type findingHandler struct {
	store *FindingStore
	pub   *Publisher
}

func newFindingHandler(store *FindingStore, pub *Publisher) *findingHandler {
	return &findingHandler{store: store, pub: pub}
}

// POST /findings
//
// Accepts a normalized finding from any scanner.
// Request body:
//
//	{
//	  "tenant_id":   "acme-corp",
//	  "target":      "nginx:1.19.0",
//	  "scanner":     "trivy",
//	  "severity":    "high",
//	  "cve_id":      "CVE-2021-3618",
//	  "title":       "ALPACA Attack",
//	  "description": "...",
//	  "raw_json":    "{...}"
//	}
func (h *findingHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID    string `json:"tenant_id"`
		Target      string `json:"target"`
		Scanner     string `json:"scanner"`
		Severity    string `json:"severity"`
		CVEID       string `json:"cve_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		RawJSON     string `json:"raw_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := schema.ValidateTenantID(body.TenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if body.Target == "" {
		writeError(w, http.StatusUnprocessableEntity, "target must not be empty")
		return
	}
	if !validScanners[body.Scanner] {
		writeError(w, http.StatusUnprocessableEntity, "scanner must be one of: nuclei, trivy, grype, manual")
		return
	}
	if !validSeverities[body.Severity] {
		writeError(w, http.StatusUnprocessableEntity, "severity must be one of: critical, high, medium, low, informational")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title must not be empty")
		return
	}

	f, err := h.store.Create(r.Context(), Finding{
		TenantID:    body.TenantID,
		Target:      body.Target,
		Scanner:     body.Scanner,
		Severity:    body.Severity,
		CVEID:       body.CVEID,
		Title:       body.Title,
		Description: body.Description,
		Status:      "open",
		RawJSON:     body.RawJSON,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishFindingEvent(FindingEvent{
		FindingID: f.ID, TenantID: f.TenantID, Severity: f.Severity, Action: "created",
	})
	writeJSON(w, http.StatusCreated, f)
}

// GET /findings?tenant_id=acme-corp&severity=high&status=open&limit=50
func (h *findingHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id: "+err.Error())
		return
	}
	severity := r.URL.Query().Get("severity")
	status := r.URL.Query().Get("status")
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	findings, err := h.store.List(r.Context(), tenantID, severity, status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if findings == nil {
		findings = []Finding{}
	}
	writeJSON(w, http.StatusOK, findings)
}

// GET /findings/{findingID}
func (h *findingHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "findingID")
	f, err := h.store.Get(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "finding not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// PATCH /findings/{findingID}
//
// Updates triage status. Request body: { "status": "acknowledged" }
func (h *findingHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "findingID")
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if !validStatuses[body.Status] {
		writeError(w, http.StatusUnprocessableEntity, "status must be one of: open, acknowledged, resolved, false-positive")
		return
	}
	f, err := h.store.UpdateStatus(r.Context(), id, body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "finding not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishFindingEvent(FindingEvent{
		FindingID: f.ID, TenantID: f.TenantID, Severity: f.Severity, Action: "status_changed",
	})
	writeJSON(w, http.StatusOK, f)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
