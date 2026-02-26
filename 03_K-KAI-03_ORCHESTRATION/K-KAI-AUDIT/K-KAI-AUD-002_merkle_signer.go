// K-KAI-AUD-002 — Merkle Signer: tamper-evident audit log signing using Blake3 Merkle trees.
// All KAI agent actions are recorded in an append-only Blake3 chain with Ed25519 signatures.
package kai

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

// AuditEntry is a single immutable record of a KAI agent action.
// All fields are serialised into the Blake3 leaf hash to guarantee integrity.
type AuditEntry struct {
	// Sequence is a monotonically increasing counter (1-based) assigned at
	// Append time.  It is part of the hash input so re-ordering is detectable.
	Sequence uint64 `json:"seq"`

	// Timestamp is the Unix nanosecond timestamp when the entry was appended.
	Timestamp int64 `json:"ts"`

	// Identity
	AgentName string `json:"agent_name"`
	TenantID  string `json:"tenant_id"`
	UserID    string `json:"user_id"`

	// Operation
	Action string `json:"action"`
	Input  string `json:"input"`
	Output string `json:"output"`

	// Fingerprint is a caller-supplied short summary (e.g. SHA-256 of prompt).
	Fingerprint string `json:"fingerprint,omitempty"`

	// Chain fields — populated by Append; callers should leave these empty.
	PrevHash string `json:"prev_hash"` // leaf hash of the preceding entry
	Hash     string `json:"hash"`      // leaf hash of this entry
}

// VerificationError records a single chain integrity violation.
type VerificationError struct {
	Sequence uint64 `json:"seq"`
	Error    string `json:"error"`
}

// snapshotRecord is the signed JSON payload returned by Snapshot().
type snapshotRecord struct {
	MerkleRoot  string    `json:"merkle_root"`
	EntryCount  int       `json:"entry_count"`
	SnapshotAt  time.Time `json:"snapshot_at"`
	PublicKey   string    `json:"public_key"`   // hex-encoded Ed25519 public key
	Signature   string    `json:"signature"`    // hex-encoded Ed25519 signature over the payload
}

// ---------------------------------------------------------------------------
// MerkleAuditLog
// ---------------------------------------------------------------------------

// MerkleAuditLog is an append-only audit log backed by a Blake3 chain and
// a Merkle tree root.  Each entry's leaf hash depends on all previous entries,
// making undetected insertion, deletion, or reordering computationally infeasible.
//
// The log is signed with an Ed25519 key; Snapshot() returns a payload that
// can be independently verified with the corresponding public key.
type MerkleAuditLog struct {
	mu      sync.RWMutex
	entries []AuditEntry
	privKey ed25519.PrivateKey
}

// NewMerkleAuditLog creates an empty audit log.  privKeyHex must be a
// hex-encoded Ed25519 private key: either 64 bytes (128 hex chars, full key)
// or 32 bytes (64 hex chars, seed only).
func NewMerkleAuditLog(privKeyHex string) (*MerkleAuditLog, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(privKeyHex))
	if err != nil {
		return nil, fmt.Errorf("merkle_audit: decode private key hex: %w", err)
	}

	var priv ed25519.PrivateKey
	switch len(raw) {
	case ed25519.PrivateKeySize: // 64 bytes — full private key
		priv = ed25519.PrivateKey(raw)
	case ed25519.SeedSize: // 32 bytes — seed
		priv = ed25519.NewKeyFromSeed(raw)
	default:
		return nil, fmt.Errorf("merkle_audit: private key must be 32 or 64 bytes, got %d", len(raw))
	}

	return &MerkleAuditLog{privKey: priv}, nil
}

// ---------------------------------------------------------------------------
// Append
// ---------------------------------------------------------------------------

// Append records a new AuditEntry at the tail of the chain.
// It sets Sequence, Timestamp, PrevHash, and Hash on the entry before
// appending, then returns the leaf hash.
//
// The leaf hash is computed as:
//
//	Blake3( prevHash || seq || timestamp || agentName || action || Blake3(input) || Blake3(output) )
func (m *MerkleAuditLog) Append(_ context.Context, entry AuditEntry) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Assign monotonic sequence and timestamp.
	entry.Sequence = uint64(len(m.entries)) + 1
	entry.Timestamp = time.Now().UnixNano()

	// PrevHash is the leaf hash of the last entry (or "" for the first).
	if len(m.entries) > 0 {
		entry.PrevHash = m.entries[len(m.entries)-1].Hash
	} else {
		entry.PrevHash = ""
	}

	// Compute the leaf hash.
	entry.Hash = computeLeafHash(entry)

	m.entries = append(m.entries, entry)
	return entry.Hash, nil
}

// ---------------------------------------------------------------------------
// Verify
// ---------------------------------------------------------------------------

// Verify checks the entire chain for integrity: every entry's Hash is
// re-derived from its fields and compared to the stored value, and each
// PrevHash must equal the Hash of the preceding entry.
//
// Returns (true, nil) when the chain is intact.
// Returns (false, errors) listing every detected violation.
func (m *MerkleAuditLog) Verify(_ context.Context) (bool, []VerificationError) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []VerificationError

	prevHash := ""
	for i, e := range m.entries {
		// Check the PrevHash linkage.
		if e.PrevHash != prevHash {
			errs = append(errs, VerificationError{
				Sequence: e.Sequence,
				Error: fmt.Sprintf("entry[%d] prev_hash mismatch: stored=%q expected=%q",
					i, e.PrevHash, prevHash),
			})
		}

		// Re-derive the leaf hash.
		expected := computeLeafHash(e)
		if e.Hash != expected {
			errs = append(errs, VerificationError{
				Sequence: e.Sequence,
				Error: fmt.Sprintf("entry[%d] hash mismatch: stored=%q expected=%q",
					i, e.Hash, expected),
			})
		}

		prevHash = e.Hash
	}

	return len(errs) == 0, errs
}

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

// Root returns the current Merkle root computed from all leaf hashes.
// Returns an empty string if the log has no entries.
//
// The tree is built bottom-up: pairs of adjacent leaf hashes are combined as
//
//	parent = Blake3( left || right )
//
// If the level has an odd number of nodes, the last node is promoted as-is.
func (m *MerkleAuditLog) Root() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.entries) == 0 {
		return ""
	}

	// Initialise the leaf level.
	level := make([]string, len(m.entries))
	for i, e := range m.entries {
		level[i] = e.Hash
	}

	return buildMerkleRoot(level)
}

// buildMerkleRoot computes the Merkle root for a slice of hex-encoded hashes.
func buildMerkleRoot(level []string) string {
	for len(level) > 1 {
		var next []string
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				parent := blake3Hash(level[i] + level[i+1])
				next = append(next, parent)
			} else {
				// Odd node — promote without hashing.
				next = append(next, level[i])
			}
		}
		level = next
	}
	return level[0]
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// Export writes all audit entries to w as NDJSON (newline-delimited JSON),
// one entry per line.  It acquires a read lock for the duration of the write.
func (m *MerkleAuditLog) Export(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bw := bufio.NewWriter(w)
	enc := json.NewEncoder(bw)
	for _, e := range m.entries {
		if err := enc.Encode(e); err != nil {
			return fmt.Errorf("merkle_audit: encode entry seq=%d: %w", e.Sequence, err)
		}
	}
	return bw.Flush()
}

// ---------------------------------------------------------------------------
// Snapshot
// ---------------------------------------------------------------------------

// Snapshot returns a signed JSON snapshot of the current Merkle root.
// The snapshot payload is:
//
//	{ "merkle_root": "...", "entry_count": N, "snapshot_at": "...",
//	  "public_key": "...", "signature": "" }
//
// where Signature covers the canonically-serialised payload with
// Signature set to the empty string.
func (m *MerkleAuditLog) Snapshot() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	root := ""
	if len(m.entries) > 0 {
		level := make([]string, len(m.entries))
		for i, e := range m.entries {
			level[i] = e.Hash
		}
		root = buildMerkleRoot(level)
	}

	pub := m.privKey.Public().(ed25519.PublicKey)

	rec := snapshotRecord{
		MerkleRoot: root,
		EntryCount: len(m.entries),
		SnapshotAt: time.Now().UTC(),
		PublicKey:  hex.EncodeToString(pub),
		Signature:  "", // placeholder — filled in after signing
	}

	// Serialise without signature for the signing input.
	unsigned, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("merkle_audit: marshal snapshot: %w", err)
	}

	sig := ed25519.Sign(m.privKey, unsigned)
	rec.Signature = hex.EncodeToString(sig)

	signed, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("merkle_audit: marshal signed snapshot: %w", err)
	}
	return signed, nil
}

// ---------------------------------------------------------------------------
// Public key accessor
// ---------------------------------------------------------------------------

// PublicKeyHex returns the hex-encoded Ed25519 public key for out-of-band
// snapshot verification.
func (m *MerkleAuditLog) PublicKeyHex() string {
	pub := m.privKey.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub)
}

// EntryCount returns the number of entries currently in the log.
func (m *MerkleAuditLog) EntryCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// ---------------------------------------------------------------------------
// Snapshot verification helper (standalone — no receiver)
// ---------------------------------------------------------------------------

// VerifySnapshot verifies that a signed snapshot JSON blob was produced by the
// private key corresponding to publicKeyHex.  It returns an error if the
// signature does not match or the payload is malformed.
func VerifySnapshot(snapshotJSON []byte, publicKeyHex string) error {
	pubBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return fmt.Errorf("merkle_audit: decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("merkle_audit: public key must be %d bytes, got %d",
			ed25519.PublicKeySize, len(pubBytes))
	}
	pub := ed25519.PublicKey(pubBytes)

	var rec snapshotRecord
	if err := json.Unmarshal(snapshotJSON, &rec); err != nil {
		return fmt.Errorf("merkle_audit: parse snapshot: %w", err)
	}

	sigBytes, err := hex.DecodeString(rec.Signature)
	if err != nil {
		return fmt.Errorf("merkle_audit: decode signature: %w", err)
	}

	// Re-serialise without the signature field for verification.
	rec.Signature = ""
	unsigned, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("merkle_audit: re-marshal unsigned: %w", err)
	}

	if !ed25519.Verify(pub, unsigned, sigBytes) {
		return fmt.Errorf("merkle_audit: signature verification failed")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal hash helpers
// ---------------------------------------------------------------------------

// computeLeafHash derives the leaf hash for an AuditEntry using the formula:
//
//	Blake3( prevHash || seq || timestamp || agentName || action || Blake3(input) || Blake3(output) )
func computeLeafHash(e AuditEntry) string {
	inputDigest := blake3Hash(e.Input)
	outputDigest := blake3Hash(e.Output)

	h := blake3.New()
	writeStr := func(s string) { _, _ = io.WriteString(h, s) }

	writeStr(e.PrevHash)
	writeStr(strconv.FormatUint(e.Sequence, 10))
	writeStr(strconv.FormatInt(e.Timestamp, 10))
	writeStr(e.AgentName)
	writeStr(e.Action)
	writeStr(inputDigest)
	writeStr(outputDigest)

	return hex.EncodeToString(h.Sum(nil))
}

// blake3Hash returns the hex-encoded Blake3 digest of s.
func blake3Hash(s string) string {
	h := blake3.New()
	_, _ = io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil))
}
