// K-SOC-ID-002 — Neo4j graph client for AD attack-path analysis.
// Provides Bolt-protocol queries against a Neo4j/BloodHound CE graph
// for shortest-path and admin-path discovery.
//
// NOTE: AttackPath is already defined in K-SOC-ID-001_bloodhound_analysis.go
// with different fields (PathID, Steps, Techniques, Severity).  This file
// uses Neo4jPath for the Neo4j-specific wire type
// (From, To, Hops, Relationships) to avoid symbol collision.
//
// Env vars:
//
//	NEO4J_URI       bolt:// or neo4j:// URI (required)
//	NEO4J_USER      username (default: neo4j)
//	NEO4J_PASSWORD  password (required)
package soc

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Neo4jPath is an AD attack path derived from a Neo4j Cypher query.
// (Distinct from AttackPath in ID-001 which carries BloodHound API fields.)
type Neo4jPath struct {
	From          string   `json:"from"`
	To            string   `json:"to"`
	Hops          int      `json:"hops"`
	Relationships []string `json:"relationships"`
}

// GraphNode is a Neo4j node with labels and property bag.
type GraphNode struct {
	Labels     []string       `json:"labels"`
	Properties map[string]any `json:"properties"`
}

// ---------------------------------------------------------------------------
// Neo4jClient
// ---------------------------------------------------------------------------

// Neo4jClient provides read-only Cypher queries against a Neo4j instance.
// It is safe for concurrent use; the underlying driver manages a connection pool.
type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

// NewNeo4jClient creates a client from NEO4J_URI / NEO4J_USER / NEO4J_PASSWORD
// environment variables.
func NewNeo4jClient() (*Neo4jClient, error) {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		return nil, fmt.Errorf("neo4j: NEO4J_URI env var is required")
	}
	user := os.Getenv("NEO4J_USER")
	if user == "" {
		user = "neo4j"
	}
	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("neo4j: NEO4J_PASSWORD env var is required")
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: create driver: %w", err)
	}
	return &Neo4jClient{driver: driver}, nil
}

// Close releases all driver resources.
func (c *Neo4jClient) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// RunCypher executes a Cypher query and returns all rows as a slice of
// map[string]any.  Parameters are passed as a map for safe injection.
func (c *Neo4jClient) RunCypher(
	ctx context.Context,
	cypher string,
	params map[string]any,
) ([]map[string]any, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("neo4j: run cypher: %w", err)
	}

	var rows []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]any, len(record.Keys))
		for _, key := range record.Keys {
			val, _ := record.Get(key)
			row[key] = val
		}
		rows = append(rows, row)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("neo4j: cypher result: %w", err)
	}
	return rows, nil
}

// FindShortestPath finds the shortest path from fromUser to toObject in the
// BloodHound graph using Neo4j's shortestPath algorithm.
//
// maxDepth controls the maximum relationship hops ([*..maxDepth]).
func (c *Neo4jClient) FindShortestPath(
	ctx context.Context,
	fromUser, toObject string,
	maxDepth int,
) ([]GraphNode, error) {
	cypher := fmt.Sprintf(
		`MATCH p=shortestPath((a:User {name:$from})-[*..%d]->(b {name:$to}))
		 RETURN nodes(p) AS nodes, length(p) AS hops`,
		maxDepth,
	)
	rows, err := c.RunCypher(ctx, cypher, map[string]any{
		"from": fromUser,
		"to":   toObject,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	rawNodes, _ := rows[0]["nodes"].([]any)
	return nodesToGraphNodes(rawNodes), nil
}

// FindAdminPaths returns all enabled user-to-computer admin relationships
// within 5 hops in the BloodHound graph, aggregated as Neo4jPath slices.
//
// Cypher: MATCH (u:User)-[r:MemberOf|AdminTo*1..5]->(c:Computer)
//
//	WHERE u.enabled=true RETURN u,r,c
func (c *Neo4jClient) FindAdminPaths(ctx context.Context, tenantID string) ([]Neo4jPath, error) {
	cypher := `MATCH (u:User)-[r:MemberOf|AdminTo*1..5]->(c:Computer)
	           WHERE u.enabled = true
	           RETURN u.name AS from_user, c.name AS to_computer,
	                  length(r) AS hops,
	                  [rel IN r | type(rel)] AS rel_types`

	rows, err := c.RunCypher(ctx, cypher, map[string]any{})
	if err != nil {
		return nil, err
	}

	paths := make([]Neo4jPath, 0, len(rows))
	for _, row := range rows {
		from, _ := row["from_user"].(string)
		to, _ := row["to_computer"].(string)
		hopsVal, _ := row["hops"].(int64)
		relTypesRaw, _ := row["rel_types"].([]any)

		relTypes := make([]string, 0, len(relTypesRaw))
		for _, rt := range relTypesRaw {
			if s, ok := rt.(string); ok {
				relTypes = append(relTypes, s)
			}
		}
		paths = append(paths, Neo4jPath{
			From:          from,
			To:            to,
			Hops:          int(hopsVal),
			Relationships: relTypes,
		})
	}
	return paths, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// nodesToGraphNodes converts a slice of raw Neo4j node interface values
// into typed GraphNode structs.
func nodesToGraphNodes(rawNodes []any) []GraphNode {
	nodes := make([]GraphNode, 0, len(rawNodes))
	for _, raw := range rawNodes {
		n, ok := raw.(neo4j.Node)
		if !ok {
			continue
		}
		labels := n.Labels
		if labels == nil {
			labels = []string{}
		}
		nodes = append(nodes, GraphNode{
			Labels:     labels,
			Properties: n.Props,
		})
	}
	return nodes
}
