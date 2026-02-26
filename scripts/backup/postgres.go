// Package backup — PostgreSQL pg_dump backup to MinIO.
//
// Uses pgx/v5 to verify connectivity, then shells out to pg_dump
// to produce a custom-format dump. Uploads the dump to MinIO under
// kubric-backups/postgres/{date}/kubric.dump.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
)

const (
	postgresBackupPrefix = "postgres"
	postgresDumpFile     = "kubric.dump"
)

// BackupPostgres runs pg_dump against the Kubric PostgreSQL database and
// uploads the result to MinIO. Returns the MinIO object key.
func BackupPostgres(ctx context.Context) (string, error) {
	dbURL := getEnv("KUBRIC_DATABASE_URL", "postgresql://postgres:dev_password@postgres:5432/kubric")

	// Ping via pgx to confirm connectivity before shelling out
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return "", fmt.Errorf("postgres ping: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return "", fmt.Errorf("postgres ping: %w", err)
	}
	conn.Close(ctx)

	date := time.Now().UTC().Format("2006-01-02")
	objectKey := fmt.Sprintf("%s/%s/%s", postgresBackupPrefix, date, postgresDumpFile)

	slog.Info("running pg_dump", "key", objectKey)

	// Build pg_dump command (custom format, no passwords in args — use PGPASSWORD env)
	cmd := exec.CommandContext(ctx,
		"pg_dump",
		"--format=custom",
		"--no-password",
		"--dbname="+dbURL,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w\nstderr: %s", err, stderr.String())
	}

	dumpBytes := stdout.Bytes()
	slog.Info("pg_dump complete", "bytes", len(dumpBytes))

	mc, err := minioClient()
	if err != nil {
		return "", fmt.Errorf("minio connect: %w", err)
	}

	_, err = mc.PutObject(ctx, backupBucket, objectKey,
		bytes.NewReader(dumpBytes), int64(len(dumpBytes)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", fmt.Errorf("minio put postgres dump: %w", err)
	}

	slog.Info("PostgreSQL backup uploaded", "key", objectKey)
	return objectKey, nil
}
