package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/apps/api/internal/apierr"
	"github.com/sentinel-dev/sentinel/apps/api/internal/auth"
	"github.com/sentinel-dev/sentinel/apps/api/internal/cache"
	"github.com/sentinel-dev/sentinel/apps/api/internal/config"
	"github.com/sentinel-dev/sentinel/apps/api/internal/database"
	"github.com/sentinel-dev/sentinel/apps/api/internal/incidents"
	"github.com/sentinel-dev/sentinel/apps/api/internal/middleware"
	"github.com/sentinel-dev/sentinel/apps/api/internal/ops"
	"github.com/sentinel-dev/sentinel/apps/api/internal/paging"
	"github.com/sentinel-dev/sentinel/apps/api/internal/services"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

type Server struct {
	cfg       config.Config
	log       *slog.Logger
	db        *database.DB
	redis     *cache.Redis
	auth      *auth.Service
	catalog   *services.Service
	incidents *incidents.Service
	ops       *ops.Service
	http      *http.Server
}

func New(
	cfg config.Config,
	log *slog.Logger,
	db *database.DB,
	redis *cache.Redis,
	authSvc *auth.Service,
	catalog *services.Service,
	incidentsSvc *incidents.Service,
) *Server {
	s := &Server{
		cfg:       cfg,
		log:       log,
		db:        db,
		redis:     redis,
		auth:      authSvc,
		catalog:   catalog,
		incidents: incidentsSvc,
		ops:       ops.NewService(ops.NewRepository(db.Pool), ops.NewHTTPDecisionClient(cfg.RemediationURL)),
	}
	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           s.routes(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	return s
}

func (s *Server) Handler() http.Handler {
	return s.http.Handler
}

func (s *Server) HTTP() *http.Server {
	return s.http
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/me", s.me)
	mux.HandleFunc("GET /api/v1/services", s.listServices)
	mux.HandleFunc("GET /api/v1/services/{id}", s.getService)
	mux.HandleFunc("GET /api/v1/incidents", s.listIncidents)
	mux.HandleFunc("POST /api/v1/incidents", s.createIncident)
	mux.HandleFunc("GET /api/v1/incidents/{id}", s.getIncident)
	mux.HandleFunc("GET /api/v1/incidents/{id}/timeline", s.incidentTimeline)
	mux.HandleFunc("GET /api/v1/incidents/{id}/alerts", s.incidentAlerts)
	mux.HandleFunc("GET /api/v1/incidents/{id}/investigations", s.incidentInvestigations)
	mux.HandleFunc("GET /api/v1/incidents/{id}/recommendations", s.incidentRecommendations)
	mux.HandleFunc("GET /api/v1/incidents/{id}/remediations", s.incidentRemediations)
	mux.HandleFunc("GET /api/v1/investigations", s.listInvestigations)
	mux.HandleFunc("GET /api/v1/investigations/{id}", s.getInvestigation)
	mux.HandleFunc("GET /api/v1/recommendations/{id}", s.getRecommendation)
	mux.HandleFunc("GET /api/v1/remediations", s.listRemediations)
	mux.HandleFunc("GET /api/v1/remediations/{id}", s.getRemediation)
	mux.HandleFunc("POST /api/v1/remediations/{id}/approve", s.approveRemediation)
	mux.HandleFunc("POST /api/v1/remediations/{id}/reject", s.rejectRemediation)
	mux.HandleFunc("GET /api/v1/deployments", s.listDeployments)

	recoverer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					s.log.Error("panic", "request_id", middleware.RequestIDFrom(r.Context()), "panic", rec)
					writeError(w, r, s.log, apierr.Internal())
				}
			}()
			next.ServeHTTP(w, r)
		})
	}

	var h http.Handler = mux
	h = recoverer(h)
	h = middleware.Timeout(s.cfg.RequestTimeout, h)
	h = middleware.AccessLog(s.log, h)
	h = middleware.RequestID(h)
	h = telemetry.WrapHTTP(h)
	return h
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{}
	ready := true
	if err := s.db.Ping(ctx); err != nil {
		checks["postgres"] = "unavailable"
		ready = false
	} else {
		checks["postgres"] = "ok"
	}
	if s.redis.Enabled() {
		if err := s.redis.Ping(ctx); err != nil {
			checks["redis"] = "unavailable"
			ready = false
		} else {
			checks["redis"] = "ok"
		}
	}
	status := "ok"
	code := http.StatusOK
	if !ready {
		status = "unavailable"
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "checks": checks})
}

func (s *Server) currentUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	user, err := s.authenticate(r)
	if err != nil {
		writeError(w, r, s.log, err)
		return auth.User{}, false
	}
	return user, true
}

func (s *Server) authenticate(r *http.Request) (auth.User, error) {
	user, _, err := s.auth.Authenticate(r.Context(), sessionToken(r))
	return user, err
}

func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	return paging.RequireUUID(r.PathValue(name), name)
}
