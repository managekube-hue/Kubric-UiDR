// Package backup — Neo4j database dump to MinIO.
//
// Shells out to neo4j-admin database dump and uploads the result to MinIO
// under kubric-backups/neo4j/{date}/kubric.neo4j.
// The neo4j-admin binary must be available in PATH inside the container.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
)

const (
	neo4jBackupPrefix = "neo4j"
	neo4jDumpFile     = "kubric.neo4j"
)

// BackupNeo4j runs neo4j-admin database dump and uploads the result to MinIO.
// Returns the MinIO object key.
func BackupNeo4j(ctx context.Context) (string, error) {
	date := time.Now().UTC().Format("2006-01-02")
	objectKey := fmt.Sprintf("%s/%s/%s", neo4jBackupPrefix, date, neo4jDumpFile)

	// Write dump to a temp file (neo4j-admin writes to a file path)
	tmpDir, err := os.MkdirTemp("", "kubric-neo4j-backup-")
	if err != nil {
		return "", fmt.Errorf("tempdir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dumpPath := filepath.Join(tmpDir, neo4jDumpFile)

	slog.Info("running neo4j-admin database dump", "dest", dumpPath)

	cmd := exec.CommandContext(ctx,
		"neo4j-admin",
		"database", "dump",
		"--to-path="+tmpDir,
		"neo4j", // default database name
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("neo4j-admin dump failed: %w\nstderr: %s", err, stderr.String())
	}

	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return "", fmt.Errorf("open dump file: %w", err)
	}
	defer dumpFile.Close()

	fi, err := dumpFile.Stat()
	if err != nil {
		return "", fmt.Errorf("stat dump file: %w", err)
	}

	mc, err := minioClient()
	if err != nil {
		return "", fmt.Errorf("minio connect: %w", err)
	}

	_, err = mc.PutObject(ctx, backupBucket, objectKey,
		dumpFile, fi.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", fmt.Errorf("minio put neo4j dump: %w", err)
	}

	slog.Info("Neo4j backup uploaded", "key", objectKey, "bytes", fi.Size())
	return objectKey, nil
}
