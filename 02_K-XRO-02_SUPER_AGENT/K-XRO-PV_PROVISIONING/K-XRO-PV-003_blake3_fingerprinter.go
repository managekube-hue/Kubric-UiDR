// Package provisioning provides agent fingerprinting using Blake3 for hardware identity.
// This is the Go reference implementation matching agents/provisioning/src/fingerprint.rs
package provisioning

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/zeebo/blake3"
)

// AgentFingerprint computes a stable Blake3 hardware identity hash.
// Matches the Rust implementation in agents/provisioning/src/fingerprint.rs.
// When TPM is unavailable (VMs, containers), uses software fallback.
func AgentFingerprint(hostname, osName, kernel string, macs []string, cpuModel string) string {
	// Sort MACs for determinism
	sorted := make([]string, len(macs))
	copy(sorted, macs)
	sort.Strings(sorted)

	input := strings.Join([]string{hostname, osName, kernel, strings.Join(sorted, ","), cpuModel}, "|")
	h := blake3.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// CollectFingerprintInputs gathers the local machine's identity parameters.
func CollectFingerprintInputs() (hostname, osName, kernel string, macs []string, cpuModel string, err error) {
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
		err = nil
	}

	osName = runtime.GOOS
	kernel = getKernelVersion()
	macs = collectMACAddresses()
	cpuModel = getCPUModel()
	return
}

// LocalFingerprint computes the fingerprint for this machine.
func LocalFingerprint() (string, error) {
	hostname, osName, kernel, macs, cpuModel, err := CollectFingerprintInputs()
	if err != nil {
		return "", fmt.Errorf("collect fingerprint inputs: %w", err)
	}
	return AgentFingerprint(hostname, osName, kernel, macs, cpuModel), nil
}

// VerifyFingerprint checks whether the locally computed fingerprint matches
// the one stored during provisioning.
func VerifyFingerprint(storedFingerprint string) (bool, string, error) {
	computed, err := LocalFingerprint()
	if err != nil {
		return false, "", err
	}
	return computed == storedFingerprint, computed, nil
}

// BillingLedgerRoot computes a Merkle root over a list of line-item hashes.
// Matches internal/security/blake3_fingerprint.go:BillingLedgerRoot
func BillingLedgerRoot(lineItemHashes []string) (string, error) {
	if len(lineItemHashes) == 0 {
		return "", fmt.Errorf("cannot compute merkle root of empty list")
	}
	leaves := make([][]byte, len(lineItemHashes))
	for i, h := range lineItemHashes {
		b, err := hex.DecodeString(h)
		if err != nil {
			return "", fmt.Errorf("invalid hex at index %d: %w", i, err)
		}
		leaves[i] = b
	}
	root := merkleRoot(leaves)
	return hex.EncodeToString(root), nil
}

// ConfigSnapshotSign computes a Blake3 hash of a config YAML for tamper detection.
func ConfigSnapshotSign(configYAML []byte) string {
	h := blake3.New()
	h.Write(configYAML)
	return hex.EncodeToString(h.Sum(nil))
}

// PcapIntegrity computes a Blake3 hash of PCAP data for evidence chain integrity.
func PcapIntegrity(pcapData []byte) string {
	h := blake3.New()
	h.Write(pcapData)
	return hex.EncodeToString(h.Sum(nil))
}

// HashFile computes the Blake3 hash of a file by path.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}
	h := blake3.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Blake3Chain extends an audit chain: returns Blake3(prevHash || eventData).
func Blake3Chain(prevHashHex string, eventData []byte) (string, error) {
	prev, err := hex.DecodeString(prevHashHex)
	if err != nil {
		return "", fmt.Errorf("invalid previous hash hex: %w", err)
	}
	h := blake3.New()
	h.Write(prev)
	h.Write(eventData)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Blake3Verify verifies an audit chain entry.
func Blake3Verify(prevHashHex string, eventData []byte, expectedHex string) (bool, error) {
	computed, err := Blake3Chain(prevHashHex, eventData)
	if err != nil {
		return false, err
	}
	return computed == expectedHex, nil
}

// --- Internal helpers ---

func collectMACAddresses() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var macs []string
	for _, iface := range ifaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			mac := iface.HardwareAddr.String()
			// Skip loopback and all-zero virtual interfaces.
			if mac != "" && !strings.HasPrefix(mac, "00:00:00") {
				macs = append(macs, mac)
			}
		}
	}
	sort.Strings(macs)
	return macs
}

func merkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 1 {
		return leaves[0]
	}
	if len(leaves)%2 != 0 {
		leaves = append(leaves, leaves[len(leaves)-1]) // duplicate last leaf for odd count
	}
	var next [][]byte
	for i := 0; i < len(leaves); i += 2 {
		combined := make([]byte, 0, len(leaves[i])+len(leaves[i+1]))
		combined = append(combined, leaves[i]...)
		combined = append(combined, leaves[i+1]...)
		h := sha256.Sum256(combined) // SHA-256 for Merkle interior nodes (Blake3 used for leaves)
		next = append(next, h[:])
	}
	return merkleRoot(next)
}

// getKernelVersion reads /proc/version on Linux; returns GOOS/GOARCH on other platforms.
func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return runtime.GOOS + "/" + runtime.GOARCH
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		return fields[2] // "Linux version X.Y.Z-..." -> third token is the release string
	}
	return strings.TrimSpace(string(data))
}

// blake3New is a package-level constructor alias so that other files in this
// package can create a blake3 hasher without importing the external package directly.
func blake3New() *blake3.Hasher {
	return blake3.New()
}

// getCPUModel reads the first "model name" entry from /proc/cpuinfo on Linux.
func getCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}
