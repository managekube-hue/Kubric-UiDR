package kai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
)

// Server is the KAI HTTP gateway. It validates JWT auth and tenant context
// for all /kai/* routes, then reverse-proxies to the Python KAI FastAPI service.
type Server struct {
	cfg    Config
	proxy  *httputil.ReverseProxy
	router *chi.Mux
}

// NewServer parses configuration, constructs the reverse proxy, and wires the
// Chi router. No external I/O is performed beyond URL parsing.
func NewServer(cfg Config) (*Server, error) {
	target, err := url.Parse(cfg.KAIServiceURL)
	if err != nil {
		return nil, fmt.Errorf("kai: invalid KAI_SERVICE_URL %q: %w", cfg.KAIServiceURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// FlushInterval=-1 enables immediate flushing for streaming endpoints
	// (e.g. /kai/chat with SSE or chunked responses).
	proxy.FlushInterval = -1

	s := &Server{
		cfg:   cfg,
		proxy: proxy,
	}
	s.router = s.buildRouter()
	return s, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled.
// It performs a graceful shutdown with a 10-second deadline on cancellation.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      120 * time.Second, // long timeout for streaming /kai/chat
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}

// buildRouter wires all Chi routes according to the KAI OpenAPI spec.
func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health probes — unauthenticated, no tenant required, respond immediately.
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check that the Python KAI service is reachable.
		checkCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, s.cfg.KAIServiceURL+"/healthz", nil)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "kai-python unreachable",
				"error":  err.Error(),
			})
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "kai-python unreachable",
				"error":  err.Error(),
			})
			return
		}
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "kai-python unhealthy",
				"code":   fmt.Sprintf("%d", resp.StatusCode),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// All other routes require a valid JWT and a tenant context.
	r.Group(func(r chi.Router) {
		r.Use(kubricmw.JWTAuth())
		r.Use(kubricmw.TenantContext)

		// Direct non-proxy route: NATS consumer / agent health.
		r.Get("/kai/health/agent", s.handleAgentHealth)

		// Proxy routes: all require X-Kubric-Tenant-Id header.
		r.Group(func(r chi.Router) {
			r.Use(requireTenant)

			r.Post("/kai/analyze", s.proxyHandler())
			r.Post("/kai/investigate", s.proxyHandler())
			r.Post("/kai/remediate", s.proxyHandler())
			r.Post("/kai/chat", s.proxyHandler()) // may be streaming (SSE / chunked)

			r.Get("/kai/reports", s.proxyHandler())
			r.Post("/kai/reports", s.proxyHandler())
			r.Get("/kai/reports/{id}", s.proxyHandler())

			r.Get("/kai/operations/{id}", s.proxyHandler())

			r.Get("/kai/workflows", s.proxyHandler())
			r.Post("/kai/workflows", s.proxyHandler())
			r.Post("/kai/workflows/{id}/execute", s.proxyHandler())
		})
	})

	return r
}

// requireTenant is an inline middleware that enforces the presence of a tenant
// ID in context (set by kubricmw.TenantContext from X-Kubric-Tenant-Id).
// Returns HTTP 400 when the header is absent or invalid.
func requireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if kubricmw.TenantID(r.Context()) == "" {
			writeError(w, http.StatusBadRequest, "missing required header: X-Kubric-Tenant-Id")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// proxyHandler returns an http.HandlerFunc that forwards the request to the
// Python KAI FastAPI service, preserving all headers, query parameters, and body.
// The httputil.ReverseProxy Director rewrites only the scheme and host; the
// original path and query string are preserved so every route maps 1:1.
func (s *Server) proxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.proxy.ServeHTTP(w, r)
	}
}

// handleAgentHealth is a direct (non-proxy) route that reports the gateway's
// connectivity status. The actual NATS consumers live in the Python KAI service;
// this endpoint surfaces whether the gateway can reach that service and
// acknowledges the configured NATS URL so operators can verify wiring.
func (s *Server) handleAgentHealth(w http.ResponseWriter, r *http.Request) {
	checkCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	kaiReachable := true
	kaiErr := ""
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, s.cfg.KAIServiceURL+"/healthz", nil)
	if err != nil {
		kaiReachable = false
		kaiErr = err.Error()
	} else {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			kaiReachable = false
			kaiErr = err.Error()
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 500 {
				kaiReachable = false
				kaiErr = fmt.Sprintf("HTTP %d from kai-python", resp.StatusCode)
			}
		}
	}

	status := "ok"
	httpStatus := http.StatusOK
	if !kaiReachable {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, map[string]any{
		"status":         status,
		"gateway":        "kai-go",
		"kai_python_url": s.cfg.KAIServiceURL,
		"kai_reachable":  kaiReachable,
		"kai_error":      kaiErr,
		"nats_url":       s.cfg.NATSUrl,
		"checked_at":     time.Now().UTC().Format(time.RFC3339),
	})
}

// writeJSON serialises v as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
