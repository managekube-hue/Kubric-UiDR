package psa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ZammadBridge synchronises service_tickets to/from the Zammad ITSM REST API.
type ZammadBridge struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	pgPool     *pgxpool.Pool
}

// NewZammadBridge creates a ZammadBridge from environment variables.
// Required env: ZAMMAD_URL, ZAMMAD_TOKEN, DATABASE_URL.
func NewZammadBridge() (*ZammadBridge, error) {
	baseURL := os.Getenv("ZAMMAD_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("ZAMMAD_URL not set")
	}
	token := os.Getenv("ZAMMAD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("ZAMMAD_TOKEN not set")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool: %w", err)
	}
	return &ZammadBridge{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		pgPool:     pool,
	}, nil
}

// doRequest is a helper that sends a JSON request to the Zammad API and returns the raw response bytes.
// Authentication uses the Zammad API token header: Authorization: Token token={token}.
func (z *ZammadBridge) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, z.BaseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Token token="+z.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := z.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("zammad %s %s status=%d body=%s", method, path, resp.StatusCode, string(raw))
	}
	return raw, nil
}

// CreateTicket creates a ticket in Zammad and returns the Zammad ticket ID string.
func (z *ZammadBridge) CreateTicket(ctx context.Context, t *Ticket) (string, error) {
	zPriority := "3 normal"
	switch t.Priority {
	case P1:
		zPriority = "1 low" // adjust to match your Zammad priority names
	case P2:
		zPriority = "2 normal"
	}

	payload := map[string]any{
		"title":       t.Title,
		"group":       "Users",
		"priority":    zPriority,
		"state":       "new",
		"customer_id": "guess:" + t.TenantID + "@kubric.io",
		"article": map[string]any{
			"subject":  t.Title,
			"body":     t.Description,
			"type":     "note",
			"internal": false,
		},
	}

	raw, err := z.doRequest(ctx, http.MethodPost, "/api/v1/tickets", payload)
	if err != nil {
		return "", fmt.Errorf("create zammad ticket: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}
	idFloat, ok := result["id"].(float64)
	if !ok {
		return "", fmt.Errorf("zammad response missing id field")
	}
	return fmt.Sprintf("%.0f", idFloat), nil
}

// UpdateTicket updates the state of an existing Zammad ticket by its Zammad ID.
func (z *ZammadBridge) UpdateTicket(ctx context.Context, zammadID, state string) error {
	payload := map[string]any{"state": state}
	_, err := z.doRequest(ctx, http.MethodPut, "/api/v1/tickets/"+zammadID, payload)
	return err
}

// GetTicket fetches a single Zammad ticket by its Zammad ID and returns the raw map.
func (z *ZammadBridge) GetTicket(ctx context.Context, zammadID string) (map[string]any, error) {
	raw, err := z.doRequest(ctx, http.MethodGet, "/api/v1/tickets/"+zammadID, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode ticket response: %w", err)
	}
	return result, nil
}

// SyncTicket loads a ticket from PostgreSQL, creates or updates it in Zammad,
// then writes the zammad_ticket_id back to the database.
func (z *ZammadBridge) SyncTicket(ctx context.Context, ticketID string) error {
	row := z.pgPool.QueryRow(ctx, `
		SELECT id, tenant_id, title, description, priority, state,
		       COALESCE(assigned_to::text,''), created_at, updated_at,
		       resolved_at, sla_breach_at, COALESCE(external_ticket_id,'')
		FROM service_tickets WHERE id = $1`, ticketID)

	var t Ticket
	if err := row.Scan(
		&t.ID, &t.TenantID, &t.Title, &t.Description,
		&t.Priority, &t.State, &t.AssignedTo,
		&t.CreatedAt, &t.UpdatedAt, &t.ResolvedAt,
		&t.SLABreachAt, &t.ExternalTicketID,
	); err != nil {
		return fmt.Errorf("load ticket %s: %w", ticketID, err)
	}

	// Check for an existing Zammad binding.
	var existingZammadID string
	_ = z.pgPool.QueryRow(ctx,
		`SELECT zammad_ticket_id FROM zammad_ticket_sync WHERE ticket_id = $1`, ticketID,
	).Scan(&existingZammadID)

	if existingZammadID == "" {
		// First sync — create in Zammad.
		zID, err := z.CreateTicket(ctx, &t)
		if err != nil {
			return fmt.Errorf("create in zammad: %w", err)
		}
		_, err = z.pgPool.Exec(ctx, `
			INSERT INTO zammad_ticket_sync (ticket_id, zammad_ticket_id, synced_at, sync_status)
			VALUES ($1,$2,NOW(),'synced')
			ON CONFLICT (ticket_id) DO UPDATE SET
				zammad_ticket_id = EXCLUDED.zammad_ticket_id,
				synced_at        = NOW(),
				sync_status      = 'synced'`,
			ticketID, zID)
		if err != nil {
			return fmt.Errorf("save zammad_ticket_id: %w", err)
		}
		_, _ = z.pgPool.Exec(ctx,
			`UPDATE service_tickets SET external_ticket_id=$1, external_system='zammad' WHERE id=$2`,
			zID, ticketID)
		log.Printf("zammad_bridge: created ticket=%s zammad_id=%s", ticketID, zID)
	} else {
		// Subsequent sync — push current state.
		if err := z.UpdateTicket(ctx, existingZammadID, string(t.State)); err != nil {
			return fmt.Errorf("update in zammad: %w", err)
		}
		_, _ = z.pgPool.Exec(ctx,
			`UPDATE zammad_ticket_sync SET synced_at=NOW(), sync_status='synced' WHERE ticket_id=$1`,
			ticketID)
		log.Printf("zammad_bridge: updated ticket=%s zammad_id=%s state=%s", ticketID, existingZammadID, t.State)
	}
	return nil
}

// PollAndSync polls for unsynced or stale tickets and syncs them to Zammad on the given interval.
func (z *ZammadBridge) PollAndSync(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = 2 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("zammad_bridge: starting poll loop interval=%v", interval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("zammad_bridge: shutting down")
			return
		case <-ticker.C:
			rows, err := z.pgPool.Query(ctx, `
				SELECT t.id FROM service_tickets t
				LEFT JOIN zammad_ticket_sync s ON s.ticket_id = t.id
				WHERE s.ticket_id IS NULL
				   OR s.sync_status = 'pending'
				   OR t.updated_at > s.synced_at
				LIMIT 50`)
			if err != nil {
				log.Printf("zammad_bridge: poll query error: %v", err)
				continue
			}

			var ids []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil {
					ids = append(ids, id)
				}
			}
			rows.Close()

			for _, id := range ids {
				if err := z.SyncTicket(ctx, id); err != nil {
					// Mark as error so it retries next cycle.
					_, _ = z.pgPool.Exec(ctx,
						`UPDATE zammad_ticket_sync SET sync_status='error' WHERE ticket_id=$1`, id)
					log.Printf("zammad_bridge: sync ticket=%s error: %v", id, err)
				}
			}
		}
	}
}
