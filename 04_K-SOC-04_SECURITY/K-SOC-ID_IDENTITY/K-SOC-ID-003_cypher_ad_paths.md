# K-SOC-ID-003 -- BloodHound Cypher Queries for AD Attack Path Analysis

**Vendored at:** `vendor/bloodhound/cypher/*.cypher`  
**License:** BloodHound queries are knowledge artifacts — safe to vendor.  
**Role:** Active Directory attack path discovery via Cypher queries against Neo4j with ingested BloodHound data.

---

## 1. Architecture

```
┌──────────────────────────────────────────────────┐
│  Kubric ITDR Pipeline                             │
│                                                   │
│  BloodHound       Neo4j            Go Service     │
│  Collector  ──►  Graph DB  ◄──►  (KIC / SOC)     │
│  (SharpHound)     :7687           Cypher queries  │
│                                       │           │
│                                       │ NATS      │
│                                       ▼           │
│                              kubric.itdr.ad.>     │
└──────────────────────────────────────────────────┘
```

---

## 2. Vendored Cypher Queries

### 2.1 Shortest Path to Domain Admin

```cypher
-- vendor/bloodhound/cypher/shortest_path_to_da.cypher
-- Find shortest attack paths from any user to Domain Admin group.
MATCH p = shortestPath(
    (u:User {enabled: true})-[:MemberOf|HasSession|AdminTo|CanRDP|
     ExecuteDCOM|AllowedToDelegate|ForceChangePassword|GenericAll|
     GenericWrite|WriteDacl|WriteOwner|Owns|AddMember|
     SQLAdmin|ReadLAPSPassword|ReadGMSAPassword|HasSIDHistory*1..]->(g:Group)
)
WHERE g.name =~ '(?i)DOMAIN ADMINS@.*'
  AND u <> g
RETURN
    u.name AS start_user,
    length(p) AS path_length,
    [n IN nodes(p) | n.name] AS path_nodes,
    [r IN relationships(p) | type(r)] AS path_edges
ORDER BY path_length ASC
LIMIT 25
```

### 2.2 Kerberoastable Users

```cypher
-- vendor/bloodhound/cypher/kerberoastable_users.cypher
-- Find all Kerberoastable user accounts (users with SPNs set).
MATCH (u:User {hasspn: true, enabled: true})
OPTIONAL MATCH (u)-[:AdminTo]->(c:Computer)
OPTIONAL MATCH (u)-[:MemberOf*1..]->(g:Group)
WHERE g.name =~ '(?i)DOMAIN ADMINS@.*'
   OR g.name =~ '(?i)ENTERPRISE ADMINS@.*'
   OR g.highvalue = true
RETURN
    u.name AS username,
    u.serviceprincipalnames AS spns,
    u.pwdlastset AS password_last_set,
    u.lastlogontimestamp AS last_logon,
    CASE WHEN g IS NOT NULL THEN true ELSE false END AS is_privileged,
    collect(DISTINCT g.name) AS privileged_groups,
    count(DISTINCT c) AS admin_to_computers
ORDER BY is_privileged DESC, admin_to_computers DESC
```

### 2.3 AS-REP Roastable Accounts

```cypher
-- vendor/bloodhound/cypher/asrep_roastable.cypher
-- Find accounts that do not require Kerberos pre-authentication.
MATCH (u:User {dontreqpreauth: true, enabled: true})
OPTIONAL MATCH (u)-[:MemberOf*1..]->(g:Group {highvalue: true})
RETURN
    u.name AS username,
    u.description AS description,
    u.pwdlastset AS password_last_set,
    u.lastlogontimestamp AS last_logon,
    CASE WHEN g IS NOT NULL THEN true ELSE false END AS is_privileged,
    collect(DISTINCT g.name) AS high_value_groups
ORDER BY is_privileged DESC
```

### 2.4 Unconstrained Delegation Computers

```cypher
-- vendor/bloodhound/cypher/unconstrained_delegation.cypher
-- Find computers with unconstrained delegation enabled.
MATCH (c:Computer {unconstraineddelegation: true, enabled: true})
WHERE NOT c.name CONTAINS 'DC'
OPTIONAL MATCH (u:User)-[:HasSession]->(c)
RETURN
    c.name AS computer,
    c.operatingsystem AS os,
    collect(DISTINCT u.name) AS active_sessions,
    c.lastlogontimestamp AS last_logon
ORDER BY size(active_sessions) DESC
```

### 2.5 ACL Abuse Paths (WriteDACL / GenericAll)

```cypher
-- vendor/bloodhound/cypher/acl_abuse_paths.cypher
-- Find users with dangerous ACL permissions on high-value targets.
MATCH (u:User {enabled: true})-[r:GenericAll|GenericWrite|WriteDacl|
      WriteOwner|ForceChangePassword|Owns]->(target)
WHERE target.highvalue = true OR target:Group OR target:Computer
RETURN
    u.name AS attacker,
    type(r) AS permission,
    labels(target)[0] AS target_type,
    target.name AS target_name
ORDER BY permission, target_type
```

### 2.6 Shadow Admin Detection

```cypher
-- vendor/bloodhound/cypher/shadow_admins.cypher
-- Identify users with effective admin rights not in admin groups.
MATCH p = (u:User {enabled: true})-[:GenericAll|GenericWrite|WriteDacl|
           WriteOwner|Owns|AllExtendedRights*1..3]->(da:Group)
WHERE da.name =~ '(?i)DOMAIN ADMINS@.*'
  AND NOT (u)-[:MemberOf*1..]->(da)
RETURN
    u.name AS shadow_admin,
    length(p) AS hops,
    [n IN nodes(p) | n.name] AS path,
    [r IN relationships(p) | type(r)] AS permissions
ORDER BY hops ASC
```

---

## 3. Go Neo4j Client Code

```go
// internal/identity/bloodhound.go
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	nats "github.com/nats-io/nats.go"
)

type BloodHoundAnalyzer struct {
	driver   neo4j.DriverWithContext
	nc       *nats.Conn
	tenantID string
	queryDir string
}

func NewBloodHoundAnalyzer(
	neo4jURI, username, password, tenantID string,
	nc *nats.Conn,
) (*BloodHoundAnalyzer, error) {
	driver, err := neo4j.NewDriverWithContext(
		neo4jURI,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("neo4j driver: %w", err)
	}
	return &BloodHoundAnalyzer{
		driver:   driver,
		nc:       nc,
		tenantID: tenantID,
		queryDir: "vendor/bloodhound/cypher",
	}, nil
}

// AttackPath represents a discovered attack path in the AD graph.
type AttackPath struct {
	StartUser  string   `json:"start_user"`
	PathLength int      `json:"path_length"`
	PathNodes  []string `json:"path_nodes"`
	PathEdges  []string `json:"path_edges"`
	Risk       string   `json:"risk"` // critical, high, medium
}

// FindPathsToDomainAdmin discovers shortest attack paths to DA.
func (bha *BloodHoundAnalyzer) FindPathsToDomainAdmin(ctx context.Context) ([]AttackPath, error) {
	cypher, err := os.ReadFile(
		filepath.Join(bha.queryDir, "shortest_path_to_da.cypher"),
	)
	if err != nil {
		return nil, fmt.Errorf("read query: %w", err)
	}

	session := bha.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, string(cypher), nil)
	if err != nil {
		return nil, fmt.Errorf("neo4j query: %w", err)
	}

	var paths []AttackPath
	for result.Next(ctx) {
		record := result.Record()
		startUser, _ := record.Get("start_user")
		pathLength, _ := record.Get("path_length")
		pathNodes, _ := record.Get("path_nodes")
		pathEdges, _ := record.Get("path_edges")

		risk := "medium"
		if pl, ok := pathLength.(int64); ok && pl <= 2 {
			risk = "critical"
		} else if pl, ok := pathLength.(int64); ok && pl <= 4 {
			risk = "high"
		}

		path := AttackPath{
			StartUser:  fmt.Sprintf("%v", startUser),
			PathLength: int(pathLength.(int64)),
			Risk:       risk,
		}

		if nodes, ok := pathNodes.([]interface{}); ok {
			for _, n := range nodes {
				path.PathNodes = append(path.PathNodes, fmt.Sprintf("%v", n))
			}
		}
		if edges, ok := pathEdges.([]interface{}); ok {
			for _, e := range edges {
				path.PathEdges = append(path.PathEdges, fmt.Sprintf("%v", e))
			}
		}

		paths = append(paths, path)
	}

	return paths, nil
}

// KerberoastableUser represents a user account with SPN set.
type KerberoastableUser struct {
	Username        string   `json:"username"`
	SPNs            []string `json:"spns"`
	PasswordLastSet string   `json:"password_last_set"`
	IsPrivileged    bool     `json:"is_privileged"`
	Groups          []string `json:"privileged_groups"`
	AdminToCount    int      `json:"admin_to_computers"`
}

// FindKerberoastable returns all Kerberoastable users.
func (bha *BloodHoundAnalyzer) FindKerberoastable(ctx context.Context) ([]KerberoastableUser, error) {
	cypher, err := os.ReadFile(
		filepath.Join(bha.queryDir, "kerberoastable_users.cypher"),
	)
	if err != nil {
		return nil, fmt.Errorf("read query: %w", err)
	}

	session := bha.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, string(cypher), nil)
	if err != nil {
		return nil, fmt.Errorf("neo4j query: %w", err)
	}

	var users []KerberoastableUser
	for result.Next(ctx) {
		record := result.Record()
		user := KerberoastableUser{
			Username: fmt.Sprintf("%v", record.Values[0]),
		}
		if spns, ok := record.Values[1].([]interface{}); ok {
			for _, s := range spns {
				user.SPNs = append(user.SPNs, fmt.Sprintf("%v", s))
			}
		}
		user.PasswordLastSet = fmt.Sprintf("%v", record.Values[2])
		user.IsPrivileged, _ = record.Values[4].(bool)
		if groups, ok := record.Values[5].([]interface{}); ok {
			for _, g := range groups {
				user.Groups = append(user.Groups, fmt.Sprintf("%v", g))
			}
		}
		if count, ok := record.Values[6].(int64); ok {
			user.AdminToCount = int(count)
		}
		users = append(users, user)
	}

	return users, nil
}

// PublishFindings publishes AD analysis results to NATS.
func (bha *BloodHoundAnalyzer) PublishFindings(ctx context.Context) error {
	paths, err := bha.FindPathsToDomainAdmin(ctx)
	if err != nil {
		return err
	}

	for _, path := range paths {
		ocsf := map[string]interface{}{
			"class_uid":    2001, // OCSF SecurityFinding
			"activity_id":  1,   // Create
			"category_uid": 2,   // Findings
			"severity_id":  severityFromRisk(path.Risk),
			"time":         time.Now().UTC().Format(time.RFC3339),
			"finding_info": map[string]interface{}{
				"title": fmt.Sprintf("AD Attack Path: %s → Domain Admin (%d hops)",
					path.StartUser, path.PathLength),
				"uid":   fmt.Sprintf("bh-path-%s-%d", path.StartUser, path.PathLength),
				"types": []string{"Attack Path"},
				"analytic": map[string]string{
					"name": "BloodHound AD Path Analysis",
					"type": "Graph Query",
				},
			},
			"metadata": map[string]interface{}{
				"product":    map[string]string{"name": "BloodHound", "vendor_name": "SpecterOps"},
				"tenant_uid": bha.tenantID,
			},
			"unmapped": map[string]interface{}{
				"path_nodes": path.PathNodes,
				"path_edges": path.PathEdges,
			},
		}

		data, _ := json.Marshal(ocsf)
		subject := fmt.Sprintf("kubric.itdr.ad.%s", bha.tenantID)
		if err := bha.nc.Publish(subject, data); err != nil {
			return fmt.Errorf("nats publish: %w", err)
		}
	}

	return nil
}

func severityFromRisk(risk string) int {
	switch risk {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	default:
		return 2
	}
}

func (bha *BloodHoundAnalyzer) Close(ctx context.Context) error {
	return bha.driver.Close(ctx)
}
```

---

## 4. Graph Visualization Data Structure

```go
// internal/identity/graph_viz.go
package identity

// GraphVizData is the response format for the portal's attack path visualizer.
type GraphVizData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Type  string            `json:"type"` // User, Computer, Group, Domain
	Props map[string]string `json:"properties,omitempty"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"` // MemberOf, AdminTo, HasSession, etc.
}

// PathsToGraphViz converts attack paths to a visualizable graph structure.
func PathsToGraphViz(paths []AttackPath) GraphVizData {
	nodeMap := make(map[string]GraphNode)
	var edges []GraphEdge

	for _, path := range paths {
		for i, nodeName := range path.PathNodes {
			nodeType := "User"
			if i == len(path.PathNodes)-1 {
				nodeType = "Group"
			} else if i > 0 {
				nodeType = "Object"
			}

			nodeMap[nodeName] = GraphNode{
				ID:    nodeName,
				Label: nodeName,
				Type:  nodeType,
			}

			if i < len(path.PathEdges) && i+1 < len(path.PathNodes) {
				edges = append(edges, GraphEdge{
					Source: nodeName,
					Target: path.PathNodes[i+1],
					Label:  path.PathEdges[i],
				})
			}
		}
	}

	var nodes []GraphNode
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	return GraphVizData{Nodes: nodes, Edges: edges}
}
```

---

## 5. Scheduled Analysis

```go
// Run attack path analysis every 6 hours
ticker := time.NewTicker(6 * time.Hour)
defer ticker.Stop()

for {
    select {
    case <-ctx.Done():
        return
    case <-ticker.C:
        if err := analyzer.PublishFindings(ctx); err != nil {
            log.Printf("BloodHound analysis failed: %v", err)
        }
    }
}
```
