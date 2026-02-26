// cmd/correlation — standalone Kubric correlation engine service.
//
// The correlation engine ingests events from all Kubric agents and third-party
// security tools via NATS JetStream, evaluates built-in correlation rules, and
// dispatches incidents to TheHive and Shuffle.
//
// # Configuration (all from environment variables)
//
//	KUBRIC_NATS_URL       NATS server URL           (default: nats://localhost:4222)
//	THEHIVE_URL           TheHive 5.x base URL       (default: empty → disabled)
//	THEHIVE_API_KEY       TheHive API key            (default: "")
//	SHUFFLE_URL           Shuffle base URL           (default: empty → disabled)
//	SHUFFLE_API_KEY       Shuffle API key            (default: "")
//	CORRELATION_LISTEN    health check HTTP addr     (default: :9099)
//	CORRELATION_LOG_LEVEL log level (debug/info)     (default: info)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/managekube-hue/Kubric-UiDR/internal/correlation"
	"github.com/managekube-hue/Kubric-UiDR/internal/shuffle"
	"github.com/managekube-hue/Kubric-UiDR/internal/thehive"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type config struct {
	NATSUrl       string
	TheHiveURL    string
	TheHiveAPIKey string
	ShuffleURL    string
	ShuffleAPIKey string
	ListenAddr    string
}

func loadConfig() config {
	return config{
		NATSUrl:       getenv("KUBRIC_NATS_URL", "nats://localhost:4222"),
		TheHiveURL:    getenv("THEHIVE_URL", ""),
		TheHiveAPIKey: getenv("THEHIVE_API_KEY", ""),
		ShuffleURL:    getenv("SHUFFLE_URL", ""),
		ShuffleAPIKey: getenv("SHUFFLE_API_KEY", ""),
		ListenAddr:    getenv("CORRELATION_LISTEN", ":9099"),
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("correlation: starting Kubric correlation engine")

	cfg := loadConfig()

	// ---------------------------------------------------------------------------
	// Optional integration clients
	// ---------------------------------------------------------------------------

	var th *thehive.Client
	if cfg.TheHiveURL != "" {
		c, err := thehive.New(cfg.TheHiveURL, cfg.TheHiveAPIKey)
		if err != nil {
			log.Printf("correlation: warn — TheHive client init failed: %v", err)
		} else {
			th = c
			log.Printf("correlation: TheHive integration enabled (%s)", cfg.TheHiveURL)
		}
	} else {
		log.Println("correlation: TheHive integration disabled (THEHIVE_URL not set)")
	}

	var sh *shuffle.Client
	if cfg.ShuffleURL != "" {
		c, err := shuffle.New(cfg.ShuffleURL, cfg.ShuffleAPIKey)
		if err != nil {
			log.Printf("correlation: warn — Shuffle client init failed: %v", err)
		} else {
			sh = c
			log.Printf("correlation: Shuffle integration enabled (%s)", cfg.ShuffleURL)
		}
	} else {
		log.Println("correlation: Shuffle integration disabled (SHUFFLE_URL not set)")
	}

	// ---------------------------------------------------------------------------
	// Build the correlation engine
	// ---------------------------------------------------------------------------

	engine, err := correlation.New(cfg.NATSUrl, th, sh)
	if err != nil {
		fmt.Fprintf(os.Stderr, "correlation: engine init failed: %v\n", err)
		os.Exit(1)
	}
	log.Printf("correlation: engine created; NATS=%s", cfg.NATSUrl)

	// Print the loaded rule set summary.
	rules := correlation.DefaultRules()
	log.Printf("correlation: %d built-in correlation rules loaded:", len(rules))
	for _, r := range rules {
		log.Printf("  [%s] %s (severity=%d, window=%s, MITRE=%s/%s)",
			r.ID, r.Name, r.Severity, r.Window, r.MITRETactic, r.MITRETechnique)
	}

	// ---------------------------------------------------------------------------
	// Context / signal handling
	// ---------------------------------------------------------------------------

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ---------------------------------------------------------------------------
	// Health check HTTP server
	// ---------------------------------------------------------------------------

	healthSrv := startHealthServer(cfg.ListenAddr, engine)
	log.Printf("correlation: health check HTTP server listening on %s", cfg.ListenAddr)

	// ---------------------------------------------------------------------------
	// Run the engine (blocks until ctx is cancelled)
	// ---------------------------------------------------------------------------

	engineErrCh := make(chan error, 1)
	go func() {
		if err := engine.Start(ctx); err != nil {
			engineErrCh <- err
		}
	}()

	// ---------------------------------------------------------------------------
	// Wait for shutdown
	// ---------------------------------------------------------------------------

	select {
	case err := <-engineErrCh:
		fmt.Fprintf(os.Stderr, "correlation: engine error: %v\n", err)
		stop()
	case <-ctx.Done():
		log.Println("correlation: shutdown signal received")
	}

	// Graceful shutdown of the health HTTP server.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("correlation: health server shutdown: %v", err)
	}

	// Clean up integration clients.
	if th != nil {
		th.Close()
	}
	if sh != nil {
		sh.Close()
	}

	log.Println("correlation: stopped")
}

// ---------------------------------------------------------------------------
// Health check HTTP server
// ---------------------------------------------------------------------------

// startHealthServer spawns a minimal HTTP server on addr with two endpoints:
//
//	GET /healthz — liveness probe (always 200)
//	GET /readyz  — current engine metrics as JSON
func startHealthServer(addr string, engine *correlation.Engine) *http.Server {
	mux := http.NewServeMux()

	// Liveness probe — always returns 200 once the service is up.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Readiness probe — returns 200 with current metrics.
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		metrics := engine.Metrics()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"metrics": metrics,
		})
	})

	// Metrics endpoint — Prometheus-compatible text format (counter values only).
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics := engine.Metrics()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		for name, val := range metrics {
			metricName := "kubric_correlation_" + strings.ReplaceAll(name, "-", "_")
			_, _ = fmt.Fprintf(w, "# TYPE %s counter\n%s %d\n", metricName, metricName, val)
		}
	})

	// Rules summary — returns the active rule set as JSON.
	mux.HandleFunc("/rules", func(w http.ResponseWriter, r *http.Request) {
		type ruleSummary struct {
			ID             string        `json:"id"`
			Name           string        `json:"name"`
			Severity       int           `json:"severity"`
			Window         string        `json:"window"`
			MITRETactic    string        `json:"mitre_tactic"`
			MITRETechnique string        `json:"mitre_technique"`
			Threshold      int           `json:"threshold,omitempty"`
			Conditions     int           `json:"condition_count"`
		}
		var summaries []ruleSummary
		for _, rule := range correlation.DefaultRules() {
			summaries = append(summaries, ruleSummary{
				ID:             rule.ID,
				Name:           rule.Name,
				Severity:       rule.Severity,
				Window:         rule.Window.String(),
				MITRETactic:    rule.MITRETactic,
				MITRETechnique: rule.MITRETechnique,
				Threshold:      rule.Threshold,
				Conditions:     len(rule.Conditions),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(summaries)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("correlation: health HTTP server error: %v", err)
		}
	}()

	return srv
}
