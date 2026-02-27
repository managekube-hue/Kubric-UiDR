// Package noc provides NOC operations tooling.
// K-NOC-BR-001 — Restic Backup Scheduler: schedule and run Restic backups via subprocess.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ResticSummary captures key metrics returned by restic backup --json.
type ResticSummary struct {
	FilesNew      int     `json:"files_new"`
	FilesChanged  int     `json:"files_changed"`
	FilesDone     int     `json:"files_done"`
	DataAdded     int64   `json:"data_added"`
	TotalDuration float64 `json:"total_duration"`
	SnapshotID    string  `json:"snapshot_id"`
}

// ResticSnapshot represents a single snapshot returned by restic snapshots --json.
type ResticSnapshot struct {
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Tags     []string  `json:"tags"`
	Paths    []string  `json:"paths"`
}

// ResticScheduler schedules and runs Restic backups.
type ResticScheduler struct {
	Repo         string
	PasswordFile string
	BackupPaths  []string
}

// NewResticScheduler reads RESTIC_REPOSITORY and RESTIC_PASSWORD_FILE from the environment.
func NewResticScheduler() *ResticScheduler {
	repo := os.Getenv("RESTIC_REPOSITORY")
	if repo == "" {
		repo = "s3:s3.amazonaws.com/kubric-backups"
	}
	pwFile := os.Getenv("RESTIC_PASSWORD_FILE")
	if pwFile == "" {
		pwFile = "/etc/restic/password"
	}
	var paths []string
	if p := os.Getenv("RESTIC_BACKUP_PATHS"); p != "" {
		paths = strings.Split(p, ":")
	} else {
		paths = []string{"/var/lib/kubric", "/etc/kubric"}
	}
	return &ResticScheduler{
		Repo:         repo,
		PasswordFile: pwFile,
		BackupPaths:  paths,
	}
}

func (rs *ResticScheduler) baseArgs() []string {
	return []string{"-r", rs.Repo, "--password-file", rs.PasswordFile}
}

// RunBackup runs restic backup for a single path and parses the JSON summary.
func (rs *ResticScheduler) RunBackup(ctx context.Context, backupPath string) (*ResticSummary, error) {
	args := append(rs.baseArgs(), "backup", backupPath, "--json")
	cmd := exec.CommandContext(ctx, "restic", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("restic backup %q: %w — stderr: %s", backupPath, err, stderr.String())
	}

	// restic --json emits multiple JSON objects; the final one is the summary.
	var summary ResticSummary
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var raw json.RawMessage
		if decErr := dec.Decode(&raw); decErr != nil {
			break
		}
		var candidate ResticSummary
		if err := json.Unmarshal(raw, &candidate); err == nil && candidate.SnapshotID != "" {
			summary = candidate
		}
	}
	return &summary, nil
}

// ForgetOld runs restic forget --prune with the standard retention policy.
func (rs *ResticScheduler) ForgetOld(ctx context.Context) error {
	args := append(rs.baseArgs(),
		"forget", "--json", "--prune",
		"--keep-daily", "7",
		"--keep-weekly", "4",
		"--keep-monthly", "6",
	)
	cmd := exec.CommandContext(ctx, "restic", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic forget: %w — stderr: %s", err, stderr.String())
	}
	return nil
}

// CheckIntegrity runs restic check to verify repository integrity.
func (rs *ResticScheduler) CheckIntegrity(ctx context.Context) error {
	args := append(rs.baseArgs(), "check")
	cmd := exec.CommandContext(ctx, "restic", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restic check: %w — stderr: %s", err, stderr.String())
	}
	return nil
}

// ListSnapshots returns all snapshots stored in the Restic repository.
func (rs *ResticScheduler) ListSnapshots(ctx context.Context) ([]ResticSnapshot, error) {
	args := append(rs.baseArgs(), "snapshots", "--json")
	cmd := exec.CommandContext(ctx, "restic", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("restic snapshots: %w — stderr: %s", err, stderr.String())
	}

	var snapshots []ResticSnapshot
	if err := json.Unmarshal(stdout.Bytes(), &snapshots); err != nil {
		return nil, fmt.Errorf("parse restic snapshots JSON: %w", err)
	}
	return snapshots, nil
}

// RunSchedule runs backup loops for all configured paths on the given interval.
// It blocks until ctx is cancelled.
func (rs *ResticScheduler) RunSchedule(ctx context.Context, interval time.Duration) {
	rs.executeBackupCycle(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rs.executeBackupCycle(ctx)
		}
	}
}

func (rs *ResticScheduler) executeBackupCycle(ctx context.Context) {
	for _, path := range rs.BackupPaths {
		select {
		case <-ctx.Done():
			return
		default:
		}
		summary, err := rs.RunBackup(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[restic] backup error path=%s err=%v\n", path, err)
			continue
		}
		fmt.Printf("[restic] backup ok path=%s snapshot=%s files_new=%d data_added=%d dur=%.2fs\n",
			path, summary.SnapshotID, summary.FilesNew, summary.DataAdded, summary.TotalDuration)
	}
	if err := rs.ForgetOld(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[restic] forget error: %v\n", err)
	}
}
