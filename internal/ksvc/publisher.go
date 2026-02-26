package ksvc

import (
	"encoding/json"
	"fmt"

	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
	"github.com/nats-io/nats.go"
)

// TenantEvent is the payload published to NATS when a tenant lifecycle event occurs.
// Subject pattern: kubric.{tenant_id}.tenant.lifecycle.v1
type TenantEvent struct {
	TenantID string `json:"tenant_id"`
	Action   string `json:"action"` // "created" | "updated" | "deleted"
}

// Publisher wraps a NATS connection and publishes typed Kubric events.
// A nil Publisher is safe to use — all methods are no-ops when the receiver is nil.
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher connects to NATS and returns a Publisher ready for use.
func NewPublisher(natsURL string) (*Publisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Publisher{conn: conn}, nil
}

// Close drains the NATS connection, flushing any buffered messages.
func (p *Publisher) Close() {
	if p == nil || p.conn == nil {
		return
	}
	_ = p.conn.Drain()
}

// PublishTenantEvent publishes a TenantEvent to:
//
//	kubric.{tenant_id}.tenant.lifecycle.v1
//
// Returns nil if the publisher is nil (NATS unavailable — events are best-effort).
func (p *Publisher) PublishTenantEvent(e TenantEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}
	subject := schema.NATSSubject(e.TenantID, "tenant", "lifecycle")
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal tenant event: %w", err)
	}
	return p.conn.Publish(subject, data)
}
