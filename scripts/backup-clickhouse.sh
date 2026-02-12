#!/bin/bash
set -e

BACKUP_DIR="${BACKUP_DIR:-.backups}"
RETENTION_DAYS="${RETENTION_DAYS:-90}"

echo "🔄 Starting ClickHouse backup..."

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/clickhouse_backup_$TIMESTAMP.tar.gz"

CLICKHOUSE_POD=$(kubectl get pod -n kubric -l app=clickhouse -o jsonpath='{.items[0].metadata.name}')

# Create backup
echo "Backing up to $BACKUP_FILE..."
kubectl exec -n kubric "$CLICKHOUSE_POD" -- bash -c \
  "tar -czf /tmp/backup_$TIMESTAMP.tar.gz /var/lib/clickhouse/data"

# Copy to local
kubectl cp "kubric/$CLICKHOUSE_POD:/tmp/backup_$TIMESTAMP.tar.gz" "$BACKUP_FILE"

echo "✅ Backup complete: $BACKUP_FILE"

# Clean old backups
echo "Cleaning backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "clickhouse_backup_*.tar.gz" -mtime +"$RETENTION_DAYS" -delete

echo "✅ Cleanup complete!"
