// Package grc implements the CISO-Assistant GRC bridge service.
//
// This service connects the customer portal's compliance features with:
//   - KIC assessment store (compliance control results)
//   - KAI RAG CISO-Assistant (AI-grounded security policy answers)
//   - GRC evidence vault (BLAKE3-signed audit trail)
//   - NATS message bus (kubric.grc.ciso.v1 events)
//
// Architecture:
//
//	┌──────────┐   POST /v1/ciso/ask   ┌──────────┐   HTTP   ┌──────────────┐
//	│  Portal  │ ────────────────────── │  KIC API │ ──────── │  KAI RAG     │
//	│ (Next.js)│                        │  (Go)    │          │  (Python)    │
//	└──────────┘                        └─────┬────┘          └──────────────┘
//	                                          │
//	                            ┌──────────────┴──────────────┐
//	                            │                             │
//	                      ┌─────▼─────┐               ┌──────▼──────┐
//	                      │ Postgres  │               │  NATS       │
//	                      │ (assess.) │               │ grc.ciso.v1 │
//	                      └───────────┘               └─────────────┘
//
// Endpoints registered on the KIC Chi router:
//   POST /ciso/ask         — AI-grounded compliance Q&A
//   GET  /ciso/frameworks  — list supported compliance frameworks
//   GET  /ciso/posture     — current compliance posture summary
//
// NATS subject: kubric.grc.ciso.v1.<tenant_id>
// See: docs/message-bus/subject-mapping/K-MB-SUB-016_grc.ciso.v1.md
//
// Python RAG service: 03_K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-003_ciso_assistant.py
// Compliance assessor: 07_K-GRC-07_COMPLIANCE/K-GRC-CA/K-GRC-CA-001_compliance_assessor.go
// Evidence vault: 07_K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-002_blake3_signer.go
package grc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CompliancePosture represents the aggregated compliance state for a tenant.
// Used by the CISO-Assistant to provide context-aware answers about a tenant's
// current security posture.
type CompliancePosture struct {
	TenantID    string                `json:"tenant_id"`
	Overall     float64               `json:"overall_score"`
	ByFramework []FrameworkScore      `json:"by_framework"`
	AssessedAt  time.Time             `json:"assessed_at"`
}

// FrameworkScore is the per-framework compliance breakdown.
type FrameworkScore struct {
	Framework    string  `json:"framework"`
	Score        float64 `json:"score"`          // 0–100
	Compliant    int     `json:"compliant"`
	NonCompliant int     `json:"non_compliant"`
	Partial      int     `json:"partial"`
	Total        int     `json:"total"`
}

// CISOQueryRecord is the audit-trail record for a CISO-Assistant interaction.
// Persisted to the evidence vault for GRC compliance.
type CISOQueryRecord struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Question   string    `json:"question"`
	Answer     string    `json:"answer"`
	Sources    []string  `json:"sources"`        // OSCAL control IDs
	Confidence float64   `json:"confidence"`
	QueriedAt  time.Time `json:"queried_at"`
}

// CISOBridge coordinates compliance data with AI-driven guidance.
// It is instantiated by the KIC server and delegates to:
//   - AssessmentStore (Postgres) for compliance state
//   - RAG service (Python HTTP) for AI answers
//   - Publisher (NATS) for audit events
type CISOBridge struct {
	ragBaseURL string
}

// NewCISOBridge creates a bridge with the given RAG service URL.
func NewCISOBridge(ragBaseURL string) *CISOBridge {
	if ragBaseURL == "" {
		ragBaseURL = "http://kai-rag:8090"
	}
	return &CISOBridge{ragBaseURL: ragBaseURL}
}

// BuildPostureContext creates a JSON context block that can be injected into
// RAG queries to give the AI model awareness of the tenant's current compliance state.
func (b *CISOBridge) BuildPostureContext(posture *CompliancePosture) (string, error) {
	if posture == nil {
		return "", nil
	}
	ctx := map[string]interface{}{
		"compliance_context": map[string]interface{}{
			"overall_score": posture.Overall,
			"frameworks":    posture.ByFramework,
			"as_of":         posture.AssessedAt.Format(time.RFC3339),
		},
	}
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal posture context: %w", err)
	}
	return string(data), nil
}

// AuditRecord creates a CISOQueryRecord suitable for evidence vault persistence.
func (b *CISOBridge) AuditRecord(
	_ context.Context,
	tenantID, question, answer string,
	sources []string,
	confidence float64,
) CISOQueryRecord {
	return CISOQueryRecord{
		ID:         fmt.Sprintf("ciso-%d", time.Now().UnixNano()),
		TenantID:   tenantID,
		Question:   question,
		Answer:     answer,
		Sources:    sources,
		Confidence: confidence,
		QueriedAt:  time.Now().UTC(),
	}
}

// SupportedFrameworks returns the list of frameworks the CISO-Assistant can advise on.
func SupportedFrameworks() []string {
	return []string{
		"NIST-800-53",
		"CIS-K8s-1.8",
		"PCI-DSS-4.0",
		"SOC2",
		"ISO-27001",
	}
}
