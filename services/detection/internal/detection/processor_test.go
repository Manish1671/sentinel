package detection

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
	"github.com/sentinel-dev/sentinel/services/detection/internal/database"
	"github.com/sentinel-dev/sentinel/services/detection/internal/events"
	"github.com/sentinel-dev/sentinel/services/detection/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/detection/internal/rules"
)

func TestPoisonCommitsWithoutPanic(t *testing.T) {
	p := NewProcessor(config.Config{}, slog.New(slog.NewTextHandler(os.Stdout, nil)), nil, rules.NewEngine(config.Rules{}, rules.NewWindows()), stubPub{})
	res, err := p.Handle(context.Background(), []byte("{"), "telemetry.events", 0, 1)
	if err != nil || !res.Commit || res.Outcome != "poison" {
		t.Fatalf("%+v %v", res, err)
	}
}

type stubPub struct{ n int }

func (s stubPub) Publish(context.Context, kafka.Message) error { return nil }

func liveDB(t *testing.T) *database.DB {
	t.Helper()
	urls := []string{os.Getenv("DATABASE_URL"),
		"postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable",
		"postgres://sentinel:sentinel@localhost:5433/sentinel?sslmode=disable",
	}
	var last error
	for _, url := range urls {
		if url == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		db, err := database.Connect(ctx, url)
		cancel()
		if err != nil {
			last = err
			continue
		}
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		t.Cleanup(db.Close)
		if err := database.Migrate(ctx, db, database.DefaultMigrationsPath()); err != nil {
			last = err
			db.Close()
			continue
		}
		_ = database.SeedIfEmpty(ctx, db, database.DefaultSeedsPath())
		return db
	}
	t.Skipf("postgres: %v", last)
	return nil
}

func TestAlertInsertAndEventDedup(t *testing.T) {
	db := liveDB(t)
	store := database.NewStore(db)
	cfg := config.Config{Rules: config.Rules{LatencyThresholdMS: 1000, AlertCooldown: 5 * time.Minute}}
	eng := rules.NewEngine(cfg.Rules, rules.NewWindows())
	pub := &countingPub{}
	p := NewProcessor(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), store, eng, pub)

	env := events.Envelope{
		EventID:       uuid.New(),
		EventType:     events.TypeMetric,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: uuid.New(),
		Payload: map[string]any{
			"service_id":   "22222222-2222-4222-8222-222222222221",
			"service_slug": "payments-api",
			"name":         "http_request_latency",
			"value":        1500.0,
		},
	}
	raw, _ := json.Marshal(env)
	res, err := p.Handle(context.Background(), raw, "telemetry.events", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != "created" || res.NewPublish != 1 {
		t.Fatalf("%+v", res)
	}
	if pub.n != 1 {
		t.Fatalf("published %d", pub.n)
	}
	res2, err := p.Handle(context.Background(), raw, "telemetry.events", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Outcome != "duplicate" {
		t.Fatalf("dedup %+v", res2)
	}
	if pub.n != 1 {
		t.Fatalf("duplicate published %d", pub.n)
	}
	n, err := store.CountOpenByFingerprint(context.Background(), uuid.MustParse("22222222-2222-4222-8222-222222222221"), res.AlertIDs[0].String())
	if err != nil {
		t.Fatal(err)
	}
	_ = n
	open, err := store.CountOpenByFingerprint(context.Background(), uuid.MustParse("22222222-2222-4222-8222-222222222221"), rules.Fingerprint(rules.HighLatency, uuid.MustParse("22222222-2222-4222-8222-222222222221"), "http_request_latency"))
	if err != nil || open != 1 {
		t.Fatalf("open=%d err=%v", open, err)
	}
}

type countingPub struct{ n int }

func (c *countingPub) Publish(context.Context, kafka.Message) error {
	c.n++
	return nil
}
