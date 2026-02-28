# K-SOC-TI-005 -- HaveIBeenPwned API Integration

**API:** HIBP Breached Account API v3  
**Rate Limit:** 1 request per 1.5 seconds (free tier), 10/sec with paid API key  
**Role:** Check customer employee emails against known data breaches for credential exposure risk.

---

## 1. Go HTTP Client

```go
// internal/threatintel/hibp.go
package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
)

const (
	hibpBaseURL    = "https://haveibeenpwned.com/api/v3"
	hibpRateLimit  = 1500 * time.Millisecond // 1 req / 1.5s (free tier)
)

// HIBPClient queries the HaveIBeenPwned API.
type HIBPClient struct {
	httpClient *http.Client
	apiKey     string
	nc         *nats.Conn
	tenantID   string
	rateLimiter <-chan time.Time
}

func NewHIBPClient(nc *nats.Conn, tenantID string) *HIBPClient {
	return &HIBPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     os.Getenv("HIBP_API_KEY"),
		nc:         nc,
		tenantID:   tenantID,
		rateLimiter: time.Tick(hibpRateLimit),
	}
}

// Breach represents a single data breach from HIBP.
type Breach struct {
	Name         string   `json:"Name"`
	Title        string   `json:"Title"`
	Domain       string   `json:"Domain"`
	BreachDate   string   `json:"BreachDate"`
	AddedDate    string   `json:"AddedDate"`
	ModifiedDate string   `json:"ModifiedDate"`
	PwnCount     int64    `json:"PwnCount"`
	Description  string   `json:"Description"`
	LogoPath     string   `json:"LogoPath"`
	DataClasses  []string `json:"DataClasses"`
	IsVerified   bool     `json:"IsVerified"`
	IsFabricated bool     `json:"IsFabricated"`
	IsSensitive  bool     `json:"IsSensitive"`
	IsRetired    bool     `json:"IsRetired"`
	IsSpamList   bool     `json:"IsSpamList"`
}

// CheckEmail queries HIBP for breaches containing the given email.
func (h *HIBPClient) CheckEmail(ctx context.Context, email string) ([]Breach, error) {
	// Rate limiting
	<-h.rateLimiter

	url := fmt.Sprintf("%s/breachedaccount/%s?truncateResponse=false",
		hibpBaseURL, email)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("hibp-api-key", h.apiKey)
	req.Header.Set("User-Agent", "Kubric-SecurityPlatform/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hibp request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, _ := io.ReadAll(resp.Body)
		var breaches []Breach
		if err := json.Unmarshal(body, &breaches); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		return breaches, nil

	case http.StatusNotFound:
		return nil, nil // No breaches found — clean

	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("rate limited, retry after %s seconds", retryAfter)

	case http.StatusUnauthorized:
		return nil, fmt.Errorf("invalid HIBP API key")

	default:
		return nil, fmt.Errorf("hibp api error: status %d", resp.StatusCode)
	}
}

// CheckPasswordHash uses the k-Anonymity API to check password exposure.
// Only sends the first 5 chars of the SHA-1 hash.
func (h *HIBPClient) CheckPasswordHash(ctx context.Context, sha1Prefix string) (map[string]int, error) {
	if len(sha1Prefix) < 5 {
		return nil, fmt.Errorf("sha1 prefix must be at least 5 characters")
	}

	url := fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", sha1Prefix[:5])
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	results := make(map[string]int)
	for _, line := range splitLines(string(body)) {
		if len(line) > 36 {
			suffix := line[:35]
			var count int
			fmt.Sscanf(line[36:], "%d", &count)
			results[suffix] = count
		}
	}
	return results, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
```

---

## 2. Batch Email Checking

```go
// internal/threatintel/hibp_batch.go
package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// BreachResult contains the HIBP check result for one email.
type BreachResult struct {
	Email       string   `json:"email"`
	BreachCount int      `json:"breach_count"`
	Breaches    []Breach `json:"breaches,omitempty"`
	CheckedAt   string   `json:"checked_at"`
	RiskLevel   string   `json:"risk_level"` // none, low, medium, high, critical
}

// CheckEmailBatch processes a list of employee emails.
func (h *HIBPClient) CheckEmailBatch(
	ctx context.Context,
	emails []string,
) ([]BreachResult, error) {
	var results []BreachResult

	for _, email := range emails {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		breaches, err := h.CheckEmail(ctx, email)
		if err != nil {
			// Log error but continue with remaining emails
			results = append(results, BreachResult{
				Email:     email,
				CheckedAt: time.Now().UTC().Format(time.RFC3339),
				RiskLevel: "unknown",
			})
			continue
		}

		risk := assessBreachRisk(breaches)
		results = append(results, BreachResult{
			Email:       email,
			BreachCount: len(breaches),
			Breaches:    breaches,
			CheckedAt:   time.Now().UTC().Format(time.RFC3339),
			RiskLevel:   risk,
		})
	}

	return results, nil
}

// assessBreachRisk scores the risk based on breach characteristics.
func assessBreachRisk(breaches []Breach) string {
	if len(breaches) == 0 {
		return "none"
	}

	hasPasswords := false
	hasRecent := false
	hasVerified := false

	for _, b := range breaches {
		for _, dc := range b.DataClasses {
			if dc == "Passwords" || dc == "Password hints" {
				hasPasswords = true
			}
		}
		if t, err := time.Parse("2006-01-02", b.BreachDate); err == nil {
			if time.Since(t) < 365*24*time.Hour {
				hasRecent = true
			}
		}
		if b.IsVerified {
			hasVerified = true
		}
	}

	switch {
	case hasPasswords && hasRecent && hasVerified:
		return "critical"
	case hasPasswords && hasVerified:
		return "high"
	case hasPasswords || len(breaches) > 5:
		return "medium"
	default:
		return "low"
	}
}

// PublishResults sends breach results to NATS and caches in ClickHouse.
func (h *HIBPClient) PublishResults(
	ctx context.Context,
	results []BreachResult,
) error {
	for _, r := range results {
		if r.RiskLevel == "none" {
			continue
		}

		severity := map[string]int{
			"low": 2, "medium": 3, "high": 4, "critical": 5,
		}[r.RiskLevel]

		event := map[string]interface{}{
			"class_uid":    2001,
			"activity_id":  1,
			"category_uid": 2,
			"severity_id":  severity,
			"time":         time.Now().UTC().Format(time.RFC3339),
			"finding_info": map[string]interface{}{
				"title": fmt.Sprintf("Email %s found in %d data breaches",
					r.Email, r.BreachCount),
				"uid":   fmt.Sprintf("hibp-%s", r.Email),
				"types": []string{"Credential Exposure"},
				"analytic": map[string]string{
					"name": "HIBP Breach Check",
					"type": "Threat Intelligence",
				},
			},
			"metadata": map[string]interface{}{
				"product":    map[string]string{"name": "HIBP", "vendor_name": "Troy Hunt"},
				"tenant_uid": h.tenantID,
			},
			"unmapped": map[string]interface{}{
				"risk_level":   r.RiskLevel,
				"breach_count": r.BreachCount,
			},
		}

		data, _ := json.Marshal(event)
		subject := fmt.Sprintf("kubric.ti.credential.%s", h.tenantID)
		if err := h.nc.Publish(subject, data); err != nil {
			return err
		}
	}
	return nil
}
```

---

## 3. ClickHouse Result Caching

```sql
-- ClickHouse table for caching HIBP results
CREATE TABLE IF NOT EXISTS kubric.hibp_breach_cache (
    email         String,
    tenant_id     String,
    breach_name   String,
    breach_domain String,
    breach_date   Date,
    pwn_count     UInt64,
    data_classes  Array(String),
    is_verified   UInt8,
    risk_level    LowCardinality(String),
    checked_at    DateTime64(3),
    INDEX idx_email email TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = ReplacingMergeTree(checked_at)
ORDER BY (tenant_id, email, breach_name)
TTL checked_at + INTERVAL 30 DAY;
```

---

## 4. KAI-SENTINEL Credential Risk Scoring

```go
// Integrate with KAI-SENTINEL for aggregate credential risk score.
// Published to kubric.kai.sentinel.credential.{tenant_id}
func computeCredentialRiskScore(results []BreachResult) float64 {
	if len(results) == 0 {
		return 0
	}

	var totalRisk float64
	riskWeights := map[string]float64{
		"critical": 1.0,
		"high":     0.75,
		"medium":   0.5,
		"low":      0.25,
		"none":     0.0,
	}

	for _, r := range results {
		totalRisk += riskWeights[r.RiskLevel]
	}

	// Normalize to 0-100 scale
	score := (totalRisk / float64(len(results))) * 100.0
	if score > 100 {
		score = 100
	}
	return score
}
```

---

## 5. Scheduled Checking

```go
// Run HIBP checks weekly on Sunday at 03:00 UTC
func scheduleHIBPChecks(h *HIBPClient, db *sql.DB) {
	ticker := time.NewTicker(24 * time.Hour)
	for t := range ticker.C {
        if t.Weekday() != time.Sunday {
            continue
        }

        ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)

        // Fetch employee emails from PostgreSQL
        rows, err := db.QueryContext(ctx,
            `SELECT email FROM customer_employees
             WHERE tenant_id = $1 AND active = true`,
            h.tenantID,
        )
        if err != nil {
            cancel()
            continue
        }

        var emails []string
        for rows.Next() {
            var email string
            rows.Scan(&email)
            emails = append(emails, email)
        }
        rows.Close()

        results, _ := h.CheckEmailBatch(ctx, emails)
        h.PublishResults(ctx, results)
        cancel()
    }
}
```
