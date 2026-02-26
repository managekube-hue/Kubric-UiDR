// K-XRO-WD-003 — Blake3 manifest signer for watchdog OTA updates.
//
// Builds and verifies signed update manifests for agent binary distribution.
//
// # Manifest format (JSON)
//
//  {
//    "version": "1.2.3",
//    "entries": [
//      {"filename": "kubric-coresec", "blake3_hash": "abc123…", "size": 12345678}
//    ],
//    "signature": "ed25519-hex-encoded-sig"
//  }
//
// # Signing scheme
//  1. Collect all files in the release directory.
//  2. Hash each file with Blake3 (256-bit).
//  3. Serialise the entries array to canonical JSON (sorted keys).
//  4. Sign the canonical JSON with Ed25519 (crypto/ed25519 stdlib).
//  5. Encode the signature as hex and attach to the manifest.
//
// # Verification
//  VerifyManifest decodes the hex signature and verifies it against the
//  canonical JSON of the entries array using the distributor's Ed25519 public key.

package watchdog

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/zeebo/blake3"
)

// ManifestEntry describes one file in the update package.
type ManifestEntry struct {
	// Filename is the base name of the file (no directory component).
	Filename string `json:"filename"`
	// Blake3Hash is the lowercase hex-encoded 256-bit Blake3 digest.
	Blake3Hash string `json:"blake3_hash"`
	// Size is the file size in bytes.
	Size int64 `json:"size"`
}

// UpdateManifest is the signed release manifest.
type UpdateManifest struct {
	// Version is the semantic version string for this update package.
	Version string `json:"version"`
	// Entries lists all files included in this update.
	Entries []ManifestEntry `json:"entries"`
	// Signature is the Ed25519 signature of the canonical JSON of Entries,
	// hex-encoded (128 hex characters / 64 bytes).
	Signature string `json:"signature"`
}

// HashFile returns the lowercase hex-encoded Blake3-256 digest of the file
// at the given path.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash file open %s: %w", path, err)
	}
	defer f.Close()

	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file read %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// BuildManifest scans dir for regular files, hashes each one with Blake3,
// and returns an unsigned UpdateManifest with the given version.
//
// Only regular (non-directory, non-symlink) files are included.
// Entries are sorted lexicographically by filename for determinism.
func BuildManifest(version string, dir string) (*UpdateManifest, error) {
	if version == "" {
		return nil, fmt.Errorf("build manifest: version must not be empty")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("build manifest: read dir %s: %w", dir, err)
	}

	var manifestEntries []ManifestEntry

	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			// Skip directories, symlinks, devices, etc.
			continue
		}

		path := filepath.Join(dir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("build manifest: stat %s: %w", path, err)
		}

		hash, err := HashFile(path)
		if err != nil {
			return nil, fmt.Errorf("build manifest: hash %s: %w", path, err)
		}

		manifestEntries = append(manifestEntries, ManifestEntry{
			Filename:   entry.Name(),
			Blake3Hash: hash,
			Size:       info.Size(),
		})
	}

	// Sort for deterministic canonical JSON
	sort.Slice(manifestEntries, func(i, j int) bool {
		return manifestEntries[i].Filename < manifestEntries[j].Filename
	})

	return &UpdateManifest{
		Version: version,
		Entries: manifestEntries,
	}, nil
}

// SignManifest computes the Ed25519 signature over the canonical JSON of the
// manifest's Entries array and stores it in manifest.Signature.
//
// The canonical message is:
//
//	json.Marshal(manifest.Entries)
//
// (deterministic because entries are sorted before signing)
func SignManifest(manifest *UpdateManifest, privateKey ed25519.PrivateKey) error {
	if manifest == nil {
		return fmt.Errorf("sign manifest: manifest is nil")
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return fmt.Errorf("sign manifest: invalid private key length %d (want %d)",
			len(privateKey), ed25519.PrivateKeySize)
	}

	message, err := canonicalEntriesJSON(manifest.Entries)
	if err != nil {
		return fmt.Errorf("sign manifest: serialise entries: %w", err)
	}

	sig := ed25519.Sign(privateKey, message)
	manifest.Signature = hex.EncodeToString(sig)
	return nil
}

// VerifyManifest verifies the Ed25519 signature stored in manifest.Signature
// against the canonical JSON of manifest.Entries.
//
// Returns (true, nil) on success, (false, nil) on signature mismatch,
// and (false, error) on malformed input.
func VerifyManifest(manifest *UpdateManifest, publicKey ed25519.PublicKey) (bool, error) {
	if manifest == nil {
		return false, fmt.Errorf("verify manifest: manifest is nil")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("verify manifest: invalid public key length %d (want %d)",
			len(publicKey), ed25519.PublicKeySize)
	}
	if manifest.Signature == "" {
		return false, fmt.Errorf("verify manifest: signature is empty")
	}

	sigBytes, err := hex.DecodeString(manifest.Signature)
	if err != nil {
		return false, fmt.Errorf("verify manifest: decode signature hex: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return false, fmt.Errorf("verify manifest: signature wrong length %d (want %d)",
			len(sigBytes), ed25519.SignatureSize)
	}

	message, err := canonicalEntriesJSON(manifest.Entries)
	if err != nil {
		return false, fmt.Errorf("verify manifest: serialise entries: %w", err)
	}

	return ed25519.Verify(publicKey, message, sigBytes), nil
}

// VerifyEntryFile verifies that the file at path matches the Blake3 hash
// and size recorded in entry.
func VerifyEntryFile(entry ManifestEntry, basedir string) error {
	path := filepath.Join(basedir, entry.Filename)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("verify entry %s: stat: %w", entry.Filename, err)
	}
	if info.Size() != entry.Size {
		return fmt.Errorf("verify entry %s: size mismatch (expected=%d actual=%d)",
			entry.Filename, entry.Size, info.Size())
	}

	hash, err := HashFile(path)
	if err != nil {
		return fmt.Errorf("verify entry %s: hash: %w", entry.Filename, err)
	}
	if hash != entry.Blake3Hash {
		return fmt.Errorf("verify entry %s: hash mismatch (expected=%s actual=%s)",
			entry.Filename, entry.Blake3Hash, hash)
	}

	return nil
}

// VerifyAllEntries verifies every file in the manifest against basedir.
// Returns a slice of errors (one per failed entry), or nil if all pass.
func VerifyAllEntries(manifest *UpdateManifest, basedir string) []error {
	var errs []error
	for _, entry := range manifest.Entries {
		if err := VerifyEntryFile(entry, basedir); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// canonicalEntriesJSON returns the deterministic JSON encoding of entries.
func canonicalEntriesJSON(entries []ManifestEntry) ([]byte, error) {
	return json.Marshal(entries)
}
