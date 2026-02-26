// Package backup — main entry point for the backup CLI.
//
// Usage:
//
//	go run ./scripts/backup clickhouse
//	go run ./scripts/backup vault
//	go run ./scripts/backup postgres
//	go run ./scripts/backup neo4j
//	go run ./scripts/backup all
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	backup "github.com/managekube-hue/Kubric-UiDR/scripts/backup"
)

func main() {
	target := "all"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var err error
	switch target {
	case "clickhouse":
		_, err = backup.BackupClickHouse(ctx)
	case "vault":
		_, err = backup.BackupVault(ctx)
	case "postgres":
		_, err = backup.BackupPostgres(ctx)
	case "neo4j":
		_, err = backup.BackupNeo4j(ctx)
	case "all":
		slog.Info("running all backups...")
		for _, fn := range []struct {
			name string
			fn   func(context.Context) (string, error)
		}{
			{"clickhouse", backup.BackupClickHouse},
			{"vault", backup.BackupVault},
			{"postgres", backup.BackupPostgres},
			{"neo4j", backup.BackupNeo4j},
		} {
			key, backerr := fn.fn(ctx)
			if backerr != nil {
				slog.Error("backup failed", "target", fn.name, "err", backerr)
				err = backerr
			} else {
				slog.Info("backup complete", "target", fn.name, "key", key)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown backup target %q\nUsage: backup [clickhouse|vault|postgres|neo4j|all]\n", target)
		os.Exit(1)
	}

	if err != nil {
		slog.Error("backup failed", "err", err)
		os.Exit(1)
	}
}
