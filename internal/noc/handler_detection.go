package noc

// handler_detection.go — HTTP routes for the correlation engine detection API.
//
// Routes mounted under /detection:
//
//	GET  /detection/incidents              — list incidents (filter: tenant_id, severity, status, limit, offset)
//	GET  /detection/incidents/{id}         — get single incident by ID
//	PATCH /detection/incidents/{id}        — update incident status (investigating|resolved)
//	GET  /detection/timeline               — unified event timeline across all sources
//	GET  /detection/health                 — correlation engine metrics
//	POST /detection/incidents/{id}/dispatch — re-dispatch incident to TheHive/Shuffle

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/correlation"
	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
)

// ---------------------------------------------------------------------------
// CorrelationQuerier — interface satisfied by *correlation.Engine
// ---------------------------------------------------------------------------

// CorrelationQuerier is the interface the detection handler requires.
// Using an interface keeps the handler testable and decoupled from the
// concrete engine implementation.
type CorrelationQuerier interface {
	// ListIncidents returns incidents filtered and paginated.
	ListIncidents(tenantID string, severity int, status string, limit, offset int) []correlation.Incident
	// GetIncident returns a single incident by ID.
	GetIncident(id string) (correlation.Incident, bool)
	// UpdateIncidentStatus sets the status of an incident.
	UpdateIncidentStatus(id, status string) (correlation.Incident, bool)
	// ListTimeline returns raw events for a tenant in the requested time range.
	ListTimeline(tenantID string, since, until time.Time, limit int) []correlation.NormalizedEvent
	// Metrics returns runtime counters for the correlation engine.
	Metrics() map[string]int64
	// DispatchByID re-dispatches an incident to TheHive / Shuffle.
	DispatchByID(ctx context.Context, incidentID string) error
}

// ---------------------------------------------------------------------------
// detectionHandler
// ---------------------------------------------------------------------------

type detectionHandler struct {
	engine CorrelationQuerier
}

// NewDetectionHandler creates a detection HTTP handler that wraps a
// CorrelationQuerier (normally a *correlation.Engine).
func NewDetectionHandler(engine CorrelationQuerier) *detectionHandler {
	return &detectionHandler{engine: engine}
}

// RegisterRoutes mounts all /detection routes.
// Call this from Server.buildRouter() inside a route group that has already
// applied JWT auth and role middleware.
func (h *detectionHandler) RegisterRoutes(r chi.Router) {
	r.Route("/detection", func(r chi.Router) {
		// Read endpoints: analyst and readonly roles
		r.Group(func(r chi.Router) {
			r.Use(kubricmw.RequireAnyRole("kubric:analyst", "kubric:readonly"))
			r.Get("/incidents", h.listIncidents)
			r.Get("/incidents/{id}", h.getIncident)
			r.Get("/timeline", h.listTimeline)
			r.Get("/health", h.engineHealth)
		})

		// Write endpoints: admin and analyst roles only
		r.Group(func(r chi.Router) {
			r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
			r.Patch("/incidents/{id}", h.patchIncident)
			r.Post("/incidents/{id}/dispatch", h.dispatchIncident)
		})
	})
}

// ---------------------------------------------------------------------------
// GET /detection/incidents
// ---------------------------------------------------------------------------

// listIncidents returns a paginated, filtered list of correlated incidents.
//
// Query parameters:
//
//	tenant_id  — required: filter by tenant
//	severity   — optional int 1–5: filter by exact severity
//	status     — optional: "new" | "investigating" | "resolved"
//	limit      — optional int (default 50, max 500)
//	offset     — optional int (default 0)
func (h *detectionHandler) listIncidents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id is required")
		return
	}

	severity := 0
	if raw := r.URL.Query().Get("severity"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 5 {
			severity = n
		} else {
			writeError(w, http.StatusUnprocessableEntity, "severity must be an integer 1–5")
			return
		}
	}

	status := r.URL.Query().Get("status")
	if status != "" && status != "new" && status != "investigating" && status != "resolved" {
		writeError(w, http.StatusUnprocessableEntity, "status must be one of: new, investigating, resolved")
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			offset = n
		}
	}

	incidents := h.engine.ListIncidents(tenantID, severity, status, limit, offset)
	if incidents == nil {
		incidents = []correlation.Incident{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":   incidents,
		"count":  len(incidents),
		"offset": offset,
		"limit":  limit,
	})
}

// ---------------------------------------------------------------------------
// GET /detection/incidents/{id}
// ---------------------------------------------------------------------------

// getIncident returns a single incident by UUID.
func (h *detectionHandler) getIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "incident id is required")
		return
	}

	inc, ok := h.engine.GetIncident(id)
	if !ok {
		writeError(w, http.StatusNotFound, "incident not found")
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

// ---------------------------------------------------------------------------
// PATCH /detection/incidents/{id}
// ---------------------------------------------------------------------------

// patchIncident updates the status of an incident.
//
// Request body:
//
//	{ "status": "investigating" }   or   { "status": "resolved" }
func (h *detectionHandler) patchIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "incident id is required")
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	switch body.Status {
	case "investigating", "resolved":
		// valid
	case "new":
		writeError(w, http.StatusUnprocessableEntity, "cannot set status back to 'new'; use 'investigating' or 'resolved'")
		return
	default:
		writeError(w, http.StatusUnprocessableEntity, "status must be 'investigating' or 'resolved'")
		return
	}

	inc, ok := h.engine.UpdateIncidentStatus(id, body.Status)
	if !ok {
		writeError(w, http.StatusNotFound, "incident not found")
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

// ---------------------------------------------------------------------------
// GET /detection/timeline
// ---------------------------------------------------------------------------

// listTimeline returns a unified, chronological stream of NormalizedEvents
// across all sources for the requested tenant and time range.
//
// Query parameters:
//
//	tenant_id  — required
//	since      — optional RFC3339; defaults to 5 minutes ago
//	until      — optional RFC3339; defaults to now
//	limit      — optional int (default 200, max 2000)
func (h *detectionHandler) listTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id is required")
		return
	}

	var since, until time.Time

	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "since: invalid RFC3339 timestamp: "+err.Error())
			return
		}
		since = t
	} else {
		since = time.Now().Add(-5 * time.Minute)
	}

	if raw := r.URL.Query().Get("until"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "until: invalid RFC3339 timestamp: "+err.Error())
			return
		}
		until = t
	} else {
		until = time.Now()
	}

	if !since.IsZero() && !until.IsZero() && since.After(until) {
		writeError(w, http.StatusUnprocessableEntity, "since must be before until")
		return
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 2000 {
			limit = n
		}
	}

	events := h.engine.ListTimeline(tenantID, since, until, limit)
	if events == nil {
		events = []correlation.NormalizedEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  events,
		"count": len(events),
		"since": since.Format(time.RFC3339),
		"until": until.Format(time.RFC3339),
	})
}

// ---------------------------------------------------------------------------
// GET /detection/health
// ---------------------------------------------------------------------------

// engineHealth returns runtime metrics for the correlation engine.
// No authentication filter beyond the route-group requirement.
func (h *detectionHandler) engineHealth(w http.ResponseWriter, r *http.Request) {
	m := h.engine.Metrics()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"metrics": m,
	})
}

// ---------------------------------------------------------------------------
// POST /detection/incidents/{id}/dispatch
// ---------------------------------------------------------------------------

// dispatchIncident manually re-dispatches an incident to TheHive and Shuffle.
// Useful for re-triggering failed dispatches or testing SOAR playbooks.
func (h *detectionHandler) dispatchIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "incident id is required")
		return
	}

	// Verify the incident exists before dispatching.
	if _, ok := h.engine.GetIncident(id); !ok {
		writeError(w, http.StatusNotFound, "incident not found")
		return
	}

	if err := h.engine.DispatchByID(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "dispatch failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":      "dispatched",
		"incident_id": id,
	})
}
