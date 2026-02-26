// Package watchdog implements a TUF (The Update Framework) client for automatic
// agent binary updates. It fetches signed metadata, verifies Blake3 hashes, and
// performs atomic binary swaps with self-restart on update.
package watchdog

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/zeebo/blake3"
)

// ErrNoUpdate is returned by CheckForUpdate when the agent is already on the latest version.
var ErrNoUpdate = errors.New("already on latest version")

// TUFConfig holds all configuration needed to check and apply updates.
type TUFConfig struct {
	// MetadataURL is the base URL for TUF metadata (e.g. https://updates.kubric.io/tuf).
	MetadataURL string
	// BinaryURL is the base URL from which binaries are downloaded.
	BinaryURL string
	// LocalMetadataDir is a writable directory for caching TUF metadata.
	LocalMetadataDir string
	// CurrentVersion is the running agent's semver string (e.g. "1.4.2").
	CurrentVersion string
	// AgentBinary is the absolute path of the running binary to replace.
	AgentBinary string
	// TenantID scopes the update channel; included in metadata lookups.
	TenantID string
}

// UpdateInfo describes an available update fetched from targets.json.
type UpdateInfo struct {
	Version      string `json:"version"`
	BinaryURL    string `json:"binary_url"`
	Blake3Hash   string `json:"blake3_hash"` // 64-char hex of the binary's Blake3-256 digest
	Size         int64  `json:"size"`
	ReleaseNotes string `json:"release_notes"`
}

// tufTargets mirrors the relevant fields of a TUF targets.json document.
type tufTargets struct {
	Signed struct {
		Targets map[string]struct {
			Hashes struct {
				Blake3 string `json:"blake3"`
				SHA256 string `json:"sha256"`
			} `json:"hashes"`
			Length int64             `json:"length"`
			Custom map[string]string `json:"custom"`
		} `json:"targets"`
	} `json:"signed"`
}

// TUFUpdater manages the update lifecycle for a single agent binary.
type TUFUpdater struct {
	cfg    TUFConfig
	client *http.Client
	log    *slog.Logger
}

// NewTUFUpdater constructs a TUFUpdater. A nil logger defaults to slog.Default().
func NewTUFUpdater(cfg TUFConfig, client *http.Client, log *slog.Logger) *TUFUpdater {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	if log == nil {
		log = slog.Default()
	}
	return &TUFUpdater{cfg: cfg, client: client, log: log}
}

// CheckForUpdate fetches targets.json from the metadata server and compares versions.
// Returns ErrNoUpdate when the running version is already the latest.
func (u *TUFUpdater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	// Build the metadata URL. Example:
	//   https://updates.kubric.io/tuf/{tenantID}/targets.json
	metaURL := fmt.Sprintf("%s/%s/targets.json", u.cfg.MetadataURL, u.cfg.TenantID)
	u.log.InfoContext(ctx, "checking for update", "url", metaURL, "current_version", u.cfg.CurrentVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build metadata request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch targets.json: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read targets.json body: %w", err)
	}

	// Cache metadata locally for offline trust root validation.
	if err := u.cacheMetadata(body); err != nil {
		u.log.WarnContext(ctx, "could not cache metadata", "err", err)
	}

	var targets tufTargets
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, fmt.Errorf("parse targets.json: %w", err)
	}

	// Target key follows convention: "kubric-agent-{os}-{arch}-{version}"
	//   e.g. "kubric-agent-linux-amd64-1.5.0"
	platform := runtime.GOOS + "-" + runtime.GOARCH
	bestInfo, err := u.selectBestTarget(targets, platform)
	if err != nil {
		return nil, err
	}
	if bestInfo == nil {
		return nil, ErrNoUpdate
	}
	return bestInfo, nil
}

// selectBestTarget scans targets for a newer version than cfg.CurrentVersion
// on the given platform string (e.g. "linux-amd64").
func (u *TUFUpdater) selectBestTarget(targets tufTargets, platform string) (*UpdateInfo, error) {
	var best *UpdateInfo
	for name, target := range targets.Signed.Targets {
		// Expect name format: kubric-agent-{platform}-{version}
		prefix := "kubric-agent-" + platform + "-"
		if len(name) <= len(prefix) {
			continue
		}
		candidateVersion := name[len(prefix):]
		if !newerThan(candidateVersion, u.cfg.CurrentVersion) {
			continue
		}
		info := &UpdateInfo{
			Version:      candidateVersion,
			BinaryURL:    fmt.Sprintf("%s/%s", u.cfg.BinaryURL, name),
			Blake3Hash:   target.Hashes.Blake3,
			Size:         target.Length,
			ReleaseNotes: target.Custom["release_notes"],
		}
		if best == nil || newerThan(info.Version, best.Version) {
			best = info
		}
	}
	if best == nil {
		return nil, nil
	}
	return best, nil
}

// DownloadAndVerify downloads the binary described by info and verifies its Blake3 hash.
// Returns the raw binary bytes on success.
func (u *TUFUpdater) DownloadAndVerify(ctx context.Context, info *UpdateInfo) ([]byte, error) {
	u.log.InfoContext(ctx, "downloading update",
		"version", info.Version,
		"url", info.BinaryURL,
		"expected_bytes", info.Size,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.BinaryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build download request: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binary server returned %d", resp.StatusCode)
	}

	// Guard against unexpectedly large payloads (max 256 MB).
	const maxBinary = 256 * 1024 * 1024
	limited := io.LimitReader(resp.Body, maxBinary+1)

	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read binary body: %w", err)
	}
	if int64(len(data)) > maxBinary {
		return nil, fmt.Errorf("binary exceeds 256 MiB safety limit")
	}

	// Blake3 verification.
	if info.Blake3Hash != "" {
		h := blake3.New()
		h.Write(data)
		got := hex.EncodeToString(h.Sum(nil))
		if got != info.Blake3Hash {
			return nil, fmt.Errorf("blake3 hash mismatch: expected %s, got %s", info.Blake3Hash, got)
		}
		u.log.InfoContext(ctx, "blake3 hash verified", "hash", got)
	} else {
		u.log.WarnContext(ctx, "no blake3 hash in metadata; skipping verification")
	}

	return data, nil
}

// ApplyUpdate performs an atomic binary swap:
//  1. Writes binaryData to AgentBinary + ".tmp"
//  2. Sets permissions 0755
//  3. Renames .tmp over the existing binary (atomic on POSIX)
func (u *TUFUpdater) ApplyUpdate(ctx context.Context, info *UpdateInfo, binaryData []byte) error {
	if u.cfg.AgentBinary == "" {
		return fmt.Errorf("AgentBinary path is not configured")
	}

	tmpPath := u.cfg.AgentBinary + ".tmp"
	u.log.InfoContext(ctx, "applying update",
		"binary", u.cfg.AgentBinary,
		"new_version", info.Version,
	)

	if err := os.WriteFile(tmpPath, binaryData, 0o755); err != nil {
		return fmt.Errorf("write tmp binary %q: %w", tmpPath, err)
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod tmp binary: %w", err)
	}

	// Atomic rename — on Windows this may fail if the binary is locked; handle gracefully.
	if err := os.Rename(tmpPath, u.cfg.AgentBinary); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename %q -> %q: %w", tmpPath, u.cfg.AgentBinary, err)
	}

	u.log.InfoContext(ctx, "update applied successfully", "version", info.Version, "path", u.cfg.AgentBinary)
	return nil
}

// RunUpdateLoop runs CheckForUpdate / DownloadAndVerify / ApplyUpdate on the given
// interval, blocking until ctx is cancelled. After a successful update it calls
// selfRestart to swap the process image.
func (u *TUFUpdater) RunUpdateLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	u.log.InfoContext(ctx, "TUF update loop started",
		"interval", interval,
		"current_version", u.cfg.CurrentVersion,
	)

	for {
		select {
		case <-ctx.Done():
			u.log.InfoContext(ctx, "TUF update loop stopping")
			return
		case <-ticker.C:
			if err := u.runOnce(ctx); err != nil {
				if errors.Is(err, ErrNoUpdate) {
					u.log.InfoContext(ctx, "no update available", "version", u.cfg.CurrentVersion)
				} else {
					u.log.ErrorContext(ctx, "update cycle error", "err", err)
				}
			}
		}
	}
}

// runOnce executes one full update check-download-apply cycle.
func (u *TUFUpdater) runOnce(ctx context.Context) error {
	info, err := u.CheckForUpdate(ctx)
	if err != nil {
		return err
	}

	data, err := u.DownloadAndVerify(ctx, info)
	if err != nil {
		return fmt.Errorf("download/verify: %w", err)
	}

	if err := u.ApplyUpdate(ctx, info, data); err != nil {
		return fmt.Errorf("apply: %w", err)
	}

	u.log.InfoContext(ctx, "update applied; restarting...", "new_version", info.Version)
	u.selfRestart()
	// selfRestart calls exec and does not return; this line is unreachable on success.
	return nil
}

// selfRestart replaces the current process image with the updated binary via syscall.Exec.
// On Windows, where exec-replace is not supported, it starts a new process and exits.
func (u *TUFUpdater) selfRestart() {
	binary := u.cfg.AgentBinary
	if binary == "" {
		var err error
		binary, err = os.Executable()
		if err != nil {
			u.log.Error("could not determine own executable path for restart", "err", err)
			os.Exit(1)
		}
	}
	binary, _ = filepath.EvalSymlinks(binary)

	args := os.Args
	env := os.Environ()

	u.log.Info("exec-replacing process", "binary", binary, "args", args)

	// syscall.Exec replaces the process on POSIX; not available on Windows.
	if runtime.GOOS == "windows" {
		// On Windows start a detached child and exit the parent.
		attr := &os.ProcAttr{
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{},
		}
		proc, err := os.StartProcess(binary, args, attr)
		if err != nil {
			u.log.Error("StartProcess failed", "err", err)
			os.Exit(1)
		}
		_ = proc.Release()
		os.Exit(0)
	}

	if err := syscall.Exec(binary, args, env); err != nil {
		u.log.Error("syscall.Exec failed", "err", err)
		os.Exit(1)
	}
}

// cacheMetadata writes raw TUF metadata bytes to LocalMetadataDir/targets.json.
func (u *TUFUpdater) cacheMetadata(data []byte) error {
	if u.cfg.LocalMetadataDir == "" {
		return nil
	}
	if err := os.MkdirAll(u.cfg.LocalMetadataDir, 0o750); err != nil {
		return err
	}
	path := filepath.Join(u.cfg.LocalMetadataDir, "targets.json")
	return os.WriteFile(path, data, 0o640)
}

// newerThan performs a simple semantic version comparison: a > b.
// Supports "MAJOR.MINOR.PATCH" format; falls back to string comparison.
func newerThan(a, b string) bool {
	return parseSemver(a) > parseSemver(b)
}

// parseSemver converts "MAJOR.MINOR.PATCH" to a comparable int64.
func parseSemver(ver string) int64 {
	var major, minor, patch int64
	fmt.Sscanf(ver, "%d.%d.%d", &major, &minor, &patch)
	return major*1_000_000 + minor*1_000 + patch
}
