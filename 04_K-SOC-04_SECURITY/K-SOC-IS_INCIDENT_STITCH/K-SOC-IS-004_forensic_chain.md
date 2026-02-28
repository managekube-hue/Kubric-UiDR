# K-SOC-IS-004 -- Blake3 Forensic Chain of Custody

**Role:** Immutable hash chain for forensic integrity of security events across the Kubric pipeline.  
**Algorithm:** BLAKE3 (faster than SHA-256, cryptographically secure)  
**Pattern:** `hash(prev_hash || event_bytes)` — each event references its predecessor.

---

## 1. Architecture

```
┌────────────────┐    ┌────────────────┐    ┌────────────────┐
│ Rust Agent     │───►│ Go Service     │───►│ PostgreSQL     │
│ (CoreSec)      │    │ (KIC / SOC)    │    │ audit_log      │
│                │    │                │    │                │
│ blake3 crate   │    │ zeebo/blake3   │    │ blake3_hash    │
│ hash_event()   │    │ re_hash()      │    │ prev_hash      │
│                │    │ verify_chain() │    │ event_data     │
└────────────────┘    └────────────────┘    └────────────────┘
        │                     │
        └──── Hashes must ────┘
              MATCH (cross-layer verification)
```

---

## 2. Rust Agent-Side Hashing

```rust
// agents/coresec/src/forensics/chain.rs

use blake3::Hasher;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::sync::Mutex;

/// A single link in the forensic hash chain.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChainLink {
    /// BLAKE3 hash of (prev_hash || event_bytes)
    pub hash: String,
    /// Hash of the previous link (empty string for genesis)
    pub prev_hash: String,
    /// Sequence number
    pub seq: u64,
    /// ISO 8601 timestamp
    pub timestamp: String,
    /// Tenant identifier
    pub tenant_id: String,
    /// The event payload (canonical JSON bytes)
    pub event_data: Vec<u8>,
}

/// Maintains a per-agent hash chain.
pub struct ForensicChain {
    inner: Mutex<ChainState>,
}

struct ChainState {
    prev_hash: String,
    seq: u64,
    tenant_id: String,
}

impl ForensicChain {
    pub fn new(tenant_id: &str) -> Self {
        Self {
            inner: Mutex::new(ChainState {
                prev_hash: String::new(), // genesis
                seq: 0,
                tenant_id: tenant_id.to_string(),
            }),
        }
    }

    /// Hash an event and append to the chain.
    /// Returns the ChainLink with the computed hash.
    pub fn hash_event(&self, event_data: &[u8]) -> ChainLink {
        let mut state = self.inner.lock().unwrap();

        // Compute: blake3(prev_hash_bytes || event_data)
        let mut hasher = Hasher::new();
        hasher.update(state.prev_hash.as_bytes());
        hasher.update(event_data);
        let hash = hasher.finalize().to_hex().to_string();

        state.seq += 1;
        let link = ChainLink {
            hash: hash.clone(),
            prev_hash: state.prev_hash.clone(),
            seq: state.seq,
            timestamp: Utc::now().to_rfc3339(),
            tenant_id: state.tenant_id.clone(),
            event_data: event_data.to_vec(),
        };

        state.prev_hash = hash;
        link
    }

    /// Verify a chain link against its predecessor.
    pub fn verify_link(link: &ChainLink) -> bool {
        let mut hasher = Hasher::new();
        hasher.update(link.prev_hash.as_bytes());
        hasher.update(&link.event_data);
        let computed = hasher.finalize().to_hex().to_string();
        computed == link.hash
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_chain_integrity() {
        let chain = ForensicChain::new("tenant-001");

        let link1 = chain.hash_event(b"event-1-data");
        assert_eq!(link1.seq, 1);
        assert!(link1.prev_hash.is_empty()); // genesis
        assert!(ForensicChain::verify_link(&link1));

        let link2 = chain.hash_event(b"event-2-data");
        assert_eq!(link2.seq, 2);
        assert_eq!(link2.prev_hash, link1.hash);
        assert!(ForensicChain::verify_link(&link2));

        let link3 = chain.hash_event(b"event-3-data");
        assert_eq!(link3.prev_hash, link2.hash);
        assert!(ForensicChain::verify_link(&link3));
    }

    #[test]
    fn test_tamper_detection() {
        let chain = ForensicChain::new("tenant-001");
        let mut link = chain.hash_event(b"original-data");

        // Tamper with event data
        link.event_data = b"tampered-data".to_vec();
        assert!(!ForensicChain::verify_link(&link));
    }
}
```

---

## 3. NATS Publishing with Chain Link

```rust
// agents/coresec/src/forensics/publisher.rs

use super::chain::{ForensicChain, ChainLink};
use serde_json::json;

/// Wrap an OCSF event with forensic chain metadata before publishing.
pub async fn publish_with_chain(
    nc: &async_nats::Client,
    chain: &ForensicChain,
    subject: &str,
    event: &serde_json::Value,
) -> anyhow::Result<ChainLink> {
    // Canonicalize the event JSON (sorted keys, no whitespace)
    let canonical = serde_json::to_vec(event)?;

    // Create chain link
    let link = chain.hash_event(&canonical);

    // Wrap event with chain metadata
    let envelope = json!({
        "event": event,
        "chain": {
            "hash": &link.hash,
            "prev_hash": &link.prev_hash,
            "seq": link.seq,
            "timestamp": &link.timestamp,
            "tenant_id": &link.tenant_id,
        }
    });

    let payload = serde_json::to_vec(&envelope)?;
    nc.publish(subject.to_string(), payload.into()).await?;

    Ok(link)
}
```

---

## 4. Go Service Re-Hashing and Verification

```go
// internal/forensics/chain.go
package forensics

import (
	"encoding/json"
	"fmt"
	"sync"

	"lukechampine.com/blake3"
)

// ChainLink is a single entry in the forensic hash chain.
type ChainLink struct {
	Hash      string `json:"hash"`
	PrevHash  string `json:"prev_hash"`
	Seq       uint64 `json:"seq"`
	Timestamp string `json:"timestamp"`
	TenantID  string `json:"tenant_id"`
	EventData []byte `json:"event_data"`
}

// ChainVerifier validates chain links received from Rust agents.
type ChainVerifier struct {
	mu       sync.RWMutex
	lastHash map[string]string // keyed by tenant_id
}

func NewChainVerifier() *ChainVerifier {
	return &ChainVerifier{
		lastHash: make(map[string]string),
	}
}

// ComputeHash computes blake3(prev_hash || event_data).
func ComputeHash(prevHash string, eventData []byte) string {
	h := blake3.New(32, nil)
	h.Write([]byte(prevHash))
	h.Write(eventData)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// VerifyAndRecord checks the chain link integrity and records it.
// Returns an error if verification fails (possible tampering).
func (cv *ChainVerifier) VerifyAndRecord(link ChainLink) error {
	// Step 1: Re-compute hash on the Go side
	computed := ComputeHash(link.PrevHash, link.EventData)
	if computed != link.Hash {
		return fmt.Errorf(
			"TAMPER DETECTED: hash mismatch for tenant=%s seq=%d (expected=%s, got=%s)",
			link.TenantID, link.Seq, computed, link.Hash,
		)
	}

	// Step 2: Verify chain continuity
	cv.mu.Lock()
	defer cv.mu.Unlock()

	expectedPrev := cv.lastHash[link.TenantID]
	if expectedPrev != "" && link.PrevHash != expectedPrev {
		return fmt.Errorf(
			"CHAIN BREAK: tenant=%s seq=%d prev_hash mismatch (expected=%s, got=%s)",
			link.TenantID, link.Seq, expectedPrev, link.PrevHash,
		)
	}

	// Record this hash as the latest
	cv.lastHash[link.TenantID] = link.Hash
	return nil
}

// EventEnvelope is the NATS message format with chain metadata.
type EventEnvelope struct {
	Event json.RawMessage `json:"event"`
	Chain struct {
		Hash      string `json:"hash"`
		PrevHash  string `json:"prev_hash"`
		Seq       uint64 `json:"seq"`
		Timestamp string `json:"timestamp"`
		TenantID  string `json:"tenant_id"`
	} `json:"chain"`
}

// ParseEnvelope extracts chain link and event from NATS message.
func ParseEnvelope(data []byte) (*ChainLink, json.RawMessage, error) {
	var env EventEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	link := &ChainLink{
		Hash:      env.Chain.Hash,
		PrevHash:  env.Chain.PrevHash,
		Seq:       env.Chain.Seq,
		Timestamp: env.Chain.Timestamp,
		TenantID:  env.Chain.TenantID,
		EventData: env.Event,
	}

	return link, env.Event, nil
}
```

---

## 5. PostgreSQL Audit Log Table

```sql
-- migrations/009_forensic_chain.up.sql

CREATE TABLE IF NOT EXISTS audit_log (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       VARCHAR(64) NOT NULL,
    seq             BIGINT NOT NULL,
    blake3_hash     VARCHAR(64) NOT NULL,
    prev_hash       VARCHAR(64) NOT NULL DEFAULT '',
    event_data      JSONB NOT NULL,
    nats_subject    VARCHAR(255),
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    verified        BOOLEAN NOT NULL DEFAULT FALSE,
    verification_ts TIMESTAMPTZ,

    CONSTRAINT uq_tenant_seq UNIQUE (tenant_id, seq),
    CONSTRAINT uq_hash UNIQUE (blake3_hash)
);

CREATE INDEX idx_audit_tenant_time ON audit_log (tenant_id, event_time DESC);
CREATE INDEX idx_audit_prev_hash ON audit_log (prev_hash);
CREATE INDEX idx_audit_verified ON audit_log (verified) WHERE NOT verified;

-- Insert trigger to verify chain continuity
CREATE OR REPLACE FUNCTION verify_chain_link()
RETURNS TRIGGER AS $$
DECLARE
    expected_prev VARCHAR(64);
BEGIN
    -- Get the hash of the previous entry for this tenant
    SELECT blake3_hash INTO expected_prev
    FROM audit_log
    WHERE tenant_id = NEW.tenant_id
      AND seq = NEW.seq - 1;

    -- Genesis link (seq=1) should have empty prev_hash
    IF NEW.seq = 1 AND NEW.prev_hash = '' THEN
        NEW.verified := TRUE;
        NEW.verification_ts := NOW();
        RETURN NEW;
    END IF;

    -- Verify chain continuity
    IF expected_prev IS NOT NULL AND NEW.prev_hash = expected_prev THEN
        NEW.verified := TRUE;
        NEW.verification_ts := NOW();
    ELSE
        NEW.verified := FALSE;
        -- Log tampering alert (application layer will handle NATS notification)
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_verify_chain
    BEFORE INSERT ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION verify_chain_link();
```

---

## 6. Go Database Writer

```go
// internal/forensics/db.go
package forensics

import (
	"context"
	"database/sql"
	"fmt"
)

type AuditWriter struct {
	db *sql.DB
}

func NewAuditWriter(db *sql.DB) *AuditWriter {
	return &AuditWriter{db: db}
}

// WriteChainLink persists a verified chain link to PostgreSQL.
func (aw *AuditWriter) WriteChainLink(ctx context.Context, link ChainLink, natsSubject string) error {
	_, err := aw.db.ExecContext(ctx, `
		INSERT INTO audit_log (tenant_id, seq, blake3_hash, prev_hash, event_data, nats_subject)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, seq) DO NOTHING
	`, link.TenantID, link.Seq, link.Hash, link.PrevHash, link.EventData, natsSubject)
	if err != nil {
		return fmt.Errorf("insert audit_log: %w", err)
	}
	return nil
}

// VerifyChain checks chain integrity for a tenant over a time range.
func (aw *AuditWriter) VerifyChain(ctx context.Context, tenantID string) ([]ChainBreak, error) {
	rows, err := aw.db.QueryContext(ctx, `
		SELECT
			a.seq,
			a.blake3_hash,
			a.prev_hash,
			b.blake3_hash AS expected_prev
		FROM audit_log a
		LEFT JOIN audit_log b ON b.tenant_id = a.tenant_id AND b.seq = a.seq - 1
		WHERE a.tenant_id = $1
		  AND a.seq > 1
		  AND (b.blake3_hash IS NULL OR a.prev_hash <> b.blake3_hash)
		ORDER BY a.seq
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("verify chain query: %w", err)
	}
	defer rows.Close()

	var breaks []ChainBreak
	for rows.Next() {
		var cb ChainBreak
		var expectedPrev sql.NullString
		if err := rows.Scan(&cb.Seq, &cb.Hash, &cb.PrevHash, &expectedPrev); err != nil {
			continue
		}
		cb.ExpectedPrev = expectedPrev.String
		breaks = append(breaks, cb)
	}
	return breaks, nil
}

type ChainBreak struct {
	Seq          uint64 `json:"seq"`
	Hash         string `json:"hash"`
	PrevHash     string `json:"prev_hash"`
	ExpectedPrev string `json:"expected_prev"`
}
```

---

## 7. Chain Verification Procedure

```bash
#!/usr/bin/env bash
# scripts/verify-forensic-chain.sh
# Verify forensic chain integrity for a tenant.
set -euo pipefail

TENANT_ID="${1:?Usage: verify-forensic-chain.sh <tenant_id>}"
DB_URL="${DATABASE_URL:-postgres://kubric:kubric@localhost:5432/kubric}"

echo "[+] Verifying forensic chain for tenant: $TENANT_ID"

# Check for chain breaks
BREAKS=$(psql "$DB_URL" -t -A -c "
    SELECT count(*)
    FROM audit_log a
    LEFT JOIN audit_log b ON b.tenant_id = a.tenant_id AND b.seq = a.seq - 1
    WHERE a.tenant_id = '$TENANT_ID'
      AND a.seq > 1
      AND (b.blake3_hash IS NULL OR a.prev_hash <> b.blake3_hash)
")

TOTAL=$(psql "$DB_URL" -t -A -c "
    SELECT count(*) FROM audit_log WHERE tenant_id = '$TENANT_ID'
")

VERIFIED=$(psql "$DB_URL" -t -A -c "
    SELECT count(*) FROM audit_log WHERE tenant_id = '$TENANT_ID' AND verified = true
")

echo "[+] Total chain links: $TOTAL"
echo "[+] Verified links:    $VERIFIED"
echo "[+] Chain breaks:      $BREAKS"

if [ "$BREAKS" -gt 0 ]; then
    echo "[!] TAMPER WARNING: $BREAKS chain break(s) detected!"
    echo "[!] Details:"
    psql "$DB_URL" -c "
        SELECT a.seq, a.blake3_hash, a.prev_hash, b.blake3_hash AS expected
        FROM audit_log a
        LEFT JOIN audit_log b ON b.tenant_id = a.tenant_id AND b.seq = a.seq - 1
        WHERE a.tenant_id = '$TENANT_ID'
          AND a.seq > 1
          AND (b.blake3_hash IS NULL OR a.prev_hash <> b.blake3_hash)
        ORDER BY a.seq
    "
    exit 1
else
    echo "[+] Chain integrity VERIFIED — no tampering detected."
fi
```
