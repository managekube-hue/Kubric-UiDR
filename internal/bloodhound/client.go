// Package bloodhound provides a Go client for the BloodHound Community Edition
// (CE) REST API.
//
// BloodHound CE is an Apache-2.0-licensed Active Directory / Azure attack path
// analysis tool.  This client communicates over HTTP using HMAC-SHA256 signed
// requests (token_id + token_key) per the BH CE API specification.
//
// Default endpoint: http://localhost:8080
// Auth: HMAC-SHA256 signed requests using API token_id + token_key.
package bloodhound

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// --------------------------------------------------------------------------
// Client
// --------------------------------------------------------------------------

// Client for BloodHound CE REST API (Apache-2.0 boundary: HTTP only).
// Authentication uses HMAC-SHA256 signed requests with a token_id and
// token_key pair obtained from the BH CE UI or API.
type Client struct {
	baseURL  string
	tokenID  string
	tokenKey string
	hc       *http.Client
}

// New creates a BloodHound CE client.  If baseURL is empty the constructor
// returns (nil, nil) so callers can treat BloodHound as an optional
// integration.
func New(baseURL, tokenID, tokenKey string) (*Client, error) {
	if baseURL == "" {
		return nil, nil
	}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		tokenID:  tokenID,
		tokenKey: tokenKey,
		hc: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Close is nil-safe and releases any resources held by the client.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.hc.CloseIdleConnections()
}

// --------------------------------------------------------------------------
// Domain types
// --------------------------------------------------------------------------

// Domain represents a collected AD or Azure domain in BloodHound.
type Domain struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // active-directory, azure
	Collected bool   `json:"collected"`
	NodeCount int    `json:"node_count,omitempty"`
}

// PathFinding holds the graph result of a shortest-path or attack-path query.
type PathFinding struct {
	Nodes []PathNode `json:"nodes"`
	Edges []PathEdge `json:"edges"`
}

// PathNode is a single node in a BloodHound attack path graph.
type PathNode struct {
	ID    string                 `json:"id"`
	Kind  string                 `json:"kind"` // User, Group, Computer, OU, GPO, Domain, etc.
	Label string                 `json:"label"`
	Props map[string]interface{} `json:"properties,omitempty"`
}

// PathEdge is a directed edge between two nodes in the attack path graph.
type PathEdge struct {
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Kind   string                 `json:"kind"` // MemberOf, GenericAll, WriteDacl, AdminTo, HasSession, etc.
	Props  map[string]interface{} `json:"properties,omitempty"`
}

// CypherResult holds the response from a Cypher query against BloodHound's
// Neo4j-backed graph.
type CypherResult struct {
	Nodes []PathNode `json:"nodes"`
	Edges []PathEdge `json:"edges"`
}

// DomainStats holds aggregate entity counts for a collected domain.
type DomainStats struct {
	Users         int `json:"users"`
	Groups        int `json:"groups"`
	Computers     int `json:"computers"`
	OUs           int `json:"ous"`
	GPOs          int `json:"gpos"`
	Sessions      int `json:"sessions"`
	ACLs          int `json:"acls"`
	Relationships int `json:"relationships"`
}

// AttackPath represents a pre-built attack path finding in BloodHound CE.
type AttackPath struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	DomainID       string  `json:"domain_id"`
	PrincipalCount int     `json:"finding_count"`
	ImpactValue    float64 `json:"impact_value"`
	Exposure       float64 `json:"exposure"`
}

// --------------------------------------------------------------------------
// Internal: HMAC-SHA256 request signing
// --------------------------------------------------------------------------

// signRequest signs an outgoing HTTP request per the BloodHound CE HMAC-SHA256
// scheme.  The signature is computed over:
//
//	HMAC-SHA256(token_key, method + uri + datetime_RFC3339 + body_sha256_b64)
//
// Headers set:
//   - Authorization: bhesignature <token_id>
//   - RequestDate:   RFC-3339 timestamp
//   - Signature:     base64(HMAC-SHA256(token_key, signing_payload))
func (c *Client) signRequest(req *http.Request, body []byte) {
	now := time.Now().UTC().Format(time.RFC3339)
	req.Header.Set("RequestDate", now)

	// SHA-256 digest of the request body (empty body -> digest of zero bytes).
	bodyDigest := sha256.Sum256(body)
	bodyDigestB64 := base64.StdEncoding.EncodeToString(bodyDigest[:])

	// Build the string to sign: METHOD + URI + RequestDate + body_digest_b64.
	uri := req.URL.RequestURI()
	sigPayload := req.Method + uri + now + bodyDigestB64

	// HMAC-SHA256 with the token_key.
	mac := hmac.New(sha256.New, []byte(c.tokenKey))
	mac.Write([]byte(sigPayload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req.Header.Set("Authorization", fmt.Sprintf("bhesignature %s", c.tokenID))
	req.Header.Set("Signature", signature)
}

// do builds, signs, executes, and decodes a JSON API request.
func (c *Client) do(ctx context.Context, method, path string, body, dst interface{}) error {
	var bodyBytes []byte
	var reqBody io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("bloodhound: marshal request: %w", err)
		}
		bodyBytes = b
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("bloodhound: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Sign the request with HMAC-SHA256.
	c.signRequest(req, bodyBytes)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("bloodhound: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bloodhound: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bloodhound: %s %s returned %d: %s",
			method, path, resp.StatusCode, string(respBody))
	}

	if dst != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dst); err != nil {
			return fmt.Errorf("bloodhound: decode response: %w", err)
		}
	}
	return nil
}

// doUnwrap is like do but unwraps the BloodHound CE "data" envelope that most
// list/detail endpoints return:
//
//	{ "data": <actual payload> }
func (c *Client) doUnwrap(ctx context.Context, method, path string, body, dst interface{}) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.do(ctx, method, path, body, &envelope); err != nil {
		return err
	}
	if dst != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, dst); err != nil {
			return fmt.Errorf("bloodhound: decode data envelope: %w", err)
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Domain operations
// --------------------------------------------------------------------------

// ListDomains returns all collected AD/Azure domains.
// GET /api/v2/available-domains
func (c *Client) ListDomains(ctx context.Context) ([]Domain, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	var domains []Domain
	if err := c.doUnwrap(ctx, http.MethodGet, "/api/v2/available-domains", nil, &domains); err != nil {
		return nil, fmt.Errorf("bloodhound: list domains: %w", err)
	}
	return domains, nil
}

// GetDomainStats returns aggregate entity counts (users, groups, computers,
// etc.) for a given domain.
// GET /api/v2/domains/{domainID}
func (c *Client) GetDomainStats(ctx context.Context, domainID string) (*DomainStats, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	var stats DomainStats
	path := "/api/v2/domains/" + domainID
	if err := c.doUnwrap(ctx, http.MethodGet, path, nil, &stats); err != nil {
		return nil, fmt.Errorf("bloodhound: get domain stats %s: %w", domainID, err)
	}
	return &stats, nil
}

// --------------------------------------------------------------------------
// Graph / path-finding operations
// --------------------------------------------------------------------------

// ShortestPath finds the shortest attack path between two nodes.
// GET /api/v2/graphs/shortest-path?start_node={startNode}&end_node={endNode}
func (c *Client) ShortestPath(ctx context.Context, startNode, endNode string) (*PathFinding, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	path := fmt.Sprintf("/api/v2/graphs/shortest-path?start_node=%s&end_node=%s",
		startNode, endNode)
	var result PathFinding
	if err := c.doUnwrap(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("bloodhound: shortest path %s -> %s: %w",
			startNode, endNode, err)
	}
	return &result, nil
}

// cypherRequest is the payload for POST /api/v2/graphs/cypher.
type cypherRequest struct {
	Query             string                 `json:"query"`
	QueryParameters   map[string]interface{} `json:"query_parameters,omitempty"`
	IncludeProperties bool                   `json:"include_properties"`
}

// RunCypher executes an arbitrary Cypher query against the BloodHound graph.
// POST /api/v2/graphs/cypher
func (c *Client) RunCypher(ctx context.Context, query string, params map[string]interface{}) (*CypherResult, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	body := cypherRequest{
		Query:             query,
		QueryParameters:   params,
		IncludeProperties: true,
	}
	var result CypherResult
	if err := c.doUnwrap(ctx, http.MethodPost, "/api/v2/graphs/cypher", body, &result); err != nil {
		return nil, fmt.Errorf("bloodhound: run cypher: %w", err)
	}
	return &result, nil
}

// --------------------------------------------------------------------------
// Attack path findings
// --------------------------------------------------------------------------

// ListAttackPaths returns pre-built attack path findings for a domain.
// GET /api/v2/domains/{domainID}/attack-path-findings
func (c *Client) ListAttackPaths(ctx context.Context, domainID string) ([]AttackPath, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	path := "/api/v2/domains/" + domainID + "/attack-path-findings"
	var findings []AttackPath
	if err := c.doUnwrap(ctx, http.MethodGet, path, nil, &findings); err != nil {
		return nil, fmt.Errorf("bloodhound: list attack paths for domain %s: %w",
			domainID, err)
	}
	return findings, nil
}

// GetAttackPathDetails returns the full graph (nodes + edges) for a specific
// attack path finding.
// GET /api/v2/attack-path-findings/{attackPathID}
func (c *Client) GetAttackPathDetails(ctx context.Context, attackPathID string) (*PathFinding, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}
	path := "/api/v2/attack-path-findings/" + attackPathID
	var result PathFinding
	if err := c.doUnwrap(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("bloodhound: get attack path details %s: %w",
			attackPathID, err)
	}
	return &result, nil
}

// --------------------------------------------------------------------------
// High-value target queries
// --------------------------------------------------------------------------

// ListKerberoastable returns all user nodes that are Kerberoastable in the
// given domain by querying for users with SPNs set.
func (c *Client) ListKerberoastable(ctx context.Context, domainID string) ([]PathNode, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}

	query := `MATCH (u:User {domain_id: $domainID}) WHERE u.hasspn = true RETURN u`
	params := map[string]interface{}{
		"domainID": domainID,
	}

	result, err := c.RunCypher(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: list kerberoastable for domain %s: %w",
			domainID, err)
	}
	return result.Nodes, nil
}

// ListDCSync returns all principal nodes that have DCSync privileges
// (GetChanges + GetChangesAll) in the given domain.
func (c *Client) ListDCSync(ctx context.Context, domainID string) ([]PathNode, error) {
	if c == nil {
		return nil, fmt.Errorf("bloodhound: client is nil")
	}

	query := `MATCH p=(n)-[:GetChanges|GetChangesAll*1..]->(d:Domain {objectid: $domainID})
WHERE n.domain_id = $domainID
RETURN n`
	params := map[string]interface{}{
		"domainID": domainID,
	}

	result, err := c.RunCypher(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("bloodhound: list dcsync for domain %s: %w",
			domainID, err)
	}
	return result.Nodes, nil
}

// --------------------------------------------------------------------------
// Health check
// --------------------------------------------------------------------------

// Health checks connectivity to the BloodHound CE instance.
// GET /api/v2/self (returns current user info; 200 = healthy)
func (c *Client) Health(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("bloodhound: client is nil")
	}
	if err := c.do(ctx, http.MethodGet, "/api/v2/self", nil, nil); err != nil {
		return fmt.Errorf("bloodhound: health check failed: %w", err)
	}
	return nil
}
