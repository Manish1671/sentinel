package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/apierr"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/config"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/ingest"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

type readiness interface {
	Ready(ctx context.Context) error
}

type Server struct {
	cfg   config.Config
	log   *slog.Logger
	svc   *ingest.Service
	kafka readiness
	http  *http.Server
}

func New(cfg config.Config, log *slog.Logger, svc *ingest.Service, k readiness) *Server {
	s := &Server{cfg: cfg, log: log, svc: svc, kafka: k}
	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           s.routes(),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
	return s
}

func (s *Server) Handler() http.Handler { return s.http.Handler }
func (s *Server) HTTP() *http.Server    { return s.http }

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /ready", s.ready)
	mux.HandleFunc("POST /api/v1/telemetry/metric", s.metric)
	mux.HandleFunc("POST /api/v1/telemetry/log", s.logEvent)
	mux.HandleFunc("POST /api/v1/telemetry/trace", s.trace)
	mux.HandleFunc("POST /api/v1/deployments", s.deployment)

	var h http.Handler = mux
	h = recoverer(s.log, h)
	h = timeout(s.cfg.RequestTimeout, h)
	h = accessLog(s.log, h)
	h = requestID(h)
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
	ok := true
	if s.kafka != nil {
		if err := s.kafka.Ready(ctx); err != nil {
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

func (s *Server) metric(w http.ResponseWriter, r *http.Request) {
	var in ingest.MetricRequest
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	start := time.Now()
	result, err := s.svc.IngestMetric(r.Context(), in, r.Header.Get("Idempotency-Key"))
	s.respondIngest(w, r, result, err, start)
}

func (s *Server) logEvent(w http.ResponseWriter, r *http.Request) {
	var in ingest.LogRequest
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	start := time.Now()
	result, err := s.svc.IngestLog(r.Context(), in, r.Header.Get("Idempotency-Key"))
	s.respondIngest(w, r, result, err, start)
}

func (s *Server) trace(w http.ResponseWriter, r *http.Request) {
	var in ingest.TraceRequest
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	start := time.Now()
	result, err := s.svc.IngestTrace(r.Context(), in, r.Header.Get("Idempotency-Key"))
	s.respondIngest(w, r, result, err, start)
}

func (s *Server) deployment(w http.ResponseWriter, r *http.Request) {
	var in ingest.DeploymentRequest
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, s.log, err)
		return
	}
	start := time.Now()
	result, err := s.svc.IngestDeployment(r.Context(), in, r.Header.Get("Idempotency-Key"))
	s.respondIngest(w, r, result, err, start)
}

func (s *Server) respondIngest(w http.ResponseWriter, r *http.Request, result ingest.Result, err error, start time.Time) {
	if err != nil {
		s.log.Error("ingest_failed",
			"request_id", requestIDFrom(r.Context()),
			"latency_ms", time.Since(start).Milliseconds(),
			"error", err.Error(),
		)
		if _, ok := apierr.As(err); !ok {
			err = apierr.Unavailable("Failed to publish event.")
		}
		writeError(w, r, s.log, err)
		return
	}
	status := http.StatusAccepted
	if result.Replayed {
		status = http.StatusOK
	}
	body, _ := json.Marshal(result.Envelope)
	var envelope any
	_ = json.Unmarshal(body, &envelope)
	s.log.Info("ingested",
		"request_id", requestIDFrom(r.Context()),
		"event_id", result.Envelope.EventID,
		"event_type", result.Envelope.EventType,
		"service", result.Envelope.Payload["service_slug"],
		"kafka_topic", result.Topic,
		"latency_ms", time.Since(start).Milliseconds(),
		"replayed", result.Replayed,
	)
	writeJSON(w, status, map[string]any{
		"data": map[string]any{
			"topic":    result.Topic,
			"replayed": result.Replayed,
			"event":    envelope,
		},
	})
}
