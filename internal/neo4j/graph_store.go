// Package neo4j provides a Bolt-protocol client for Kubric's Neo4j graph
// database.  It maps infrastructure topology (clusters, namespaces, pods,
// agents, VMs) to nodes and edges, enabling graph-powered queries such as
// "which agents are reachable from this compromised pod?" or "show the
// blast radius of namespace X".
//
// The Neo4j service is declared in docker-compose.yml (neo4j:5.18-community).
package neo4j

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GraphStore wraps a Neo4j driver and provides domain-specific helpers.
type GraphStore struct {
	driver neo4j.DriverWithContext
	dbName string
}

// AssetNode represents a Kubric infrastructure asset.
type AssetNode struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"` // cluster, namespace, pod, agent, vm, host
	Name      string            `json:"name"`
	TenantID  string            `json:"tenant_id"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// Relationship represents a directed edge between two assets.
type Relationship struct {
	FromID   string `json:"from_id"`
	ToID     string `json:"to_id"`
	RelType  string `json:"rel_type"` // RUNS_ON, MONITORS, CONNECTS_TO, PART_OF
	TenantID string `json:"tenant_id"`
}

// BlastResult is a node reachable within N hops from an origin.
type BlastResult struct {
	NodeID   string `json:"node_id"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Distance int    `json:"distance"`
}

// New connects to a Neo4j Bolt endpoint, e.g. "bolt://neo4j:7687".
// Auth uses the credentials supplied via env NEO4J_USER + NEO4J_PASSWORD
// (defaults: neo4j / kubric-neo4j — matching docker-compose.yml).
func New(uri, username, password string) (*GraphStore, error) {
	if uri == "" {
		return nil, nil // disabled — caller should check nil
	}
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j driver: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(context.Background())
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}
	return &GraphStore{driver: driver, dbName: "neo4j"}, nil
}

// Close releases the driver connection pool.
func (g *GraphStore) Close() {
	if g == nil {
		return
	}
	_ = g.driver.Close(context.Background())
}

// EnsureConstraints creates uniqueness constraints for asset IDs.
func (g *GraphStore) EnsureConstraints(ctx context.Context) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: g.dbName})
	defer session.Close(ctx)
	_, err := session.Run(ctx,
		"CREATE CONSTRAINT asset_id IF NOT EXISTS FOR (a:Asset) REQUIRE a.id IS UNIQUE",
		nil)
	return err
}

// UpsertAsset merges an AssetNode (create-or-update).
func (g *GraphStore) UpsertAsset(ctx context.Context, a AssetNode) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: g.dbName})
	defer session.Close(ctx)
	_, err := session.Run(ctx,
		`MERGE (n:Asset {id: $id})
		 SET n.kind = $kind, n.name = $name, n.tenant_id = $tenant_id,
		     n.labels = $labels, n.updated_at = datetime()
		 ON CREATE SET n.created_at = datetime()`,
		map[string]any{
			"id":        a.ID,
			"kind":      a.Kind,
			"name":      a.Name,
			"tenant_id": a.TenantID,
			"labels":    a.Labels,
		})
	return err
}

// UpsertRelationship creates or updates an edge between two assets.
func (g *GraphStore) UpsertRelationship(ctx context.Context, r Relationship) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: g.dbName})
	defer session.Close(ctx)
	// APOC merge_relationship for dynamic rel types
	cypher := fmt.Sprintf(
		`MATCH (a:Asset {id: $from_id}), (b:Asset {id: $to_id})
		 MERGE (a)-[r:%s]->(b)
		 SET r.tenant_id = $tenant_id, r.updated_at = datetime()`, r.RelType)
	_, err := session.Run(ctx, cypher, map[string]any{
		"from_id":   r.FromID,
		"to_id":     r.ToID,
		"tenant_id": r.TenantID,
	})
	return err
}

// BlastRadius returns all nodes reachable within maxHops hops from originID.
func (g *GraphStore) BlastRadius(ctx context.Context, tenantID, originID string, maxHops int) ([]BlastResult, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: g.dbName})
	defer session.Close(ctx)
	cypher := fmt.Sprintf(
		`MATCH path = (origin:Asset {id: $origin_id})-[*1..%d]-(reached:Asset)
		 WHERE reached.tenant_id = $tenant_id
		 RETURN DISTINCT reached.id AS node_id, reached.kind AS kind,
		        reached.name AS name, length(path) AS distance
		 ORDER BY distance`, maxHops)
	result, err := session.Run(ctx, cypher, map[string]any{
		"origin_id": originID,
		"tenant_id": tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("blast radius query: %w", err)
	}
	var out []BlastResult
	for result.Next(ctx) {
		rec := result.Record()
		nid, _ := rec.Get("node_id")
		kind, _ := rec.Get("kind")
		name, _ := rec.Get("name")
		dist, _ := rec.Get("distance")
		out = append(out, BlastResult{
			NodeID:   nid.(string),
			Kind:     kind.(string),
			Name:     name.(string),
			Distance: int(dist.(int64)),
		})
	}
	return out, result.Err()
}

// Topology returns all assets and relationships for a given tenant.
func (g *GraphStore) Topology(ctx context.Context, tenantID string) ([]AssetNode, []Relationship, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: g.dbName})
	defer session.Close(ctx)

	// Nodes
	nodesResult, err := session.Run(ctx,
		`MATCH (a:Asset {tenant_id: $tid}) RETURN a.id AS id, a.kind AS kind, a.name AS name`,
		map[string]any{"tid": tenantID})
	if err != nil {
		return nil, nil, fmt.Errorf("topology nodes: %w", err)
	}
	var nodes []AssetNode
	for nodesResult.Next(ctx) {
		r := nodesResult.Record()
		id, _ := r.Get("id")
		kind, _ := r.Get("kind")
		name, _ := r.Get("name")
		nodes = append(nodes, AssetNode{
			ID:       id.(string),
			Kind:     kind.(string),
			Name:     name.(string),
			TenantID: tenantID,
		})
	}

	// Edges
	edgesResult, err := session.Run(ctx,
		`MATCH (a:Asset {tenant_id: $tid})-[r]->(b:Asset {tenant_id: $tid})
		 RETURN a.id AS from_id, b.id AS to_id, type(r) AS rel_type`,
		map[string]any{"tid": tenantID})
	if err != nil {
		return nodes, nil, fmt.Errorf("topology edges: %w", err)
	}
	var edges []Relationship
	for edgesResult.Next(ctx) {
		r := edgesResult.Record()
		fid, _ := r.Get("from_id")
		tid, _ := r.Get("to_id")
		rt, _ := r.Get("rel_type")
		edges = append(edges, Relationship{
			FromID:   fid.(string),
			ToID:     tid.(string),
			RelType:  rt.(string),
			TenantID: tenantID,
		})
	}
	return nodes, edges, edgesResult.Err()
}
