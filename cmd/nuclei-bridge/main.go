// cmd/nuclei-bridge — Kubric Nuclei subprocess bridge.
//
// Runs the nuclei binary as a subprocess (NOT imported as a Go library) to
// maintain the Apache 2.0 licence boundary.  Accepts scan targets via NATS
// subject kubric.{tenant_id}.scanner.nuclei.run and publishes findings back
// via the VDR HTTP API.
//
// Environment variables
//
//	KUBRIC_TENANT_ID   — tenant that owns this scanner instance
//	KUBRIC_NATS_URL    — NATS server URL (default nats://127.0.0.1:4222)
//	KUBRIC_VDR_URL     — VDR API base URL (default http://127.0.0.1:8081)
//	NUCLEI_BIN         — path to nuclei binary (default: "nuclei" on $PATH)
//	NUCLEI_TEMPLATES   — path to templates dir (default: "vendor/nuclei-templates")
//	NUCLEI_SEVERITY    — comma-separated severities to scan (default: "critical,high")
//	KUBRIC_LOG         — log level/filter
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	tenantID := mustEnv("KUBRIC_TENANT_ID")
	natsURL   := getenv("KUBRIC_NATS_URL", "nats://127.0.0.1:4222")
	vdrURL    := getenv("KUBRIC_VDR_URL", "http://127.0.0.1:8081")
	nucleiBin := getenv("NUCLEI_BIN", "nuclei")
	templates := getenv("NUCLEI_TEMPLATES", "vendor/nuclei-templates")
	severity  := getenv("NUCLEI_SEVERITY", "critical,high")

	nc, err := nats.Connect(natsURL)
	if err != nil {
		fatalf("NATS connect: %v", err)
	}
	defer nc.Drain()

	subject := fmt.Sprintf("kubric.%s.scanner.nuclei.run", tenantID)
	printf("nuclei-bridge: subscribing to %s", subject)

	_, err = nc.Subscribe(subject, func(msg *nats.Msg) {
		var req ScanRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			printf("nuclei-bridge: bad scan request: %v", err)
			return
		}
		if req.Target == "" {
			printf("nuclei-bridge: empty target — ignoring")
			return
		}
		printf("nuclei-bridge: scanning target=%s scan_id=%s", req.Target, req.ScanID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		findings, err := runNuclei(ctx, nucleiBin, templates, severity, req.Target)
		if err != nil {
			printf("nuclei-bridge: scan error target=%s: %v", req.Target, err)
			return
		}
		printf("nuclei-bridge: scan complete target=%s findings=%d", req.Target, len(findings))

		for _, f := range findings {
			if err := postFinding(vdrURL, tenantID, req.Target, f); err != nil {
				printf("nuclei-bridge: VDR post failed: %v", err)
			}
		}
	})
	if err != nil {
		fatalf("NATS subscribe: %v", err)
	}

	printf("nuclei-bridge: ready (tenant=%s)", tenantID)

	// Block until SIGINT/SIGTERM (simplified: read from /dev/null)
	sigCh := make(chan os.Signal, 1)
	<-sigCh
}

// ── Types ─────────────────────────────────────────────────────────────────────

// ScanRequest is published to kubric.{tenant_id}.scanner.nuclei.run.
type ScanRequest struct {
	ScanID   string `json:"scan_id"`
	Target   string `json:"target"`   // URL, IP, CIDR
	TenantID string `json:"tenant_id"`
}

// NucleiResult is one line of nuclei's JSONL output.
type NucleiResult struct {
	TemplateID  string `json:"template-id"`
	Info        struct {
		Name     string   `json:"name"`
		Severity string   `json:"severity"`
		Tags     []string `json:"tags"`
		Reference []string `json:"reference"`
	} `json:"info"`
	Host        string   `json:"host"`
	MatchedAt   string   `json:"matched-at"`
	Description string   `json:"description"`
	CVEID       string   `json:"cveid"` // populated by nuclei when template has cve-id tag
}

// ── nuclei subprocess ─────────────────────────────────────────────────────────

func runNuclei(ctx context.Context, bin, templates, severity, target string) ([]NucleiResult, error) {
	args := []string{
		"-target", target,
		"-t", templates,
		"-severity", severity,
		"-json",
		"-silent",
		"-no-color",
		"-timeout", "5",
		"-retries", "1",
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("nuclei start: %w", err)
	}

	var results []NucleiResult
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r NucleiResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			printf("nuclei-bridge: parse error: %v — raw: %s", err, line)
			continue
		}
		results = append(results, r)
	}

	if err := cmd.Wait(); err != nil {
		// nuclei exits 1 when it finds vulnerabilities — that is normal
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return results, nil
		}
		return results, fmt.Errorf("nuclei exit: %w", err)
	}
	return results, nil
}

// ── VDR posting ───────────────────────────────────────────────────────────────

func postFinding(vdrURL, tenantID, target string, r NucleiResult) error {
	// Map nuclei severity to VDR severity
	severity := strings.ToLower(r.Info.Severity)
	switch severity {
	case "critical", "high", "medium", "low":
		// valid
	default:
		severity = "informational"
	}

	// Extract CVE ID from tags if not directly set
	cveID := r.CVEID
	if cveID == "" {
		for _, tag := range r.Info.Tags {
			if strings.HasPrefix(strings.ToUpper(tag), "CVE-") {
				cveID = strings.ToUpper(tag)
				break
			}
		}
	}

	rawJSON, _ := json.Marshal(r)

	body := map[string]string{
		"tenant_id":   tenantID,
		"target":      target,
		"scanner":     "nuclei",
		"severity":    severity,
		"cve_id":      cveID,
		"title":       r.Info.Name,
		"description": r.Description,
		"raw_json":    string(rawJSON),
	}
	payload, _ := json.Marshal(body)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vdrURL+"/findings", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("VDR %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		fatalf("%s must be set", key)
	}
	return v
}

func printf(format string, args ...any) {
	fmt.Fprintf(os.Stdout, "[%s] "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
