package noc

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

var validProviders = map[string]bool{
	"k8s": true, "eks": true, "gke": true, "aks": true, "proxmox": true,
}
var validClusterStatuses = map[string]bool{
	"healthy": true, "degraded": true, "critical": true, "unknown": true,
}

type clusterHandler struct {
	store *NOCStore
	pub   *Publisher
}

func newClusterHandler(store *NOCStore, pub *Publisher) *clusterHandler {
	return &clusterHandler{store: store, pub: pub}
}

// POST /clusters
//
// Registers a new cluster.
// Request body:
//
//	{ "tenant_id":"acme-corp", "name":"prod-k8s", "provider":"k8s", "version":"1.29.2" }
func (h *clusterHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID string `json:"tenant_id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Version  string `json:"version"`
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
	if body.Provider == "" {
		body.Provider = "k8s"
	}
	if !validProviders[body.Provider] {
		writeError(w, http.StatusUnprocessableEntity, "provider must be one of: k8s, eks, gke, aks, proxmox")
		return
	}

	c, err := h.store.CreateCluster(r.Context(), Cluster{
		TenantID: body.TenantID,
		Name:     body.Name,
		Provider: body.Provider,
		Version:  body.Version,
		Status:   "unknown",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishClusterEvent(ClusterEvent{
		ClusterID: c.ID, TenantID: c.TenantID, Status: c.Status, Action: "registered",
	})
	writeJSON(w, http.StatusCreated, c)
}

// GET /clusters?tenant_id=acme-corp&limit=50
func (h *clusterHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id: "+err.Error())
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	clusters, err := h.store.ListClusters(r.Context(), tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if clusters == nil {
		clusters = []Cluster{}
	}
	writeJSON(w, http.StatusOK, clusters)
}

// GET /clusters/{clusterID}
func (h *clusterHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "clusterID")
	c, err := h.store.GetCluster(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "cluster not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// PATCH /clusters/{clusterID}
//
// Updates cluster status and/or version.
// Request body: { "status": "healthy", "version": "1.30.0" }
func (h *clusterHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "clusterID")
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Status != "" && !validClusterStatuses[body.Status] {
		writeError(w, http.StatusUnprocessableEntity, "status must be one of: healthy, degraded, critical, unknown")
		return
	}
	c, err := h.store.UpdateCluster(r.Context(), id, body.Status, body.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "cluster not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishClusterEvent(ClusterEvent{
		ClusterID: c.ID, TenantID: c.TenantID, Status: c.Status, Action: "updated",
	})
	writeJSON(w, http.StatusOK, c)
}

// DELETE /clusters/{clusterID}
func (h *clusterHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "clusterID")
	c, err := h.store.GetCluster(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "cluster not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.store.DeleteCluster(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishClusterEvent(ClusterEvent{
		ClusterID: id, TenantID: c.TenantID, Status: c.Status, Action: "removed",
	})
	w.WriteHeader(http.StatusNoContent)
}
