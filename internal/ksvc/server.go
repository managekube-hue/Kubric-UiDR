package ksvc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
	"github.com/managekube-hue/Kubric-UiDR/services/k-svc/billing"
)

// Server wires together the Chi router, TenantStore (pgx → Postgres),
// and Publisher (NATS) into a runnable HTTP service.
type Server struct {
	cfg    Config
	store  *TenantStore
	pub    *Publisher
	router *chi.Mux
}

// NewServer initialises all K-SVC dependencies (Postgres pool, NATS connection)
// and returns a Server ready to accept HTTP requests.
//
// NATS failure is non-fatal — the server starts and tenant events are silently
// dropped until NATS becomes available on the next restart.
func NewServer(cfg Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := NewTenantStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("tenant store: %w", err)
	}

	pub, err := NewPublisher(cfg.NATSUrl)
	if err != nil {
		fmt.Printf("ksvc: warn — NATS unavailable (%v); tenant lifecycle events will not be published\n", err)
		pub = nil
	}

	if cfg.StripeAPIKey != "" {
		billing.Init(cfg.StripeAPIKey)
	}

	s := &Server{cfg: cfg, store: store, pub: pub}
	s.router = s.buildRouter()
	return s, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled or a fatal error occurs.
// On shutdown it drains in-flight requests (up to 10 s), closes the NATS connection,
// and releases the Postgres pool.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
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
		s.store.Close()
		s.pub.Close()
		return nil
	case err := <-errCh:
		return err
	}
}

// buildRouter configures the Chi mux with middleware and all K-SVC routes.
func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	// Standard middleware stack
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(25 * time.Second))
	r.Use(kubricmw.RateLimit)
	r.Use(kubricmw.JWTAuth())
	r.Use(kubricmw.TenantContext)

	// Health probes — no auth required, used by K8s liveness/readiness probes
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.store.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "postgres unavailable",
				"error":  err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Tenant CRUD — RBAC enforced per method group
	th := newTenantHandler(s.store, s.pub)
	r.Route("/tenants", func(r chi.Router) {
		// Read: admin, analyst, readonly
		r.Group(func(r chi.Router) {
			r.Use(kubricmw.RequireAnyRole("kubric:analyst", "kubric:readonly"))
			r.Get("/", th.list)
			r.Get("/{tenantID}", th.get)
		})
		// Write: admin only
		r.Group(func(r chi.Router) {
			r.Use(kubricmw.RequireRole("kubric:admin"))
			r.Post("/", th.create)
			r.Patch("/{tenantID}", th.update)
			r.Delete("/{tenantID}", th.delete)
		})
	})

	// Billing sub-routes — mounted under /tenants/{tenantID} as a separate
	// route prefix so Chi's trie correctly distinguishes /tenants/{id} (tenant
	// read) from /tenants/{id}/subscription and /tenants/{id}/billing/portal.
	bh := newBillingHandler(s.store, s.cfg)
	r.Route("/tenants/{tenantID}", func(r chi.Router) {
		r.With(kubricmw.RequireRole("kubric:admin")).Post("/subscription", bh.createSubscription)
		r.With(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst")).Get("/billing/portal", bh.getBillingPortal)
	})

	return r
}
