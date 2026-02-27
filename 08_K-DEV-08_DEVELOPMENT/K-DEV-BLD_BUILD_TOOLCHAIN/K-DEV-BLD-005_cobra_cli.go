// Package dev provides CLI tooling for Kubric platform operators.
// File: K-DEV-BLD-005_cobra_cli.go
package dev

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Build-time variables injected via -ldflags.
var (
	Version   = "0.1.0"
	BuildDate = "unknown"
	GitSHA    = "unknown"
)

// CLI-wide flags.
var (
	flagTenantID  string
	flagOutputFmt string
	flagAPIURL    string
	flagTimeout   int
)

var rootCmd = &cobra.Command{
	Use:   "kubric",
	Short: "Kubric platform CLI",
	Long: `Kubric CLI — interact with the Kubric security platform.

Environment variables:
  KUBRIC_API_URL    Base URL of the Kubric API  (default: http://localhost:8080)
  KUBRIC_API_TOKEN  Bearer token for authentication
  KUBRIC_TENANT_ID  Default tenant UUID

Run 'kubric help <command>' for usage of any sub-command.`,
}

// ─── version ────────────────────────────────────────────────────────────────

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kubric CLI\n  version:    %s\n  build date: %s\n  git sha:    %s\n",
			Version, BuildDate, GitSHA)
	},
}

// ─── health ─────────────────────────────────────────────────────────────────

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check health of Kubric services",
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoints := []struct{ name, path string }{
			{"api-gateway", "/healthz"},
			{"kai-core", "/kai/healthz"},
			{"noc-api", "/noc/healthz"},
		}
		client := newHTTPClient()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "SERVICE\tSTATUS\tLATENCY")
		for _, ep := range endpoints {
			url := flagAPIURL + ep.path
			start := time.Now()
			resp, err := client.Get(url)
			latency := time.Since(start)
			if err != nil {
				fmt.Fprintf(w, "%s\tDOWN\t%s\n", ep.name, latency.Round(time.Millisecond))
				continue
			}
			resp.Body.Close()
			status := "OK"
			if resp.StatusCode != http.StatusOK {
				status = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", ep.name, status, latency.Round(time.Millisecond))
		}
		return w.Flush()
	},
}

// ─── tenants ─────────────────────────────────────────────────────────────────

var tenantsCmd = &cobra.Command{
	Use:   "tenants",
	Short: "Manage tenants",
}

var tenantsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newHTTPClient()
		req, err := newAuthRequest("GET", flagAPIURL+"/api/v1/admin/tenants", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
		}
		var result struct {
			Tenants []map[string]any `json:"tenants"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if flagOutputFmt == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result.Tenants)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPLAN\tSTATUS")
		for _, t := range result.Tenants {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				strVal(t, "id"), strVal(t, "name"),
				strVal(t, "plan"), strVal(t, "status"))
		}
		return w.Flush()
	},
}

// ─── alerts ──────────────────────────────────────────────────────────────────

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alerts",
}

var (
	alertSeverity string
	alertLimit    int
)

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alerts for a tenant",
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagTenantID == "" {
			return fmt.Errorf("--tenant / KUBRIC_TENANT_ID is required")
		}
		url := fmt.Sprintf("%s/api/v1/tenants/%s/alerts?limit=%d",
			flagAPIURL, flagTenantID, alertLimit)
		if alertSeverity != "" {
			url += "&severity=" + alertSeverity
		}
		client := newHTTPClient()
		req, err := newAuthRequest("GET", url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
		}
		var result struct {
			Alerts []map[string]any `json:"alerts"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if flagOutputFmt == "json" {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result.Alerts)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tSEVERITY\tTITLE\tSTATUS\tCREATED")
		for _, a := range result.Alerts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				strVal(a, "id"), strVal(a, "severity"),
				truncate(strVal(a, "title"), 60),
				strVal(a, "status"), strVal(a, "created_at"))
		}
		return w.Flush()
	},
}

// ─── init ────────────────────────────────────────────────────────────────────

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagTenantID, "tenant", "t",
		os.Getenv("KUBRIC_TENANT_ID"), "Tenant UUID ($KUBRIC_TENANT_ID)")
	rootCmd.PersistentFlags().StringVarP(&flagOutputFmt, "output", "o",
		"table", "Output format: table|json")
	rootCmd.PersistentFlags().StringVar(&flagAPIURL, "api-url",
		getEnv("KUBRIC_API_URL", "http://localhost:8080"), "API base URL ($KUBRIC_API_URL)")
	rootCmd.PersistentFlags().IntVar(&flagTimeout, "timeout", 30, "HTTP timeout in seconds")

	alertsListCmd.Flags().StringVar(&alertSeverity, "severity", "", "Filter by severity: critical|high|medium|low")
	alertsListCmd.Flags().IntVar(&alertLimit, "limit", 50, "Max results to return")

	alertsCmd.AddCommand(alertsListCmd)
	tenantsCmd.AddCommand(tenantsListCmd)

	rootCmd.AddCommand(versionCmd, healthCmd, tenantsCmd, alertsCmd)
}

// Execute is the CLI entry point — call from main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
}

func newAuthRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("KUBRIC_API_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	if flagTenantID != "" {
		req.Header.Set("X-Tenant-ID", flagTenantID)
	}
	return req, nil
}

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
