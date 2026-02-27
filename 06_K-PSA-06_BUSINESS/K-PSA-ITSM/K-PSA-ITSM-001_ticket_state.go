package psa

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TicketState is an ITSM ticket lifecycle state.
type TicketState string

const (
	StateOpen       TicketState = "open"
	StateInProgress TicketState = "in_progress"
	StatePending    TicketState = "pending"
	StateResolved   TicketState = "resolved"
	StateClosed     TicketState = "closed"
	StateCancelled  TicketState = "cancelled"
)

// TicketPriority is an ITSM ticket priority level.
type TicketPriority string

const (
	P1 TicketPriority = "P1"
	P2 TicketPriority = "P2"
	P3 TicketPriority = "P3"
	P4 TicketPriority = "P4"
)

// Ticket is an ITSM service ticket.
type Ticket struct {
	ID               string
	TenantID         string
	Title            string
	Description      string
	Priority         TicketPriority
	State            TicketState
	AssignedTo       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ResolvedAt       *time.Time
	SLABreachAt      *time.Time // nil means no SLA deadline set
	ExternalTicketID string
}

// allowedTransitions maps each TicketState to the set of valid next states.
var allowedTransitions = map[TicketState][]TicketState{
	StateOpen:       {StateInProgress, StateResolved, StateCancelled},
	StateInProgress: {StatePending, StateResolved, StateCancelled},
	StatePending:    {StateInProgress, StateResolved, StateCancelled},
	StateResolved:   {StateClosed, StateOpen},
	StateClosed:     {},
	StateCancelled:  {},
}

// TicketStateMachine validates and applies ticket state transitions.
type TicketStateMachine struct{}

// NewTicketStateMachine creates a TicketStateMachine.
func NewTicketStateMachine() *TicketStateMachine {
	return &TicketStateMachine{}
}

// IsValidTransition returns true if transitioning from → to is allowed.
func (m *TicketStateMachine) IsValidTransition(from, to TicketState) bool {
	nexts, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range nexts {
		if s == to {
			return true
		}
	}
	return false
}

// ValidNextStates returns the allowed transition targets from the current state.
func (m *TicketStateMachine) ValidNextStates(current TicketState) []TicketState {
	return allowedTransitions[current]
}

// Transition applies a state change to the ticket, updating timestamps.
// Returns an error if the transition is not allowed by the state machine.
func (m *TicketStateMachine) Transition(ticket *Ticket, newState TicketState, actorID string) error {
	if !m.IsValidTransition(ticket.State, newState) {
		return fmt.Errorf("invalid transition %s → %s for ticket %s", ticket.State, newState, ticket.ID)
	}
	ticket.State = newState
	ticket.UpdatedAt = time.Now().UTC()
	if newState == StateResolved {
		now := time.Now().UTC()
		ticket.ResolvedAt = &now
	}
	return nil
}

// ComputeSLABreach returns the expected SLA breach time based on priority.
// P1: 4 h | P2: 8 h | P3: 24 h | P4: 72 h
func ComputeSLABreach(priority TicketPriority, createdAt time.Time) time.Time {
	switch priority {
	case P1:
		return createdAt.Add(4 * time.Hour)
	case P2:
		return createdAt.Add(8 * time.Hour)
	case P3:
		return createdAt.Add(24 * time.Hour)
	default: // P4
		return createdAt.Add(72 * time.Hour)
	}
}

// IsBreached returns true if the ticket has passed its SLA deadline without resolution.
func IsBreached(ticket *Ticket) bool {
	if ticket.State == StateResolved || ticket.State == StateClosed || ticket.State == StateCancelled {
		return false
	}
	if ticket.SLABreachAt == nil {
		return false
	}
	return time.Now().UTC().After(*ticket.SLABreachAt)
}

// SaveState upserts the ticket to the service_tickets table via pgxpool.
func SaveState(ctx context.Context, ticket *Ticket, dbPool *pgxpool.Pool) error {
	_, err := dbPool.Exec(ctx, `
		INSERT INTO service_tickets
			(id, tenant_id, title, description, priority, state, assigned_to,
			 created_at, updated_at, resolved_at, sla_breach_at, external_ticket_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			state              = EXCLUDED.state,
			assigned_to        = EXCLUDED.assigned_to,
			updated_at         = EXCLUDED.updated_at,
			resolved_at        = EXCLUDED.resolved_at,
			sla_breach_at      = EXCLUDED.sla_breach_at,
			external_ticket_id = EXCLUDED.external_ticket_id`,
		ticket.ID, ticket.TenantID, ticket.Title, ticket.Description,
		string(ticket.Priority), string(ticket.State), ticket.AssignedTo,
		ticket.CreatedAt, ticket.UpdatedAt, ticket.ResolvedAt,
		ticket.SLABreachAt, ticket.ExternalTicketID)
	return err
}
