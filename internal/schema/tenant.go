// Package schema enforces tenant_id as a first-class concern across all
// Kubric Go services. tenant_id must appear in:
//   - Every NATS subject   → kubric.{tenant_id}.{category}.{class}.v1
//   - Every ClickHouse row → partitioned by tenant_id
//   - Every API route      → extracted from JWT claim or path param
//   - Every log line       → via slog/tracing context
package schema

import (
	"errors"
	"fmt"
	"regexp"
)

// tenantIDPattern enforces lowercase alphanumeric + hyphens, max 63 chars.
// Matches Kubernetes namespace naming conventions.
var tenantIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,61}[a-z0-9]$`)

// ErrEmptyTenantID is returned when a required tenant_id is missing.
var ErrEmptyTenantID = errors.New("tenant_id must not be empty")

// ErrInvalidTenantID is returned when tenant_id does not match the pattern.
var ErrInvalidTenantID = errors.New("tenant_id must match ^[a-z0-9][a-z0-9\\-]{0,61}[a-z0-9]$")

// ValidateTenantID returns nil if id is a valid tenant identifier.
func ValidateTenantID(id string) error {
	if id == "" {
		return ErrEmptyTenantID
	}
	if !tenantIDPattern.MatchString(id) {
		return fmt.Errorf("%w: got %q", ErrInvalidTenantID, id)
	}
	return nil
}

// NATSSubject builds the canonical NATS subject for a given tenant, category,
// and event class.
//
//	kubric.{tenant_id}.{category}.{eventClass}.v1
//
// Examples:
//
//	kubric.acme-corp.endpoint.process.v1
//	kubric.acme-corp.network.conn.v1
//	kubric.acme-corp.identity.auth.v1
//	kubric.acme-corp.vuln.finding.v1
func NATSSubject(tenantID, category, eventClass string) string {
	return fmt.Sprintf("kubric.%s.%s.%s.v1", tenantID, category, eventClass)
}

// NATSSubscribePattern returns a NATS wildcard subject for subscribing to all
// events for a given tenant.
//
//	kubric.{tenant_id}.>
func NATSSubscribePattern(tenantID string) string {
	return fmt.Sprintf("kubric.%s.>", tenantID)
}

// ClickHousePartitionKey returns the ClickHouse ORDER BY / PARTITION BY prefix
// that must appear in every event table.
//
// All tables must be created with:
//
//	ORDER BY (tenant_id, toStartOfHour(timestamp), event_id)
//	PARTITION BY (tenant_id, toYYYYMM(timestamp))
//
// This enables:
//   - Per-tenant data isolation (DROP PARTITION for GDPR deletion)
//   - Per-tenant retention policies via TTL expressions
//   - Efficient per-tenant queries without full-table scans
func ClickHousePartitionKey() string {
	return "tenant_id, toYYYYMM(timestamp)"
}
