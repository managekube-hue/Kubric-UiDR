// K-SEC-003 — Blake3 Fingerprint
// Cross-layer blake3 hashing for immutable audit log chain,
// PCAP integrity, config snapshot signing, and agent fingerprinting.
//
// Both zeebo/blake3 (Go) and the blake3 Rust crate produce identical
// 256-bit digests — enabling cross-layer verification.

package security

import (
	"encoding/hex"
	"fmt"

	"github.com/zeebo/blake3"
)

// Blake3Hash computes a 256-bit BLAKE3 digest and returns the hex string.
func Blake3Hash(data []byte) string {
	h := blake3.Sum256(data)
	return hex.EncodeToString(h[:])
}

// Blake3Chain computes the next hash in an immutable chain:
// hash(prev_hash_bytes || event_bytes).
// This produces a tamper-evident append-only ledger.
func Blake3Chain(prevHashHex string, eventData []byte) (string, error) {
	prevBytes, err := hex.DecodeString(prevHashHex)
	if err != nil {
		return "", fmt.Errorf("invalid prev hash hex: %w", err)
	}
	combined := append(prevBytes, eventData...)
	h := blake3.Sum256(combined)
	return hex.EncodeToString(h[:]), nil
}

// Blake3Verify checks that a given hash matches the expected chain value.
func Blake3Verify(prevHashHex string, eventData []byte, expectedHex string) (bool, error) {
	computed, err := Blake3Chain(prevHashHex, eventData)
	if err != nil {
		return false, err
	}
	return computed == expectedHex, nil
}

// AgentFingerprint generates a unique hardware fingerprint for agent identity.
// Combines: hostname + OS + kernel + MAC addresses + CPU model.
// The fingerprint is blake3-hashed and used for agent re-enrollment detection.
func AgentFingerprint(hostname, os, kernel string, macs []string, cpuModel string) string {
	combined := hostname + "|" + os + "|" + kernel + "|" + cpuModel
	for _, mac := range macs {
		combined += "|" + mac
	}
	return Blake3Hash([]byte(combined))
}

// BillingLedgerRoot computes a Merkle-like root for a billing period.
// Each invoice line item is hashed, then all hashes are combined.
func BillingLedgerRoot(lineItemHashes []string) (string, error) {
	var combined []byte
	for _, h := range lineItemHashes {
		b, err := hex.DecodeString(h)
		if err != nil {
			return "", fmt.Errorf("invalid line item hash: %w", err)
		}
		combined = append(combined, b...)
	}
	root := blake3.Sum256(combined)
	return hex.EncodeToString(root[:]), nil
}

// PcapIntegrity computes the blake3 hash of a PCAP file for chain-of-custody.
func PcapIntegrity(pcapData []byte) string {
	return Blake3Hash(pcapData)
}

// ConfigSnapshotSign hashes a config snapshot for drift detection.
// The hash is stored in ClickHouse alongside the config version.
func ConfigSnapshotSign(configYAML []byte) string {
	return Blake3Hash(configYAML)
}
