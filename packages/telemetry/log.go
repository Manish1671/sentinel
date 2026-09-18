package telemetry

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextHandler struct {
	slog.Handler
	service string
	env     string
}

func NewLogger(level, service, env string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	if service == "" {
		service = ServiceName()
	}
	if env == "" {
		env = Environment()
	}
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	return slog.New(contextHandler{Handler: base, service: service, env: env})
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	r.AddAttrs(
		slog.String("service", h.service),
		slog.String("environment", h.env),
	)
	if tid := TraceIDFrom(ctx); tid != "" {
		r.AddAttrs(slog.String("trace_id", tid))
	}
	if rid := RequestIDFrom(ctx); rid != "" {
		r.AddAttrs(slog.String("request_id", rid))
	}
	if v, ok := ctx.Value(eventIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("event_id", v))
	}
	if v, ok := ctx.Value(incidentIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("incident_id", v))
	}
	if v, ok := ctx.Value(investigationIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("investigation_id", v))
	}
	if v, ok := ctx.Value(remediationIDKey).(string); ok && v != "" {
		r.AddAttrs(slog.String("remediation_id", v))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{Handler: h.Handler.WithAttrs(attrs), service: h.service, env: h.env}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{Handler: h.Handler.WithGroup(name), service: h.service, env: h.env}
}
