package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// UsageEvent represents a single billing usage event from NATS.
type UsageEvent struct {
	TenantID  string    `json:"tenant_id"`
	EventType string    `json:"event_type"`
	Quantity  int64     `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

// MeterConfig configures the usage metering subscriber.
type MeterConfig struct {
	DB   *pgxpool.Pool
	NATS *nats.Conn
}

// StartMeteringSubscriber subscribes to NATS usage events and records them.
// Subject pattern: kubric.*.billing.usage.>
func StartMeteringSubscriber(ctx context.Context, cfg MeterConfig) error {
	_, err := cfg.NATS.Subscribe("kubric.*.billing.usage.>", func(msg *nats.Msg) {
		var evt UsageEvent
		if err := json.Unmarshal(msg.Data, &evt); err != nil {
			slog.Warn("billing: bad usage event", "error", err)
			return
		}
		if err := insertUsageEvent(ctx, cfg.DB, &evt); err != nil {
			slog.Error("billing: insert usage event", "error", err, "tenant", evt.TenantID)
		}
	})
	if err != nil {
		return fmt.Errorf("billing subscribe: %w", err)
	}
	slog.Info("billing: metering subscriber started")
	return nil
}

func insertUsageEvent(ctx context.Context, db *pgxpool.Pool, evt *UsageEvent) error {
	_, err := db.Exec(ctx,
		`INSERT INTO billing_usage (tenant_id, event_type, quantity, timestamp)
		 VALUES ($1, $2, $3, $4)`,
		evt.TenantID, evt.EventType, evt.Quantity, evt.Timestamp,
	)
	return err
}

// AggregateUsage returns total usage for a tenant in the given period.
func AggregateUsage(ctx context.Context, db *pgxpool.Pool, tenantID string, periodStart, periodEnd time.Time) (int64, error) {
	var total int64
	err := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0)
		 FROM billing_usage
		 WHERE tenant_id = $1 AND timestamp >= $2 AND timestamp < $3`,
		tenantID, periodStart, periodEnd,
	).Scan(&total)
	return total, err
}

// MerkleRoot computes a deterministic Merkle root from billing usage records.
// Uses SHA-256 over sorted (tenant_id || event_type || quantity || timestamp) entries.
func MerkleRoot(events []UsageEvent) string {
	if len(events) == 0 {
		return ""
	}

	// Sort for determinism
	sort.Slice(events, func(i, j int) bool {
		if events[i].TenantID != events[j].TenantID {
			return events[i].TenantID < events[j].TenantID
		}
		if events[i].Timestamp != events[j].Timestamp {
			return events[i].Timestamp.Before(events[j].Timestamp)
		}
		return events[i].EventType < events[j].EventType
	})

	// Build leaf hashes
	leaves := make([][]byte, len(events))
	for i, e := range events {
		data := fmt.Sprintf("%s|%s|%d|%s", e.TenantID, e.EventType, e.Quantity, e.Timestamp.UTC().Format(time.RFC3339Nano))
		h := sha256.Sum256([]byte(data))
		leaves[i] = h[:]
	}

	// Build Merkle tree bottom-up
	for len(leaves) > 1 {
		var next [][]byte
		for i := 0; i < len(leaves); i += 2 {
			if i+1 < len(leaves) {
				combined := append(leaves[i], leaves[i+1]...)
				h := sha256.Sum256(combined)
				next = append(next, h[:])
			} else {
				next = append(next, leaves[i]) // odd leaf promoted
			}
		}
		leaves = next
	}

	return hex.EncodeToString(leaves[0])
}

// StoreMerkleRoot saves the merkle root for a billing period.
func StoreMerkleRoot(ctx context.Context, db *pgxpool.Pool, tenantID string, period string, root string) error {
	_, err := db.Exec(ctx,
		`INSERT INTO billing_ledger (tenant_id, period, merkle_root, created_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (tenant_id, period) DO UPDATE SET merkle_root = $3`,
		tenantID, period, root,
	)
	return err
}
