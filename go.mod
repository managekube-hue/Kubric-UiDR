module github.com/managekube-hue/Kubric-UiDR

go 1.21

require (
	// Existing cmd/ tools
	github.com/ClickHouse/clickhouse-go/v2 v2.28.0
	github.com/nats-io/nats.go            v1.37.0
	gopkg.in/yaml.v3                       v3.0.1

	// Protobuf — wire format for OCSF events between agents and services
	google.golang.org/protobuf v1.35.0

	// Database
	github.com/jackc/pgx/v5 v5.6.0

	// API routing — used by K-SVC, VDR, KIC, NOC (Layer 1)
	github.com/go-chi/chi/v5 v5.1.0
)