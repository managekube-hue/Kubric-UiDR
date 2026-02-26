//go:build tools

// Package tools pins Go module dependencies that are used by non-main packages
// (backup scripts, code generators) so `go mod tidy` does not remove them.
package tools

import (
	_ "github.com/hashicorp/vault/api"
	_ "github.com/minio/minio-go/v7"
)
