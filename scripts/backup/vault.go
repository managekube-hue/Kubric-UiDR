// Package backup — Vault Raft snapshot backup to MinIO.
//
// Creates a Vault Raft snapshot using the Vault API (/v1/sys/storage/raft/snapshot)
// and uploads it to MinIO under kubric-backups/vault/{timestamp}/vault.snap.
// Designed to run every 5 minutes via a Kubernetes CronJob.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/minio/minio-go/v7"
)

const (
	vaultBackupPrefix = "vault"
	vaultSnapFile     = "vault.snap"
)

// BackupVault creates a Vault Raft snapshot and uploads it to MinIO.
// Returns the MinIO object key on success.
func BackupVault(ctx context.Context) (string, error) {
	cfg := vaultapi.DefaultConfig()
	cfg.Address = getEnv("VAULT_ADDR", "http://vault:8200")
	cfg.HttpClient = &http.Client{Timeout: 30 * time.Second}

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("vault client: %w", err)
	}
	client.SetToken(mustEnv("VAULT_TOKEN"))

	// GET /v1/sys/storage/raft/snapshot — returns binary snapshot stream
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	objectKey := fmt.Sprintf("%s/%s/%s", vaultBackupPrefix, ts, vaultSnapFile)

	slog.Info("creating Vault Raft snapshot", "key", objectKey)

	resp, err := client.RawRequestWithContext(ctx,
		client.NewRequest("GET", "/v1/sys/storage/raft/snapshot"),
	)
	if err != nil {
		return "", fmt.Errorf("vault snapshot request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vault snapshot failed: status=%d body=%s", resp.StatusCode, body)
	}

	snapBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read snapshot: %w", err)
	}

	mc, err := minioClient()
	if err != nil {
		return "", fmt.Errorf("minio connect: %w", err)
	}

	_, err = mc.PutObject(ctx, backupBucket, objectKey,
		bytes.NewReader(snapBytes), int64(len(snapBytes)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", fmt.Errorf("minio put vault snapshot: %w", err)
	}

	slog.Info("Vault snapshot uploaded", "key", objectKey, "bytes", len(snapBytes))
	return objectKey, nil
}
