package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sentinel-dev/sentinel/services/remediation/internal/approvals"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/remediation"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
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
	proc  *remediation.Processor
	exec  executor.Executor
	http  *http.Server
}

func New(cfg config.Config, log *slog.Logger, store *database.Store, db interface{ Ping(ctx context.Context) error }, k readiness, proc *remediation.Processor, exec executor.Executor) *Server {
	s := &Server{cfg: cfg, log: log, store: store, db: db, k: k, proc: proc, exec: exec}
	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           s.routes(),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) HTTP() *http.Server { return s.http }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("GET /api/v1/remediations/{id}", s.getRemediation)
	mux.HandleFunc("GET /api/v1/remediations/{id}/execution", s.getExecution)
	mux.HandleFunc("POST /api/v1/remediations/{id}/approve", s.approve)
	mux.HandleFunc("POST /api/v1/remediations/{id}/reject", s.reject)
	mux.HandleFunc("GET /api/v1/incidents/{id}/remediations", s.listByIncident)
	mux.HandleFunc("GET /api/v1/incidents/{id}/timeline", s.timeline)
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

func (s *Server) actor(r *http.Request) (approvals.Actor, int, string) {
	if s.store == nil {
		return approvals.Actor{}, http.StatusInternalServerError, "store unavailable"
	}
	var userID uuid.UUID
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") && s.cfg.AuthTokenSecret != "" {
		id, err := approvals.ParseBearer(s.cfg.AuthTokenSecret, h)
		if err != nil {
			return approvals.Actor{}, http.StatusUnauthorized, "invalid token"
		}
		userID = id
	} else if s.cfg.DevAuth() {
		raw := strings.TrimSpace(r.Header.Get("X-Sentinel-User-Id"))
		id, err := uuid.Parse(raw)
		if err != nil {
			return approvals.Actor{}, http.StatusUnauthorized, "authentication required"
		}
		userID = id
	} else {
		return approvals.Actor{}, http.StatusUnauthorized, "authentication required"
	}
	u, err := s.store.GetUser(r.Context(), userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return approvals.Actor{}, http.StatusUnauthorized, "unknown user"
	}
	if err != nil {
		return approvals.Actor{}, http.StatusInternalServerError, "user lookup failed"
	}
	a, err := approvals.LoadActor(u)
	if err != nil {
		return approvals.Actor{}, http.StatusUnauthorized, err.Error()
	}
	return a, 0, ""
}

func (s *Server) getRemediation(w http.ResponseWriter, r *http.Request) {
	actor, code, msg := s.actor(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	if !approvals.CanInspect(actor.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	row, err := s.store.GetRemediation(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": remDTO(row)})
}

func (s *Server) getExecution(w http.ResponseWriter, r *http.Request) {
	actor, code, msg := s.actor(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	if !approvals.CanInspect(actor.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	row, err := s.store.GetRemediation(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get failed"})
		return
	}
	st, err := s.exec.State(r.Context(), row.ServiceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{
		"remediation_id":        row.ID,
		"status":                row.Status,
		"verification_status":   row.VerificationStatus,
		"verification_details":  row.VerificationDetails,
		"simulator":             st,
	}})
}

func (s *Server) listByIncident(w http.ResponseWriter, r *http.Request) {
	actor, code, msg := s.actor(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	if !approvals.CanInspect(actor.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	items, err := s.store.ListByIncident(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	data := make([]map[string]any, 0, len(items))
	for _, it := range items {
		data = append(data, remDTO(it))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	actor, code, msg := s.actor(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	if !approvals.CanInspect(actor.Role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	items, err := s.store.Timeline(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "timeline failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, true)
}

func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, false)
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	actor, code, msg := s.actor(r)
	if code != 0 {
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Idempotency-Key required"})
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	var req struct {
		Comment string `json:"comment"`
	}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	if len(req.Comment) > 2000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "comment too long"})
		return
	}
	hash := remediation.HashRequest(r.URL.Path, string(body))
	var row database.Remediation
	if approve {
		row, err = s.proc.Approve(r.Context(), id, actor, req.Comment, idem, hash)
	} else {
		row, err = s.proc.Reject(r.Context(), id, actor, req.Comment, idem, hash)
	}
	if err != nil {
		switch {
		case remediation.IsForbidden(err):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case remediation.IsAlreadyDecided(err):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "approval_already_decided"})
		case remediation.IsIdempotencyConflict(err):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "idempotency_key_conflict"})
		case errors.Is(err, pgx.ErrNoRows):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": remDTO(row)})
}

func remDTO(r database.Remediation) map[string]any {
	return map[string]any{
		"id":                  r.ID,
		"incident_id":         r.IncidentID,
		"recommendation_id":   r.RecommendationID,
		"service_id":          r.ServiceID,
		"status":              r.Status,
		"action_type":         r.ActionType,
		"parameters":          r.Parameters,
		"requested_by_user_id": r.RequestedBy,
		"attempt_number":      r.AttemptNumber,
		"verification_status": r.VerificationStatus,
		"result_summary":      r.ResultSummary,
		"error_message":       r.ErrorMessage,
		"approval": map[string]any{
			"id":            r.ApprovalID,
			"decision":      r.ApprovalDecision,
			"actor_user_id": r.ApprovalActor,
			"comment":       r.ApprovalComment,
			"decided_at":    formatTimePtr(r.DecidedAt),
		},
	}
}

func formatTimePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
