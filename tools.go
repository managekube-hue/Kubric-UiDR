//go:build tools

// Package tools pins Go module dependencies that are used by non-main packages
// (backup scripts, billing, bridges, vendor tools) so `go mod tidy` does not
// remove them.  Every library in the Master Library Reference (playbook L0-L5)
// that has a Go import path is pinned here.
package tools

import (
	// L0 — Foundation
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/spf13/cobra"

	// L1 — Go API Services
	_ "github.com/go-chi/cors"
	_ "github.com/hashicorp/vault/api"
	_ "github.com/zeebo/blake3"

	// L2 — KAI / Identity
	_ "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	// L3 — Detection / Compliance
	_ "github.com/ossf/scorecard/v4/checks"
	_ "github.com/sigstore/sigstore/pkg/signature"

	// L4 — Production Infrastructure
	_ "github.com/minio/minio-go/v7"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/theupdateframework/go-tuf/v2/metadata"

	// L5 — Billing / PSA
	_ "github.com/cbergoon/merkletree"
	_ "github.com/stripe/stripe-go/v76"

	// Workflows
	_ "go.temporal.io/sdk/workflow"
)
