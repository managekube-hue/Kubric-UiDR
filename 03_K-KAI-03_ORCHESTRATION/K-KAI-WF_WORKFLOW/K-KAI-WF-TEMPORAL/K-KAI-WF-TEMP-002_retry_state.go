package temporal

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.temporal.io/sdk/temporal"
)

// ---------------------------------------------------------------------------
// RetryState — per-workflow retry tracking record
// ---------------------------------------------------------------------------

// RetryState tracks the retry lifecycle of a single Temporal workflow run.
type RetryState struct {
	WorkflowID     string
	RunID          string
	TenantID       string
	Attempt        int
	MaxAttempts    int
	LastError      string
	BackoffSeconds int
	CreatedAt      time.Time
	LastAttemptAt  time.Time
	mu             sync.Mutex
}

// ---------------------------------------------------------------------------
// RetryStateStore — in-memory thread-safe store
// ---------------------------------------------------------------------------

// RetryStateStore is an in-memory, mutex-protected map of WorkflowID ->
// RetryState.  It is intentionally not persisted; Temporal's own history
// is the source-of-truth for durable state.  This store is used for fast,
// in-process query and aggregation (e.g. alerting, dashboards).
type RetryStateStore struct {
	mu     sync.RWMutex
	states map[string]*RetryState
}

// NewRetryStateStore allocates and returns an empty RetryStateStore.
func NewRetryStateStore() *RetryStateStore {
	return &RetryStateStore{
		states: make(map[string]*RetryState),
	}
}

// Store inserts or fully replaces the RetryState for a given WorkflowID.
// A defensive copy is taken to prevent the caller from mutating stored state.
func (s *RetryStateStore) Store(state RetryState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := state
	copy.mu = sync.Mutex{} // do not copy the caller's mutex
	s.states[state.WorkflowID] = &copy
}

// Get returns the RetryState associated with workflowID and a boolean
// indicating whether the entry was found.
func (s *RetryStateStore) Get(workflowID string) (RetryState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.states[workflowID]
	if !ok {
		return RetryState{}, false
	}
	// Return a value copy so callers cannot mutate stored state directly.
	st.mu.Lock()
	defer st.mu.Unlock()
	return *st, true
}

// IncrementAttempt increments the attempt counter for workflowID and records
// the error message.  It returns true when there are remaining attempts, and
// false when MaxAttempts has been reached (caller should stop retrying).
// If workflowID is not found the function returns false.
func (s *RetryStateStore) IncrementAttempt(workflowID string, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.states[workflowID]
	if !ok {
		return false
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	st.Attempt++
	st.LastAttemptAt = time.Now().UTC()
	if err != nil {
		st.LastError = err.Error()
	}

	backoff := ExponentialBackoff(st.Attempt, 2*time.Second, 5*time.Minute)
	st.BackoffSeconds = int(backoff.Seconds())

	return st.Attempt < st.MaxAttempts
}

// GetAllFailed returns a snapshot slice of every RetryState whose attempt
// count has reached (or exceeded) MaxAttempts.
func (s *RetryStateStore) GetAllFailed() []RetryState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var failed []RetryState
	for _, st := range s.states {
		st.mu.Lock()
		if st.Attempt >= st.MaxAttempts {
			failed = append(failed, *st)
		}
		st.mu.Unlock()
	}
	return failed
}

// Cleanup removes entries whose LastAttemptAt timestamp is older than
// olderThan duration.  Returns the number of entries removed.
func (s *RetryStateStore) Cleanup(olderThan time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-olderThan)
	removed := 0
	for id, st := range s.states {
		st.mu.Lock()
		tooOld := st.LastAttemptAt.Before(cutoff)
		st.mu.Unlock()
		if tooOld {
			delete(s.states, id)
			removed++
		}
	}
	return removed
}

// ---------------------------------------------------------------------------
// RetryPolicy builder
// ---------------------------------------------------------------------------

// BuildRetryPolicy constructs a temporal.RetryPolicy with the provided
// parameters.  The backoff coefficient is fixed at 2.0 (exponential doubling).
func BuildRetryPolicy(maxAttempts int, initialInterval time.Duration, maxInterval time.Duration) temporal.RetryPolicy {
	return temporal.RetryPolicy{
		InitialInterval:    initialInterval,
		BackoffCoefficient: 2.0,
		MaximumInterval:    maxInterval,
		MaximumAttempts:    int32(maxAttempts),
	}
}

// ---------------------------------------------------------------------------
// ExponentialBackoff
// ---------------------------------------------------------------------------

// ExponentialBackoff returns min(base * 2^attempt, maxBackoff).
// attempt is zero-indexed: attempt=0 returns base, attempt=1 returns 2*base, etc.
func ExponentialBackoff(attempt int, base time.Duration, maxBackoff time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	multiplier := math.Pow(2, float64(attempt))
	result := time.Duration(float64(base) * multiplier)
	if result > maxBackoff {
		return maxBackoff
	}
	return result
}

// ---------------------------------------------------------------------------
// IsNonRetryableError
// ---------------------------------------------------------------------------

// IsNonRetryableError returns true when err represents a permanent failure
// that must not be retried.  It recognises two categories:
//
//  1. Temporal ApplicationError with IsNonRetryable() == true.
//  2. Any error whose message contains one of the known permanent-failure
//     sentinel strings defined in permanentErrorTypes.
func IsNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check Temporal's typed non-retryable application error.
	var appErr *temporal.ApplicationError
	if asAppErr(err, &appErr) {
		if appErr.NonRetryable() {
			return true
		}
		// Also treat well-known error type strings as non-retryable.
		for _, t := range permanentErrorTypes {
			if appErr.Type() == t {
				return true
			}
		}
	}

	// Fallback: string-match on raw error message.
	msg := err.Error()
	for _, sentinel := range permanentErrorSentinels {
		if containsString(msg, sentinel) {
			return true
		}
	}
	return false
}

// permanentErrorTypes lists Temporal ApplicationError type strings that are
// always treated as permanent, non-retryable failures.
var permanentErrorTypes = []string{
	"InvalidPackage",
	"NotificationFailed",
	"AuthorizationFailed",
	"TenantNotFound",
	"PolicyViolation",
	"SchemaValidationError",
}

// permanentErrorSentinels lists substrings that, when found anywhere in an
// error message, mark the error as permanent.
var permanentErrorSentinels = []string{
	"invalid input",
	"not found",
	"unauthorized",
	"forbidden",
	"quota exceeded",
	"schema validation",
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// asAppErr is a thin wrapper around errors.As to avoid importing "errors" at
// the package level just for this one call.
func asAppErr(err error, target **temporal.ApplicationError) bool {
	// Walk the error chain manually via Unwrap if errors.As is not available.
	for err != nil {
		if ae, ok := err.(*temporal.ApplicationError); ok {
			*target = ae
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return false
}

// containsString reports whether substr appears in s.
func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure context and fmt are used to satisfy the import requirement.
var _ = context.Background
var _ = fmt.Sprintf
