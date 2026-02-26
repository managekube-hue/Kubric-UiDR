// K-SOC-FR-001 — Evidence Capture: collects and preserves forensic evidence
// with Blake3 hash chains for court-admissible incident evidence.
package soc

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	nats "github.com/nats-io/nats.go"
	"github.com/zeebo/blake3"
)

// ---------------------------------------------------------------------------
// Evidence type constants
// ---------------------------------------------------------------------------

// EvidenceType classifies the kind of forensic artefact captured.
type EvidenceType string

const (
	TypeFile               EvidenceType = "file"
	TypeMemoryDump         EvidenceType = "memory_dump"
	TypePCAP               EvidenceType = "pcap"
	TypeLog                EvidenceType = "log"
	TypeProcessList        EvidenceType = "process_list"
	TypeNetworkConnections EvidenceType = "network_connections"
)

// evidenceBucket is the MinIO bucket used for all evidence storage.
const evidenceBucket = "kubric-evidence"

// ---------------------------------------------------------------------------
// EvidenceItem — single custody record
// ---------------------------------------------------------------------------

// EvidenceItem is an immutable record of a single forensic artefact.
// The Blake3-based chain fields (ChainPrev, ChainHash) form an append-only
// tamper-evident log anchored by the sequence of items within an incident.
type EvidenceItem struct {
	// Identity
	ID         string       `json:"id"`          // uuid v4
	Type       EvidenceType `json:"type"`
	SourceAgent string      `json:"source_agent"` // agent ID that collected the artefact
	IncidentID string       `json:"incident_id"`
	TenantID   string       `json:"tenant_id"`

	// Storage
	FilePath string `json:"file_path,omitempty"` // local path (may be empty for remote-only)

	// Integrity
	Blake3Hash  string `json:"blake3_hash"`  // hex-encoded 32-byte Blake3 digest of the raw file bytes
	CollectedAt time.Time `json:"collected_at"`
	SizeByte    int64  `json:"size_byte"`

	// Descriptive
	Description string `json:"description,omitempty"`

	// Tamper-evident chain — each item links to the previous item's ChainHash.
	ChainPrev string `json:"chain_prev"` // ChainHash of the preceding item (or "" for first)
	ChainHash string `json:"chain_hash"` // Blake3( ChainPrev || ID || Blake3Hash )
}

// ---------------------------------------------------------------------------
// EvidenceStore interface
// ---------------------------------------------------------------------------

// EvidenceStore persists and retrieves EvidenceItems for a given incident.
// Implementations may use a database, object store, or in-memory map.
type EvidenceStore interface {
	Save(ctx context.Context, item EvidenceItem) error
	List(ctx context.Context, incidentID string) ([]EvidenceItem, error)
	Get(ctx context.Context, id string) (*EvidenceItem, error)
}

// ---------------------------------------------------------------------------
// In-memory EvidenceStore (reference implementation / test double)
// ---------------------------------------------------------------------------

// MemEvidenceStore is a thread-safe in-memory implementation of EvidenceStore.
type MemEvidenceStore struct {
	mu    sync.RWMutex
	items map[string]EvidenceItem // keyed by ID
}

// NewMemEvidenceStore creates an empty in-memory store.
func NewMemEvidenceStore() *MemEvidenceStore {
	return &MemEvidenceStore{items: make(map[string]EvidenceItem)}
}

// Save persists an EvidenceItem, overwriting any existing entry with the same ID.
func (s *MemEvidenceStore) Save(_ context.Context, item EvidenceItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return nil
}

// List returns all items belonging to the given incident, ordered by CollectedAt.
func (s *MemEvidenceStore) List(_ context.Context, incidentID string) ([]EvidenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []EvidenceItem
	for _, it := range s.items {
		if it.IncidentID == incidentID {
			out = append(out, it)
		}
	}
	// Stable sort by collection time.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CollectedAt.Before(out[j-1].CollectedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

// Get retrieves a single EvidenceItem by its ID.
func (s *MemEvidenceStore) Get(_ context.Context, id string) (*EvidenceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	it, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("evidence: item %q not found", id)
	}
	return &it, nil
}

// ---------------------------------------------------------------------------
// MinIO config
// ---------------------------------------------------------------------------

// MinIOConfig holds connection parameters for the evidence object store.
type MinIOConfig struct {
	Endpoint  string // e.g. "minio:9000"
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string
}

// ---------------------------------------------------------------------------
// EvidenceCollector
// ---------------------------------------------------------------------------

// EvidenceCollector captures, hashes, and stores forensic evidence artefacts.
// It maintains a per-incident chain tail in memory so that successive captures
// within the same incident automatically extend the Blake3 chain.
type EvidenceCollector struct {
	store  EvidenceStore
	nc     *nats.Conn
	minio  *miniogo.Client // nil if MinIO not configured
	mu     sync.Mutex
	tails  map[string]string // incidentID → ChainHash of last item
}

// NewEvidenceCollector creates a collector backed by store and an optional
// NATS connection.  Call WithMinIO to also enable object-store uploads.
func NewEvidenceCollector(store EvidenceStore, nc *nats.Conn) *EvidenceCollector {
	return &EvidenceCollector{
		store: store,
		nc:    nc,
		tails: make(map[string]string),
	}
}

// WithMinIO attaches a MinIO client to the collector for evidence uploads.
func (c *EvidenceCollector) WithMinIO(cfg MinIOConfig) error {
	mc, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  miniocreds.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("evidence: minio init: %w", err)
	}
	c.minio = mc
	return nil
}

// ---------------------------------------------------------------------------
// Core capture methods
// ---------------------------------------------------------------------------

// CaptureFile reads the file at path, computes its Blake3 hash, extends the
// incident evidence chain, saves the item, and (if MinIO is configured)
// uploads the file to object storage.
func (c *EvidenceCollector) CaptureFile(
	ctx context.Context,
	path, incidentID, tenantID, description string,
) (*EvidenceItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("evidence: read %q: %w", path, err)
	}
	hash := blake3Sum(data)

	item := EvidenceItem{
		ID:          uuid.New().String(),
		Type:        TypeFile,
		SourceAgent: "local",
		IncidentID:  incidentID,
		TenantID:    tenantID,
		FilePath:    path,
		Blake3Hash:  hash,
		CollectedAt: time.Now().UTC(),
		SizeByte:    int64(len(data)),
		Description: description,
	}
	c.extendChain(&item)

	if err := c.store.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("evidence: store: %w", err)
	}
	if c.minio != nil {
		if err := c.UploadToMinIO(ctx, &item); err != nil {
			// Non-fatal: log but continue — local copy is preserved.
			_ = err
		}
	}
	return &item, nil
}

// CaptureProcessList requests a process snapshot from the named agent via a
// NATS request and stores the response as a TypeProcessList evidence item.
func (c *EvidenceCollector) CaptureProcessList(
	ctx context.Context,
	agentID, incidentID, tenantID string,
) (*EvidenceItem, error) {
	if c.nc == nil {
		return nil, fmt.Errorf("evidence: nats connection required for agent requests")
	}

	subject := fmt.Sprintf("kubric.%s.agent.%s.cmd.process_list", tenantID, agentID)
	payload := []byte(`{"action":"process_list"}`)

	msg, err := c.nc.RequestWithContext(ctx, subject, payload)
	if err != nil {
		return nil, fmt.Errorf("evidence: process list from agent %q: %w", agentID, err)
	}

	hash := blake3Sum(msg.Data)
	item := EvidenceItem{
		ID:          uuid.New().String(),
		Type:        TypeProcessList,
		SourceAgent: agentID,
		IncidentID:  incidentID,
		TenantID:    tenantID,
		Blake3Hash:  hash,
		CollectedAt: time.Now().UTC(),
		SizeByte:    int64(len(msg.Data)),
		Description: fmt.Sprintf("Process list from agent %s", agentID),
	}
	c.extendChain(&item)

	if err := c.store.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("evidence: store process list: %w", err)
	}

	// Persist raw payload to a temp file so MinIO upload works uniformly.
	if c.minio != nil {
		tmp := filepath.Join(os.TempDir(), item.ID+".json")
		if writeErr := os.WriteFile(tmp, msg.Data, 0600); writeErr == nil {
			item.FilePath = tmp
			_ = c.UploadToMinIO(ctx, &item)
		}
	}
	return &item, nil
}

// CaptureNetworkConnections requests an active-connection snapshot from the
// named agent via NATS and stores it as a TypeNetworkConnections evidence item.
func (c *EvidenceCollector) CaptureNetworkConnections(
	ctx context.Context,
	agentID, incidentID, tenantID string,
) (*EvidenceItem, error) {
	if c.nc == nil {
		return nil, fmt.Errorf("evidence: nats connection required for agent requests")
	}

	subject := fmt.Sprintf("kubric.%s.agent.%s.cmd.netconns", tenantID, agentID)
	payload := []byte(`{"action":"network_connections"}`)

	msg, err := c.nc.RequestWithContext(ctx, subject, payload)
	if err != nil {
		return nil, fmt.Errorf("evidence: network connections from agent %q: %w", agentID, err)
	}

	hash := blake3Sum(msg.Data)
	item := EvidenceItem{
		ID:          uuid.New().String(),
		Type:        TypeNetworkConnections,
		SourceAgent: agentID,
		IncidentID:  incidentID,
		TenantID:    tenantID,
		Blake3Hash:  hash,
		CollectedAt: time.Now().UTC(),
		SizeByte:    int64(len(msg.Data)),
		Description: fmt.Sprintf("Network connections from agent %s", agentID),
	}
	c.extendChain(&item)

	if err := c.store.Save(ctx, item); err != nil {
		return nil, fmt.Errorf("evidence: store network connections: %w", err)
	}

	if c.minio != nil {
		tmp := filepath.Join(os.TempDir(), item.ID+".json")
		if writeErr := os.WriteFile(tmp, msg.Data, 0600); writeErr == nil {
			item.FilePath = tmp
			_ = c.UploadToMinIO(ctx, &item)
		}
	}
	return &item, nil
}

// ---------------------------------------------------------------------------
// MinIO upload
// ---------------------------------------------------------------------------

// UploadToMinIO streams an evidence file to the kubric-evidence MinIO bucket.
// The object key is:  {tenantID}/{incidentID}/{itemID}/{filename}
// The item must have FilePath set.
func (c *EvidenceCollector) UploadToMinIO(ctx context.Context, item *EvidenceItem) error {
	if c.minio == nil {
		return fmt.Errorf("evidence: minio client not configured")
	}
	if item.FilePath == "" {
		return fmt.Errorf("evidence: item %q has no local file path for upload", item.ID)
	}

	f, err := os.Open(item.FilePath) //nolint:gosec
	if err != nil {
		return fmt.Errorf("evidence: open %q for upload: %w", item.FilePath, err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("evidence: stat %q: %w", item.FilePath, err)
	}

	// Ensure bucket exists.
	exists, err := c.minio.BucketExists(ctx, evidenceBucket)
	if err != nil {
		return fmt.Errorf("evidence: bucket check: %w", err)
	}
	if !exists {
		if err := c.minio.MakeBucket(ctx, evidenceBucket, miniogo.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("evidence: create bucket: %w", err)
		}
	}

	objectKey := fmt.Sprintf("%s/%s/%s/%s",
		item.TenantID, item.IncidentID, item.ID, filepath.Base(item.FilePath))

	opts := miniogo.PutObjectOptions{
		ContentType: "application/octet-stream",
		UserMetadata: map[string]string{
			"X-Evidence-ID":         item.ID,
			"X-Evidence-Type":       string(item.Type),
			"X-Evidence-Blake3":     item.Blake3Hash,
			"X-Evidence-ChainHash":  item.ChainHash,
			"X-Evidence-IncidentID": item.IncidentID,
			"X-Evidence-TenantID":   item.TenantID,
		},
	}

	_, err = c.minio.PutObject(ctx, evidenceBucket, objectKey, f, fi.Size(), opts)
	if err != nil {
		return fmt.Errorf("evidence: upload %q to minio: %w", objectKey, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Chain report
// ---------------------------------------------------------------------------

// chainReport is the JSON structure returned by BuildChainReport.
type chainReport struct {
	IncidentID   string         `json:"incident_id"`
	GeneratedAt  time.Time      `json:"generated_at"`
	TotalItems   int            `json:"total_items"`
	ChainIntact  bool           `json:"chain_intact"`
	Violations   []string       `json:"violations,omitempty"`
	Items        []EvidenceItem `json:"items"`
}

// BuildChainReport retrieves all evidence items for an incident, verifies the
// Blake3 chain, and returns a JSON report suitable for court submission.
// Returns an error if the store query fails; chain violations are encoded in
// the report (not as an error) so callers can distinguish between I/O errors
// and integrity issues.
func (c *EvidenceCollector) BuildChainReport(ctx context.Context, incidentID string) (string, error) {
	items, err := c.store.List(ctx, incidentID)
	if err != nil {
		return "", fmt.Errorf("evidence: list items for %q: %w", incidentID, err)
	}

	report := chainReport{
		IncidentID:  incidentID,
		GeneratedAt: time.Now().UTC(),
		TotalItems:  len(items),
		Items:       items,
		ChainIntact: true,
	}

	prevHash := ""
	for i, it := range items {
		// Re-derive the expected chain hash.
		expected := deriveChainHash(prevHash, it.ID, it.Blake3Hash)
		if it.ChainHash != expected {
			report.ChainIntact = false
			report.Violations = append(report.Violations,
				fmt.Sprintf("item[%d] id=%s: chain hash mismatch (stored=%s, expected=%s)",
					i, it.ID, it.ChainHash, expected))
		}
		prevHash = it.ChainHash
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("evidence: marshal chain report: %w", err)
	}
	return string(b), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// extendChain computes ChainHash for item based on the current tail for its
// incident and updates the collector's tail map.  Must be called before Save.
func (c *EvidenceCollector) extendChain(item *EvidenceItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.tails[item.IncidentID]
	item.ChainPrev = prev
	item.ChainHash = deriveChainHash(prev, item.ID, item.Blake3Hash)
	c.tails[item.IncidentID] = item.ChainHash
}

// deriveChainHash computes Blake3(prevHash || id || fileHash) and returns the
// hex-encoded 32-byte digest.  This forms the per-item link in the evidence
// hash-chain.
func deriveChainHash(prevHash, id, fileHash string) string {
	h := blake3.New()
	_, _ = io.WriteString(h, prevHash)
	_, _ = io.WriteString(h, id)
	_, _ = io.WriteString(h, fileHash)
	return hex.EncodeToString(h.Sum(nil))
}

// blake3Sum returns the hex-encoded Blake3 digest of data.
func blake3Sum(data []byte) string {
	h := blake3.New()
	_, _ = io.Copy(h, bytes.NewReader(data))
	return hex.EncodeToString(h.Sum(nil))
}
