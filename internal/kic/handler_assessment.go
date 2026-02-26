package kic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

var validStatuses = map[string]bool{
	"pass": true, "fail": true, "not-applicable": true, "not-reviewed": true,
}
var validFrameworks = map[string]bool{
	"NIST-800-53": true, "CIS-K8s-1.8": true, "PCI-DSS-4.0": true,
	"SOC2": true, "ISO-27001": true,
}
var validAssessors = map[string]bool{
	"lula": true, "kube-bench": true, "openscap": true, "manual": true,
}

type assessmentHandler struct {
	store *AssessmentStore
	pub   *Publisher
}

func newAssessmentHandler(store *AssessmentStore, pub *Publisher) *assessmentHandler {
	return &assessmentHandler{store: store, pub: pub}
}

// POST /assessments
//
// Records a single control assessment result.
// Request body:
//
//	{
//	  "tenant_id":     "acme-corp",
//	  "framework":     "CIS-K8s-1.8",
//	  "control_id":    "CIS.5.1.1",
//	  "title":         "Ensure RBAC is enabled",
//	  "status":        "pass",
//	  "assessed_by":   "kube-bench",
//	  "assessed_at":   "2026-02-25T20:00:00Z",
//	  "evidence_json": "{...}"
//	}
func (h *assessmentHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID     string `json:"tenant_id"`
		Framework    string `json:"framework"`
		ControlID    string `json:"control_id"`
		Title        string `json:"title"`
		Status       string `json:"status"`
		EvidenceJSON string `json:"evidence_json"`
		AssessedBy   string `json:"assessed_by"`
		AssessedAt   string `json:"assessed_at"` // RFC3339 or empty = now
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := schema.ValidateTenantID(body.TenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if !validFrameworks[body.Framework] {
		writeError(w, http.StatusUnprocessableEntity,
			"framework must be one of: NIST-800-53, CIS-K8s-1.8, PCI-DSS-4.0, SOC2, ISO-27001")
		return
	}
	if body.ControlID == "" {
		writeError(w, http.StatusUnprocessableEntity, "control_id must not be empty")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusUnprocessableEntity, "title must not be empty")
		return
	}
	if body.Status == "" {
		body.Status = "not-reviewed"
	}
	if !validStatuses[body.Status] {
		writeError(w, http.StatusUnprocessableEntity, "status must be one of: pass, fail, not-applicable, not-reviewed")
		return
	}
	if body.AssessedBy == "" {
		body.AssessedBy = "manual"
	}
	if !validAssessors[body.AssessedBy] {
		writeError(w, http.StatusUnprocessableEntity, "assessed_by must be one of: lula, kube-bench, openscap, manual")
		return
	}
	assessedAt := time.Now().UTC()
	if body.AssessedAt != "" {
		if t, err := time.Parse(time.RFC3339, body.AssessedAt); err == nil {
			assessedAt = t
		}
	}

	a, err := h.store.Create(r.Context(), Assessment{
		TenantID:     body.TenantID,
		Framework:    body.Framework,
		ControlID:    body.ControlID,
		Title:        body.Title,
		Status:       body.Status,
		EvidenceJSON: body.EvidenceJSON,
		AssessedBy:   body.AssessedBy,
		AssessedAt:   assessedAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishAssessmentEvent(AssessmentEvent{
		AssessmentID: a.ID, TenantID: a.TenantID, Framework: a.Framework,
		ControlID: a.ControlID, Status: a.Status, Action: "created",
	})
	writeJSON(w, http.StatusCreated, a)
}

// GET /assessments?tenant_id=acme-corp&framework=CIS-K8s-1.8&status=fail&limit=50
func (h *assessmentHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id: "+err.Error())
		return
	}
	framework := r.URL.Query().Get("framework")
	status := r.URL.Query().Get("status")
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	assessments, err := h.store.List(r.Context(), tenantID, framework, status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if assessments == nil {
		assessments = []Assessment{}
	}
	writeJSON(w, http.StatusOK, assessments)
}

// GET /assessments/{assessmentID}
func (h *assessmentHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "assessmentID")
	a, err := h.store.Get(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "assessment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// PATCH /assessments/{assessmentID}
//
// Updates status and optionally replaces evidence.
// Request body: { "status": "pass", "evidence_json": "{...}" }
func (h *assessmentHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "assessmentID")
	var body struct {
		Status       string `json:"status"`
		EvidenceJSON string `json:"evidence_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if !validStatuses[body.Status] {
		writeError(w, http.StatusUnprocessableEntity, "status must be one of: pass, fail, not-applicable, not-reviewed")
		return
	}
	a, err := h.store.UpdateStatus(r.Context(), id, body.Status, body.EvidenceJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "assessment not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishAssessmentEvent(AssessmentEvent{
		AssessmentID: a.ID, TenantID: a.TenantID, Framework: a.Framework,
		ControlID: a.ControlID, Status: a.Status, Action: "status_changed",
	})
	writeJSON(w, http.StatusOK, a)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
