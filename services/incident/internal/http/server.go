package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/services/incident/internal/config"
	"github.com/sentinel-dev/sentinel/services/incident/internal/database"
)

type readiness interface {
	Ready(ctx context.Context) error
}

type Server struct {
	cfg   config.Config
	log   *slog.Logger
	store *database.Store
	db    interface{ Ping(ctx context.Context) error }
	k     readiness
	http  *http.Server
}

func New(cfg config.Config, log *slog.Logger, store *database.Store, db interface{ Ping(ctx context.Context) error }, k readiness) *Server {
	s := &Server{cfg: cfg, log: log, store: store, db: db, k: k}
	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           s.routes(),
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) HTTP() *http.Server { return s.http }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/v1/incidents", s.listIncidents)
	mux.HandleFunc("GET /api/v1/incidents/{id}", s.getIncident)
	mux.HandleFunc("GET /api/v1/incidents/{id}/timeline", s.timeline)
	return mux
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

func (s *Server) listIncidents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.store.ListIncidents(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, item := range items {
		data = append(data, listDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	item, err := s.store.GetIncident(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": detailDTO(item)})
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, err := s.store.GetIncident(r.Context(), id); errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get failed"})
		return
	}
	items, err := s.store.Timeline(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "timeline failed"})
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, ev := range items {
		src, _ := ev.Payload["source"].(string)
		if src == "" {
			src = "system"
		}
		summary, _ := ev.Payload["summary"].(string)
		if summary == "" {
			switch ev.Kind {
			case "created":
				summary = "Incident opened"
			case "alert_attached":
				summary = "Alert attached"
			default:
				summary = ev.Kind
			}
		}
		data = append(data, map[string]any{
			"id":            ev.ID,
			"kind":          ev.Kind,
			"event_type":    ev.Kind,
			"occurred_at":   ev.OccurredAt.UTC().Format(time.RFC3339Nano),
			"timestamp":     ev.OccurredAt.UTC().Format(time.RFC3339Nano),
			"actor_user_id": ev.Actor,
			"source":        src,
			"summary":       summary,
			"metadata":      ev.Payload,
			"payload":       ev.Payload,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func listDTO(item database.IncidentRow) map[string]any {
	return map[string]any{
		"id":                item.ID,
		"reference":         item.Reference,
		"service_id":        item.ServiceID,
		"title":             item.Title,
		"severity":          item.Severity,
		"status":            item.Status,
		"commander_user_id": item.Commander,
		"detected_at":       item.DetectedAt.UTC().Format(time.RFC3339Nano),
		"version":           item.Version,
	}
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func detailDTO(item database.IncidentRow) map[string]any {
	return map[string]any{
		"id":                 item.ID,
		"reference":          item.Reference,
		"service_id":         item.ServiceID,
		"title":              item.Title,
		"summary":            item.Summary,
		"severity":           item.Severity,
		"status":             item.Status,
		"created_by_user_id": item.CreatedBy,
		"commander_user_id":  item.Commander,
		"detected_at":        item.DetectedAt.UTC().Format(time.RFC3339Nano),
		"resolved_at":        formatTimePtr(item.ResolvedAt),
		"closed_at":          formatTimePtr(item.ClosedAt),
		"version":            item.Version,
		"alert_ids":          item.AlertIDs,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
