// Package backup implements ClickHouse backup-to-MinIO for Kubric DR.
//
// Connects to ClickHouse via clickhouse-go/v2, triggers a ClickHouse backup
// using the BACKUP TABLE command, then uploads the result to MinIO under
// kubric-backups/clickhouse/{date}/.
//
// Usage:
//
//	go run ./scripts/backup clickhouse
package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	backupBucket   = "kubric-backups"
	chBackupPrefix = "clickhouse"
)

// BackupClickHouse creates a ClickHouse backup for partitions older than 1 day
// and uploads the result to MinIO. It returns the MinIO object key of the backup.
func BackupClickHouse(ctx context.Context) (string, error) {
	chConn, err := connectClickHouse()
	if err != nil {
		return "", fmt.Errorf("clickhouse connect: %w", err)
	}
	defer chConn.Close()

	date := time.Now().UTC().Format("2006-01-02")
	backupName := fmt.Sprintf("kubric-%s", date)
	s3Path := fmt.Sprintf("s3('%s/%s/%s/', '%s', '%s')",
		minioEndpoint(), chBackupPrefix, date,
		mustEnv("MINIO_ACCESS_KEY"),
		mustEnv("MINIO_SECRET_KEY"),
	)

	// Trigger backup — ClickHouse will write directly to MinIO via S3 function
	backupSQL := fmt.Sprintf(
		`BACKUP DATABASE kubric TO %s SETTINGS id='%s'`,
		s3Path, backupName,
	)
	slog.Info("triggering ClickHouse backup", "name", backupName)
	if err := chConn.Exec(ctx, backupSQL); err != nil {
		return "", fmt.Errorf("backup exec: %w", err)
	}

	objectKey := filepath.Join(chBackupPrefix, date, backupName)
	slog.Info("ClickHouse backup complete", "object", objectKey)

	// Verify by listing the MinIO path
	mc, err := minioClient()
	if err != nil {
		return objectKey, fmt.Errorf("minio verify connect: %w", err)
	}
	found := false
	for obj := range mc.ListObjects(ctx, backupBucket, minio.ListObjectsOptions{
		Prefix:    filepath.Join(chBackupPrefix, date) + "/",
		Recursive: true,
	}) {
		if obj.Err != nil {
			return objectKey, fmt.Errorf("list objects: %w", obj.Err)
		}
		slog.Info("backup object verified", "key", obj.Key, "size", obj.Size)
		found = true
	}
	if !found {
		return objectKey, fmt.Errorf("backup verification failed: no objects found at %s/%s", chBackupPrefix, date)
	}

	// Write audit log
	if err := writeAuditLog(ctx, chConn, "clickhouse_backup", backupName, "success"); err != nil {
		slog.Warn("audit log write failed", "err", err)
	}

	return objectKey, nil
}

// writeAuditLog inserts a backup audit entry into a kubric.backup_audit table.
func writeAuditLog(_ context.Context, conn driver.Conn, backupType, backupName, status string) error {
	return conn.Exec(context.Background(),
		`INSERT INTO kubric.backup_audit (backup_type, backup_name, status, created_at) VALUES (?, ?, ?, ?)`,
		backupType, backupName, status, time.Now().UTC(),
	)
}

// ─── helpers ────────────────────────────────────────────────────────────────

func connectClickHouse() (driver.Conn, error) {
	addr := getEnv("CLICKHOUSE_ADDR", "clickhouse:9000")
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: getEnv("CLICKHOUSE_DB", "kubric"),
			Username: getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		DialTimeout:     10 * time.Second,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 30 * time.Minute,
	})
}

func minioClient() (*minio.Client, error) {
	endpoint := minioEndpoint()
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(mustEnv("MINIO_ACCESS_KEY"), mustEnv("MINIO_SECRET_KEY"), ""),
		Secure: false,
	})
}

func minioEndpoint() string {
	return getEnv("MINIO_ENDPOINT", "minio:9000")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required env var %s is not set", key))
	}
	return v
}
