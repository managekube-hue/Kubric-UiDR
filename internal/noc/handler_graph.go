// handler_graph.go — NOC HTTP handlers for Neo4j graph topology operations.
package noc

import (
	"encoding/json"
	"net/http"
	"strconv"

	kubricneo4j "github.com/managekube-hue/Kubric-UiDR/internal/neo4j"
)

type graphHandler struct {
	graph *kubricneo4j.GraphStore
}

func (h *graphHandler) upsertAsset(w http.ResponseWriter, r *http.Request) {
	var node kubricneo4j.AssetNode
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if node.ID == "" || node.Kind == "" || node.Name == "" {
		writeError(w, http.StatusBadRequest, "id, kind, and name are required")
		return
	}
	if err := h.graph.UpsertAsset(r.Context(), node); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "upserted", "id": node.ID})
}

func (h *graphHandler) upsertRelationship(w http.ResponseWriter, r *http.Request) {
	var rel kubricneo4j.Relationship
	if err := json.NewDecoder(r.Body).Decode(&rel); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if rel.FromID == "" || rel.ToID == "" || rel.RelType == "" {
		writeError(w, http.StatusBadRequest, "from_id, to_id, and rel_type are required")
		return
	}
	if err := h.graph.UpsertRelationship(r.Context(), rel); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "upserted"})
}

func (h *graphHandler) topology(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id required")
		return
	}
	nodes, edges, err := h.graph.Topology(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes, "edges": edges})
}

func (h *graphHandler) blastRadius(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	originID := r.URL.Query().Get("origin_id")
	if tenantID == "" || originID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and origin_id required")
		return
	}
	maxHops := 3
	if v := r.URL.Query().Get("max_hops"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 10 {
			maxHops = n
		}
	}
	results, err := h.graph.BlastRadius(r.Context(), tenantID, originID, maxHops)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}
