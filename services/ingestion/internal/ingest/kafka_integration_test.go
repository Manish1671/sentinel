package ingest

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/config"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/events"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/kafka"
)

func TestPublishAndConsumeMetric(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	cfg := config.Config{
		KafkaBrokers:   strings.Split(brokers, ","),
		PublishTimeout: 10 * time.Second,
	}
	p := kafka.NewProducer(cfg)
	defer p.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := p.Ready(ctx); err != nil {
		t.Skipf("kafka not available: %v", err)
	}

	svc := NewService(p, NewMemory())
	res, err := svc.IngestMetric(ctx, MetricRequest{
		ServiceSlug: "payments-api",
		Name:        "payments.capture.latency_p99",
		Value:       value(42),
		Unit:        "ms",
		OccurredAt:  time.Now().UTC().Format(time.RFC3339),
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       events.TopicTelemetry,
		GroupID:     "sentinel-ingestion-test-" + res.Envelope.EventID.String(),
		StartOffset: kafkago.FirstOffset,
		MinBytes:    1,
		MaxBytes:    1e6,
	})
	defer reader.Close()
	deadline, cancelRead := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRead()
	for {
		msg, err := reader.ReadMessage(deadline)
		if err != nil {
			t.Fatalf("consume: %v", err)
		}
		var env map[string]any
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			continue
		}
		if env["event_id"] == res.Envelope.EventID.String() {
			if env["event_type"] != events.TypeMetric {
				t.Fatalf("type %v", env["event_type"])
			}
			if string(msg.Key) != "22222222-2222-4222-8222-222222222221" {
				t.Fatalf("key %s", msg.Key)
			}
			return
		}
	}
}
