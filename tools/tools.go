//go:build tools

// Package tools pins build-time and integration dependencies so that
// `go mod tidy` does not prune them from go.mod.
//
// These packages are imported at runtime by integration code paths:
//   - go-duckdb   → internal/analytics/ (CGO — requires C compiler + DuckDB libs)
//   - zmq4        → internal/messaging/ (pure Go ZMQ4)
//   - twilio-go   → internal/alerting/  (pure Go REST client)
//
// To build the tools package explicitly:
//   go build -tags tools ./tools/
package tools

import (
	_ "github.com/marcboeker/go-duckdb"
	_ "github.com/go-zeromq/zmq4"
	_ "github.com/twilio/twilio-go"
)
