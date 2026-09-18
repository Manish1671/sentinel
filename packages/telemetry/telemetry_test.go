package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	shutdown, err := Init(context.Background(), Config{ServiceName: "telemetry-test", Environment: "test"})
	if err != nil {
		panic(err)
	}
	code := m.Run()
	_ = shutdown(context.Background())
	os.Exit(code)
}

func TestNormalizeRouteStripsIDs(t *testing.T) {
	got := NormalizeRoute("/api/v1/incidents/22222222-2222-4222-8222-222222222221")
	if got != "/api/v1/incidents/{id}" {
		t.Fatalf("got %s", got)
	}
	if NormalizeRoute("/incidents/INC-2026-0012") != "/incidents/{id}" {
		t.Fatalf("reference: %s", NormalizeRoute("/incidents/INC-2026-0012"))
	}
}

func TestCountAndHTTPMetrics(t *testing.T) {
	Count(context.Background(), DetectionEvents, "outcome", "created")
	Count(context.Background(), DetectionCreated, "severity", "critical")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFrom(r.Context()) == "" {
			t.Error("missing request_id")
		}
		w.WriteHeader(http.StatusTeapot)
	})
	rec := httptest.NewRecorder()
	WrapHTTP(inner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status %d", rec.Code)
	}

	mrec := httptest.NewRecorder()
	WrapHTTP(inner).ServeHTTP(mrec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := mrec.Body.String()
	if !strings.Contains(body, "sentinel_http_requests") && !strings.Contains(body, "sentinel_detection_events_processed") {
		t.Fatalf("metrics body missing counters:\n%s", body)
	}
}

func TestLogFields(t *testing.T) {
	var buf bytes.Buffer
	h := contextHandler{
		Handler: slog.NewJSONHandler(&buf, nil),
		service: "api",
		env:     "test",
	}
	logger := slog.New(h)
	ctx := WithRequestID(context.Background(), "req-1")
	ctx = WithIncidentID(ctx, "inc-1")
	logger.InfoContext(ctx, "hello", "level_check", true)
	var row map[string]any
	if err := json.Unmarshal(buf.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if row["service"] != "api" || row["environment"] != "test" {
		t.Fatalf("service fields: %#v", row)
	}
	if row["request_id"] != "req-1" || row["incident_id"] != "inc-1" {
		t.Fatalf("ids: %#v", row)
	}
	if row["msg"] != "hello" {
		t.Fatalf("message: %#v", row)
	}
}

func TestKafkaHeadersRoundTrip(t *testing.T) {
	ctx, span := Start(WithRequestID(context.Background(), "abc"), "sentinel.test")
	headers := InjectKafka(ctx, nil)
	span.End()
	out := ExtractKafka(context.Background(), headers)
	if RequestIDFrom(out) != "abc" {
		t.Fatalf("request_id %q", RequestIDFrom(out))
	}
	if TraceIDFrom(out) == "" {
		t.Fatal("expected trace_id after extract")
	}
}

func TestDBOpRecordsDurationAndErrors(t *testing.T) {
	err := DBOp(context.Background(), "ping", func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = DBOp(context.Background(), "ping", func(context.Context) error {
		return errTestDB
	})
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "sentinel_db_operation_duration") {
		t.Fatalf("missing db duration:\n%s", body)
	}
	if !strings.Contains(body, "sentinel_db_errors") {
		t.Fatalf("missing db errors:\n%s", body)
	}
}

var errTestDB = errString("db boom")

type errString string

func (e errString) Error() string { return string(e) }

func TestErrorIncrements(t *testing.T) {
	Count(context.Background(), KafkaFailures, "operation", "consume", "topic", "telemetry.metrics")
	Count(context.Background(), AIFailed, "status", "failed")
	Count(context.Background(), RemFailed, "action_type", "rollback_deployment")
	Count(context.Background(), DetectionRules, "rule", "high_latency", "severity", "high")
	Observe(context.Background(), DetectionDuration, 0.01, "outcome", "created")
	time.Sleep(20 * time.Millisecond)
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, name := range []string{
		"sentinel_kafka_processing_failures",
		"sentinel_ai_investigations_failed",
		"sentinel_remediation_failed",
		"sentinel_detection_rule_triggers",
	} {
		if !strings.Contains(body, name) {
			t.Fatalf("missing %s in %s", name, body)
		}
	}
}
