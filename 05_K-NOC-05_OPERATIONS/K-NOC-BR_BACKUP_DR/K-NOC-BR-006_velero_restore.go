// Package noc provides NOC operations tooling.
// K-NOC-BR-006 — Velero Kubernetes Restore: restore namespaces from Velero backups.
package noc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"
)

// VeleroRestoreInfo holds details for a single Velero restore object.
type VeleroRestoreInfo struct {
	Name                string    `json:"name"`
	BackupName          string    `json:"backupName"`
	Status              string    `json:"status"`
	StartTimestamp      time.Time `json:"startTimestamp"`
	CompletionTimestamp time.Time `json:"completionTimestamp"`
	Warnings            int       `json:"warnings"`
	Errors              int       `json:"errors"`
}

// VeleroRestore restores Kubernetes namespaces from Velero backups.
type VeleroRestore struct {
	Namespace      string
	KubeconfigPath string
}

// NewVeleroRestore reads VELERO_NAMESPACE and KUBECONFIG from the environment.
func NewVeleroRestore() *VeleroRestore {
	ns := os.Getenv("VELERO_NAMESPACE")
	if ns == "" {
		ns = "velero"
	}
	return &VeleroRestore{
		Namespace:      ns,
		KubeconfigPath: os.Getenv("KUBECONFIG"),
	}
}

func (vr *VeleroRestore) veleroCmd(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append(args, "--namespace", vr.Namespace)
	if vr.KubeconfigPath != "" {
		fullArgs = append(fullArgs, "--kubeconfig", vr.KubeconfigPath)
	}
	return exec.CommandContext(ctx, "velero", fullArgs...)
}

// CreateRestore creates a Velero restore from the given backup into targetNS.
func (vr *VeleroRestore) CreateRestore(ctx context.Context, restoreName, backupName, targetNS string) error {
	args := []string{
		"restore", "create", restoreName,
		"--from-backup", backupName,
		"--include-namespaces", targetNS,
	}
	cmd := vr.veleroCmd(ctx, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("velero restore create %q: %w — stderr: %s", restoreName, err, stderr.String())
	}
	return nil
}

// GetRestoreStatus returns the phase of the named restore.
func (vr *VeleroRestore) GetRestoreStatus(ctx context.Context, name string) (string, error) {
	cmd := vr.veleroCmd(ctx, "restore", "get", name, "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("velero restore get %q: %w", name, err)
	}

	var obj struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}
	if parseErr := json.Unmarshal(out, &obj); parseErr != nil {
		return "", fmt.Errorf("parse velero restore JSON: %w", parseErr)
	}
	return obj.Status.Phase, nil
}

// WaitForCompletion polls until the restore finishes or the timeout elapses.
func (vr *VeleroRestore) WaitForCompletion(ctx context.Context, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		phase, err := vr.GetRestoreStatus(ctx, name)
		if err != nil {
			return err
		}
		switch phase {
		case "Completed":
			return nil
		case "Failed", "PartiallyFailed":
			return fmt.Errorf("restore %q finished with phase %q", name, phase)
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("timeout waiting for restore %q after %s", name, timeout)
}

// ListRestores returns all Velero restore objects.
func (vr *VeleroRestore) ListRestores(ctx context.Context) ([]VeleroRestoreInfo, error) {
	cmd := vr.veleroCmd(ctx, "restore", "get", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("velero restore list: %w", err)
	}

	var list struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				BackupName string `json:"backupName"`
			} `json:"spec"`
			Status struct {
				Phase               string    `json:"phase"`
				StartTimestamp      time.Time `json:"startTimestamp"`
				CompletionTimestamp time.Time `json:"completionTimestamp"`
				Warnings            int       `json:"warnings"`
				Errors              int       `json:"errors"`
			} `json:"status"`
		} `json:"items"`
	}

	if parseErr := json.Unmarshal(out, &list); parseErr != nil {
		return nil, fmt.Errorf("parse velero restore list JSON: %w", parseErr)
	}

	result := make([]VeleroRestoreInfo, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, VeleroRestoreInfo{
			Name:                item.Metadata.Name,
			BackupName:          item.Spec.BackupName,
			Status:              item.Status.Phase,
			StartTimestamp:      item.Status.StartTimestamp,
			CompletionTimestamp: item.Status.CompletionTimestamp,
			Warnings:            item.Status.Warnings,
			Errors:              item.Status.Errors,
		})
	}
	return result, nil
}

// RestoreLatestBackup finds the most recent completed backup for the namespace
// and initiates a restore from it.
func (vr *VeleroRestore) RestoreLatestBackup(ctx context.Context, namespace string) error {
	// Re-use the VeleroBackup helper to list backups.
	lister := &VeleroBackup{Namespace: vr.Namespace, KubeconfigPath: vr.KubeconfigPath}
	backups, err := lister.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("list backups for namespace %q: %w", namespace, err)
	}

	// Filter to completed backups only.
	var completed []VeleroBackupInfo
	for _, b := range backups {
		if b.Status == "Completed" {
			completed = append(completed, b)
		}
	}
	if len(completed) == 0 {
		return fmt.Errorf("no completed backups found for namespace %q", namespace)
	}

	// Sort by completion timestamp descending to pick newest.
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].CompletionTimestamp.After(completed[j].CompletionTimestamp)
	})

	latest := completed[0]
	restoreName := fmt.Sprintf("restore-%s-%d", latest.Name, time.Now().Unix())
	return vr.CreateRestore(ctx, restoreName, latest.Name, namespace)
}
