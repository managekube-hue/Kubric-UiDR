// ─────────────────────────────────────────────────────────────────────────────
// Kubric-UiDR — Reference go.mod (spec target for dependency convergence)
// This file documents the FULL intended dependency set.  The canonical go.mod
// lives at the repo root; this is the reference spec from Orchestration §38.
// ─────────────────────────────────────────────────────────────────────────────

module github.com/managekube-hue/Kubric-UiDR

go 1.25.5

require (
	// ── HTTP / API ────────────────────────────────────────────────────────────
	github.com/go-chi/chi/v5                v5.1.0
	github.com/go-chi/cors                  v1.2.1
	github.com/go-chi/httprate              v0.9.0
	github.com/go-chi/render                v1.0.3

	// ── gRPC / Protobuf ──────────────────────────────────────────────────────
	google.golang.org/grpc                  v1.65.0
	google.golang.org/protobuf              v1.34.2
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0
	github.com/grpc-ecosystem/grpc-gateway/v2      v2.20.0

	// ── PostgreSQL ───────────────────────────────────────────────────────────
	github.com/jackc/pgx/v5                 v5.6.0
	github.com/jackc/pgconn                 v1.14.3
	github.com/jackc/pgtype                 v1.14.3

	// ── ClickHouse ───────────────────────────────────────────────────────────
	github.com/ClickHouse/clickhouse-go/v2  v2.26.0

	// ── NATS ─────────────────────────────────────────────────────────────────
	github.com/nats-io/nats.go              v1.36.0
	github.com/nats-io/nkeys               v0.4.7

	// ── ZeroMQ ───────────────────────────────────────────────────────────────
	github.com/go-zeromq/zmq4              v0.17.0

	// ── Neo4j ────────────────────────────────────────────────────────────────
	github.com/neo4j/neo4j-go-driver/v5    v5.22.0

	// ── DuckDB ───────────────────────────────────────────────────────────────
	github.com/marcboeker/go-duckdb        v1.7.0

	// ── Object Storage ───────────────────────────────────────────────────────
	github.com/minio/minio-go/v7           v7.0.74

	// ── Security / Supply-Chain ──────────────────────────────────────────────
	github.com/ossf/scorecard/v4           v4.13.1
	github.com/sigstore/cosign/v2          v2.3.0
	github.com/sigstore/sigstore           v1.8.4
	github.com/theupdateframework/go-tuf/v2 v2.0.0

	// ── Crypto ───────────────────────────────────────────────────────────────
	github.com/zeebo/blake3                v0.2.3
	github.com/cbergoon/merkletree         v0.2.0
	github.com/golang-jwt/jwt/v5           v5.2.1

	// ── CLI ──────────────────────────────────────────────────────────────────
	github.com/spf13/cobra                 v1.8.1
	github.com/spf13/viper                 v1.19.0

	// ── Billing ──────────────────────────────────────────────────────────────
	github.com/stripe/stripe-go/v76        v76.25.0

	// ── Workflow / Temporal ──────────────────────────────────────────────────
	go.temporal.io/sdk                     v1.27.0
	go.temporal.io/api                     v1.33.0

	// ── Secrets ──────────────────────────────────────────────────────────────
	github.com/hashicorp/vault/api         v1.14.0
	github.com/hashicorp/vault/sdk         v0.13.0

	// ── Notifications ────────────────────────────────────────────────────────
	github.com/twilio/twilio-go            v1.22.2

	// ── Observability ────────────────────────────────────────────────────────
	github.com/prometheus/client_golang    v1.19.1
	github.com/prometheus/common           v0.55.0
	go.opentelemetry.io/otel              v1.28.0
	go.opentelemetry.io/otel/sdk          v1.28.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/exporters/prometheus                     v0.50.0

	// ── Database Migrations ──────────────────────────────────────────────────
	github.com/golang-migrate/migrate/v4   v4.17.1

	// ── Compression ──────────────────────────────────────────────────────────
	github.com/klauspost/compress          v1.17.9

	// ── Utilities ────────────────────────────────────────────────────────────
	github.com/google/uuid                 v1.6.0
	gopkg.in/yaml.v3                       v3.0.1
	go.uber.org/zap                        v1.27.0
	go.uber.org/fx                         v1.22.1
	golang.org/x/sync                      v0.7.0
	golang.org/x/crypto                    v0.25.0

	// ── Testing ──────────────────────────────────────────────────────────────
	github.com/stretchr/testify            v1.9.0
	github.com/testcontainers/testcontainers-go v0.31.0
)
