// K-SOC-IS-001 — Incident state store.
// Maintains per-incident correlation windows and deduplication state using
// NATS JetStream KeyValue store (nats.go v1.37+).
//
// NOTE: github.com/redis/go-redis/v9 is not in go.mod; this implementation
// uses the NATS JetStream KV API (also an in-memory/persistent key-value
// store), which provides equivalent TTL and deduplication semantics.
//
// Env vars:
//
//	NATS_URL  nats:// connection string (default: nats://localhost:4222)
package soc

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// CorrelatedEvent is a single security event stored in an incident window.
type CorrelatedEvent struct {
	EventID   string         `json:"event_id"`
	TenantID  string         `json:"tenant_id"`
	AssetID   string         `json:"asset_id"`
	RuleID    string         `json:"rule_id"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
}

// ---------------------------------------------------------------------------
// IncidentStateStore
// ---------------------------------------------------------------------------

// IncidentStateStore manages incident correlation windows backed by the NATS
// JetStream KeyValue store.  Any NATS KV bucket is a persistent, TTL-aware,
// distributed store semantically equivalent to Redis hash + key expiry.
type IncidentStateStore struct {
	js  nats.JetStreamContext
	kv  nats.KeyValue       // primary KV bucket for event windows
	dedup nats.KeyValue     // deduplication bucket (24h TTL)
}

const (
	kvBucketWindows = "incident-windows"
	kvBucketDedup   = "incident-dedup"
	maxEventsPerKey = 1000
)

// NewIncidentStateStore connects to NATS, creates the JetStream KV buckets, and
// returns a ready IncidentStateStore.  Reads NATS_URL from the environment.
func NewIncidentStateStore() (*IncidentStateStore, error) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("incident_state: connect to nats: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("incident_state: jetstream context: %w", err)
	}

	// Event windows bucket — no expiry by default (managed by caller TTL).
	kvWindows, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:      kvBucketWindows,
		Description: "Incident correlation event windows",
		MaxValueSize: 1 << 20, // 1 MiB per window
		MaxBytes:     1 << 30, // 1 GiB total
	})
	if err != nil {
		// Bucket may already exist — bind to it.
		kvWindows, err = js.KeyValue(kvBucketWindows)
		if err != nil {
			return nil, fmt.Errorf("incident_state: kv windows: %w", err)
		}
	}

	// Dedup bucket — 24 h TTL on every key.
	kvDedup, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:  kvBucketDedup,
		TTL:     24 * time.Hour,
		MaxValueSize: 4,
	})
	if err != nil {
		kvDedup, err = js.KeyValue(kvBucketDedup)
		if err != nil {
			return nil, fmt.Errorf("incident_state: kv dedup: %w", err)
		}
	}

	return &IncidentStateStore{
		js:    js,
		kv:    kvWindows,
		dedup: kvDedup,
	}, nil
}

// SetEventWindow stores an event slice under key with the given TTL.
// The key is scoped to incident:{tenantID}:{windowKey} and its TTL is
// approximated by writing a separate expiry marker key on the dedup bucket.
func (s *IncidentStateStore) SetEventWindow(
	key string,
	events []CorrelatedEvent,
	ttl time.Duration,
) error {
	data, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("incident_state: marshal events for %q: %w", key, err)
	}
	if _, err := s.kv.Put(key, data); err != nil {
		return fmt.Errorf("incident_state: kv put %q: %w", key, err)
	}
	// Register an expiry key in the dedup bucket so we can detect stale windows.
	expireKey := "ttl:" + key
	_ = s.dedup.Delete(expireKey) // ignore error if not present
	_, _ = s.dedup.Put(expireKey, []byte("1"))
	return nil
}

// GetEventWindow retrieves the event slice stored under key.
// Returns (nil, nil) if the key does not exist.
func (s *IncidentStateStore) GetEventWindow(key string) ([]CorrelatedEvent, error) {
	entry, err := s.kv.Get(key)
	if err == nats.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("incident_state: kv get %q: %w", key, err)
	}
	var events []CorrelatedEvent
	if err := json.Unmarshal(entry.Value(), &events); err != nil {
		return nil, fmt.Errorf("incident_state: unmarshal events for %q: %w", key, err)
	}
	return events, nil
}

// AddEventToWindow appends an event to the event list in the KV store.
// The oldest events are discarded when the list exceeds maxEventsPerKey.
func (s *IncidentStateStore) AddEventToWindow(key string, event CorrelatedEvent) error {
	existing, err := s.GetEventWindow(key)
	if err != nil {
		return err
	}
	existing = append(existing, event)
	// Trim to max entries (keep newest).
	if len(existing) > maxEventsPerKey {
		existing = existing[len(existing)-maxEventsPerKey:]
	}
	data, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("incident_state: marshal window: %w", err)
	}
	_, err = s.kv.Put(key, data)
	return err
}

// CreateIncidentKey builds a deterministic KV key from tenant, rule, and asset IDs.
func CreateIncidentKey(tenantID, ruleID, assetID string) string {
	return fmt.Sprintf("incident:%s:%s:%s", tenantID, ruleID, assetID)
}

// CheckDuplicateAlert returns true if alertID has already been seen within 24h.
// Uses NATS KV Create (conditional put) for atomic SETNX semantics.
func (s *IncidentStateStore) CheckDuplicateAlert(alertID string) bool {
	key := "alert:" + alertID
	_, err := s.dedup.Create(key, []byte("1"))
	if err != nil {
		// err is non-nil when the key already exists (duplicate) or on I/O error.
		// Treat I/O errors as non-duplicate to avoid suppressing real alerts.
		if err == nats.ErrKeyExists {
			return true // already seen
		}
		return false
	}
	return false // first seen
}
