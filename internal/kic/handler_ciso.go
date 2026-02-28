package kic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
)

// ─── CISO-Assistant handler ──────────────────────────────────────────────────
// Bridges the customer portal to the KAI RAG CISO-Assistant service.
// Architecture:
//   Portal (Next.js) → Go API (this handler) → Python RAG service (K-KAI-RAG-003)
//   Results are enriched with compliance assessment context from the assessment store,
//   then published as NATS events on kubric.grc.ciso.v1.<tenant_id>.
//
// Endpoints:
//   POST /ciso/ask         — submit a security/compliance question
//   GET  /ciso/frameworks  — list available compliance frameworks
//   GET  /ciso/posture     — get current compliance posture summary

// CISOQuestion is the request body for a CISO-Assistant query.
type CISOQuestion struct {
	Question    string   `json:"question"`
	Frameworks  []string `json:"frameworks,omitempty"`  // filter to specific frameworks
	IncludeRefs bool     `json:"include_refs,omitempty"` // include OSCAL control references
}

// CISOAnswer is the structured response from the CISO-Assistant.
type CISOAnswer struct {
	Answer       string           `json:"answer"`
	Sources      []string         `json:"sources"`       // OSCAL control IDs (e.g., AC-2, CC6.1)
	Confidence   float64          `json:"confidence"`     // 0.0–1.0
	Frameworks   []string         `json:"frameworks"`     // applicable frameworks
	Posture      *PostureSummary  `json:"posture,omitempty"` // current compliance posture if relevant
	RetrievedDocs []string        `json:"retrieved_docs,omitempty"`
	RespondedAt  string           `json:"responded_at"`
}

// PostureSummary aggregates compliance scores across all frameworks for a tenant.
type PostureSummary struct {
	TenantID    string              `json:"tenant_id"`
	Overall     float64             `json:"overall_score"`   // 0–100
	ByFramework []FrameworkPosture  `json:"by_framework"`
	AssessedAt  string              `json:"assessed_at"`
}

// FrameworkPosture is the compliance score for a single framework.
type FrameworkPosture struct {
	Framework        string  `json:"framework"`
	Score            float64 `json:"score"`             // 0–100
	TotalControls    int     `json:"total_controls"`
	CompliantCount   int     `json:"compliant_count"`
	NonCompliant     int     `json:"non_compliant_count"`
	PartialCount     int     `json:"partial_count"`
	LastAssessedAt   string  `json:"last_assessed_at"`
}

// cisoHandler is the HTTP handler for CISO-Assistant endpoints.
type cisoHandler struct {
	store    *AssessmentStore
	pub      *Publisher
	ragURL   string // base URL of the Python K-KAI-RAG-003 service
}

func newCISOHandler(store *AssessmentStore, pub *Publisher, ragURL string) *cisoHandler {
	if ragURL == "" {
		ragURL = "http://kai-rag:8090"
	}
	return &cisoHandler{store: store, pub: pub, ragURL: ragURL}
}

// POST /ciso/ask — answer a security/compliance question via RAG
func (h *cisoHandler) ask(w http.ResponseWriter, r *http.Request) {
	tenantID := kubricmw.TenantFromContext(r.Context())
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid tenant: "+err.Error())
		return
	}

	var q CISOQuestion
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if strings.TrimSpace(q.Question) == "" {
		writeError(w, http.StatusUnprocessableEntity, "question must not be empty")
		return
	}
	if len(q.Question) > 2000 {
		writeError(w, http.StatusUnprocessableEntity, "question must be ≤ 2000 characters")
		return
	}

	// 1) Call the Python RAG service for AI-grounded answer
	ragAnswer, err := h.callRAG(r.Context(), tenantID, q)
	if err != nil {
		// RAG service unavailable — return degraded response with posture data only
		fmt.Printf("kic/ciso: warn — RAG service unavailable (%v); returning posture-only response\n", err)
		posture, _ := h.buildPosture(r.Context(), tenantID)
		writeJSON(w, http.StatusOK, CISOAnswer{
			Answer:      "CISO-AI is temporarily unavailable. Below is your current compliance posture.",
			Sources:     []string{},
			Confidence:  0.0,
			Frameworks:  supportedFrameworks(),
			Posture:     posture,
			RespondedAt: time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// 2) Enrich with compliance posture if question is posture-related
	var posture *PostureSummary
	if q.IncludeRefs || isPostureQuestion(q.Question) {
		posture, _ = h.buildPosture(r.Context(), tenantID)
	}

	answer := CISOAnswer{
		Answer:        ragAnswer.Answer,
		Sources:       ragAnswer.Sources,
		Confidence:    ragAnswer.Confidence,
		Frameworks:    effectiveFrameworks(q.Frameworks),
		Posture:       posture,
		RetrievedDocs: ragAnswer.RetrievedDocs,
		RespondedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	// 3) Publish NATS event for audit trail
	if h.pub != nil {
		h.publishCISOEvent(tenantID, q.Question, answer)
	}

	writeJSON(w, http.StatusOK, answer)
}

// GET /ciso/frameworks — list available compliance frameworks
func (h *cisoHandler) frameworks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"frameworks": []map[string]string{
			{"id": "NIST-800-53", "name": "NIST SP 800-53 Rev 5", "category": "Federal"},
			{"id": "CIS-K8s-1.8", "name": "CIS Kubernetes Benchmark v1.8", "category": "Infrastructure"},
			{"id": "PCI-DSS-4.0", "name": "PCI DSS v4.0", "category": "Payment"},
			{"id": "SOC2", "name": "SOC 2 Type II", "category": "Trust Services"},
			{"id": "ISO-27001", "name": "ISO/IEC 27001:2022", "category": "InfoSec Management"},
		},
	})
}

// GET /ciso/posture — current compliance posture summary
func (h *cisoHandler) posture(w http.ResponseWriter, r *http.Request) {
	tenantID := kubricmw.TenantFromContext(r.Context())
	if err := schema.ValidateTenantID(tenantID); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid tenant: "+err.Error())
		return
	}

	posture, err := h.buildPosture(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute posture: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, posture)
}

// ─── Internal helpers ────────────────────────────────────────────────────────

// ragResponse mirrors the Python service JSON response.
type ragResponse struct {
	Answer        string   `json:"answer"`
	Sources       []string `json:"sources"`
	Confidence    float64  `json:"confidence"`
	RetrievedDocs []string `json:"retrieved_docs"`
}

// callRAG sends the question to the Python K-KAI-RAG-003 CISO-Assistant service.
func (h *cisoHandler) callRAG(ctx context.Context, tenantID string, q CISOQuestion) (*ragResponse, error) {
	reqBody := map[string]interface{}{
		"question":  q.Question,
		"tenant_id": tenantID,
		"top_k":     5,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal RAG request: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", h.ragURL+"/v1/ciso/ask", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("build RAG request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("RAG service call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RAG service returned %d", resp.StatusCode)
	}

	var result ragResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode RAG response: %w", err)
	}
	return &result, nil
}

// buildPosture aggregates compliance assessment data into a posture summary.
func (h *cisoHandler) buildPosture(ctx context.Context, tenantID string) (*PostureSummary, error) {
	frameworks := supportedFrameworks()
	var byFw []FrameworkPosture
	var totalScore float64
	counted := 0

	for _, fw := range frameworks {
		stats, err := h.store.GetFrameworkStats(ctx, tenantID, fw)
		if err != nil {
			continue
		}
		fp := FrameworkPosture{
			Framework:      fw,
			Score:          stats.Score,
			TotalControls:  stats.Total,
			CompliantCount: stats.Compliant,
			NonCompliant:   stats.NonCompliant,
			PartialCount:   stats.Partial,
			LastAssessedAt: stats.LastAssessed.Format(time.RFC3339),
		}
		byFw = append(byFw, fp)
		totalScore += stats.Score
		counted++
	}

	overall := 0.0
	if counted > 0 {
		overall = totalScore / float64(counted)
	}

	return &PostureSummary{
		TenantID:    tenantID,
		Overall:     overall,
		ByFramework: byFw,
		AssessedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// publishCISOEvent sends a CISO-Assistant interaction event to NATS for audit trail.
func (h *cisoHandler) publishCISOEvent(tenantID, question string, answer CISOAnswer) {
	event := map[string]interface{}{
		"type":       "ciso.query.v1",
		"tenant_id":  tenantID,
		"question":   question,
		"confidence": answer.Confidence,
		"sources":    answer.Sources,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(event)
	subject := fmt.Sprintf("kubric.grc.ciso.v1.%s", tenantID)
	h.pub.Publish(subject, data)
}

// isPostureQuestion heuristically detects compliance-posture questions.
func isPostureQuestion(q string) bool {
	q = strings.ToLower(q)
	keywords := []string{"posture", "score", "compliance", "status", "overview", "dashboard", "summary", "gap"}
	for _, kw := range keywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func supportedFrameworks() []string {
	return []string{"NIST-800-53", "CIS-K8s-1.8", "PCI-DSS-4.0", "SOC2", "ISO-27001"}
}

func effectiveFrameworks(requested []string) []string {
	if len(requested) > 0 {
		return requested
	}
	return supportedFrameworks()
}
