package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/rules"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

type readiness interface {
	Ready(ctx context.Context) error
}

type dbPing interface {
	Ping(ctx context.Context) error
}

type Server struct {
	cfg  config.Config
	log  *slog.Logger
	db   dbPing
	k    readiness
	http *http.Server
}

func New(cfg config.Config, log *slog.Logger, db dbPing, k readiness) *Server {
	s := &Server{cfg: cfg, log: log, db: db, k: k}
	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           s.routes(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) HTTP() *http.Server { return s.http }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/v1/rules", s.rules)
	return telemetry.WrapHTTP(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	checks := map[string]string{}
	ok := true
	if s.db != nil {
		if err := s.db.Ping(ctx); err != nil {
			checks["postgres"] = "unavailable"
			ok = false
		} else {
			checks["postgres"] = "ok"
		}
	}
	if s.k != nil {
		if err := s.k.Ready(ctx); err != nil {
			checks["kafka"] = "unavailable"
			ok = false
		} else {
			checks["kafka"] = "ok"
		}
	}
	status, code := "ok", http.StatusOK
	if !ok {
		status, code = "unavailable", http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"status": status, "checks": checks})
}

func (s *Server) rules(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"consumer_group":   config.ConsumerGroup,
			"rules":            rules.Catalog(s.cfg.Rules),
			"cooldown_seconds": int(s.cfg.Rules.AlertCooldown.Seconds()),
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
