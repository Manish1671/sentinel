package detection

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/database"
	"github.com/sentinel-dev/sentinel/services/detection/internal/events"
	"github.com/sentinel-dev/sentinel/services/detection/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/detection/internal/rules"
)

func TestPublishConsumeAlertCreated(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cfg := config.Config{
		KafkaBrokers:   strings.Split(brokers, ","),
		KafkaClientID:  "sentinel-detection-test",
		PublishTimeout: 8 * time.Second,
		DatabaseURL:    dbURL,
		Rules:          config.Rules{LatencyThresholdMS: 1000, AlertCooldown: time.Minute},
	}
	prod := kafka.NewProducer(cfg)
	defer prod.Close()
	if err := prod.Ready(ctx); err != nil {
		t.Skipf("kafka: %v", err)
	}
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("postgres: %v", err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, database.DefaultMigrationsPath()); err != nil {
		t.Skip(err)
	}
	_ = database.SeedIfEmpty(ctx, db, database.DefaultSeedsPath())

	store := database.NewStore(db)
	p := NewProcessor(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), store, rules.NewEngine(cfg.Rules, rules.NewWindows()), prod)
	env := events.Envelope{
		EventID:       uuid.New(),
		EventType:     events.TypeMetric,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: uuid.New(),
		Payload: map[string]any{
			"service_id":   "22222222-2222-4222-8222-222222222221",
			"service_slug": "payments-api",
			"name":         "http_request_latency",
			"value":        1700.0,
		},
	}
	raw, _ := json.Marshal(env)
	res, err := p.Handle(ctx, raw, events.TopicTelemetry, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != "created" || len(res.AlertIDs) == 0 {
		t.Fatalf("%+v", res)
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       events.TopicAlerts,
		GroupID:     "sentinel-detection-itest-" + res.AlertIDs[0].String(),
		StartOffset: kafkago.FirstOffset,
		MinBytes:    1,
		MaxBytes:    1e6,
	})
	defer reader.Close()
	deadline, cancelRead := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRead()
	want := res.AlertIDs[0].String()
	for {
		msg, err := reader.ReadMessage(deadline)
		if err != nil {
			t.Fatalf("consume alerts.created: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(msg.Value, &got); err != nil {
			continue
		}
		payload, _ := got["payload"].(map[string]any)
		if payload["alert_id"] == want && got["event_type"] == events.TypeAlert {
			if string(msg.Key) != "22222222-2222-4222-8222-222222222221" {
				t.Fatalf("key %s", msg.Key)
			}
			return
		}
	}
}
