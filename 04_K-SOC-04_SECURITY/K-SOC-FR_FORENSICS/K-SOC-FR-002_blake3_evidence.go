// K-SOC-FR-002 — Evidence integrity hashing with BLAKE3.
// Provides deep file hashing, directory manifests, and tamper-verification
// utilities for the forensic evidence pipeline.
//
// blake3Sum and blake3 package usage: zeebo/blake3 (in go.mod).
// blake3Sum (internal helper) is defined in K-SOC-FR-001_evidence_capture.go
// and is available throughout this package.
package soc

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// ---------------------------------------------------------------------------
// Manifest types
// ---------------------------------------------------------------------------

// FileHash is a single file entry in an EvidenceManifest.
type FileHash struct {
	RelPath string `json:"rel_path"`
	Hash    string `json:"hash"` // BLAKE3 hex digest
	SizeBytes int64 `json:"size_bytes"`
}

// EvidenceManifest is a tamper-evident list of hashed evidence files.
type EvidenceManifest struct {
	Files     []FileHash `json:"files"`
	CreatedAt time.Time  `json:"created_at"`
	SignedBy  string     `json:"signed_by,omitempty"`
}

// ---------------------------------------------------------------------------
// EvidenceHasher
// ---------------------------------------------------------------------------

// EvidenceHasher provides BLAKE3-based integrity operations over individual
// files and directory trees.
type EvidenceHasher struct{}

// NewEvidenceHasher returns a ready EvidenceHasher.
func NewEvidenceHasher() *EvidenceHasher {
	return &EvidenceHasher{}
}

// HashFile computes the BLAKE3 digest of the file at path.
// Returns hex-encoded 32-byte digest or an error.
func (h *EvidenceHasher) HashFile(path string) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("evidence_hasher: read %q: %w", path, err)
	}
	// blake3Sum is defined in K-SOC-FR-001_evidence_capture.go (same package).
	return blake3Sum(data), nil
}

// HashBytes computes the BLAKE3 digest of a byte slice and returns the
// hex-encoded result.
func (h *EvidenceHasher) HashBytes(data []byte) string {
	return blake3Sum(data)
}

// VerifyFile recomputes the BLAKE3 hash of the file at path and compares it
// to expectedHash.  Returns (true, nil) if they match.
func (h *EvidenceHasher) VerifyFile(path, expectedHash string) (bool, error) {
	actual, err := h.HashFile(path)
	if err != nil {
		return false, err
	}
	return actual == expectedHash, nil
}

// HashDirectory walks dirPath recursively, hashes every regular file, and
// returns a map of relativePath → BLAKE3-hex-digest.
func (h *EvidenceHasher) HashDirectory(dirPath string) (map[string]string, error) {
	hashes := make(map[string]string)

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("evidence_hasher: walk %q: %w", path, walkErr)
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			// Skip symlinks to avoid loops.
			return nil
		}

		digest, err := h.HashFile(path)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dirPath, path)
		if err != nil {
			rel = path
		}
		hashes[rel] = digest
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hashes, nil
}

// ---------------------------------------------------------------------------
// Manifest operations
// ---------------------------------------------------------------------------

// WriteManifest computes BLAKE3 hashes for all files in roots (files or
// directories), populates an EvidenceManifest, and writes it as pretty-
// printed JSON to outPath.
func (h *EvidenceHasher) WriteManifest(outPath string, manifest EvidenceManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("evidence_hasher: marshal manifest: %w", err)
	}
	if err := os.WriteFile(outPath, data, 0600); err != nil {
		return fmt.Errorf("evidence_hasher: write manifest %q: %w", outPath, err)
	}
	return nil
}

// BuildManifest hashes a directory tree and returns a populated EvidenceManifest.
// Pass signedBy = "" to omit the signature field.
func (h *EvidenceHasher) BuildManifest(dirPath, signedBy string) (EvidenceManifest, error) {
	hashes, err := h.HashDirectory(dirPath)
	if err != nil {
		return EvidenceManifest{}, err
	}

	files := make([]FileHash, 0, len(hashes))
	for rel, digest := range hashes {
		fullPath := filepath.Join(dirPath, rel)
		fi, statErr := os.Stat(fullPath)
		var size int64
		if statErr == nil {
			size = fi.Size()
		}
		files = append(files, FileHash{
			RelPath:   rel,
			Hash:      digest,
			SizeBytes: size,
		})
	}

	return EvidenceManifest{
		Files:     files,
		CreatedAt: time.Now().UTC(),
		SignedBy:  signedBy,
	}, nil
}

// VerifyManifest reads the manifest at manifestPath, re-hashes all listed
// files (relative to baseDir), and returns:
//   - ok: true if all hashes match
//   - invalid: list of rel-paths whose hashes have changed or files are missing
//   - error: I/O error reading manifest or files
func (h *EvidenceHasher) VerifyManifest(manifestPath, baseDir string) (bool, []string, error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec
	if err != nil {
		return false, nil, fmt.Errorf("evidence_hasher: read manifest %q: %w", manifestPath, err)
	}

	var manifest EvidenceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return false, nil, fmt.Errorf("evidence_hasher: parse manifest: %w", err)
	}

	var invalid []string
	for _, entry := range manifest.Files {
		fullPath := filepath.Join(baseDir, entry.RelPath)
		actual, err := h.HashFile(fullPath)
		if err != nil {
			invalid = append(invalid, fmt.Sprintf("%s: read error: %v", entry.RelPath, err))
			continue
		}
		if actual != entry.Hash {
			invalid = append(invalid, fmt.Sprintf("%s: hash mismatch (stored=%s actual=%s)",
				entry.RelPath, entry.Hash, actual))
		}
	}

	return len(invalid) == 0, invalid, nil
}
