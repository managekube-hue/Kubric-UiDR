package kic

import (
	"encoding/json"
	"fmt"

	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
	"github.com/nats-io/nats.go"
)

// AssessmentEvent is published to NATS when a compliance assessment is created or updated.
// Subject: kubric.{tenant_id}.compliance.assessment.v1
type AssessmentEvent struct {
	AssessmentID string `json:"assessment_id"`
	TenantID     string `json:"tenant_id"`
	Framework    string `json:"framework"`
	ControlID    string `json:"control_id"`
	Status       string `json:"status"`
	Action       string `json:"action"` // "created" | "status_changed"
}

// Publisher wraps a NATS connection. A nil Publisher is safe — all methods are no-ops.
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher connects to NATS and returns a Publisher.
func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Publisher{conn: conn}, nil
}

func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	_ = p.conn.Drain()
}

// Publish sends raw data to an arbitrary NATS subject.
// Used by CISO-Assistant and other handlers that need custom subject patterns.
func (p *Publisher) Publish(subject string, data []byte) error {
	if p == nil || p.conn == nil {
		return nil
	}
	return p.conn.Publish(subject, data)
}

// PublishAssessmentEvent publishes to kubric.{tenant_id}.compliance.assessment.v1.
func (p *Publisher) PublishAssessmentEvent(e AssessmentEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}
	subject := schema.NATSSubject(e.TenantID, "compliance", "assessment")
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal assessment event: %w", err)
	}
	return p.conn.Publish(subject, data)
}
