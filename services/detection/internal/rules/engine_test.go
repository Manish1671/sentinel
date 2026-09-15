package rules

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/events"
)

var payments = uuid.MustParse("22222222-2222-4222-8222-222222222221")

func engine() *Engine {
	return NewEngine(config.Rules{
		LatencyThresholdMS:          1000,
		ErrorRateThreshold:          0.05,
		DBConnectionThreshold:       0.90,
		ErrorBurstCount:             3,
		ErrorBurstWindow:            time.Minute,
		DeploymentCorrelationWindow: 15 * time.Minute,
		ErrorBurstSeverity:          "high",
	}, NewWindows())
}

func metric(name string, value float64, at time.Time) events.Envelope {
	return events.Envelope{
		EventID:    uuid.New(),
		EventType:  events.TypeMetric,
		OccurredAt: at,
		Payload: map[string]any{
			"service_id":   payments.String(),
			"service_slug": "payments-api",
			"name":         name,
			"value":        value,
		},
	}
}

func TestHighLatencyThreshold(t *testing.T) {
	e := engine()
	at := time.Now().UTC()
	if hits := e.Evaluate(metric("payments.capture.latency_p99", 1000, at)); len(hits) != 0 {
		t.Fatalf("boundary 1000 should not fire: %+v", hits)
	}
	hits := e.Evaluate(metric("payments.capture.latency_p99", 1000.1, at))
	if len(hits) != 1 || hits[0].DetectorID != HighLatency || hits[0].Severity != "high" {
		t.Fatalf("%+v", hits)
	}
	if hits[0].Fingerprint != Fingerprint(HighLatency, payments, "payments.capture.latency_p99") {
		t.Fatalf("fp %s", hits[0].Fingerprint)
	}
}

func TestHighErrorRatePercentAndRatio(t *testing.T) {
	e := engine()
	at := time.Now().UTC()
	if hits := e.Evaluate(metric("http_error_rate", 0.05, at)); len(hits) != 0 {
		t.Fatal("0.05 should not fire")
	}
	hits := e.Evaluate(metric("payments.capture.error_rate", 0.11, at))
	if len(hits) != 1 || hits[0].DetectorID != HighErrorRate {
		t.Fatalf("%+v", hits)
	}
	hits = e.Evaluate(metric("http_error_rate", 6, at))
	if len(hits) != 1 {
		t.Fatalf("percent 6 should be 0.06: %+v", hits)
	}
}

func TestDBSaturation(t *testing.T) {
	e := engine()
	at := time.Now().UTC()
	if hits := e.Evaluate(metric("db_connection_utilization", 0.90, at)); len(hits) != 0 {
		t.Fatal("90% boundary")
	}
	hits := e.Evaluate(metric("db_connection_utilization", 0.91, at))
	if len(hits) != 1 || hits[0].Severity != "critical" || hits[0].DetectorID != DBSaturation {
		t.Fatalf("%+v", hits)
	}
}

func TestErrorBurstWindow(t *testing.T) {
	e := engine()
	at := time.Date(2026, 9, 14, 4, 18, 0, 0, time.UTC)
	log := func(ts time.Time) events.Envelope {
		return events.Envelope{
			EventID: uuid.New(), EventType: events.TypeLog, OccurredAt: ts,
			Payload: map[string]any{"service_id": payments.String(), "severity": "error", "message": "x"},
		}
	}
	if hits := e.Evaluate(log(at)); len(hits) != 0 {
		t.Fatal("1 log")
	}
	if hits := e.Evaluate(log(at.Add(time.Second))); len(hits) != 0 {
		t.Fatal("2 logs")
	}
	hits := e.Evaluate(log(at.Add(2 * time.Second)))
	if len(hits) != 1 || hits[0].DetectorID != ErrorLogBurst {
		t.Fatalf("%+v", hits)
	}
	info := events.Envelope{
		EventID: uuid.New(), EventType: events.TypeLog, OccurredAt: at.Add(3 * time.Second),
		Payload: map[string]any{"service_id": payments.String(), "severity": "info", "message": "ok"},
	}
	if hits := e.Evaluate(info); len(hits) != 0 {
		t.Fatal("info should not count")
	}
}

func TestDeploymentCorrelation(t *testing.T) {
	e := engine()
	at := time.Date(2026, 9, 14, 4, 12, 0, 0, time.UTC)
	e.Evaluate(events.Envelope{
		EventID: uuid.New(), EventType: events.TypeDeployment, OccurredAt: at,
		Payload: map[string]any{
			"service_id": payments.String(), "deployment_id": "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "version": "1.18.0", "started_at": at.Format(time.RFC3339),
		},
	})
	hits := e.Evaluate(metric("http_request_latency", 1500, at.Add(2*time.Minute)))
	if len(hits) != 1 || hits[0].Labels["deployment_associated"] != "true" {
		t.Fatalf("%+v", hits)
	}
	if hits[0].Labels["deployment_id"] != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("labels %+v", hits[0].Labels)
	}
	late := e.Evaluate(metric("http_request_latency", 1500, at.Add(16*time.Minute)))
	if len(late) != 1 || late[0].Labels["deployment_associated"] == "true" {
		t.Fatalf("outside window %+v", late)
	}
}

func TestTraceDoesNotFireLatency(t *testing.T) {
	e := engine()
	env := events.Envelope{
		EventID: uuid.New(), EventType: events.TypeTrace, OccurredAt: time.Now().UTC(),
		Payload: map[string]any{"service_id": payments.String(), "duration_ms": 5000.0, "status": "error"},
	}
	if hits := e.Evaluate(env); len(hits) != 0 {
		t.Fatalf("%+v", hits)
	}
}

func TestWindowCleanup(t *testing.T) {
	w := NewWindows()
	id := payments
	old := time.Now().UTC().Add(-2 * time.Hour)
	w.ObserveError(id, old)
	w.ObserveDeployment(id, Deployment{ID: "x", StartedAt: old})
	w.Cleanup(time.Now().UTC(), time.Minute)
	if w.ErrorCount(id, old) != 0 {
		t.Fatal("errors should be cleaned")
	}
	if _, ok := w.RecentDeployment(id, time.Now().UTC(), time.Hour); ok {
		t.Fatal("deploys should be cleaned")
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint(HighLatency, payments, "http_request_latency")
	b := Fingerprint(HighLatency, payments, "http_request_latency")
	if a != b || a == "" {
		t.Fatal(a, b)
	}
}
