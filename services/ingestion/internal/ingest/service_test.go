package ingest

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/apierr"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/events"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/kafka"
)

type stubPub struct {
	mu   sync.Mutex
	msgs []kafka.Message
	err  error
}

func (s *stubPub) Publish(_ context.Context, msg kafka.Message) error {
	if s.err != nil {
		return s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs = append(s.msgs, msg)
	return nil
}

func (s *stubPub) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.msgs)
}

func value(v float64) *float64 { return &v }

func TestMetricEnvelopeAndTopic(t *testing.T) {
	pub := &stubPub{}
	svc := NewService(pub, NewMemory())
	corr := uuid.MustParse("dddddddd-0000-4000-8000-0000000000d1")
	res, err := svc.IngestMetric(context.Background(), MetricRequest{
		ServiceSlug:   "payments-api",
		Environment:   "production",
		Name:          "payments.capture.latency_p99",
		Value:         value(180),
		Unit:          "ms",
		OccurredAt:    "2026-09-14T04:18:00Z",
		CorrelationID: corr.String(),
		Labels:        map[string]any{"route": "POST /capture"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Topic != events.TopicTelemetry || res.Envelope.EventType != events.TypeMetric {
		t.Fatalf("topic/type %s %s", res.Topic, res.Envelope.EventType)
	}
	if res.Envelope.Source != events.Source {
		t.Fatalf("source %s", res.Envelope.Source)
	}
	if res.Envelope.CorrelationID != corr {
		t.Fatalf("correlation %s", res.Envelope.CorrelationID)
	}
	if res.Envelope.CausationID != nil {
		t.Fatal("origin events must have null causation_id")
	}
	if res.Envelope.Payload["service_slug"] != "payments-api" {
		t.Fatalf("payload %+v", res.Envelope.Payload)
	}
	if pub.count() != 1 {
		t.Fatalf("published %d", pub.count())
	}
	if string(pub.msgs[0].Key) != "22222222-2222-4222-8222-222222222221" {
		t.Fatalf("key %s", pub.msgs[0].Key)
	}
}

func TestCorrelationGeneratedWhenMissing(t *testing.T) {
	svc := NewService(&stubPub{}, NewMemory())
	res, err := svc.IngestLog(context.Background(), LogRequest{
		ServiceSlug: "auth-service",
		Severity:    "info",
		Message:     "login ok",
		OccurredAt:  time.Now().UTC().Format(time.RFC3339),
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Envelope.CorrelationID == uuid.Nil {
		t.Fatal("expected generated correlation_id")
	}
}

func TestTimestampNormalizedUTC(t *testing.T) {
	svc := NewService(&stubPub{}, NewMemory())
	res, err := svc.IngestMetric(context.Background(), MetricRequest{
		ServiceSlug: "payments-api",
		Name:        "n",
		Value:       value(1),
		OccurredAt:  "2026-09-14T09:48:00+05:30",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Envelope.OccurredAt.Location() != time.UTC {
		t.Fatalf("tz %s", res.Envelope.OccurredAt.Location())
	}
	if got := res.Envelope.OccurredAt.Format(time.RFC3339); got != "2026-09-14T04:18:00Z" {
		t.Fatalf("occurred_at %s", got)
	}
}

func TestIdempotentRetryDoesNotRepublish(t *testing.T) {
	pub := &stubPub{}
	svc := NewService(pub, NewMemory())
	id := uuid.New().String()
	req := MetricRequest{
		ServiceSlug: "payments-api",
		Name:        "payments.capture.error_rate",
		Value:       value(0.01),
		OccurredAt:  "2026-09-14T04:18:00Z",
		EventID:     id,
	}
	a, err := svc.IngestMetric(context.Background(), req, "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.IngestMetric(context.Background(), req, "")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Replayed || a.Envelope.EventID != b.Envelope.EventID {
		t.Fatalf("replay %+v %+v", a, b)
	}
	if pub.count() != 1 {
		t.Fatalf("published %d", pub.count())
	}
}

func TestIdempotencyConflict(t *testing.T) {
	svc := NewService(&stubPub{}, NewMemory())
	key := "ingest-key-12345"
	req := MetricRequest{
		ServiceSlug: "payments-api",
		Name:        "n",
		Value:       value(1),
		OccurredAt:  "2026-09-14T04:18:00Z",
	}
	if _, err := svc.IngestMetric(context.Background(), req, key); err != nil {
		t.Fatal(err)
	}
	req.Name = "other"
	_, err := svc.IngestMetric(context.Background(), req, key)
	if err == nil {
		t.Fatal("expected conflict")
	}
	api, ok := apierr.As(err)
	if !ok || api.Code != "idempotency_key_conflict" {
		t.Fatalf("err %v", err)
	}
}

func TestMalformedRejected(t *testing.T) {
	svc := NewService(&stubPub{}, NewMemory())
	_, err := svc.IngestMetric(context.Background(), MetricRequest{ServiceSlug: "payments-api", Value: value(1), OccurredAt: "2026-09-14T04:18:00Z"}, "")
	if err == nil {
		t.Fatal("expected validation")
	}
	_, err = svc.IngestLog(context.Background(), LogRequest{ServiceSlug: "payments-api", Message: "x", Severity: "nope", OccurredAt: "2026-09-14T04:18:00Z"}, "")
	if err == nil {
		t.Fatal("expected severity validation")
	}
	_, err = svc.IngestMetric(context.Background(), MetricRequest{ServiceSlug: "unknown-svc", Name: "n", Value: value(1), OccurredAt: "2026-09-14T04:18:00Z"}, "")
	if err == nil {
		t.Fatal("expected unknown service")
	}
}

func TestDeploymentTopic(t *testing.T) {
	pub := &stubPub{}
	svc := NewService(pub, NewMemory())
	res, err := svc.IngestDeployment(context.Background(), DeploymentRequest{
		ServiceSlug: "payments-api",
		Version:     "1.18.0",
		Status:      "succeeded",
		StartedAt:   "2026-09-14T04:12:00Z",
		CompletedAt: "2026-09-14T04:17:00Z",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Topic != events.TopicDeployments || res.Envelope.EventType != events.TypeDeployment {
		t.Fatalf("%s %s", res.Topic, res.Envelope.EventType)
	}
}

func TestPublishFailureSurfaces(t *testing.T) {
	pub := &stubPub{err: context.DeadlineExceeded}
	svc := NewService(pub, NewMemory())
	_, err := svc.IngestMetric(context.Background(), MetricRequest{
		ServiceSlug: "payments-api",
		Name:        "n",
		Value:       value(1),
		OccurredAt:  "2026-09-14T04:18:00Z",
	}, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvelopeJSONHasRequiredFields(t *testing.T) {
	svc := NewService(&stubPub{}, NewMemory())
	res, err := svc.IngestTrace(context.Background(), TraceRequest{
		ServiceSlug: "payments-api",
		TraceID:     "4bf2c91a7e3d00aa",
		SpanID:      "91aa00e3",
		Name:        "inventory.reserve",
		DurationMS:  value(12),
		Status:      "ok",
		OccurredAt:  "2026-09-14T04:18:22Z",
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(res.Envelope)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, k := range []string{"event_id", "event_type", "event_version", "occurred_at", "source", "correlation_id", "payload"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s in %s", k, raw)
		}
	}
}
