package incidents

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/sentinel-dev/sentinel/services/incident/internal/config"
	"github.com/sentinel-dev/sentinel/services/incident/internal/database"
	"github.com/sentinel-dev/sentinel/services/incident/internal/events"
	"github.com/sentinel-dev/sentinel/services/incident/internal/kafka"
)

type memPub struct {
	mu   sync.Mutex
	msgs []kafka.Message
}

func (m *memPub) Publish(_ context.Context, msg kafka.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func testStore(t *testing.T) (*database.Store, *database.DB, context.Context) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("postgres: %v", err)
	}
	t.Cleanup(db.Close)
	if err := database.Migrate(ctx, db, database.DefaultMigrationsPath()); err != nil {
		t.Skip(err)
	}
	_ = database.SeedIfEmpty(ctx, db, database.DefaultSeedsPath())
	return database.NewStore(db), db, ctx
}

func insertService(t *testing.T, ctx context.Context, db *database.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	slug := "t" + strings.ReplaceAll(id.String(), "-", "")[:12]
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO services (id, slug, name, environment, description, health_status)
		VALUES ($1, $2, $2, 'production', 'test', 'unknown')
	`, id, slug)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func insertAlert(t *testing.T, ctx context.Context, db *database.DB, serviceID uuid.UUID, detector, severity string, started time.Time, labels map[string]string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	body, _ := json.Marshal(labels)
	if labels == nil {
		body = []byte("{}")
	}
	fp := detector + ":" + id.String()
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO alerts (id, service_id, detector_id, severity, title, summary, fingerprint, labels, started_at)
		VALUES ($1, $2, $3, $4::severity_level, $5, $6, $7, $8::jsonb, $9)
	`, id, serviceID, detector, severity, detector+" alert", "test", fp, body, started)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func alertEnvelope(alertID, serviceID uuid.UUID, detector, severity string, started time.Time, labels map[string]string) ([]byte, uuid.UUID) {
	eid := uuid.New()
	env := events.Envelope{
		EventID:       eid,
		EventType:     events.TypeAlert,
		EventVersion:  1,
		OccurredAt:    started,
		Source:        "services.detection",
		CorrelationID: uuid.New(),
		Payload: map[string]any{
			"alert_id":    alertID.String(),
			"service_id":  serviceID.String(),
			"detector_id": detector,
			"severity":    severity,
			"title":       detector,
			"summary":     "test",
			"fingerprint": detector + ":" + alertID.String(),
			"labels":      labels,
			"started_at":  started.UTC().Format(time.RFC3339Nano),
		},
	}
	raw, _ := json.Marshal(env)
	return raw, eid
}

func TestCorrelateThreeAlertsOneIncident(t *testing.T) {
	store, db, ctx := testStore(t)
	svc := insertService(t, ctx, db)
	pub := &memPub{}
	p := NewProcessor(config.Config{CorrelationWindow: 5 * time.Minute, PublishTimeout: time.Second}, slog.Default(), store, pub)
	now := time.Now().UTC()
	labels := map[string]string{"deployment_id": "d1", "deployment_version": "1.19.0", "deployment_associated": "true"}
	a1 := insertAlert(t, ctx, db, svc, "high_latency", "high", now, labels)
	a2 := insertAlert(t, ctx, db, svc, "high_error_rate", "high", now.Add(time.Second), labels)
	a3 := insertAlert(t, ctx, db, svc, "db_connection_saturation", "critical", now.Add(2*time.Second), labels)

	raw, _ := alertEnvelope(a1, svc, "high_latency", "high", now, labels)
	r1, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 1)
	if err != nil || r1.Outcome != "created" {
		t.Fatalf("create %+v %v", r1, err)
	}
	raw, _ = alertEnvelope(a2, svc, "high_error_rate", "high", now.Add(time.Second), labels)
	r2, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 2)
	if err != nil || r2.Outcome != "attached" || r2.IncidentID != r1.IncidentID {
		t.Fatalf("attach2 %+v %v", r2, err)
	}
	raw, _ = alertEnvelope(a3, svc, "db_connection_saturation", "critical", now.Add(2*time.Second), labels)
	r3, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 3)
	if err != nil || r3.Outcome != "attached" || r3.IncidentID != r1.IncidentID {
		t.Fatalf("attach3 %+v %v", r3, err)
	}

	in, err := store.GetIncident(ctx, r1.IncidentID)
	if err != nil {
		t.Fatal(err)
	}
	if in.Severity != "critical" || in.Status != "open" || len(in.AlertIDs) != 3 {
		t.Fatalf("incident %+v", in)
	}
	tl, err := store.Timeline(ctx, in.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tl) != 4 {
		t.Fatalf("timeline %d %+v", len(tl), tl)
	}
	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.msgs) != 3 {
		t.Fatalf("published %d", len(pub.msgs))
	}
	if pub.msgs[0].Topic != events.TopicCreated {
		t.Fatalf("first topic %s", pub.msgs[0].Topic)
	}
}

func TestDuplicateAlertEventIdempotent(t *testing.T) {
	store, db, ctx := testStore(t)
	svc := insertService(t, ctx, db)
	pub := &memPub{}
	p := NewProcessor(config.Config{CorrelationWindow: 5 * time.Minute}, slog.Default(), store, pub)
	now := time.Now().UTC()
	id := insertAlert(t, ctx, db, svc, "high_latency", "high", now, nil)
	raw, _ := alertEnvelope(id, svc, "high_latency", "high", now, nil)
	r1, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 10)
	if err != nil || r1.Outcome != "created" {
		t.Fatalf("%+v %v", r1, err)
	}
	r2, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 10)
	if err != nil || r2.Outcome != "duplicate" {
		t.Fatalf("replay %+v %v", r2, err)
	}
	in, _ := store.GetIncident(ctx, r1.IncidentID)
	if len(in.AlertIDs) != 1 {
		t.Fatalf("alerts %v", in.AlertIDs)
	}
	tl, _ := store.Timeline(ctx, in.ID)
	if len(tl) != 2 {
		t.Fatalf("timeline %d", len(tl))
	}
}

func TestDifferentServiceNewIncident(t *testing.T) {
	store, db, ctx := testStore(t)
	pub := &memPub{}
	p := NewProcessor(config.Config{CorrelationWindow: 5 * time.Minute}, slog.Default(), store, pub)
	now := time.Now().UTC()
	svc := insertService(t, ctx, db)
	other := insertService(t, ctx, db)
	id := insertAlert(t, ctx, db, svc, "high_latency", "high", now, nil)
	raw, _ := alertEnvelope(id, svc, "high_latency", "high", now, nil)
	r1, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	aid := insertAlert(t, ctx, db, other, "high_latency", "high", now, nil)
	raw2, _ := alertEnvelope(aid, other, "high_latency", "high", now, nil)
	r2, err := p.Handle(ctx, raw2, events.TopicAlerts, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if r2.IncidentID == r1.IncidentID {
		t.Fatal("different services must not share an incident")
	}
}

func TestTransactionRollbackLeavesNoIncident(t *testing.T) {
	store, db, ctx := testStore(t)
	svc := insertService(t, ctx, db)
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	err = database.TxInsertIncident(ctx, tx, database.IncidentRow{
		ID: id, Reference: "INC-2099-0099", ServiceID: svc,
		Title: "rollback", Summary: "x", Severity: "low", DetectedAt: time.Now().UTC(),
	}, "alert:"+id.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = store.GetIncident(ctx, id)
	if err == nil {
		t.Fatal("rolled back incident should not exist")
	}
}

func TestKafkaPublishConsumeIncidentCreated(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	store, db, ctx := testStore(t)
	svc := insertService(t, ctx, db)
	cfg := config.Config{
		KafkaBrokers:      strings.Split(brokers, ","),
		KafkaClientID:     "sentinel-incident-test",
		PublishTimeout:    8 * time.Second,
		CorrelationWindow: 5 * time.Minute,
		DatabaseURL:       os.Getenv("DATABASE_URL"),
	}
	prod := kafka.NewProducer(cfg)
	defer prod.Close()
	readyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := prod.Ready(readyCtx); err != nil {
		t.Skipf("kafka: %v", err)
	}
	p := NewProcessor(cfg, slog.Default(), store, prod)
	now := time.Now().UTC()
	labels := map[string]string{"deployment_associated": "true", "deployment_version": "1.19.0"}
	a1 := insertAlert(t, ctx, db, svc, "high_latency", "high", now, labels)
	a2 := insertAlert(t, ctx, db, svc, "high_error_rate", "high", now.Add(time.Second), labels)
	raw, _ := alertEnvelope(a1, svc, "high_latency", "high", now, labels)
	r1, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 1)
	if err != nil || r1.Outcome != "created" {
		t.Fatalf("%+v %v", r1, err)
	}
	raw, _ = alertEnvelope(a2, svc, "high_error_rate", "high", now.Add(time.Second), labels)
	r2, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 2)
	if err != nil || r2.IncidentID != r1.IncidentID || r2.Outcome != "attached" {
		t.Fatalf("second %+v %v", r2, err)
	}
	raw, _ = alertEnvelope(a1, svc, "high_latency", "high", now, labels)
	r3, err := p.Handle(ctx, raw, events.TopicAlerts, 0, 1)
	if err != nil || r3.Outcome != "duplicate" {
		t.Fatalf("replay %+v %v", r3, err)
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       events.TopicCreated,
		GroupID:     "sentinel-incident-itest-" + r1.IncidentID.String(),
		StartOffset: kafkago.FirstOffset,
		MinBytes:    1,
		MaxBytes:    1e6,
	})
	defer reader.Close()
	deadline, cancelRead := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRead()
	want := r1.IncidentID.String()
	for {
		msg, err := reader.ReadMessage(deadline)
		if err != nil {
			t.Fatalf("consume incidents.created: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(msg.Value, &got); err != nil {
			continue
		}
		payload, _ := got["payload"].(map[string]any)
		if payload["incident_id"] == want && got["event_type"] == events.TypeCreated {
			if string(msg.Key) != want {
				t.Fatalf("key %s", msg.Key)
			}
			return
		}
	}
}
