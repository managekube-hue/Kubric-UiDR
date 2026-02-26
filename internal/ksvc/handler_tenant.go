package ksvc

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

// tenantHandler groups the tenant CRUD HTTP handlers.
type tenantHandler struct {
	store *TenantStore
	pub   *Publisher
}

func newTenantHandler(store *TenantStore, pub *Publisher) *tenantHandler {
	return &tenantHandler{store: store, pub: pub}
}

// POST /tenants
//
// Request body:
//
//	{ "tenant_id": "acme-corp", "name": "Acme Corp", "plan": "starter" }
func (h *tenantHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID string `json:"tenant_id"`
		Name     string `json:"name"`
		Plan     string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := schema.ValidateTenantID(body.TenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "name must not be empty")
		return
	}
	if body.Plan == "" {
		body.Plan = "starter"
	}

	t, err := h.store.Create(r.Context(), Tenant{
		TenantID: body.TenantID,
		Name:     body.Name,
		Plan:     body.Plan,
		Status:   "active",
	})
	if err != nil {
		writeError(w, http.StatusConflict, "tenant_id already exists or DB error: "+err.Error())
		return
	}

	_ = h.pub.PublishTenantEvent(TenantEvent{TenantID: t.TenantID, Action: "created"})
	writeJSON(w, http.StatusCreated, t)
}

// GET /tenants/{tenantID}
func (h *tenantHandler) get(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	t, err := h.store.Get(r.Context(), tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// GET /tenants?limit=50
func (h *tenantHandler) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	tenants, err := h.store.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tenants == nil {
		tenants = []Tenant{}
	}
	writeJSON(w, http.StatusOK, tenants)
}

// PATCH /tenants/{tenantID}
//
// Request body (all fields optional):
//
//	{ "name": "New Name", "plan": "pro", "status": "suspended" }
func (h *tenantHandler) update(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var body struct {
		Name   string `json:"name"`
		Plan   string `json:"plan"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	t, err := h.store.Update(r.Context(), tenantID, body.Name, body.Plan, body.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.pub.PublishTenantEvent(TenantEvent{TenantID: tenantID, Action: "updated"})
	writeJSON(w, http.StatusOK, t)
}

// DELETE /tenants/{tenantID}
func (h *tenantHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if err := h.store.Delete(r.Context(), tenantID); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_ = h.pub.PublishTenantEvent(TenantEvent{TenantID: tenantID, Action: "deleted"})
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON encodes v as JSON and writes it with the given HTTP status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a standard { "error": "..." } JSON response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
