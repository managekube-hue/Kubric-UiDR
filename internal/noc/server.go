package noc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/managekube-hue/Kubric-UiDR/internal/bloodhound"
	"github.com/managekube-hue/Kubric-UiDR/internal/correlation"
	"github.com/managekube-hue/Kubric-UiDR/internal/cortex"
	"github.com/managekube-hue/Kubric-UiDR/internal/falco"
	kubricmw "github.com/managekube-hue/Kubric-UiDR/internal/middleware"
	"github.com/managekube-hue/Kubric-UiDR/internal/osquery"
	"github.com/managekube-hue/Kubric-UiDR/internal/shuffle"
	"github.com/managekube-hue/Kubric-UiDR/internal/thehive"
	"github.com/managekube-hue/Kubric-UiDR/internal/velociraptor"
	"github.com/managekube-hue/Kubric-UiDR/internal/wazuh"

	"github.com/managekube-hue/Kubric-UiDR/internal/alerting"
	"github.com/managekube-hue/Kubric-UiDR/internal/messaging"
	kubricneo4j "github.com/managekube-hue/Kubric-UiDR/internal/neo4j"
	"github.com/managekube-hue/Kubric-UiDR/internal/storage"
)

// Server wires together the Chi router, NOCStore (pgx → Postgres), Publisher (NATS),
// and optional security-tool integration clients.
type Server struct {
	cfg               Config
	store             *NOCStore
	pub               *Publisher
	router            *chi.Mux
	integrations      *integrationHandler
	correlationEngine *correlation.Engine
	falcoReceiver     *falco.WebhookReceiver
	zmqPub            *messaging.Publisher  // ZMQ4 high-throughput fanout (optional)
	twilioAlerter     *alerting.TwilioAlerter // Twilio escalation (optional)
	graphStore        *kubricneo4j.GraphStore // Neo4j asset topology (optional)
	objectStore       *storage.ObjectStore    // MinIO evidence/SBOM/scan store (optional)
}

// NewServer initialises all NOC dependencies and returns a ready-to-run Server.
func NewServer(cfg Config) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store, err := NewNOCStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("noc store: %w", err)
	}

	pub, err := NewPublisher(cfg.NATSUrl)
	if err != nil {
		fmt.Printf("noc: warn — NATS unavailable (%v); NOC events will not be published\n", err)
		pub = nil
	}

	ih := initIntegrations(cfg)

	// Start the correlation engine as a background goroutine.  If NATS is
	// unavailable the NOC server continues without detection routes.
	var corrEngine *correlation.Engine
	if ce, err := correlation.New(cfg.NATSUrl, ih.thehive, ih.shuffle); err != nil {
		fmt.Printf("noc: warn — correlation engine init failed: %v\n", err)
	} else {
		corrEngine = ce
		go corrEngine.Start(context.Background())
	}

	falcoReceiver := falco.NewWebhookReceiver()
	if corrEngine != nil {
		falcoReceiver.OnAlert(func(a falco.Alert) {
			corrEngine.IngestFalcoAlert(a.Rule, a.Priority, a.Hostname, a.Output, a.OutputFields)
		})
	}

	s := &Server{
		cfg:               cfg,
		store:             store,
		pub:               pub,
		integrations:      ih,
		correlationEngine: corrEngine,
		falcoReceiver:     falcoReceiver,
	}

	// ── ZMQ fanout (optional) ────────────────────────────────────────────
	if cfg.ZMQPublishAddr != "" {
		zmqPub, err := messaging.NewPublisher(context.Background(), cfg.ZMQPublishAddr)
		if err != nil {
			fmt.Printf("noc: warn — ZMQ publisher init failed: %v\n", err)
		} else {
			s.zmqPub = zmqPub
		}
	}

	// ── Twilio alerter (optional) ────────────────────────────────────────
	if cfg.TwilioAccountSID != "" {
		s.twilioAlerter = alerting.NewTwilioAlerter(
			cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber,
		)
	}

	// ── Neo4j graph store (optional) ─────────────────────────────────────
	if gs, err := kubricneo4j.New(cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword); err != nil {
		fmt.Printf("noc: warn — Neo4j graph store init failed: %v\n", err)
	} else if gs != nil {
		s.graphStore = gs
		if err := gs.EnsureConstraints(context.Background()); err != nil {
			fmt.Printf("noc: warn — Neo4j constraints: %v\n", err)
		}
	}

	// ── MinIO object store (optional) ────────────────────────────────────
	if os, err := storage.New(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, false); err != nil {
		fmt.Printf("noc: warn — MinIO init failed: %v\n", err)
	} else if os != nil {
		s.objectStore = os
		if err := os.EnsureDefaultBuckets(context.Background()); err != nil {
			fmt.Printf("noc: warn — MinIO default buckets: %v\n", err)
		}
	}

	s.router = s.buildRouter()
	return s, nil
}

// initIntegrations creates all optional integration clients.  Each client
// constructor returns (nil, nil) when its URL is empty, so no error handling
// is needed for disabled integrations.  Errors from enabled-but-misconfigured
// integrations are logged as warnings but do not prevent the NOC from starting.
func initIntegrations(cfg Config) *integrationHandler {
	ih := &integrationHandler{}

	if c, err := wazuh.New(cfg.WazuhURL, cfg.WazuhUser, cfg.WazuhPassword); err != nil {
		fmt.Printf("noc: warn — wazuh client init failed: %v\n", err)
	} else {
		ih.wazuh = c
	}

	if c, err := velociraptor.New(cfg.VelociraptorURL, cfg.VelociraptorAPIKey); err != nil {
		fmt.Printf("noc: warn — velociraptor client init failed: %v\n", err)
	} else {
		ih.velociraptor = c
	}

	if c, err := thehive.New(cfg.TheHiveURL, cfg.TheHiveAPIKey); err != nil {
		fmt.Printf("noc: warn — thehive client init failed: %v\n", err)
	} else {
		ih.thehive = c
	}

	if c, err := cortex.New(cfg.CortexURL, cfg.CortexAPIKey); err != nil {
		fmt.Printf("noc: warn — cortex client init failed: %v\n", err)
	} else {
		ih.cortex = c
	}

	if c, err := falco.New(cfg.FalcoURL); err != nil {
		fmt.Printf("noc: warn — falco client init failed: %v\n", err)
	} else {
		ih.falco = c
	}

	if c, err := osquery.New(cfg.OsqueryURL, cfg.OsqueryAPIKey); err != nil {
		fmt.Printf("noc: warn — osquery client init failed: %v\n", err)
	} else {
		ih.osquery = c
	}

	if c, err := shuffle.New(cfg.ShuffleURL, cfg.ShuffleAPIKey); err != nil {
		fmt.Printf("noc: warn — shuffle client init failed: %v\n", err)
	} else {
		ih.shuffle = c
	}

	if c, err := bloodhound.New(cfg.BloodHoundURL, cfg.BloodHoundTokenID, cfg.BloodHoundTokenKey); err != nil {
		fmt.Printf("noc: warn — bloodhound client init failed: %v\n", err)
	} else {
		ih.bloodhound = c
	}

	return ih
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
		s.closeIntegrations()
		if s.zmqPub != nil {
			_ = s.zmqPub.Close()
		}
		s.graphStore.Close()
		s.objectStore.Close()
		return nil
	case err := <-errCh:
		return err
	}
}

// closeIntegrations releases resources held by all integration clients.
func (s *Server) closeIntegrations() {
	if s.integrations == nil {
		return
	}
	s.integrations.wazuh.Close()
	s.integrations.velociraptor.Close()
	s.integrations.thehive.Close()
	s.integrations.cortex.Close()
	s.integrations.falco.Close()
	s.integrations.osquery.Close()
	s.integrations.shuffle.Close()
	s.integrations.bloodhound.Close()
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(25 * time.Second))
	r.Use(kubricmw.RateLimit)

	// Webhook routes bypass JWT — authenticated by shared secret header instead.
	webhookSecret := os.Getenv("KUBRIC_WEBHOOK_SECRET")
	r.Group(func(r chi.Router) {
		r.Use(webhookSecretAuth(webhookSecret))
		r.Post("/webhooks/falco", s.falcoReceiver.ServeHTTP)
	})

	// Health probes — no auth, must be reachable by Kubernetes kubelet.
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

	// All authenticated routes require a valid JWT.
	r.Group(func(r chi.Router) {
		r.Use(kubricmw.JWTAuth())
		r.Use(kubricmw.TenantContext)

		ch := newClusterHandler(s.store, s.pub)
		r.Route("/clusters", func(r chi.Router) {
			// Read: admin, analyst, readonly
			r.Group(func(r chi.Router) {
				r.Use(kubricmw.RequireAnyRole("kubric:analyst", "kubric:readonly"))
				r.Get("/", ch.list)
				r.Get("/{clusterID}", ch.get)
			})
			// Write: admin only (cluster lifecycle)
			r.Group(func(r chi.Router) {
				r.Use(kubricmw.RequireRole("kubric:admin"))
				r.Post("/", ch.create)
				r.Patch("/{clusterID}", ch.update)
				r.Delete("/{clusterID}", ch.delete)
			})
		})

		ah := newAgentHandler(s.store, s.pub)
		r.Route("/agents", func(r chi.Router) {
			// Heartbeat: agent role (CoreSec/NetGuard agents call this)
			r.With(kubricmw.RequireRole("kubric:agent")).Post("/heartbeat", ah.heartbeat)
			// Read: admin, analyst, readonly
			r.Group(func(r chi.Router) {
				r.Use(kubricmw.RequireAnyRole("kubric:analyst", "kubric:readonly"))
				r.Get("/", ah.list)
				r.Get("/{agentID}", ah.get)
			})
			// Write: admin and analyst (status updates, tag mutations)
			r.Group(func(r chi.Router) {
				r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
				r.Patch("/{agentID}", ah.update)
			})
		})

		// Integration routes: admin and analyst can read and write
		r.Group(func(r chi.Router) {
			r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
			s.integrations.RegisterRoutes(r)
		})

		// Detection routes: backed by the correlation engine.
		// Only mounted when the engine initialised successfully.
		if s.correlationEngine != nil {
			r.Mount("/detection", s.detectionRoutes())
		}

		// ── Graph topology routes (Neo4j) ─────────────────────────────
		if s.graphStore != nil {
			gh := &graphHandler{graph: s.graphStore}
			r.Route("/graph", func(r chi.Router) {
				r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
				r.Post("/assets", gh.upsertAsset)
				r.Post("/relationships", gh.upsertRelationship)
				r.Get("/topology", gh.topology)
				r.Get("/blast-radius", gh.blastRadius)
			})
		}

		// ── Object storage routes (MinIO) ─────────────────────────────
		if s.objectStore != nil {
			sh := &storageHandler{store: s.objectStore}
			r.Route("/storage", func(r chi.Router) {
				r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
				r.Get("/buckets", sh.bucketStats)
				r.Get("/{bucket}", sh.list)
				r.Put("/{bucket}/{key}", sh.upload)
				r.Get("/{bucket}/{key}", sh.download)
				r.Get("/{bucket}/{key}/presign", sh.presign)
				r.Delete("/{bucket}/{key}", sh.deleteObj)
			})
		}
	})

	return r
}

// writeJSON and writeError are shared by all NOC handlers via this package.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// webhookSecretAuth returns a middleware that validates the X-Kubric-Webhook-Secret header.
// When secret is empty the middleware is a no-op (useful in local dev without a secret configured).
func webhookSecretAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret != "" && r.Header.Get("X-Kubric-Webhook-Secret") != secret {
				http.Error(w, `{"error":"invalid webhook secret"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// detectionRoutes builds a chi.Router for the /detection sub-tree.
// Routes are relative to the mount point (/detection) so they do NOT repeat
// the /detection prefix.  RBAC mirrors the roles used in handler_detection.go.
func (s *Server) detectionRoutes() chi.Router {
	dh := NewDetectionHandler(s.correlationEngine)
	r := chi.NewRouter()

	// Read endpoints — analyst and readonly roles.
	r.Group(func(r chi.Router) {
		r.Use(kubricmw.RequireAnyRole("kubric:analyst", "kubric:readonly"))
		r.Get("/incidents", dh.listIncidents)
		r.Get("/incidents/{id}", dh.getIncident)
		r.Get("/timeline", dh.listTimeline)
		r.Get("/health", dh.engineHealth)
	})

	// Write endpoints — admin and analyst roles only.
	r.Group(func(r chi.Router) {
		r.Use(kubricmw.RequireAnyRole("kubric:admin", "kubric:analyst"))
		r.Patch("/incidents/{id}", dh.patchIncident)
		r.Post("/incidents/{id}/dispatch", dh.dispatchIncident)
	})

	return r
}
