package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	eventIDKey
	incidentIDKey
	investigationIDKey
	remediationIDKey
)

func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, eventIDKey, id)
}

func WithIncidentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, incidentIDKey, id)
}

func WithInvestigationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, investigationIDKey, id)
}

func WithRemediationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, remediationIDKey, id)
}

func Start(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("sentinel").Start(ctx, name, trace.WithAttributes(attrs...))
	if rid := RequestIDFrom(ctx); rid != "" {
		span.SetAttributes(attribute.String("request_id", rid))
	}
	return ctx, span
}

func End(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

func TraceIDFrom(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}
