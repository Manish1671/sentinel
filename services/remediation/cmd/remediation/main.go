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

	"github.com/sentinel-dev/sentinel/services/remediation/internal/config"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/database"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/executor"
	httpapi "github.com/sentinel-dev/sentinel/services/remediation/internal/http"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/observability"
	"github.com/sentinel-dev/sentinel/services/remediation/internal/remediation"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "remediation: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	telShutdown, err := telemetry.Init(context.Background(), telemetry.FromEnv("sentinel-remediation"))
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

	producer := kafka.NewProducer(cfg)
	defer func() { _ = producer.Close() }()
	consumer := kafka.NewConsumer(cfg)
	defer func() { _ = consumer.Close() }()

	store := database.NewStore(db)
	sim := executor.NewSimulator(db)
	proc := remediation.NewProcessor(cfg, log, store, sim, producer)
	server := httpapi.New(cfg, log, store, db, producer, proc, sim)

	errCh := make(chan error, 2)
	go func() {
		log.Info("listening", "addr", server.HTTP().Addr, "group", cfg.KafkaGroup)
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

func consumeLoop(ctx context.Context, log *slog.Logger, consumer *kafka.Consumer, proc *remediation.Processor) error {
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
		for {
			res, err := proc.Handle(msgCtx, msg.Value, msg.Topic, msg.Partition, msg.Offset)
			if err != nil {
				telemetry.Count(msgCtx, telemetry.KafkaFailures, "operation", "consume", "topic", msg.Topic)
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
			telemetry.Count(msgCtx, telemetry.KafkaConsumed, "topic", msg.Topic, "operation", "consume")
			telemetry.Observe(msgCtx, telemetry.KafkaDuration, time.Since(start).Seconds(), "topic", msg.Topic, "operation", "consume")
			telemetry.Gauge(msgCtx, telemetry.KafkaLag, float64(consumer.Lag()), "topic", msg.Topic)
			break
		}
	}
}
