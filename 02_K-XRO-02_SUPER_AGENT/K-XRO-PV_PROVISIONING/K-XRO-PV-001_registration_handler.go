// Package provisioning handles agent registration requests from the provisioning API.
// Agents POST their hardware fingerprint and receive NATS credentials + Vault roles.
package provisioning

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RegistrationRequest is the JSON body sent by an agent at first boot.
type RegistrationRequest struct {
	AgentType   string `json:"agent_type"`
	Fingerprint string `json:"fingerprint"`
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
	Version     string `json:"version"`
	NATSUrl     string `json:"nats_url"`
}

// validate returns a human-readable error if required fields are missing.
func (r *RegistrationRequest) validate() error {
	if strings.TrimSpace(r.AgentType) == "" {
		return fmt.Errorf("agent_type is required")
	}
	if strings.TrimSpace(r.Fingerprint) == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if len(r.Fingerprint) != 64 {
		return fmt.Errorf("fingerprint must be a 64-char hex string (Blake3-256), got %d chars", len(r.Fingerprint))
	}
	if strings.TrimSpace(r.Hostname) == "" {
		return fmt.Errorf("hostname is required")
	}
	return nil
}

// RegistrationResponse is sent back to a successfully registered agent.
type RegistrationResponse struct {
	AgentID             string `json:"agent_id"`
	NATSCredentials    string `json:"nats_credentials"`
	VaultRoleID        string `json:"vault_role_id"`
	VaultSecretID      string `json:"vault_secret_id"`
	PollingIntervalSecs int   `json:"polling_interval_secs"`
	IssuedAt           int64  `json:"issued_at"`
}

// AgentRecord is the persistent record stored after registration.
type AgentRecord struct {
	ID           string
	TenantID     string
	AgentType    string
	Fingerprint  string
	Hostname     string
	OS           string
	Arch         string
	Version      string
	Status       string
	RegisteredAt time.Time
}

// ProvisioningStore abstracts the persistence layer for agent records.
type ProvisioningStore interface {
	// IsKnownFingerprint returns true when the fingerprint has already been registered.
	IsKnownFingerprint(ctx context.Context, fingerprint string) (bool, error)
	// RegisterAgent persists a new agent record and returns the created record.
	RegisterAgent(ctx context.Context, req RegistrationRequest) (AgentRecord, error)
	// GetAgentByFingerprint retrieves an existing record by its hardware fingerprint.
	GetAgentByFingerprint(ctx context.Context, fingerprint string) (*AgentRecord, error)
}

// rateLimitEntry tracks registration attempts per source IP.
type rateLimitEntry struct {
	count    int
	windowAt time.Time
}

const (
	rateLimitWindow = time.Hour
	rateLimitMax    = 10
)

// RegistrationHandler is an http.Handler that processes agent registration requests.
type RegistrationHandler struct {
	store     ProvisioningStore
	vaultAddr string
	natsURL   string

	mu        sync.Mutex
	rateLimit map[string]*rateLimitEntry
}

// NewRegistrationHandler creates a RegistrationHandler with its dependencies injected.
func NewRegistrationHandler(store ProvisioningStore, vaultAddr, natsURL string) *RegistrationHandler {
	return &RegistrationHandler{
		store:     store,
		vaultAddr: vaultAddr,
		natsURL:   natsURL,
		rateLimit: make(map[string]*rateLimitEntry),
	}
}

// ServeHTTP implements http.Handler.
//
//	POST /provision/register
//
// Flow:
//  1. Extract and validate remote IP for rate limiting.
//  2. Decode and validate JSON body.
//  3. If fingerprint already registered, return existing credentials.
//  4. Otherwise register and issue fresh credentials.
func (h *RegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := remoteIP(r)
	if !h.checkRateLimit(ip) {
		jsonError(w, "rate limit exceeded: max 10 registrations per hour per IP", http.StatusTooManyRequests)
		return
	}

	var req RegistrationRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		jsonError(w, fmt.Sprintf("invalid JSON body: %s", err), http.StatusBadRequest)
		return
	}

	if err := req.validate(); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Check for an already-registered fingerprint to enable idempotent re-registration.
	known, err := h.store.IsKnownFingerprint(ctx, req.Fingerprint)
	if err != nil {
		jsonError(w, "store lookup failed", http.StatusInternalServerError)
		return
	}

	if known {
		record, err := h.store.GetAgentByFingerprint(ctx, req.Fingerprint)
		if err != nil || record == nil {
			jsonError(w, "failed to retrieve existing agent record", http.StatusInternalServerError)
			return
		}
		resp, err := h.buildResponse(ctx, record.ID, record.TenantID, record.AgentType)
		if err != nil {
			jsonError(w, "credential generation failed", http.StatusInternalServerError)
			return
		}
		jsonOK(w, resp, http.StatusOK)
		return
	}

	// New registration path.
	record, err := h.store.RegisterAgent(ctx, req)
	if err != nil {
		jsonError(w, fmt.Sprintf("registration failed: %s", err), http.StatusInternalServerError)
		return
	}

	resp, err := h.buildResponse(ctx, record.ID, record.TenantID, record.AgentType)
	if err != nil {
		jsonError(w, "credential generation failed after registration", http.StatusInternalServerError)
		return
	}
	jsonOK(w, resp, http.StatusCreated)
}

// buildResponse manufactures a RegistrationResponse by issuing Vault approle credentials
// and constructing the NATS credentials string.
func (h *RegistrationHandler) buildResponse(ctx context.Context, agentID, tenantID, agentType string) (RegistrationResponse, error) {
	roleID, secretID, err := h.issueVaultApprole(ctx, agentID, tenantID, agentType)
	if err != nil {
		return RegistrationResponse{}, fmt.Errorf("vault approle: %w", err)
	}

	natsCreds := h.buildNATSCredentials(agentID, tenantID)

	return RegistrationResponse{
		AgentID:             agentID,
		NATSCredentials:    natsCreds,
		VaultRoleID:        roleID,
		VaultSecretID:      secretID,
		PollingIntervalSecs: 60,
		IssuedAt:           time.Now().Unix(),
	}, nil
}

// issueVaultApprole creates or fetches a Vault AppRole for the given agent.
// In a real deployment this calls the Vault HTTP API; here it is stubbed to return
// deterministic hex tokens derived from the agent ID so that the handler compiles
// and runs correctly without a live Vault instance.
func (h *RegistrationHandler) issueVaultApprole(_ context.Context, agentID, tenantID, agentType string) (roleID, secretID string, err error) {
	// Production implementation would POST to:
	//   h.vaultAddr + "/v1/auth/approle/role/" + roleName + "/secret-id"
	// For now generate stable mock credentials.
	roleName := fmt.Sprintf("kubric-%s-%s-%s", tenantID, agentType, agentID[:8])
	roleID = hashString(roleName + ":role")
	secretID = hashString(roleName + ":secret:" + time.Now().Format("2006-01-02"))
	return
}

// buildNATSCredentials returns a minimal NATS credentials file content.
// In production this is fetched from the NATS account server (NSC).
func (h *RegistrationHandler) buildNATSCredentials(agentID, tenantID string) string {
	return fmt.Sprintf(
		"-----BEGIN NATS USER JWT-----\n%s\n-----END NATS USER JWT-----\n\n"+
			"-----BEGIN USER NKEY SEED-----\n%s\n-----END USER NKEY SEED-----\n",
		hashString("jwt:"+agentID+":"+tenantID),
		hashString("seed:"+agentID+":"+tenantID),
	)
}

// checkRateLimit returns false when the caller has exceeded rateLimitMax registrations
// within the current rateLimitWindow. The map is cleaned lazily on each call.
func (h *RegistrationHandler) checkRateLimit(ip string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	entry, ok := h.rateLimit[ip]
	if !ok || now.Sub(entry.windowAt) > rateLimitWindow {
		h.rateLimit[ip] = &rateLimitEntry{count: 1, windowAt: now}
		h.cleanExpiredLocked(now)
		return true
	}
	if entry.count >= rateLimitMax {
		return false
	}
	entry.count++
	return true
}

// cleanExpiredLocked removes entries older than rateLimitWindow. Caller must hold h.mu.
func (h *RegistrationHandler) cleanExpiredLocked(now time.Time) {
	for ip, entry := range h.rateLimit {
		if now.Sub(entry.windowAt) > rateLimitWindow {
			delete(h.rateLimit, ip)
		}
	}
}

// --- HTTP helpers ---

type errorBody struct {
	Error string `json:"error"`
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorBody{Error: msg})
}

func jsonOK(w http.ResponseWriter, v any, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// remoteIP extracts the client IP from X-Forwarded-For or RemoteAddr.
func remoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- Miscellaneous helpers ---

// GenerateAgentID returns a random 32-byte hex string (UUID-like, no dashes).
func GenerateAgentID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hashString returns a short deterministic hex digest of s, used in mock credential generation.
func hashString(s string) string {
	h := blake3New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:32]
}
