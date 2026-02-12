#!/bin/bash
set -e

echo "🗄️  Initializing Kubric databases..."

POSTGRES_POD=$(kubectl get pod -n kubric -l app=postgres -o jsonpath='{.items[0].metadata.name}')

# Wait for PostgreSQL to be ready
echo "✓ Waiting for PostgreSQL..."
kubectl exec -n kubric "$POSTGRES_POD" -- pg_isready -U postgres || sleep 5

# Initialize UAR schema
echo "✓ Creating UAR schema..."
kubectl exec -n kubric "$POSTGRES_POD" -- psql -U postgres -d kubric < config/postgres/init/01_uar_schema.sql

# Apply RLS policies
echo "✓ Applying RLS policies..."
kubectl exec -n kubric "$POSTGRES_POD" -- psql -U postgres -d kubric < config/postgres/init/02_rls_policies.sql

# Initialize ClickHouse
CLICKHOUSE_POD=$(kubectl get pod -n kubric -l app=clickhouse -o jsonpath='{.items[0].metadata.name}')

echo "✓ Creating ClickHouse databases..."
kubectl exec -n kubric "$CLICKHOUSE_POD" -- clickhouse-client <<EOF
CREATE DATABASE IF NOT EXISTS kubric;
CREATE DATABASE IF NOT EXISTS kubric_siem;

CREATE TABLE IF NOT EXISTS kubric.events (
  timestamp DateTime,
  source String,
  event_type String,
  payload String
) ENGINE = MergeTree()
ORDER BY (timestamp, source);
EOF

echo "✅ Database initialization complete!"
