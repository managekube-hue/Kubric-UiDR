package noc

import (
	"encoding/json"
	"fmt"

	"github.com/managekube-hue/Kubric-UiDR/internal/schema"
	"github.com/nats-io/nats.go"
)

// ClusterEvent is published when a cluster is registered or its status changes.
// Subject: kubric.{tenant_id}.noc.cluster.v1
type ClusterEvent struct {
	ClusterID string `json:"cluster_id"`
	TenantID  string `json:"tenant_id"`
	Status    string `json:"status"`
	Action    string `json:"action"` // "registered" | "updated" | "removed"
}

// AgentEvent is published when an agent sends a heartbeat or goes offline.
// Subject: kubric.{tenant_id}.noc.agent.v1
type AgentEvent struct {
	AgentID   string `json:"agent_id"`
	TenantID  string `json:"tenant_id"`
	Hostname  string `json:"hostname"`
	AgentType string `json:"agent_type"`
	Action    string `json:"action"` // "heartbeat" | "registered"
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

// PublishClusterEvent publishes to kubric.{tenant_id}.noc.cluster.v1.
func (p *Publisher) PublishClusterEvent(e ClusterEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}
	subject := schema.NATSSubject(e.TenantID, "noc", "cluster")
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal cluster event: %w", err)
	}
	return p.conn.Publish(subject, data)
}

// PublishAgentEvent publishes to kubric.{tenant_id}.noc.agent.v1.
func (p *Publisher) PublishAgentEvent(e AgentEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}
	subject := schema.NATSSubject(e.TenantID, "noc", "agent")
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal agent event: %w", err)
	}
	return p.conn.Publish(subject, data)
}
