// K-XRO-WD-002 — Go wrapper for zstd delta compression (OTA updates).
//
// Uses github.com/klauspost/compress/zstd with the old binary as a
// zstd dictionary to produce a compact delta patch.
//
// Delta compression flow:
//   1. The current agent binary ("old") is used as a zstd dictionary.
//   2. The new binary is compressed against that dictionary.
//   3. The resulting patch is far smaller than the full binary.
//   4. Agents download the patch and reconstruct the new binary locally.
//
// Integrity:
//   - SHA-256 of both old and new data is stored in DeltaPatch.
//   - DecompressDelta verifies old hash before decompressing and new hash
//     after, rejecting any tampered patch.
//
// Dictionary window size: 4 MiB (1<<22) — covers typical agent binaries.

package watchdog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// defaultWindowSize is the zstd window size (4 MiB).
const defaultWindowSize = 1 << 22

// DeltaPatch holds a compressed delta and the integrity digests.
type DeltaPatch struct {
	// CompressedData is the zstd-compressed delta bytes.
	CompressedData []byte
	// OldSHA256 is the hex-encoded SHA-256 of the original (old) data.
	OldSHA256 string
	// NewSHA256 is the hex-encoded SHA-256 of the reconstructed (new) data.
	NewSHA256 string
	// OriginalSize is the uncompressed size of the new data in bytes.
	OriginalSize int
}

// CompressDelta compresses newData using oldData as a zstd dictionary.
//
// The dictionary allows zstd to encode only the differences between the
// two blobs, resulting in much smaller deltas for incremental updates.
//
// Dictionary size is capped at defaultWindowSize to bound memory usage.
func CompressDelta(oldData, newData []byte) (*DeltaPatch, error) {
	if len(newData) == 0 {
		return nil, fmt.Errorf("compress delta: newData is empty")
	}

	// Cap dictionary size
	dict := oldData
	if len(dict) > defaultWindowSize {
		dict = dict[:defaultWindowSize]
	}

	// Build zstd encoder with dictionary
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(
		&buf,
		zstd.WithWindowSize(defaultWindowSize),
		zstd.WithEncoderDict(dict),
		zstd.WithEncoderLevel(zstd.SpeedBestCompression),
	)
	if err != nil {
		return nil, fmt.Errorf("compress delta: create encoder: %w", err)
	}

	if _, err := enc.Write(newData); err != nil {
		return nil, fmt.Errorf("compress delta: write: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("compress delta: close encoder: %w", err)
	}

	return &DeltaPatch{
		CompressedData: buf.Bytes(),
		OldSHA256:      hashSHA256(oldData),
		NewSHA256:      hashSHA256(newData),
		OriginalSize:   len(newData),
	}, nil
}

// DecompressDelta reconstructs newData from oldData and a DeltaPatch.
//
// Verification steps:
//  1. SHA-256(oldData) must match patch.OldSHA256
//  2. Decompress using oldData as dictionary
//  3. SHA-256(result) must match patch.NewSHA256
func DecompressDelta(oldData []byte, patch *DeltaPatch) ([]byte, error) {
	if patch == nil {
		return nil, fmt.Errorf("decompress delta: patch is nil")
	}

	// Step 1: verify old data integrity
	actualOld := hashSHA256(oldData)
	if actualOld != patch.OldSHA256 {
		return nil, fmt.Errorf(
			"decompress delta: old data hash mismatch: expected=%s got=%s",
			patch.OldSHA256, actualOld,
		)
	}

	// Cap dictionary
	dict := oldData
	if len(dict) > defaultWindowSize {
		dict = dict[:defaultWindowSize]
	}

	// Step 2: decompress with dictionary
	dec, err := zstd.NewReader(
		bytes.NewReader(patch.CompressedData),
		zstd.WithDecoderDicts(dict),
		zstd.WithDecoderMaxMemory(uint64(patch.OriginalSize)*2+defaultWindowSize),
	)
	if err != nil {
		return nil, fmt.Errorf("decompress delta: create decoder: %w", err)
	}
	defer dec.Close()

	newData := make([]byte, 0, patch.OriginalSize)
	var readBuf bytes.Buffer
	if _, err := readBuf.ReadFrom(dec); err != nil {
		return nil, fmt.Errorf("decompress delta: read: %w", err)
	}
	newData = readBuf.Bytes()

	// Step 3: verify new data integrity
	actualNew := hashSHA256(newData)
	if actualNew != patch.NewSHA256 {
		return nil, fmt.Errorf(
			"decompress delta: new data hash mismatch: expected=%s got=%s",
			patch.NewSHA256, actualNew,
		)
	}

	return newData, nil
}

// CompressRaw compresses data with zstd at best-compression level (no dictionary).
// Used for full binary distribution when no previous version is available.
func CompressRaw(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf,
		zstd.WithEncoderLevel(zstd.SpeedBestCompression),
		zstd.WithWindowSize(defaultWindowSize),
	)
	if err != nil {
		return nil, fmt.Errorf("compress raw: %w", err)
	}
	if _, err := enc.Write(data); err != nil {
		return nil, fmt.Errorf("compress raw write: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("compress raw close: %w", err)
	}
	return buf.Bytes(), nil
}

// DecompressRaw decompresses a zstd stream (no dictionary).
func DecompressRaw(compressed []byte, maxSize int) ([]byte, error) {
	dec, err := zstd.NewReader(
		bytes.NewReader(compressed),
		zstd.WithDecoderMaxMemory(uint64(maxSize)),
	)
	if err != nil {
		return nil, fmt.Errorf("decompress raw: %w", err)
	}
	defer dec.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(dec); err != nil {
		return nil, fmt.Errorf("decompress raw read: %w", err)
	}
	return buf.Bytes(), nil
}

// hashSHA256 returns the hex-encoded SHA-256 digest of data.
func hashSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// PatchStats returns human-readable statistics for a DeltaPatch.
func PatchStats(patch *DeltaPatch) string {
	if patch == nil {
		return "nil patch"
	}
	ratio := 0.0
	if patch.OriginalSize > 0 {
		ratio = float64(len(patch.CompressedData)) / float64(patch.OriginalSize) * 100
	}
	return fmt.Sprintf(
		"original=%d compressed=%d ratio=%.1f%% old_sha=%s… new_sha=%s…",
		patch.OriginalSize,
		len(patch.CompressedData),
		ratio,
		truncate(patch.OldSHA256, 8),
		truncate(patch.NewSHA256, 8),
	)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
