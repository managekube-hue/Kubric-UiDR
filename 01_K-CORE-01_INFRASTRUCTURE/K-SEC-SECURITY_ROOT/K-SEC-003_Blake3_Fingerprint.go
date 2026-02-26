//go:build ignore

// K-SEC-003 — Blake3 Fingerprint (reference stub)
//
// Production implementation: internal/security/blake3_fingerprint.go
//
// Functions:
//   Blake3Hash(data []byte) string
//   Blake3Chain(prevHashHex string, eventData []byte) (string, error)
//   Blake3Verify(prevHashHex string, eventData []byte, expectedHex string) (bool, error)
//   AgentFingerprint(hostname, os, kernel string, macs []string, cpuModel string) string
//   BillingLedgerRoot(lineItemHashes []string) (string, error)
//   PcapIntegrity(pcapData []byte) string
//   ConfigSnapshotSign(configYAML []byte) string
//
// Go uses github.com/zeebo/blake3 (256-bit).
// Rust agents use the blake3 crate — identical digests for cross-layer verification.

package stub
