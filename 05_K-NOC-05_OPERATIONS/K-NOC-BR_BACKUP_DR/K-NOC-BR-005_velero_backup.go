// Package noc provides NOC operations tooling.
// K-NOC-BR-005 — Velero Kubernetes Backup: trigger and monitor Velero backup operations.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// VeleroBackupInfo holds details for a single Velero backup object.
type VeleroBackupInfo struct {
	Name                string    `json:"name"`
	Namespace           string    `json:"namespace"`
	Status              string    `json:"status"`
	StartTimestamp      time.Time `json:"startTimestamp"`
	CompletionTimestamp time.Time `json:"completionTimestamp"`
	Expiration          time.Time `json:"expiration"`
}

// VeleroBackup triggers and monitors Velero backup operations.
type VeleroBackup struct {
	Namespace      string
	KubeconfigPath string
}

// NewVeleroBackup reads VELERO_NAMESPACE and KUBECONFIG from the environment.
func NewVeleroBackup() *VeleroBackup {
	ns := os.Getenv("VELERO_NAMESPACE")
	if ns == "" {
		ns = "velero"
	}
	return &VeleroBackup{
		Namespace:      ns,
		KubeconfigPath: os.Getenv("KUBECONFIG"),
	}
}

func (vb *VeleroBackup) veleroCmd(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append(args, "--namespace", vb.Namespace)
	if vb.KubeconfigPath != "" {
		fullArgs = append(fullArgs, "--kubeconfig", vb.KubeconfigPath)
	}
	return exec.CommandContext(ctx, "velero", fullArgs...)
}

// CreateBackup creates a Velero backup including the given namespace with optional labels.
func (vb *VeleroBackup) CreateBackup(ctx context.Context, name, namespace string, labels map[string]string) error {
	args := []string{"backup", "create", name, "--include-namespaces", namespace, "--wait=false"}
	for k, v := range labels {
		args = append(args, "--labels", fmt.Sprintf("%s=%s", k, v))
	}
	cmd := vb.veleroCmd(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("velero backup create %q: %w — stderr: %s", name, err, stderr.String())
	}
	return nil
}

// GetBackupStatus returns the phase of the named backup.
func (vb *VeleroBackup) GetBackupStatus(ctx context.Context, name string) (string, error) {
	cmd := vb.veleroCmd(ctx, "backup", "get", name, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("velero backup get %q: %w", name, err)
	}

	var obj struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}
	if parseErr := json.Unmarshal(out, &obj); parseErr != nil {
		return "", fmt.Errorf("parse velero backup JSON: %w", parseErr)
	}
	return obj.Status.Phase, nil
}

// WaitForCompletion polls until the backup reaches a terminal phase or timeout.
func (vb *VeleroBackup) WaitForCompletion(ctx context.Context, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		phase, err := vb.GetBackupStatus(ctx, name)
		if err != nil {
			return err
		}
		switch phase {
		case "Completed":
			return nil
		case "Failed", "PartiallyFailed":
			return fmt.Errorf("backup %q finished with phase %q", name, phase)
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout waiting for backup %q after %s", name, timeout)
}

// ListBackups returns all Velero backups.
func (vb *VeleroBackup) ListBackups(ctx context.Context) ([]VeleroBackupInfo, error) {
	cmd := vb.veleroCmd(ctx, "backup", "get", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("velero backup get list: %w", err)
	}

	var list struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Status struct {
				Phase               string    `json:"phase"`
				StartTimestamp      time.Time `json:"startTimestamp"`
				CompletionTimestamp time.Time `json:"completionTimestamp"`
				Expiration          time.Time `json:"expiration"`
			} `json:"status"`
		} `json:"items"`
	}

	if parseErr := json.Unmarshal(out, &list); parseErr != nil {
		// velero may return a single object when there is only one backup
		return nil, fmt.Errorf("parse velero list JSON: %w", parseErr)
	}

	result := make([]VeleroBackupInfo, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, VeleroBackupInfo{
			Name:                item.Metadata.Name,
			Namespace:           item.Metadata.Namespace,
			Status:              item.Status.Phase,
			StartTimestamp:      item.Status.StartTimestamp,
			CompletionTimestamp: item.Status.CompletionTimestamp,
			Expiration:          item.Status.Expiration,
		})
	}
	return result, nil
}

// DeleteBackup deletes the named backup.
func (vb *VeleroBackup) DeleteBackup(ctx context.Context, name string) error {
	cmd := vb.veleroCmd(ctx, "backup", "delete", name, "--confirm")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("velero backup delete %q: %w — stderr: %s", name, err, stderr.String())
	}
	return nil
}

// ScheduleDaily creates a Velero daily schedule for the target namespace at 02:00 UTC
// with a 30-day TTL.
func (vb *VeleroBackup) ScheduleDaily(ctx context.Context, scheduleName, targetNS string) error {
	args := []string{
		"schedule", "create", scheduleName,
		"--schedule", "0 2 * * *",
		"--include-namespaces", targetNS,
		"--ttl", "720h",
	}
	cmd := vb.veleroCmd(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		// Ignore "already exists" errors.
		if strings.Contains(stderr.String(), "already exists") {
			return nil
		}
		return fmt.Errorf("velero schedule create %q: %w — stderr: %s", scheduleName, err, stderr.String())
	}
	_, _ = io.Discard.Write(out)
	return nil
}
