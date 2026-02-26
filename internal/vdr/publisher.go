package vdr

import (
	"encoding/json"
	"fmt"

	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
	"github.com/nats-io/nats.go"
)

// FindingEvent is published to NATS when a vulnerability finding is created or triaged.
// Subject: kubric.{tenant_id}.vuln.finding.v1
type FindingEvent struct {
	FindingID string `json:"finding_id"`
	TenantID  string `json:"tenant_id"`
	Severity  string `json:"severity"`
	Action    string `json:"action"` // "created" | "status_changed"
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

// PublishFindingEvent publishes a FindingEvent to kubric.{tenant_id}.vuln.finding.v1.
func (p *Publisher) PublishFindingEvent(e FindingEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}
	subject := schema.NATSSubject(e.TenantID, "vuln", "finding")
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal finding event: %w", err)
	}
	return p.conn.Publish(subject, data)
}
