package kic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
)

// Server wires together the Chi router, AssessmentStore (pgx → Postgres), and Publisher (NATS).
type Server struct {
	cfg    Config
	store  *AssessmentStore
	pub    *Publisher
	router *chi.Mux
}

// NewServer initialises all KIC dependencies and returns a ready-to-run Server.
func NewServer(cfg Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := NewAssessmentStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("assessment store: %w", err)
	}

	pub, err := NewPublisher(cfg.NATSUrl)
	if err != nil {
		fmt.Printf("kic: warn — NATS unavailable (%v); compliance events will not be published\n", err)
		pub = nil
	}

	s := &Server{cfg: cfg, store: store, pub: pub}
	s.router = s.buildRouter()
	return s, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled.
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

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(25 * time.Second))
	r.Use(kubricmw.RateLimit)
	r.Use(kubricmw.JWTAuth())
	r.Use(kubricmw.TenantContext)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.store.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "postgres unavailable", "error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	ah := newAssessmentHandler(s.store, s.pub)
	r.Route("/assessments", func(r chi.Router) {
		r.Get("/", ah.list)
		r.Post("/", ah.create)
		r.Get("/{assessmentID}", ah.get)
		r.Patch("/{assessmentID}", ah.updateStatus)
	})

	return r
}
