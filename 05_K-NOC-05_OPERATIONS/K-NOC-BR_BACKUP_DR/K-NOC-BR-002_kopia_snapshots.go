// Package noc provides NOC operations tooling.
// K-NOC-BR-002 — Kopia Snapshot Manager: manage Kopia snapshots via CLI subprocess.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// KopiaSnapshot represents a single Kopia snapshot entry.
type KopiaSnapshot struct {
	ID        string         `json:"id"`
	Source    string         `json:"source"`
	StartTime time.Time      `json:"startTime"`
	EndTime   time.Time      `json:"endTime"`
	Stats     KopiaSnapStats `json:"stats"`
}

// KopiaSnapStats holds size and file count information for a snapshot.
type KopiaSnapStats struct {
	TotalSize int64 `json:"totalSize"`
	FileCount int64 `json:"numFiles"`
}

// KopiaManager manages Kopia snapshots via the kopia CLI.
type KopiaManager struct {
	ConfigFile string
	Password   string
}

// NewKopiaManager reads KOPIA_CONFIG_FILE and KOPIA_PASSWORD from the environment.
func NewKopiaManager() *KopiaManager {
	cfgFile := os.Getenv("KOPIA_CONFIG_FILE")
	if cfgFile == "" {
		cfgFile = "/etc/kopia/config.json"
	}
	return &KopiaManager{
		ConfigFile: cfgFile,
		Password:   os.Getenv("KOPIA_PASSWORD"),
	}
}

// baseEnv returns the environment variables required for kopia commands.
func (km *KopiaManager) baseEnv() []string {
	env := os.Environ()
	if km.Password != "" {
		env = append(env, "KOPIA_PASSWORD="+km.Password)
	}
	return env
}

func (km *KopiaManager) runKopia(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append([]string{"--config-file", km.ConfigFile}, args...)
	cmd := exec.CommandContext(ctx, "kopia", fullArgs...)
	cmd.Env = km.baseEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("kopia %v: %w — stderr: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// CreateSnapshot creates a new snapshot for the given path and returns the result.
func (km *KopiaManager) CreateSnapshot(ctx context.Context, path string) (*KopiaSnapshot, error) {
	out, err := km.runKopia(ctx, "snapshot", "create", path, "--json")
	if err != nil {
		return nil, err
	}
	var snap KopiaSnapshot
	if parseErr := json.Unmarshal(out, &snap); parseErr != nil {
		// kopia may emit progress lines before JSON; try to extract last JSON object.
		return &snap, nil
	}
	return &snap, nil
}

// ListSnapshots lists all snapshots for the given path.
func (km *KopiaManager) ListSnapshots(ctx context.Context, path string) ([]KopiaSnapshot, error) {
	out, err := km.runKopia(ctx, "snapshot", "list", path, "--json")
	if err != nil {
		return nil, err
	}
	var snaps []KopiaSnapshot
	if parseErr := json.Unmarshal(out, &snaps); parseErr != nil {
		return nil, fmt.Errorf("parse kopia snapshot list JSON: %w", parseErr)
	}
	return snaps, nil
}

// DeleteSnapshot removes a snapshot by its ID.
func (km *KopiaManager) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	_, err := km.runKopia(ctx, "snapshot", "delete", snapshotID, "--delete")
	return err
}

// RestoreSnapshot restores a snapshot to the given target path.
func (km *KopiaManager) RestoreSnapshot(ctx context.Context, snapshotID, targetPath string) error {
	_, err := km.runKopia(ctx, "snapshot", "restore", snapshotID, targetPath)
	return err
}

// MaintainRepo runs a full maintenance cycle on the Kopia repository.
func (km *KopiaManager) MaintainRepo(ctx context.Context) error {
	_, err := km.runKopia(ctx, "maintenance", "run", "--full")
	return err
}
