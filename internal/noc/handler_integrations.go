package noc

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/managekube-hue/Kubric-UiDR/internal/bloodhound"
	"github.com/managekube-hue/Kubric-UiDR/internal/cortex"
	"github.com/managekube-hue/Kubric-UiDR/internal/falco"
	"github.com/managekube-hue/Kubric-UiDR/internal/itdr"
	"github.com/managekube-hue/Kubric-UiDR/internal/osquery"
	"github.com/managekube-hue/Kubric-UiDR/internal/shuffle"
	"github.com/managekube-hue/Kubric-UiDR/internal/thehive"
	"github.com/managekube-hue/Kubric-UiDR/internal/velociraptor"
	"github.com/managekube-hue/Kubric-UiDR/internal/wazuh"
)

// integrationHandler groups HTTP handlers for all security-tool integrations.
// Each field is optional: when nil, the corresponding routes return 503
// "integration not configured".
type integrationHandler struct {
	wazuh        *wazuh.Client
	velociraptor *velociraptor.Client
	thehive      *thehive.Client
	cortex       *cortex.Client
	falco        *falco.Client
	osquery      *osquery.Client
	shuffle      *shuffle.Client
	bloodhound   *bloodhound.Client
	itdr         *itdr.Service
}

// RegisterRoutes mounts all integration sub-routes under /integrations.
func (h *integrationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/integrations", func(r chi.Router) {
		// ------------------------------------------------------------------
		// Wazuh
		// ------------------------------------------------------------------
		r.Route("/wazuh", func(r chi.Router) {
			r.Get("/agents", h.wazuhListAgents)
			r.Get("/agents/{agentID}", h.wazuhGetAgent)
			r.Get("/agents/{agentID}/alerts", h.wazuhListAlerts)
			r.Get("/agents/{agentID}/sca", h.wazuhListSCA)
			r.Get("/agents/{agentID}/syscheck", h.wazuhListSyscheck)
			r.Post("/agents/{agentID}/active-response", h.wazuhActiveResponse)
			r.Post("/agents/{agentID}/restart", h.wazuhRestartAgent)
		})

		// ------------------------------------------------------------------
		// Velociraptor
		// ------------------------------------------------------------------
		r.Route("/velociraptor", func(r chi.Router) {
			r.Get("/clients", h.vrSearchClients)
			r.Get("/clients/{clientID}", h.vrGetClient)
			r.Post("/hunts", h.vrCreateHunt)
			r.Get("/hunts", h.vrListHunts)
			r.Get("/hunts/{huntID}", h.vrGetHunt)
			r.Get("/hunts/{huntID}/results", h.vrGetHuntResults)
			r.Post("/collect", h.vrCollectArtifact)
			r.Post("/vql", h.vrRunVQL)
		})

		// ------------------------------------------------------------------
		// TheHive
		// ------------------------------------------------------------------
		r.Route("/thehive", func(r chi.Router) {
			r.Post("/alerts", h.thCreateAlert)
			r.Get("/alerts/{alertID}", h.thGetAlert)
			r.Post("/alerts/{alertID}/promote", h.thPromoteAlert)
			r.Post("/cases", h.thCreateCase)
			r.Get("/cases", h.thListCases)
			r.Get("/cases/{caseID}", h.thGetCase)
			r.Patch("/cases/{caseID}", h.thUpdateCase)
			r.Post("/cases/{caseID}/observables", h.thAddObservable)
			r.Post("/cases/{caseID}/tasks", h.thAddTask)
		})

		// ------------------------------------------------------------------
		// Cortex
		// ------------------------------------------------------------------
		r.Route("/cortex", func(r chi.Router) {
			r.Get("/analyzers", h.cxListAnalyzers)
			r.Post("/analyzers/{analyzerID}/run", h.cxRunAnalyzer)
			r.Get("/jobs/{jobID}", h.cxGetJob)
			r.Get("/jobs/{jobID}/report", h.cxGetJobReport)
		})

		// ------------------------------------------------------------------
		// osquery (FleetDM)
		// ------------------------------------------------------------------
		r.Route("/osquery", func(r chi.Router) {
			r.Get("/hosts", h.oqListHosts)
			r.Get("/hosts/{hostID}", h.oqGetHost)
			r.Post("/queries/run", h.oqRunLiveQuery)
		})

		// ------------------------------------------------------------------
		// BloodHound CE
		// ------------------------------------------------------------------
		r.Route("/bloodhound", func(r chi.Router) {
			r.Get("/domains", h.bhListDomains)
			r.Post("/cypher", h.bhRunCypher)
			r.Get("/domains/{domainID}/attack-paths", h.bhListAttackPaths)
		})

		// ------------------------------------------------------------------
		// ITDR
		// ------------------------------------------------------------------
		r.Route("/itdr", func(r chi.Router) {
			r.Get("/assets", h.itdrListAssets)
			r.Get("/misp/taxonomies", h.itdrListMispTaxonomies)
			r.Post("/bloodhound/cypher/run", h.itdrRunBloodHoundCypherFile)
			r.Get("/otx/{indicatorType}/{indicator}", h.itdrLookupOTX)
			r.Post("/responders/{name}/run", h.itdrRunIdentityResponder)
			r.Post("/cortex/responders/{responderID}/run", h.itdrRunCortexResponder)
		})

		// ------------------------------------------------------------------
		// Shuffle SOAR
		// ------------------------------------------------------------------
		r.Route("/shuffle", func(r chi.Router) {
			r.Get("/workflows", h.shListWorkflows)
			r.Post("/workflows/{workflowID}/execute", h.shExecuteWorkflow)
		})

		// ------------------------------------------------------------------
		// Aggregate health
		// ------------------------------------------------------------------
		r.Get("/health", h.healthAll)
	})
}

// =========================================================================
// Wazuh handlers
// =========================================================================

// GET /integrations/wazuh/agents?status=active&limit=100&offset=0
func (h *integrationHandler) wazuhListAgents(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	limit := queryInt(q, "limit", 100)
	offset := queryInt(q, "offset", 0)

	agents, total, err := h.wazuh.ListAgents(r.Context(), status, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": agents,
		"total": total,
	})
}

// GET /integrations/wazuh/agents/{agentID}
func (h *integrationHandler) wazuhGetAgent(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")
	agent, err := h.wazuh.GetAgent(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

// GET /integrations/wazuh/agents/{agentID}/alerts?min_level=7&limit=50&offset=0
func (h *integrationHandler) wazuhListAlerts(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")
	q := r.URL.Query()
	minLevel := queryInt(q, "min_level", 0)
	limit := queryInt(q, "limit", 100)
	offset := queryInt(q, "offset", 0)

	alerts, total, err := h.wazuh.ListAlerts(r.Context(), agentID, minLevel, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": alerts,
		"total": total,
	})
}

// GET /integrations/wazuh/agents/{agentID}/sca
func (h *integrationHandler) wazuhListSCA(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")
	policies, err := h.wazuh.ListSCAPolicies(r.Context(), agentID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

// GET /integrations/wazuh/agents/{agentID}/syscheck?limit=100&offset=0
func (h *integrationHandler) wazuhListSyscheck(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")
	q := r.URL.Query()
	limit := queryInt(q, "limit", 100)
	offset := queryInt(q, "offset", 0)

	files, total, err := h.wazuh.ListSyscheckFiles(r.Context(), agentID, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": files,
		"total": total,
	})
}

// POST /integrations/wazuh/agents/{agentID}/active-response
// Body: { "command": "...", "arguments": ["..."], "alert": {...} }
func (h *integrationHandler) wazuhActiveResponse(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")

	var body wazuh.ActiveResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := h.wazuh.RunActiveResponse(r.Context(), agentID, body); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /integrations/wazuh/agents/{agentID}/restart
func (h *integrationHandler) wazuhRestartAgent(w http.ResponseWriter, r *http.Request) {
	if h.wazuh == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	agentID := chi.URLParam(r, "agentID")
	if err := h.wazuh.RestartAgent(r.Context(), agentID); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// =========================================================================
// Velociraptor handlers
// =========================================================================

// GET /integrations/velociraptor/clients?q=host:prod&limit=100
func (h *integrationHandler) vrSearchClients(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	q := r.URL.Query()
	query := q.Get("q")
	limit := queryInt(q, "limit", 100)

	clients, err := h.velociraptor.SearchClients(r.Context(), query, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

// GET /integrations/velociraptor/clients/{clientID}
func (h *integrationHandler) vrGetClient(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	clientID := chi.URLParam(r, "clientID")
	client, err := h.velociraptor.GetClient(r.Context(), clientID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, client)
}

// POST /integrations/velociraptor/hunts
// Body: { "description": "...", "artifacts": ["..."], "parameters": {"key":"val"} }
func (h *integrationHandler) vrCreateHunt(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body struct {
		Description string            `json:"description"`
		Artifacts   []string          `json:"artifacts"`
		Parameters  map[string]string `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if len(body.Artifacts) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "artifacts must not be empty")
		return
	}

	hunt, err := h.velociraptor.CreateHunt(r.Context(), body.Description, body.Artifacts, body.Parameters)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, hunt)
}

// GET /integrations/velociraptor/hunts?limit=50&offset=0
func (h *integrationHandler) vrListHunts(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	q := r.URL.Query()
	limit := queryInt(q, "limit", 50)
	offset := queryInt(q, "offset", 0)

	hunts, err := h.velociraptor.ListHunts(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hunts)
}

// GET /integrations/velociraptor/hunts/{huntID}
func (h *integrationHandler) vrGetHunt(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	huntID := chi.URLParam(r, "huntID")
	hunt, err := h.velociraptor.GetHunt(r.Context(), huntID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hunt)
}

// GET /integrations/velociraptor/hunts/{huntID}/results?artifact=Generic.Client.Info&limit=1000
func (h *integrationHandler) vrGetHuntResults(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	huntID := chi.URLParam(r, "huntID")
	q := r.URL.Query()
	artifact := q.Get("artifact")
	limit := queryInt(q, "limit", 1000)

	rows, err := h.velociraptor.GetHuntResults(r.Context(), huntID, artifact, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// POST /integrations/velociraptor/collect
// Body: { "client_id":"C.abc", "artifacts":["..."], "specs":{"key":"val"}, "urgent":false }
func (h *integrationHandler) vrCollectArtifact(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body velociraptor.CollectRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.ClientID == "" || len(body.Artifacts) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "client_id and artifacts are required")
		return
	}

	flow, err := h.velociraptor.CollectArtifact(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, flow)
}

// POST /integrations/velociraptor/vql
// Body: { "query":"SELECT ...", "env":{"key":"val"} }
func (h *integrationHandler) vrRunVQL(w http.ResponseWriter, r *http.Request) {
	if h.velociraptor == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body velociraptor.VQLRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Query == "" {
		writeError(w, http.StatusUnprocessableEntity, "query must not be empty")
		return
	}

	rows, err := h.velociraptor.RunVQL(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// =========================================================================
// TheHive handlers
// =========================================================================

// POST /integrations/thehive/alerts
func (h *integrationHandler) thCreateAlert(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body thehive.Alert
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	alert, err := h.thehive.CreateAlert(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, alert)
}

// GET /integrations/thehive/alerts/{alertID}
func (h *integrationHandler) thGetAlert(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	alertID := chi.URLParam(r, "alertID")
	alert, err := h.thehive.GetAlert(r.Context(), alertID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alert)
}

// POST /integrations/thehive/alerts/{alertID}/promote
func (h *integrationHandler) thPromoteAlert(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	alertID := chi.URLParam(r, "alertID")
	cas, err := h.thehive.PromoteAlert(r.Context(), alertID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cas)
}

// POST /integrations/thehive/cases
func (h *integrationHandler) thCreateCase(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body thehive.Case
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	cas, err := h.thehive.CreateCase(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cas)
}

// GET /integrations/thehive/cases?status=New&limit=50&offset=0
func (h *integrationHandler) thListCases(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "New"
	}
	limit := queryInt(q, "limit", 50)
	offset := queryInt(q, "offset", 0)

	cases, err := h.thehive.ListCases(r.Context(), status, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cases)
}

// GET /integrations/thehive/cases/{caseID}
func (h *integrationHandler) thGetCase(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	caseID := chi.URLParam(r, "caseID")
	cas, err := h.thehive.GetCase(r.Context(), caseID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cas)
}

// PATCH /integrations/thehive/cases/{caseID}
// Body: { "status": "InProgress", ... }
func (h *integrationHandler) thUpdateCase(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	caseID := chi.URLParam(r, "caseID")

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := h.thehive.UpdateCase(r.Context(), caseID, body); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// POST /integrations/thehive/cases/{caseID}/observables
func (h *integrationHandler) thAddObservable(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	caseID := chi.URLParam(r, "caseID")

	var body thehive.Observable
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	obs, err := h.thehive.AddObservable(r.Context(), caseID, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, obs)
}

// POST /integrations/thehive/cases/{caseID}/tasks
func (h *integrationHandler) thAddTask(w http.ResponseWriter, r *http.Request) {
	if h.thehive == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	caseID := chi.URLParam(r, "caseID")

	var body thehive.Task
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	task, err := h.thehive.CreateTask(r.Context(), caseID, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

// =========================================================================
// Cortex handlers
// =========================================================================

// GET /integrations/cortex/analyzers?dataType=ip
func (h *integrationHandler) cxListAnalyzers(w http.ResponseWriter, r *http.Request) {
	if h.cortex == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	dataType := r.URL.Query().Get("dataType")
	analyzers, err := h.cortex.ListAnalyzers(r.Context(), dataType)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, analyzers)
}

// POST /integrations/cortex/analyzers/{analyzerID}/run
// Body: { "data":"8.8.8.8", "dataType":"ip", "tlp":2, "pap":2, "message":"..." }
func (h *integrationHandler) cxRunAnalyzer(w http.ResponseWriter, r *http.Request) {
	if h.cortex == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	analyzerID := chi.URLParam(r, "analyzerID")

	var body cortex.AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Data == "" || body.DataType == "" {
		writeError(w, http.StatusUnprocessableEntity, "data and dataType are required")
		return
	}

	job, err := h.cortex.Analyze(r.Context(), analyzerID, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

// GET /integrations/cortex/jobs/{jobID}
func (h *integrationHandler) cxGetJob(w http.ResponseWriter, r *http.Request) {
	if h.cortex == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	jobID := chi.URLParam(r, "jobID")
	job, err := h.cortex.GetJob(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// GET /integrations/cortex/jobs/{jobID}/report
func (h *integrationHandler) cxGetJobReport(w http.ResponseWriter, r *http.Request) {
	if h.cortex == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	jobID := chi.URLParam(r, "jobID")
	report, err := h.cortex.GetJobReport(r.Context(), jobID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// =========================================================================
// osquery (FleetDM) handlers
// =========================================================================

// GET /integrations/osquery/hosts?status=online&limit=50&offset=0
func (h *integrationHandler) oqListHosts(w http.ResponseWriter, r *http.Request) {
	if h.osquery == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	q := r.URL.Query()
	status := q.Get("status")
	limit := queryInt(q, "limit", 50)
	offset := queryInt(q, "offset", 0)

	hosts, total, err := h.osquery.ListHosts(r.Context(), status, limit, offset)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": hosts,
		"total": total,
	})
}

// GET /integrations/osquery/hosts/{hostID}
func (h *integrationHandler) oqGetHost(w http.ResponseWriter, r *http.Request) {
	if h.osquery == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	hostIDStr := chi.URLParam(r, "hostID")
	hostID, err := strconv.Atoi(hostIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "hostID must be an integer")
		return
	}

	host, err := h.osquery.GetHost(r.Context(), hostID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, host)
}

// POST /integrations/osquery/queries/run
// Body: { "query":"SELECT * FROM os_version;", "host_ids":[1,2,3] }
func (h *integrationHandler) oqRunLiveQuery(w http.ResponseWriter, r *http.Request) {
	if h.osquery == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body struct {
		Query   string `json:"query"`
		HostIDs []int  `json:"host_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Query == "" {
		writeError(w, http.StatusUnprocessableEntity, "query must not be empty")
		return
	}
	if len(body.HostIDs) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "host_ids must not be empty")
		return
	}

	result, err := h.osquery.RunLiveQuery(r.Context(), body.Query, body.HostIDs)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// =========================================================================
// BloodHound CE handlers
// =========================================================================

// GET /integrations/bloodhound/domains
func (h *integrationHandler) bhListDomains(w http.ResponseWriter, r *http.Request) {
	if h.bloodhound == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	domains, err := h.bloodhound.ListDomains(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domains)
}

// POST /integrations/bloodhound/cypher
// Body: { "query":"MATCH (n:User) RETURN n LIMIT 10", "parameters": {...} }
func (h *integrationHandler) bhRunCypher(w http.ResponseWriter, r *http.Request) {
	if h.bloodhound == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body struct {
		Query      string                 `json:"query"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Query == "" {
		writeError(w, http.StatusUnprocessableEntity, "query must not be empty")
		return
	}

	result, err := h.bloodhound.RunCypher(r.Context(), body.Query, body.Parameters)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /integrations/bloodhound/domains/{domainID}/attack-paths
func (h *integrationHandler) bhListAttackPaths(w http.ResponseWriter, r *http.Request) {
	if h.bloodhound == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	domainID := chi.URLParam(r, "domainID")
	paths, err := h.bloodhound.ListAttackPaths(r.Context(), domainID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

// =========================================================================
// ITDR handlers
// =========================================================================

// GET /integrations/itdr/assets
func (h *integrationHandler) itdrListAssets(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	inv, err := h.itdr.ListAssets()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

// GET /integrations/itdr/misp/taxonomies
func (h *integrationHandler) itdrListMispTaxonomies(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	taxonomies, err := h.itdr.ListMispTaxonomies()
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taxonomies)
}

// POST /integrations/itdr/bloodhound/cypher/run
// Body: { "file":"path.cypher", "parameters": { ... } }
func (h *integrationHandler) itdrRunBloodHoundCypherFile(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	var body struct {
		File       string                 `json:"file"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.File == "" {
		writeError(w, http.StatusUnprocessableEntity, "file must not be empty")
		return
	}

	result, err := h.itdr.RunCypherQueryFile(r.Context(), body.File, body.Parameters)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /integrations/itdr/otx/{indicatorType}/{indicator}
func (h *integrationHandler) itdrLookupOTX(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	indicatorType := chi.URLParam(r, "indicatorType")
	indicator := chi.URLParam(r, "indicator")
	result, err := h.itdr.LookupOTXIndicator(r.Context(), indicatorType, indicator)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /integrations/itdr/responders/{name}/run
func (h *integrationHandler) itdrRunIdentityResponder(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	name := chi.URLParam(r, "name")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	result, err := h.itdr.RunIdentityResponderScript(r.Context(), name, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// POST /integrations/itdr/cortex/responders/{responderID}/run
// Body: { "object_type":"case", "object_id":"...", "parameters": { ... } }
func (h *integrationHandler) itdrRunCortexResponder(w http.ResponseWriter, r *http.Request) {
	if h.itdr == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	responderID := chi.URLParam(r, "responderID")
	var body struct {
		ObjectType string                 `json:"object_type"`
		ObjectID   string                 `json:"object_id"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	action, err := h.itdr.RunCortexIdentityResponder(r.Context(), responderID, body.ObjectType, body.ObjectID, body.Parameters)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, action)
}

// =========================================================================
// Shuffle SOAR handlers
// =========================================================================

// GET /integrations/shuffle/workflows
func (h *integrationHandler) shListWorkflows(w http.ResponseWriter, r *http.Request) {
	if h.shuffle == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	workflows, err := h.shuffle.ListWorkflows(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workflows)
}

// POST /integrations/shuffle/workflows/{workflowID}/execute
// Body: { "argument": { ... } }
func (h *integrationHandler) shExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	if h.shuffle == nil {
		writeError(w, http.StatusServiceUnavailable, "integration not configured")
		return
	}
	workflowID := chi.URLParam(r, "workflowID")

	var body struct {
		Argument map[string]interface{} `json:"argument"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	exec, err := h.shuffle.ExecuteWorkflow(r.Context(), workflowID, body.Argument)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, exec)
}

// =========================================================================
// Aggregate health check
// =========================================================================

// GET /integrations/health
func (h *integrationHandler) healthAll(w http.ResponseWriter, r *http.Request) {
	type integrationHealth struct {
		Name      string `json:"name"`
		Status    string `json:"status"`    // "ok", "unavailable", "not_configured"
		Error     string `json:"error,omitempty"`
	}

	integrations := []struct {
		name   string
		check  func() error
		active bool
	}{
		{"wazuh", func() error { return h.wazuh.Health(r.Context()) }, h.wazuh != nil},
		{"velociraptor", func() error { return h.velociraptor.Health(r.Context()) }, h.velociraptor != nil},
		{"thehive", func() error { return h.thehive.Health(r.Context()) }, h.thehive != nil},
		{"cortex", func() error { return h.cortex.Health(r.Context()) }, h.cortex != nil},
		{"falco", func() error { return h.falco.Health(r.Context()) }, h.falco != nil},
		{"osquery", func() error { return h.osquery.Health(r.Context()) }, h.osquery != nil},
		{"shuffle", func() error { return h.shuffle.Health(r.Context()) }, h.shuffle != nil},
		{"bloodhound", func() error { return h.bloodhound.Health(r.Context()) }, h.bloodhound != nil},
		{"itdr", func() error { return h.itdr.Health(r.Context()) }, h.itdr != nil},
	}

	results := make([]integrationHealth, len(integrations))
	var wg sync.WaitGroup

	for i, ig := range integrations {
		if !ig.active {
			results[i] = integrationHealth{Name: ig.name, Status: "not_configured"}
			continue
		}

		wg.Add(1)
		go func(idx int, name string, check func() error) {
			defer wg.Done()
			if err := check(); err != nil {
				results[idx] = integrationHealth{
					Name:   name,
					Status: "unavailable",
					Error:  err.Error(),
				}
			} else {
				results[idx] = integrationHealth{
					Name:   name,
					Status: "ok",
				}
			}
		}(i, ig.name, ig.check)
	}

	wg.Wait()

	// Determine overall status.
	overall := "ok"
	for _, r := range results {
		if r.Status == "unavailable" {
			overall = "degraded"
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       overall,
		"integrations": results,
	})
}

// =========================================================================
// Helpers
// =========================================================================

// queryInt extracts an integer query parameter with a default value.
func queryInt(vals interface{ Get(string) string }, key string, defaultVal int) int {
	raw := vals.Get(key)
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
