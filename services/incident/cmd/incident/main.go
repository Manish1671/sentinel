package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sentinel-dev/sentinel/services/incident/internal/config"
	"github.com/sentinel-dev/sentinel/services/incident/internal/database"
	httpapi "github.com/sentinel-dev/sentinel/services/incident/internal/http"
	"github.com/sentinel-dev/sentinel/services/incident/internal/incidents"
	"github.com/sentinel-dev/sentinel/services/incident/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/incident/internal/observability"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "incident: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	telShutdown, err := telemetry.Init(context.Background(), telemetry.FromEnv("sentinel-incident"))
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() { _ = telShutdown(context.Background()) }()
	log := observability.NewLogger(cfg.LogLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	mig := cfg.MigrationsPath
	if mig == "" {
		mig = database.DefaultMigrationsPath()
	}
	if err := database.Migrate(ctx, db, mig); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if os.Getenv("SEED_ON_START") == "true" {
		seeds := os.Getenv("SEEDS_PATH")
		if seeds == "" {
			seeds = database.DefaultSeedsPath()
		}
		if err := database.SeedIfEmpty(ctx, db, seeds); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}

	producer := kafka.NewProducer(cfg)
	defer func() { _ = producer.Close() }()
	consumer := kafka.NewConsumer(cfg)
	defer func() { _ = consumer.Close() }()

	store := database.NewStore(db)
	proc := incidents.NewProcessor(cfg, log, store, producer)
	server := httpapi.New(cfg, log, store, db, producer)

	errCh := make(chan error, 2)
	go func() {
		log.Info("listening", "addr", server.HTTP().Addr, "group", cfg.KafkaGroup, "window_s", int(cfg.CorrelationWindow.Seconds()))
		if err := server.HTTP().ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	consumeCtx, consumeCancel := context.WithCancel(context.Background())
	go func() {
		if err := consumeLoop(consumeCtx, log, consumer, proc); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			consumeCancel()
			return err
		}
	}
	consumeCancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutCancel()
	_ = consumer.Close()
	return server.HTTP().Shutdown(shutCtx)
}

func consumeLoop(ctx context.Context, log *slog.Logger, consumer *kafka.Consumer, proc *incidents.Processor) error {
	for {
		msg, err := consumer.Fetch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		msgCtx := consumer.Context(ctx, msg)
		start := time.Now()
		var loopErr error
		for {
			res, err := proc.Handle(msgCtx, msg.Value, msg.Topic, msg.Partition, msg.Offset)
			if err != nil {
				loopErr = err
				log.Error("process_failed", "error", err.Error(), "kafka_topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second):
				}
				continue
			}
			if res.Commit {
				if err := consumer.Commit(msgCtx, msg); err != nil {
					return err
				}
			}
			break
		}
		if loopErr != nil {
			telemetry.Count(msgCtx, telemetry.KafkaFailures, "operation", "consume", "topic", msg.Topic)
		} else {
			telemetry.Count(msgCtx, telemetry.KafkaConsumed, "topic", msg.Topic, "operation", "consume")
		}
		telemetry.Observe(msgCtx, telemetry.KafkaDuration, time.Since(start).Seconds(), "topic", msg.Topic, "operation", "consume")
		telemetry.Gauge(msgCtx, telemetry.KafkaLag, float64(consumer.Lag()), "topic", msg.Topic)
	}
}
