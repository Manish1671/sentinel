package validation

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/apierr"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/catalog"
)

type ServiceRef struct {
	catalog.Service
}

type CommonInput struct {
	ServiceID      string
	ServiceSlug    string
	Environment    string
	Timestamp      string
	OccurredAt     string
	CorrelationID  string
	EventID        string
	IdempotencyKey string
	TenantID       string
}

func ParseUUID(raw, field string, required bool) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return nil, fieldErr(field, "required", field+" is required")
		}
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fieldErr(field, "invalid", field+" must be a UUID")
	}
	return &id, nil
}

func ParseTime(raw, field string, required bool, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if required {
			return time.Time{}, fieldErr(field, "required", field+" is required")
		}
		return fallback.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return time.Time{}, fieldErr(field, "invalid", field+" must be an RFC3339 timestamp")
	}
	return t.UTC(), nil
}

func Environment(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		v = "production"
	}
	switch v {
	case "production", "staging", "development":
		return v, nil
	default:
		return "", fieldErr("environment", "invalid", "environment is invalid")
	}
}

func ResolveService(idRaw, slug, env string) (catalog.Service, error) {
	env, err := Environment(env)
	if err != nil {
		return catalog.Service{}, err
	}
	var id *uuid.UUID
	if strings.TrimSpace(idRaw) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(idRaw))
		if err != nil {
			return catalog.Service{}, fieldErr("service_id", "invalid", "service_id must be a UUID")
		}
		id = &parsed
	}
	slug = strings.TrimSpace(slug)
	svc, ok := catalog.Resolve(id, slug, env)
	if !ok {
		return catalog.Service{}, fieldErr("service", "invalid", "unknown service identity (provide a known service_id or service_slug)")
	}
	return svc, nil
}

func IdempotencyKey(header, body, eventID string) (string, error) {
	key := strings.TrimSpace(header)
	if key == "" {
		key = strings.TrimSpace(body)
	}
	if key == "" && eventID != "" {
		key = eventID
	}
	if key == "" {
		return "", nil
	}
	n := utf8.RuneCountInString(key)
	if n < 8 || n > 128 {
		return "", fieldErr("idempotency_key", "invalid", "idempotency_key must be 8-128 characters")
	}
	for _, c := range key {
		ok := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-'
		if !ok {
			return "", fieldErr("idempotency_key", "invalid", "idempotency_key contains invalid characters")
		}
	}
	return key, nil
}

func OccurredAt(timestamp, occurredAt string, now time.Time) (time.Time, error) {
	raw := strings.TrimSpace(occurredAt)
	if raw == "" {
		raw = strings.TrimSpace(timestamp)
	}
	return ParseTime(raw, "occurred_at", true, now)
}

func CorrelationID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.New(), nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fieldErr("correlation_id", "invalid", "correlation_id must be a UUID")
	}
	return id, nil
}

func EventID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.New(), nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fieldErr("event_id", "invalid", "event_id must be a UUID")
	}
	return id, nil
}

func StringLabels(in map[string]any, field string) (map[string]string, error) {
	if in == nil {
		return map[string]string{}, nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		s, ok := v.(string)
		if !ok {
			return nil, fieldErr(field+"."+k, "invalid", field+" values must be strings")
		}
		out[k] = s
	}
	return out, nil
}

func FiniteNumber(v float64, field string) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fieldErr(field, "invalid", field+" must be a finite number")
	}
	return nil
}

func OneOf(v, field string, allowed ...string) error {
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return fieldErr(field, "invalid", field+" is invalid")
}

func NonEmpty(v, field string) error {
	if strings.TrimSpace(v) == "" {
		return fieldErr(field, "required", field+" is required")
	}
	return nil
}

func fieldErr(path, code, message string) error {
	return apierr.Validation([]map[string]string{apierr.Field(path, code, message)})
}

func RFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func PayloadMetric(svc catalog.Service, name string, value float64, unit string, occurredAt time.Time, labels map[string]string) map[string]any {
	p := map[string]any{
		"service_id":   svc.ID.String(),
		"service_slug": svc.Slug,
		"environment":  svc.Environment,
		"name":         name,
		"value":        value,
		"occurred_at":  RFC3339(occurredAt),
	}
	if unit != "" {
		p["unit"] = unit
	}
	if len(labels) > 0 {
		p["labels"] = labels
	}
	return p
}

func PayloadLog(svc catalog.Service, severity, message string, occurredAt time.Time, traceID, spanID *string, attrs map[string]any) map[string]any {
	p := map[string]any{
		"service_id":   svc.ID.String(),
		"service_slug": svc.Slug,
		"environment":  svc.Environment,
		"severity":     severity,
		"message":      message,
		"occurred_at":  RFC3339(occurredAt),
	}
	if traceID != nil {
		p["trace_id"] = *traceID
	}
	if spanID != nil {
		p["span_id"] = *spanID
	}
	if len(attrs) > 0 {
		p["attributes"] = attrs
	}
	return p
}

func PayloadTrace(svc catalog.Service, traceID, spanID, name string, parent *string, duration float64, status string, occurredAt time.Time, attrs map[string]any) map[string]any {
	p := map[string]any{
		"service_id":   svc.ID.String(),
		"service_slug": svc.Slug,
		"environment":  svc.Environment,
		"trace_id":     traceID,
		"span_id":      spanID,
		"name":         name,
		"duration_ms":  duration,
		"status":       status,
		"occurred_at":  RFC3339(occurredAt),
	}
	if parent != nil {
		p["parent_span_id"] = *parent
	}
	if len(attrs) > 0 {
		p["attributes"] = attrs
	}
	return p
}

func PayloadDeployment(svc catalog.Service, deploymentID uuid.UUID, version, gitSHA, status, changelog string, startedAt time.Time, completedAt *time.Time, deployedBy *uuid.UUID) map[string]any {
	p := map[string]any{
		"deployment_id": deploymentID.String(),
		"service_id":    svc.ID.String(),
		"service_slug":  svc.Slug,
		"environment":   svc.Environment,
		"version":       version,
		"status":        status,
		"started_at":    RFC3339(startedAt),
	}
	if gitSHA != "" {
		p["git_sha"] = gitSHA
	}
	if changelog != "" {
		p["changelog"] = changelog
	}
	if completedAt != nil {
		p["completed_at"] = RFC3339(*completedAt)
	}
	if deployedBy != nil {
		p["deployed_by_user_id"] = deployedBy.String()
	}
	return p
}
