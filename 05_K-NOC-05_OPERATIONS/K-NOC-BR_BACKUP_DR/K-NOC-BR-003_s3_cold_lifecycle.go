// Package noc provides NOC operations tooling.
// K-NOC-BR-003 — S3 Cold Storage Lifecycle Manager: manage AWS S3 lifecycle rules
// to transition old backups to Glacier using the AWS SDK (minio-go is in go.mod; use
// net/http with plain S3 REST for lifecycle since minio-go does not expose lifecycle APIs).
package noc

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// StorageClass represents an S3 storage class identifier.
type StorageClass string

const (
	StorageClassGlacier            StorageClass = "GLACIER"
	StorageClassDeepArchive        StorageClass = "DEEP_ARCHIVE"
	StorageClassIntelligentTiering StorageClass = "INTELLIGENT_TIERING"
)

// Transition defines when objects transition to a given storage class.
type Transition struct {
	Days         int          `xml:"Days"`
	StorageClass StorageClass `xml:"StorageClass"`
}

// Expiration defines when objects expire.
type Expiration struct {
	Days int `xml:"Days"`
}

// LifecycleRule is a single S3 lifecycle configuration rule.
type LifecycleRule struct {
	ID          string       `xml:"ID"`
	Status      string       `xml:"Status"`
	Prefix      string       `xml:"Filter>Prefix,omitempty"`
	Transitions []Transition `xml:"Transition,omitempty"`
	Expiration  *Expiration  `xml:"Expiration,omitempty"`
}

// lifecycleConfig is the XML root element for S3 lifecycle API calls.
type lifecycleConfig struct {
	XMLName xml.Name        `xml:"LifecycleConfiguration"`
	Rules   []LifecycleRule `xml:"Rule"`
}

// S3LifecycleManager manages AWS S3 lifecycle rules via the S3 REST API with SigV4.
type S3LifecycleManager struct {
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	client    *http.Client
}

// NewS3LifecycleManager reads AWS credentials and bucket config from the environment.
func NewS3LifecycleManager() *S3LifecycleManager {
	return &S3LifecycleManager{
		Bucket:    os.Getenv("S3_BACKUP_BUCKET"),
		Region:    os.Getenv("AWS_REGION"),
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateDefaultRules returns a standard backup retention lifecycle policy.
func (m *S3LifecycleManager) CreateDefaultRules() []LifecycleRule {
	return []LifecycleRule{
		{
			ID:     "glacier-transition-90d",
			Status: "Enabled",
			Prefix: "backups/",
			Transitions: []Transition{
				{Days: 90, StorageClass: StorageClassGlacier},
			},
			Expiration: &Expiration{Days: 365},
		},
		{
			ID:     "deep-archive-transition-180d",
			Status: "Enabled",
			Prefix: "cold/",
			Transitions: []Transition{
				{Days: 180, StorageClass: StorageClassDeepArchive},
			},
			Expiration: &Expiration{Days: 730},
		},
	}
}

// PutLifecycleConfig uploads a lifecycle configuration to the S3 bucket.
func (m *S3LifecycleManager) PutLifecycleConfig(ctx context.Context, rules []LifecycleRule) error {
	cfg := lifecycleConfig{Rules: rules}
	body, err := xml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal lifecycle config: %w", err)
	}

	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/?lifecycle", m.Bucket, m.Region)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build PUT lifecycle request: %w", err)
	}
	req.Header.Set("Content-Type", "application/xml")

	if signErr := m.signRequest(req, body); signErr != nil {
		return fmt.Errorf("sign request: %w", signErr)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("PUT lifecycle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("PUT lifecycle returned %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// GetLifecycleConfig retrieves the lifecycle configuration from the S3 bucket.
func (m *S3LifecycleManager) GetLifecycleConfig(ctx context.Context) ([]LifecycleRule, error) {
	endpoint := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/?lifecycle", m.Bucket, m.Region)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build GET lifecycle request: %w", err)
	}
	if signErr := m.signRequest(req, nil); signErr != nil {
		return nil, fmt.Errorf("sign request: %w", signErr)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET lifecycle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("GET lifecycle returned %d: %s", resp.StatusCode, respBody)
	}

	var cfg lifecycleConfig
	if err := xml.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode lifecycle XML: %w", err)
	}
	return cfg.Rules, nil
}

// ---- AWS SigV4 signing helpers ----

func (m *S3LifecycleManager) signRequest(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	dateISO := now.Format("20060102T150405Z")
	dateShort := now.Format("20060102")

	if body == nil {
		body = []byte{}
	}
	payloadHash := sha256Hex(body)

	req.Header.Set("x-amz-date", dateISO)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Host", req.URL.Host)

	// Canonical request.
	canonicalHeaders, signedHeaders := buildCanonicalHeaders(req)
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}
	canonicalQuery := req.URL.RawQuery
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign.
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateShort, m.Region)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		dateISO,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// Signing key.
	signingKey := deriveSigningKey(m.SecretKey, dateShort, m.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		m.AccessKey, credentialScope, signedHeaders, signature,
	)
	req.Header.Set("Authorization", authHeader)
	return nil
}

func buildCanonicalHeaders(req *http.Request) (canonicalHeaders, signedHeaders string) {
	headers := make(map[string]string)
	for k, v := range req.Header {
		headers[strings.ToLower(k)] = strings.TrimSpace(v[0])
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var ch, sh strings.Builder
	for i, k := range keys {
		ch.WriteString(k)
		ch.WriteByte(':')
		ch.WriteString(headers[k])
		ch.WriteByte('\n')
		if i > 0 {
			sh.WriteByte(';')
		}
		sh.WriteString(k)
	}
	return ch.String(), sh.String()
}

func deriveSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
