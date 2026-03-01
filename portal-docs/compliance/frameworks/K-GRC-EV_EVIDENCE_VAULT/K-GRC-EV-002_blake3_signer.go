package grc

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zeebo/blake3"
)

// AuditEvent is a single entry in the append-only audit_events table.
type AuditEvent struct {
	ID           string
	TenantID     string
	EventType    string
	ActorID      string
	ActorEmail   string
	ResourceType string
	ResourceID   string
	Action       string
	Outcome      string
	IPAddress    string
	UserAgent    string
	Payload      map[string]any
	PreviousHash string
	EventHash    string
	CreatedAt    time.Time
}

// EvidenceSigner computes BLAKE3 digests for audit chain integrity.
// The zeebo/blake3 package is available in go.mod.
type EvidenceSigner struct{}

// NewEvidenceSigner creates an EvidenceSigner (no configuration required).
func NewEvidenceSigner() *EvidenceSigner {
	return &EvidenceSigner{}
}

// Hash returns a lowercase hex-encoded BLAKE3 digest of data.
func (s *EvidenceSigner) Hash(data []byte) string {
	h := blake3.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashFile returns the BLAKE3 digest of a file.
func (s *EvidenceSigner) HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	return s.Hash(data), nil
}

// HashChain derives the hash for a new chain link.
// Input: previousHash + "|" + eventID + "|" + timestamp
func (s *EvidenceSigner) HashChain(previousHash, eventID, timestamp string) string {
	input := previousHash + "|" + eventID + "|" + timestamp
	return s.Hash([]byte(input))
}

// SignAuditEvent fills in the EventHash field on an AuditEvent.
// The event's CreatedAt must be set before calling this.
func (s *EvidenceSigner) SignAuditEvent(event AuditEvent) AuditEvent {
	ts := event.CreatedAt.UTC().Format(time.RFC3339Nano)
	event.EventHash = s.HashChain(event.PreviousHash, event.ID, ts)
	return event
}

// VerifyChain verifies the integrity of a slice of audit events.
// Returns (true, nil) on success, or (false, []broken_positions) on failure.
func (s *EvidenceSigner) VerifyChain(events []AuditEvent) (bool, []string) {
	var broken []string
	prevHash := ""
	for i, ev := range events {
		if ev.PreviousHash != prevHash {
			broken = append(broken, fmt.Sprintf("event[%d] id=%s: previous_hash mismatch (expected %q, got %q)",
				i, ev.ID, prevHash, ev.PreviousHash))
		}
		expected := s.HashChain(ev.PreviousHash, ev.ID, ev.CreatedAt.UTC().Format(time.RFC3339Nano))
		if ev.EventHash != expected {
			broken = append(broken, fmt.Sprintf("event[%d] id=%s: event_hash mismatch (expected %q, got %q)",
				i, ev.ID, expected, ev.EventHash))
		}
		prevHash = ev.EventHash
	}
	return len(broken) == 0, broken
}

// SaveEvent inserts an AuditEvent into the append-only audit_events table.
// The table has an immutability trigger that blocks UPDATE and DELETE.
func (s *EvidenceSigner) SaveEvent(ctx context.Context, event AuditEvent, pgPool *pgxpool.Pool) error {
	payloadJSON := "{}"
	if len(event.Payload) > 0 {
		b := "{"
		i := 0
		for k, v := range event.Payload {
			if i > 0 {
				b += ","
			}
			b += fmt.Sprintf(`"%s":"%v"`, k, v)
			i++
		}
		payloadJSON = b + "}"
	}

	_, err := pgPool.Exec(ctx, `
		INSERT INTO audit_events
			(id, tenant_id, event_type, actor_id, actor_email,
			 resource_type, resource_id, action, outcome,
			 ip_address, user_agent, payload,
			 previous_hash, event_hash, created_at)
		VALUES ($1,$2,$3,
		        NULLIF($4,'')::uuid, NULLIF($5,''),
		        $6, NULLIF($7,''), $8, $9,
		        NULLIF($10,'')::inet, NULLIF($11,''),
		        $12::jsonb, $13, $14, $15)`,
		event.ID, event.TenantID, event.EventType,
		event.ActorID, event.ActorEmail,
		event.ResourceType, event.ResourceID, event.Action, event.Outcome,
		event.IPAddress, event.UserAgent,
		payloadJSON,
		event.PreviousHash, event.EventHash, event.CreatedAt)
	return err
}

// GetLatestHash fetches the most recent event_hash for a tenant (for chain continuation).
func (s *EvidenceSigner) GetLatestHash(ctx context.Context, tenantID string, pgPool *pgxpool.Pool) (string, error) {
	var hash string
	err := pgPool.QueryRow(ctx, `
		SELECT event_hash
		FROM audit_events
		WHERE tenant_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, tenantID).Scan(&hash)
	if err != nil {
		// No prior events is a valid state — chain starts with empty hash
		return "", nil
	}
	return hash, nil
}
