# K-SOC-ID-004 -- BloodHound Azure/Entra ID OAuth Abuse Queries

**Vendored at:** `vendor/bloodhound/cypher/azure/oauth/*.cypher`  
**Role:** Detect OAuth consent grant abuse, overprivileged service principals, and app registration attack paths in Azure AD/Entra ID environments.

---

## 1. Vendored Cypher Queries

### 1.1 Overprivileged Service Principals

```cypher
-- vendor/bloodhound/cypher/azure/oauth/overprivileged_sp.cypher
-- Find service principals with dangerous Microsoft Graph permissions.
MATCH (sp:AZServicePrincipal)-[:AZHasAppRole]->(role:AZAppRole)
WHERE role.value IN [
    'RoleManagement.ReadWrite.Directory',
    'Application.ReadWrite.All',
    'AppRoleAssignment.ReadWrite.All',
    'Directory.ReadWrite.All',
    'GroupMember.ReadWrite.All',
    'Group.ReadWrite.All',
    'User.ReadWrite.All',
    'Mail.ReadWrite',
    'Files.ReadWrite.All',
    'Sites.ReadWrite.All'
]
OPTIONAL MATCH (sp)-[:AZRunsAs]->(app:AZApp)
OPTIONAL MATCH (owner:AZUser)-[:AZOwns]->(app)
RETURN
    sp.displayname AS service_principal,
    sp.appid AS app_id,
    collect(DISTINCT role.value) AS dangerous_permissions,
    app.displayname AS application_name,
    collect(DISTINCT owner.userprincipalname) AS owners
ORDER BY size(dangerous_permissions) DESC
```

### 1.2 Consent Grant Chains

```cypher
-- vendor/bloodhound/cypher/azure/oauth/consent_grant_chains.cypher
-- Discover delegated permission consent grants that create privilege escalation.
MATCH (u:AZUser)-[:AZConsented]->(grant:AZOAuth2PermissionGrant)
MATCH (grant)-[:AZGrantedTo]->(sp:AZServicePrincipal)
WHERE grant.scope CONTAINS 'ReadWrite'
   OR grant.scope CONTAINS '.All'
   OR grant.scope CONTAINS 'full_access'
OPTIONAL MATCH (sp)-[:AZHasAppRole]->(role:AZAppRole)
WHERE role.value CONTAINS 'ReadWrite' OR role.value CONTAINS '.All'
RETURN
    u.userprincipalname AS consenting_user,
    sp.displayname AS service_principal,
    grant.scope AS granted_scopes,
    collect(DISTINCT role.value) AS app_roles,
    grant.startdatetime AS grant_date
ORDER BY grant.startdatetime DESC
```

### 1.3 App Registration Abuse Paths

```cypher
-- vendor/bloodhound/cypher/azure/oauth/app_registration_abuse.cypher
-- Find paths where compromising an app owner leads to tenant-wide access.
MATCH (owner:AZUser)-[:AZOwns]->(app:AZApp)
MATCH (app)-[:AZRunsAs]->(sp:AZServicePrincipal)
MATCH (sp)-[:AZHasAppRole]->(role:AZAppRole)
WHERE role.value IN [
    'Application.ReadWrite.All',
    'RoleManagement.ReadWrite.Directory',
    'AppRoleAssignment.ReadWrite.All'
]
OPTIONAL MATCH p = shortestPath(
    (attacker:AZUser)-[:AZResetPassword|AZOwns|AZAddOwner|
     AZGlobalAdmin|AZPrivilegedRoleAdmin*1..4]->(owner)
)
WHERE attacker <> owner
RETURN
    attacker.userprincipalname AS attacker,
    owner.userprincipalname AS app_owner,
    app.displayname AS target_app,
    sp.displayname AS service_principal,
    collect(DISTINCT role.value) AS escalation_permissions,
    length(p) AS attack_hops,
    [n IN nodes(p) | n.displayname] AS path
ORDER BY attack_hops ASC
LIMIT 20
```

### 1.4 Stale App Registrations with High Privileges

```cypher
-- vendor/bloodhound/cypher/azure/oauth/stale_privileged_apps.cypher
-- Find app registrations not used recently but with high privileges.
MATCH (app:AZApp)-[:AZRunsAs]->(sp:AZServicePrincipal)
MATCH (sp)-[:AZHasAppRole]->(role:AZAppRole)
WHERE role.value CONTAINS 'ReadWrite'
  AND (sp.lastlogontimestamp IS NULL
       OR sp.lastlogontimestamp < datetime() - duration('P90D'))
OPTIONAL MATCH (owner:AZUser)-[:AZOwns]->(app)
RETURN
    app.displayname AS application,
    sp.appid AS app_id,
    sp.lastlogontimestamp AS last_used,
    collect(DISTINCT role.value) AS permissions,
    collect(DISTINCT owner.userprincipalname) AS owners
ORDER BY sp.lastlogontimestamp ASC
```

### 1.5 Cross-Tenant Trust Abuse

```cypher
-- vendor/bloodhound/cypher/azure/oauth/cross_tenant_trust.cypher
-- Identify multi-tenant apps with cross-tenant access.
MATCH (sp:AZServicePrincipal)
WHERE sp.appownertenantid <> sp.tenantid
MATCH (sp)-[:AZHasAppRole]->(role:AZAppRole)
RETURN
    sp.displayname AS service_principal,
    sp.appid AS app_id,
    sp.appownertenantid AS owner_tenant,
    sp.tenantid AS resource_tenant,
    collect(DISTINCT role.value) AS permissions
ORDER BY size(permissions) DESC
```

---

## 2. Neo4j Integration Code

```go
// internal/identity/azure_oauth.go
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

type AzureOAuthAnalyzer struct {
	driver   neo4j.DriverWithContext
	nc       *nats.Conn
	tenantID string
	queryDir string
}

func NewAzureOAuthAnalyzer(
	neo4jURI, user, pass, tenantID string,
	nc *nats.Conn,
) (*AzureOAuthAnalyzer, error) {
	driver, err := neo4j.NewDriverWithContext(
		neo4jURI,
		neo4j.BasicAuth(user, pass, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("neo4j driver: %w", err)
	}
	return &AzureOAuthAnalyzer{
		driver:   driver,
		nc:       nc,
		tenantID: tenantID,
		queryDir: "vendor/bloodhound/cypher/azure/oauth",
	}, nil
}

type OverprivilegedSP struct {
	Name         string   `json:"service_principal"`
	AppID        string   `json:"app_id"`
	Permissions  []string `json:"dangerous_permissions"`
	AppName      string   `json:"application_name"`
	Owners       []string `json:"owners"`
}

// FindOverprivilegedServicePrincipals queries for SPs with dangerous permissions.
func (a *AzureOAuthAnalyzer) FindOverprivilegedServicePrincipals(
	ctx context.Context,
) ([]OverprivilegedSP, error) {
	cypher, err := os.ReadFile(
		filepath.Join(a.queryDir, "overprivileged_sp.cypher"),
	)
	if err != nil {
		return nil, err
	}

	session := a.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, string(cypher), nil)
	if err != nil {
		return nil, fmt.Errorf("neo4j exec: %w", err)
	}

	var findings []OverprivilegedSP
	for result.Next(ctx) {
		rec := result.Record()
		sp := OverprivilegedSP{
			Name:    asString(rec, "service_principal"),
			AppID:   asString(rec, "app_id"),
			AppName: asString(rec, "application_name"),
		}
		sp.Permissions = asStringSlice(rec, "dangerous_permissions")
		sp.Owners = asStringSlice(rec, "owners")
		findings = append(findings, sp)
	}

	return findings, nil
}

// PublishOAuthFindings runs all Azure OAuth queries and publishes findings.
func (a *AzureOAuthAnalyzer) PublishOAuthFindings(ctx context.Context) error {
	sps, err := a.FindOverprivilegedServicePrincipals(ctx)
	if err != nil {
		return err
	}

	for _, sp := range sps {
		ocsf := map[string]interface{}{
			"class_uid":    2001,
			"activity_id":  1,
			"category_uid": 2,
			"severity_id":  4, // High
			"time":         time.Now().UTC().Format(time.RFC3339),
			"finding_info": map[string]interface{}{
				"title": fmt.Sprintf("Overprivileged Azure SP: %s (%d dangerous perms)",
					sp.Name, len(sp.Permissions)),
				"uid":   fmt.Sprintf("azure-sp-%s", sp.AppID),
				"types": []string{"Identity Risk", "OAuth Abuse"},
				"analytic": map[string]string{
					"name": "BloodHound Azure OAuth Analysis",
					"type": "Graph Query",
				},
			},
			"metadata": map[string]interface{}{
				"product":    map[string]string{"name": "BloodHound", "vendor_name": "SpecterOps"},
				"tenant_uid": a.tenantID,
			},
			"unmapped": map[string]interface{}{
				"permissions":  sp.Permissions,
				"owners":       sp.Owners,
				"app_name":     sp.AppName,
			},
		}

		data, _ := json.Marshal(ocsf)
		_ = a.nc.Publish(
			fmt.Sprintf("kubric.itdr.azure.%s", a.tenantID), data,
		)
	}

	return nil
}

func asString(rec *neo4j.Record, key string) string {
	val, ok := rec.Get(key)
	if !ok || val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

func asStringSlice(rec *neo4j.Record, key string) []string {
	val, ok := rec.Get(key)
	if !ok || val == nil {
		return nil
	}
	if slice, ok := val.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, v := range slice {
			result = append(result, fmt.Sprintf("%v", v))
		}
		return result
	}
	return nil
}
```

---

## 3. Azure AD Audit Log Ingestion to Neo4j

```go
// internal/identity/azure_audit_ingest.go
package identity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// AzureAuditIngester pulls Azure AD audit logs and updates the Neo4j graph.
type AzureAuditIngester struct {
	driver      neo4j.DriverWithContext
	graphAPIURL string
	accessToken string
}

// IngestAuditLogs pulls sign-in and audit logs from Microsoft Graph API
// and creates/updates nodes in the BloodHound-compatible graph.
func (aai *AzureAuditIngester) IngestAuditLogs(ctx context.Context) error {
	// Pull sign-in logs
	url := fmt.Sprintf("%s/v1.0/auditLogs/signIns?$top=100&$orderby=createdDateTime desc",
		aai.graphAPIURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+aai.accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("graph api: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Value []struct {
			UserPrincipalName string `json:"userPrincipalName"`
			AppDisplayName    string `json:"appDisplayName"`
			AppID             string `json:"appId"`
			IPAddress         string `json:"ipAddress"`
			Status            struct {
				ErrorCode      int    `json:"errorCode"`
				FailureReason  string `json:"failureReason"`
			} `json:"status"`
			CreatedDateTime   string `json:"createdDateTime"`
			ConditionalAccess []struct {
				Result string `json:"result"`
			} `json:"conditionalAccessPolicies"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse sign-ins: %w", err)
	}

	// Update Neo4j with sign-in data
	session := aai.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	for _, signIn := range result.Value {
		_, err := session.Run(ctx, `
			MERGE (u:AZUser {userprincipalname: $upn})
			MERGE (sp:AZServicePrincipal {appid: $appid})
			SET sp.displayname = $appname
			MERGE (u)-[r:AZSignedInTo]->(sp)
			SET r.lastSignIn = $ts,
			    r.ipAddress = $ip,
			    r.status = $status
		`, map[string]interface{}{
			"upn":     signIn.UserPrincipalName,
			"appid":   signIn.AppID,
			"appname": signIn.AppDisplayName,
			"ts":      signIn.CreatedDateTime,
			"ip":      signIn.IPAddress,
			"status":  signIn.Status.ErrorCode,
		})
		if err != nil {
			continue
		}
	}

	return nil
}
```

---

## 4. OCSF AuthenticationActivity Events

```go
// Publish identity anomalies as OCSF AuthenticationActivity (class 3002)
func publishIdentityAnomaly(
	nc *nats.Conn,
	tenantID, userPrincipal, appName, anomalyType string,
	severity int,
) error {
	event := map[string]interface{}{
		"class_uid":    3002,
		"activity_id":  1,
		"category_uid": 3, // Identity & Access Management
		"severity_id":  severity,
		"time":         time.Now().UTC().Format(time.RFC3339),
		"actor": map[string]interface{}{
			"user": map[string]string{
				"name": userPrincipal,
				"type": "AzureAD",
			},
		},
		"auth_protocol": "OAuth2",
		"service": map[string]string{
			"name": appName,
		},
		"finding_info": map[string]interface{}{
			"title": fmt.Sprintf("Azure Identity Anomaly: %s - %s", anomalyType, userPrincipal),
			"types": []string{anomalyType},
		},
		"metadata": map[string]interface{}{
			"product":    map[string]string{"name": "Kubric ITDR", "vendor_name": "Kubric"},
			"tenant_uid": tenantID,
		},
	}

	data, _ := json.Marshal(event)
	return nc.Publish(
		fmt.Sprintf("kubric.itdr.azure.%s", tenantID), data,
	)
}
```
