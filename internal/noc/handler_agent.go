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

var validAgentTypes = map[string]bool{
	"coresec": true, "netguard": true, "perftrace": true, "watchdog": true,
}

type agentHandler struct {
	store *NOCStore
	pub   *Publisher
}

func newAgentHandler(store *NOCStore, pub *Publisher) *agentHandler {
	return &agentHandler{store: store, pub: pub}
}

// POST /agents/heartbeat
//
// Called by every Kubric agent binary on each heartbeat tick (every 30–60 s).
// Creates the agent record on first call; refreshes last_heartbeat on subsequent calls.
// Agents are identified by (tenant_id, hostname, agent_type) — no stored ID needed.
//
// Request body:
//
//	{
//	  "tenant_id":  "acme-corp",
//	  "cluster_id": "uuid-or-empty",
//	  "hostname":   "node-01.acme.internal",
//	  "agent_type": "coresec",
//	  "version":    "0.1.0"
//	}
func (h *agentHandler) heartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TenantID  string `json:"tenant_id"`
		ClusterID string `json:"cluster_id"`
		Hostname  string `json:"hostname"`
		AgentType string `json:"agent_type"`
		Version   string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := schema.ValidateTenantID(body.TenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if body.Hostname == "" {
		writeError(w, http.StatusUnprocessableEntity, "hostname must not be empty")
		return
	}
	if !validAgentTypes[body.AgentType] {
		writeError(w, http.StatusUnprocessableEntity, "agent_type must be one of: coresec, netguard, perftrace, watchdog")
		return
	}

	a, err := h.store.Heartbeat(r.Context(), Agent{
		TenantID:  body.TenantID,
		ClusterID: body.ClusterID,
		Hostname:  body.Hostname,
		AgentType: body.AgentType,
		Version:   body.Version,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.pub.PublishAgentEvent(AgentEvent{
		AgentID: a.ID, TenantID: a.TenantID, Hostname: a.Hostname,
		AgentType: a.AgentType, Action: "heartbeat",
	})
	writeJSON(w, http.StatusOK, a)
}

// GET /agents?tenant_id=acme-corp&cluster_id=uuid&limit=50
func (h *agentHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "tenant_id: "+err.Error())
		return
	}
	clusterID := r.URL.Query().Get("cluster_id")
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	agents, err := h.store.ListAgents(r.Context(), tenantID, clusterID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agents == nil {
		agents = []Agent{}
	}
	writeJSON(w, http.StatusOK, agents)
}

// GET /agents/{agentID}
func (h *agentHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "agentID")
	a, err := h.store.GetAgent(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}
