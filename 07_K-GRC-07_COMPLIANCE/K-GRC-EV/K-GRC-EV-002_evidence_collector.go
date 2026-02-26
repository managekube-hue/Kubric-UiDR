// K-GRC-EV-002 — GRC Evidence Collector: automated evidence collection for compliance audits.
// Collects and preserves evidence for audit trails with Blake3 chain integrity.
package grc

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/zeebo/blake3"
)

// ─── Evidence categories ─────────────────────────────────────────────────────

// Framework identifies a compliance framework (duplicated from K-GRC-CA to avoid import cycle).
type Framework string

// EvidenceCategory classifies the kind of evidence being collected.
type EvidenceCategory string

const (
	CatTechnical EvidenceCategory = "technical" // system configs, scan outputs
	CatPolicy    EvidenceCategory = "policy"    // written policies, procedures
	CatProcess   EvidenceCategory = "process"   // incident records, change tickets
	CatPeople    EvidenceCategory = "people"    // training records, org charts
)

// ─── Request & bundle types ───────────────────────────────────────────────────

// EvidenceRequest is an auditor-defined task requesting a specific piece of
// compliance evidence for a control in a given framework.
type EvidenceRequest struct {
	ID              string           `json:"id"`
	ControlID       string           `json:"control_id"`
	FrameworkID     Framework        `json:"framework_id"`
	Category        EvidenceCategory `json:"category"`
	RequiredBy      time.Time        `json:"required_by"`
	AssignedTo      string           `json:"assigned_to"`
	TenantID        string           `json:"tenant_id"`
	Description     string           `json:"description"`
	AutoCollectable bool             `json:"auto_collectable"`
}

// EvidenceArtifact is a single piece of collected evidence (file, query result, etc.).
type EvidenceArtifact struct {
	Type        string    `json:"type"`        // "file", "osquery_result", "api_response"
	Path        string    `json:"path"`        // S3 key or local path
	Hash        string    `json:"hash"`        // Blake3 hex hash of content
	Description string    `json:"description"`
	Source      string    `json:"source"` // originating system
	CollectedAt time.Time `json:"collected_at"`
	SizeBytes   int64     `json:"size_bytes"`
}

// EvidenceBundle is a signed, tamper-evident collection of EvidenceArtifacts
// for a single EvidenceRequest.
type EvidenceBundle struct {
	RequestID    string             `json:"request_id"`
	Items        []EvidenceArtifact `json:"items"`
	CollectedAt  time.Time          `json:"collected_at"`
	CollectedBy  string             `json:"collected_by"`
	Blake3Root   string             `json:"blake3_root"`   // Merkle-like root over item hashes
	SignatureHex string             `json:"signature_hex"` // Ed25519 over Blake3Root
}

// ─── S3 / MinIO abstraction ───────────────────────────────────────────────────

// ObjectStorageClient is the minimal interface used for evidence uploads.
// Compatible with *minio.Client from minio-go/v7.
type ObjectStorageClient interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts interface{}) (interface{}, error)
	BucketExists(ctx context.Context, bucketName string) (bool, error)
}

// PolicyDocLister lists policy documents stored in an S3-compatible bucket.
type PolicyDocLister interface {
	ListObjects(ctx context.Context, bucketName, prefix string) ([]StoredObject, error)
	GetObjectBytes(ctx context.Context, bucketName, objectName string) ([]byte, error)
}

// StoredObject represents a minimal S3/MinIO object descriptor.
type StoredObject struct {
	Key       string
	SizeBytes int64
	ETag      string
}

// NOCReportFetcher retrieves recent NOC incident reports for process evidence.
type NOCReportFetcher interface {
	ListRecentIncidents(ctx context.Context, tenantID string, since time.Time) ([]NOCIncidentSummary, error)
}

// NOCIncidentSummary is a lightweight incident record used as process evidence.
type NOCIncidentSummary struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Severity   string    `json:"severity"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
}

// EvidenceStore persists and queries EvidenceRequests.
type EvidenceStore interface {
	ListPendingByTenant(ctx context.Context, tenantID string, before time.Time) ([]EvidenceRequest, error)
	SaveBundle(ctx context.Context, bundle *EvidenceBundle) error
}

// OsqueryClient is the minimal interface for running osquery SQL on managed endpoints.
// Compatible with *osquery.FleetClient from internal/osquery.
type OsqueryClient interface {
	Query(ctx context.Context, agentID, query string) ([]map[string]interface{}, error)
}

// ─── Collector ────────────────────────────────────────────────────────────────

// GRCEvidenceCollector automates the gathering and preservation of compliance
// evidence across technical, policy, process, and people categories.
type GRCEvidenceCollector struct {
	osquery      OsqueryClient    // re-used from compliance assessor package
	policyDocs   PolicyDocLister
	nocReports   NOCReportFetcher
	store        EvidenceStore
	bucketName   string // MinIO/S3 bucket for artifact upload
	collectorID  string // identity used in CollectedBy field
}

// NewGRCEvidenceCollector constructs a collector with all required backends.
func NewGRCEvidenceCollector(
	oq OsqueryClient,
	pl PolicyDocLister,
	noc NOCReportFetcher,
	store EvidenceStore,
	bucket, collectorID string,
) *GRCEvidenceCollector {
	return &GRCEvidenceCollector{
		osquery:     oq,
		policyDocs:  pl,
		nocReports:  noc,
		store:       store,
		bucketName:  bucket,
		collectorID: collectorID,
	}
}

// ─── AutoCollect ─────────────────────────────────────────────────────────────

// AutoCollect gathers evidence for an AutoCollectable EvidenceRequest.
// It dispatches based on Category:
//   - CatPolicy    → lists files from the S3 policies/ prefix
//   - CatTechnical → runs osquery system-config queries
//   - CatProcess   → fetches recent NOC incident reports
//   - CatPeople    → returns a guidance artifact (manual step required)
func (c *GRCEvidenceCollector) AutoCollect(ctx context.Context, req EvidenceRequest) (*EvidenceBundle, error) {
	if !req.AutoCollectable {
		return nil, fmt.Errorf("evidence request %s is not auto-collectable", req.ID)
	}

	bundle := &EvidenceBundle{
		RequestID:   req.ID,
		CollectedAt: time.Now().UTC(),
		CollectedBy: c.collectorID,
	}

	var collectErr error
	switch req.Category {
	case CatPolicy:
		bundle.Items, collectErr = c.collectPolicyDocs(ctx, req)
	case CatTechnical:
		bundle.Items, collectErr = c.collectTechnicalEvidence(ctx, req)
	case CatProcess:
		bundle.Items, collectErr = c.collectProcessEvidence(ctx, req)
	case CatPeople:
		bundle.Items = []EvidenceArtifact{{
			Type:        "guidance",
			Description: "People-category evidence requires manual upload (training records, org charts)",
			Source:      "kubric-grc",
			CollectedAt: time.Now().UTC(),
		}}
	default:
		return nil, fmt.Errorf("unsupported evidence category: %s", req.Category)
	}

	if collectErr != nil {
		return nil, fmt.Errorf("collect evidence for request %s: %w", req.ID, collectErr)
	}

	// Compute Blake3 Merkle root over all item hashes.
	if err := computeBlake3Root(bundle); err != nil {
		return nil, fmt.Errorf("compute blake3 root: %w", err)
	}

	if err := c.store.SaveBundle(ctx, bundle); err != nil {
		return nil, fmt.Errorf("save evidence bundle: %w", err)
	}

	return bundle, nil
}

// collectPolicyDocs lists and hashes policy documents from S3 policies/ prefix.
func (c *GRCEvidenceCollector) collectPolicyDocs(ctx context.Context, req EvidenceRequest) ([]EvidenceArtifact, error) {
	objects, err := c.policyDocs.ListObjects(ctx, c.bucketName, "policies/")
	if err != nil {
		return nil, fmt.Errorf("list policy docs: %w", err)
	}

	var artifacts []EvidenceArtifact
	for _, obj := range objects {
		data, err := c.policyDocs.GetObjectBytes(ctx, c.bucketName, obj.Key)
		if err != nil {
			// Non-fatal: record a failed artifact rather than aborting the bundle.
			artifacts = append(artifacts, EvidenceArtifact{
				Type:        "file",
				Path:        obj.Key,
				Description: fmt.Sprintf("Failed to retrieve: %v", err),
				Source:      "s3:" + c.bucketName,
				CollectedAt: time.Now().UTC(),
				SizeBytes:   obj.SizeBytes,
			})
			continue
		}

		h := blake3.Sum256(data)
		artifacts = append(artifacts, EvidenceArtifact{
			Type:        "file",
			Path:        obj.Key,
			Hash:        hex.EncodeToString(h[:]),
			Description: fmt.Sprintf("Policy document: %s", path.Base(obj.Key)),
			Source:      "s3:" + c.bucketName,
			CollectedAt: time.Now().UTC(),
			SizeBytes:   int64(len(data)),
		})
	}

	return artifacts, nil
}

// technicalOsqueries maps control IDs to osquery SQL statements.
var technicalOsqueries = map[string]string{
	"default": "SELECT hostname, cpu_type, cpu_brand, physical_memory, hardware_model FROM system_info;",
	"CC6.1":   "SELECT username, uid, gid, shell FROM users WHERE shell NOT LIKE '%nologin%' AND shell NOT LIKE '%false%';",
	"CC7.1":   "SELECT path, sha256, mtime FROM file WHERE path LIKE '/etc/%' ORDER BY mtime DESC LIMIT 50;",
	"CIS-1":   "SELECT name, version, status FROM os_version;",
}

// collectTechnicalEvidence runs osquery against all tenant agents.
func (c *GRCEvidenceCollector) collectTechnicalEvidence(ctx context.Context, req EvidenceRequest) ([]EvidenceArtifact, error) {
	if c.osquery == nil {
		return nil, fmt.Errorf("osquery client not configured")
	}

	query, ok := technicalOsqueries[req.ControlID]
	if !ok {
		query = technicalOsqueries["default"]
	}

	// For technical evidence we run against a synthetic agentID derived from tenantID.
	// In production this would iterate over all tenant agents.
	agentID := req.TenantID + "-primary"
	rows, err := c.osquery.Query(ctx, agentID, query)
	if err != nil {
		return nil, fmt.Errorf("osquery query for control %s: %w", req.ControlID, err)
	}

	jsonData, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("marshal osquery result: %w", err)
	}

	h := blake3.Sum256(jsonData)
	return []EvidenceArtifact{{
		Type:        "osquery_result",
		Path:        fmt.Sprintf("evidence/%s/%s/osquery.json", req.TenantID, req.ControlID),
		Hash:        hex.EncodeToString(h[:]),
		Description: fmt.Sprintf("osquery result for control %s: %s", req.ControlID, query),
		Source:      "osquery:" + agentID,
		CollectedAt: time.Now().UTC(),
		SizeBytes:   int64(len(jsonData)),
	}}, nil
}

// collectProcessEvidence fetches recent NOC incident reports as process evidence.
func (c *GRCEvidenceCollector) collectProcessEvidence(ctx context.Context, req EvidenceRequest) ([]EvidenceArtifact, error) {
	if c.nocReports == nil {
		return nil, fmt.Errorf("NOC report fetcher not configured")
	}

	since := time.Now().Add(-90 * 24 * time.Hour) // 90-day lookback
	incidents, err := c.nocReports.ListRecentIncidents(ctx, req.TenantID, since)
	if err != nil {
		return nil, fmt.Errorf("list noc incidents: %w", err)
	}

	jsonData, err := json.Marshal(incidents)
	if err != nil {
		return nil, fmt.Errorf("marshal noc incidents: %w", err)
	}

	h := blake3.Sum256(jsonData)
	return []EvidenceArtifact{{
		Type:        "api_response",
		Path:        fmt.Sprintf("evidence/%s/%s/incidents.json", req.TenantID, req.ControlID),
		Hash:        hex.EncodeToString(h[:]),
		Description: fmt.Sprintf("%d NOC incidents (last 90 days)", len(incidents)),
		Source:      "kubric-noc",
		CollectedAt: time.Now().UTC(),
		SizeBytes:   int64(len(jsonData)),
	}}, nil
}

// ─── Blake3 root computation ──────────────────────────────────────────────────

// computeBlake3Root builds a deterministic Merkle-like root by hashing all
// item Blake3 hashes (sorted for determinism) with a final Blake3 pass.
func computeBlake3Root(bundle *EvidenceBundle) error {
	hashes := make([]string, 0, len(bundle.Items))
	for _, item := range bundle.Items {
		if item.Hash != "" {
			hashes = append(hashes, item.Hash)
		}
	}
	sort.Strings(hashes)

	combined := strings.Join(hashes, "")
	root := blake3.Sum256([]byte(combined))
	bundle.Blake3Root = hex.EncodeToString(root[:])
	return nil
}

// ─── Bundle signing & verification ───────────────────────────────────────────

// SignBundle creates an Ed25519 signature over the bundle's Blake3Root and
// stores it as a hex-encoded string in bundle.SignatureHex.
// privKeyHex must be a 64-byte (128 hex chars) Ed25519 private key seed.
func SignBundle(bundle *EvidenceBundle, privKeyHex string) error {
	if bundle.Blake3Root == "" {
		return fmt.Errorf("bundle.Blake3Root is empty; call computeBlake3Root first")
	}

	privBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return fmt.Errorf("decode private key hex: %w", err)
	}

	var privKey ed25519.PrivateKey
	switch len(privBytes) {
	case ed25519.SeedSize:
		privKey = ed25519.NewKeyFromSeed(privBytes)
	case ed25519.PrivateKeySize:
		privKey = ed25519.PrivateKey(privBytes)
	default:
		return fmt.Errorf("invalid private key length %d (want %d or %d)", len(privBytes), ed25519.SeedSize, ed25519.PrivateKeySize)
	}

	msg := []byte(bundle.Blake3Root)
	sig := ed25519.Sign(privKey, msg)
	bundle.SignatureHex = hex.EncodeToString(sig)
	return nil
}

// VerifyBundle checks the Ed25519 signature on the bundle's Blake3Root.
// pubKeyHex must be a 32-byte (64 hex chars) Ed25519 public key.
func VerifyBundle(bundle *EvidenceBundle, pubKeyHex string) (bool, error) {
	if bundle.Blake3Root == "" || bundle.SignatureHex == "" {
		return false, fmt.Errorf("bundle is unsigned or has no Blake3Root")
	}

	pubBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("decode public key hex: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key length %d (want %d)", len(pubBytes), ed25519.PublicKeySize)
	}

	sig, err := hex.DecodeString(bundle.SignatureHex)
	if err != nil {
		return false, fmt.Errorf("decode signature hex: %w", err)
	}

	ok := ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(bundle.Blake3Root), sig)
	return ok, nil
}

// ─── Bundle upload ────────────────────────────────────────────────────────────

// UploadBundle serialises the EvidenceBundle as JSON and uploads it to the
// configured object storage bucket. Returns the canonical S3 URL.
// minioClient must satisfy ObjectStorageClient (typically *minio.Client).
func (c *GRCEvidenceCollector) UploadBundle(ctx context.Context, bundle *EvidenceBundle, minioClient interface{}) (string, error) {
	osc, ok := minioClient.(ObjectStorageClient)
	if !ok {
		return "", fmt.Errorf("minioClient does not implement ObjectStorageClient")
	}

	exists, err := osc.BucketExists(ctx, c.bucketName)
	if err != nil {
		return "", fmt.Errorf("check bucket existence: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("evidence bucket %q does not exist", c.bucketName)
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal bundle: %w", err)
	}

	objectKey := fmt.Sprintf("evidence-bundles/%s/%s.json",
		bundle.CollectedAt.Format("2006/01/02"),
		bundle.RequestID,
	)

	_, err = osc.PutObject(ctx, c.bucketName, objectKey,
		bytes.NewReader(data), int64(len(data)), nil)
	if err != nil {
		return "", fmt.Errorf("upload bundle to s3: %w", err)
	}

	return fmt.Sprintf("s3://%s/%s", c.bucketName, objectKey), nil
}

// ─── Pending request listing ──────────────────────────────────────────────────

// ListPendingRequests returns all EvidenceRequests for tenantID that are due
// within the given dueWithin duration from now.
func (c *GRCEvidenceCollector) ListPendingRequests(ctx context.Context, tenantID string, dueWithin time.Duration) ([]EvidenceRequest, error) {
	deadline := time.Now().Add(dueWithin)
	reqs, err := c.store.ListPendingByTenant(ctx, tenantID, deadline)
	if err != nil {
		return nil, fmt.Errorf("list pending evidence requests for tenant %s: %w", tenantID, err)
	}

	// Sort by RequiredBy ascending (most urgent first).
	sort.Slice(reqs, func(i, j int) bool {
		return reqs[i].RequiredBy.Before(reqs[j].RequiredBy)
	})

	return reqs, nil
}
