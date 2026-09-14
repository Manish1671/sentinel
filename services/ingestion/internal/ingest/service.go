package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/apierr"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/events"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/validation"
)

type Publisher interface {
	Publish(ctx context.Context, msg kafka.Message) error
}

type Result struct {
	Envelope events.Envelope
	Topic    string
	Replayed bool
}

type Service struct {
	pub Publisher
	idm *Memory
	now func() time.Time
}

func NewService(pub Publisher, idm *Memory) *Service {
	return &Service{pub: pub, idm: idm, now: time.Now}
}

type MetricRequest struct {
	ServiceID      string         `json:"service_id"`
	ServiceSlug    string         `json:"service_slug"`
	Service        string         `json:"service"`
	Environment    string         `json:"environment"`
	Name           string         `json:"name"`
	Value          *float64       `json:"value"`
	Unit           string         `json:"unit"`
	Timestamp      string         `json:"timestamp"`
	OccurredAt     string         `json:"occurred_at"`
	Labels         map[string]any `json:"labels"`
	Attributes     map[string]any `json:"attributes"`
	CorrelationID  string         `json:"correlation_id"`
	EventID        string         `json:"event_id"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type LogRequest struct {
	ServiceID      string         `json:"service_id"`
	ServiceSlug    string         `json:"service_slug"`
	Service        string         `json:"service"`
	Environment    string         `json:"environment"`
	Severity       string         `json:"severity"`
	Message        string         `json:"message"`
	Timestamp      string         `json:"timestamp"`
	OccurredAt     string         `json:"occurred_at"`
	TraceID        *string        `json:"trace_id"`
	SpanID         *string        `json:"span_id"`
	Attributes     map[string]any `json:"attributes"`
	CorrelationID  string         `json:"correlation_id"`
	EventID        string         `json:"event_id"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type TraceRequest struct {
	ServiceID      string         `json:"service_id"`
	ServiceSlug    string         `json:"service_slug"`
	Service        string         `json:"service"`
	Environment    string         `json:"environment"`
	TraceID        string         `json:"trace_id"`
	SpanID         string         `json:"span_id"`
	ParentSpanID   *string        `json:"parent_span_id"`
	Name           string         `json:"name"`
	Operation      string         `json:"operation"`
	DurationMS     *float64       `json:"duration_ms"`
	Duration       *float64       `json:"duration"`
	Status         string         `json:"status"`
	Timestamp      string         `json:"timestamp"`
	OccurredAt     string         `json:"occurred_at"`
	Attributes     map[string]any `json:"attributes"`
	CorrelationID  string         `json:"correlation_id"`
	EventID        string         `json:"event_id"`
	IdempotencyKey string         `json:"idempotency_key"`
}

type DeploymentRequest struct {
	ServiceID        string `json:"service_id"`
	ServiceSlug      string `json:"service_slug"`
	Service          string `json:"service"`
	Environment      string `json:"environment"`
	DeploymentID     string `json:"deployment_id"`
	Version          string `json:"version"`
	GitSHA           string `json:"git_sha"`
	Status           string `json:"status"`
	Changelog        string `json:"changelog"`
	StartedAt        string `json:"started_at"`
	CompletedAt      string `json:"completed_at"`
	DeployedByUserID string `json:"deployed_by_user_id"`
	CorrelationID    string `json:"correlation_id"`
	EventID          string `json:"event_id"`
	IdempotencyKey   string `json:"idempotency_key"`
}

func (s *Service) IngestMetric(ctx context.Context, in MetricRequest, headerKey string) (Result, error) {
	slug := first(in.ServiceSlug, in.Service)
	svc, err := validation.ResolveService(in.ServiceID, slug, in.Environment)
	if err != nil {
		return Result{}, err
	}
	if err := validation.NonEmpty(in.Name, "name"); err != nil {
		return Result{}, err
	}
	if in.Value == nil {
		return Result{}, validationErr("value", "required", "value is required")
	}
	if err := validation.FiniteNumber(*in.Value, "value"); err != nil {
		return Result{}, err
	}
	occurred, err := validation.OccurredAt(in.Timestamp, in.OccurredAt, s.now())
	if err != nil {
		return Result{}, err
	}
	labelsIn := in.Labels
	if labelsIn == nil {
		labelsIn = in.Attributes
	}
	labels, err := validation.StringLabels(labelsIn, "labels")
	if err != nil {
		return Result{}, err
	}
	payload := validation.PayloadMetric(svc, strings.TrimSpace(in.Name), *in.Value, strings.TrimSpace(in.Unit), occurred, labels)
	return s.publish(ctx, events.TypeMetric, svc.ID.String(), in.EventID, in.CorrelationID, first(headerKey, in.IdempotencyKey), occurred, payload)
}

func (s *Service) IngestLog(ctx context.Context, in LogRequest, headerKey string) (Result, error) {
	slug := first(in.ServiceSlug, in.Service)
	svc, err := validation.ResolveService(in.ServiceID, slug, in.Environment)
	if err != nil {
		return Result{}, err
	}
	if err := validation.NonEmpty(in.Message, "message"); err != nil {
		return Result{}, err
	}
	sev := strings.ToLower(strings.TrimSpace(in.Severity))
	if err := validation.OneOf(sev, "severity", "debug", "info", "warn", "error", "fatal"); err != nil {
		return Result{}, err
	}
	occurred, err := validation.OccurredAt(in.Timestamp, in.OccurredAt, s.now())
	if err != nil {
		return Result{}, err
	}
	payload := validation.PayloadLog(svc, sev, strings.TrimSpace(in.Message), occurred, emptyToNil(in.TraceID), emptyToNil(in.SpanID), in.Attributes)
	return s.publish(ctx, events.TypeLog, svc.ID.String(), in.EventID, in.CorrelationID, first(headerKey, in.IdempotencyKey), occurred, payload)
}

func (s *Service) IngestTrace(ctx context.Context, in TraceRequest, headerKey string) (Result, error) {
	slug := first(in.ServiceSlug, in.Service)
	svc, err := validation.ResolveService(in.ServiceID, slug, in.Environment)
	if err != nil {
		return Result{}, err
	}
	name := first(in.Name, in.Operation)
	if err := validation.NonEmpty(in.TraceID, "trace_id"); err != nil {
		return Result{}, err
	}
	if err := validation.NonEmpty(in.SpanID, "span_id"); err != nil {
		return Result{}, err
	}
	if err := validation.NonEmpty(name, "name"); err != nil {
		return Result{}, err
	}
	dur := in.DurationMS
	if dur == nil {
		dur = in.Duration
	}
	if dur == nil {
		return Result{}, validationErr("duration_ms", "required", "duration_ms is required")
	}
	if err := validation.FiniteNumber(*dur, "duration_ms"); err != nil {
		return Result{}, err
	}
	if *dur < 0 {
		return Result{}, validationErr("duration_ms", "invalid", "duration_ms must be >= 0")
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "unset"
	}
	if err := validation.OneOf(status, "status", "ok", "error", "unset"); err != nil {
		return Result{}, err
	}
	occurred, err := validation.OccurredAt(in.Timestamp, in.OccurredAt, s.now())
	if err != nil {
		return Result{}, err
	}
	payload := validation.PayloadTrace(svc, strings.TrimSpace(in.TraceID), strings.TrimSpace(in.SpanID), strings.TrimSpace(name), emptyToNil(in.ParentSpanID), *dur, status, occurred, in.Attributes)
	return s.publish(ctx, events.TypeTrace, svc.ID.String(), in.EventID, in.CorrelationID, first(headerKey, in.IdempotencyKey), occurred, payload)
}

func (s *Service) IngestDeployment(ctx context.Context, in DeploymentRequest, headerKey string) (Result, error) {
	slug := first(in.ServiceSlug, in.Service)
	svc, err := validation.ResolveService(in.ServiceID, slug, in.Environment)
	if err != nil {
		return Result{}, err
	}
	if err := validation.NonEmpty(in.Version, "version"); err != nil {
		return Result{}, err
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if status == "" {
		status = "succeeded"
	}
	if err := validation.OneOf(status, "status", "in_progress", "succeeded", "failed", "rolled_back"); err != nil {
		return Result{}, err
	}
	started, err := validation.ParseTime(in.StartedAt, "started_at", true, s.now())
	if err != nil {
		return Result{}, err
	}
	var completed *time.Time
	if strings.TrimSpace(in.CompletedAt) != "" {
		t, err := validation.ParseTime(in.CompletedAt, "completed_at", true, s.now())
		if err != nil {
			return Result{}, err
		}
		completed = &t
	}
	depID := uuid.New()
	if strings.TrimSpace(in.DeploymentID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(in.DeploymentID))
		if err != nil {
			return Result{}, validationErr("deployment_id", "invalid", "deployment_id must be a UUID")
		}
		depID = id
	}
	var deployedBy *uuid.UUID
	if strings.TrimSpace(in.DeployedByUserID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(in.DeployedByUserID))
		if err != nil {
			return Result{}, validationErr("deployed_by_user_id", "invalid", "deployed_by_user_id must be a UUID")
		}
		deployedBy = &id
	}
	payload := validation.PayloadDeployment(svc, depID, strings.TrimSpace(in.Version), strings.TrimSpace(in.GitSHA), status, in.Changelog, started, completed, deployedBy)
	occurred := started
	return s.publish(ctx, events.TypeDeployment, svc.ID.String(), in.EventID, in.CorrelationID, first(headerKey, in.IdempotencyKey), occurred, payload)
}

func (s *Service) publish(ctx context.Context, eventType, key, eventIDRaw, corrRaw, idemRaw string, occurred time.Time, payload map[string]any) (Result, error) {
	eventID, err := validation.EventID(eventIDRaw)
	if err != nil {
		return Result{}, err
	}
	corr, err := validation.CorrelationID(corrRaw)
	if err != nil {
		return Result{}, err
	}
	idem, err := validation.IdempotencyKey(idemRaw, "", eventIDRaw)
	if err != nil {
		return Result{}, err
	}
	if idem == "" {
		idem = eventID.String()
	}
	hash, err := events.CanonicalHash(eventType, payload)
	if err != nil {
		return Result{}, err
	}
	if existing, ok, err := s.idm.Check(idem, hash); err != nil {
		return Result{}, err
	} else if ok {
		return Result{Envelope: existing, Topic: events.TopicFor(eventType), Replayed: true}, nil
	}
	env := events.Build(eventType, eventID, corr, occurred, idem, payload)
	body, err := json.Marshal(env)
	if err != nil {
		return Result{}, err
	}
	if err := s.pub.Publish(ctx, kafka.Message{Topic: events.TopicFor(eventType), Key: key, Value: body}); err != nil {
		return Result{}, fmt.Errorf("publish: %w", err)
	}
	s.idm.Remember(idem, hash, env)
	return Result{Envelope: env, Topic: events.TopicFor(eventType), Replayed: false}, nil
}

func first(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func emptyToNil(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func validationErr(path, code, message string) error {
	return apierr.Validation([]map[string]string{apierr.Field(path, code, message)})
}
