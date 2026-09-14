package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/config"
	httpapi "github.com/sentinel-dev/sentinel/services/ingestion/internal/http"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/ingest"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/kafka"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/observability"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ingestion: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := observability.NewLogger(cfg.LogLevel)
	producer := kafka.NewProducer(cfg)
	defer func() { _ = producer.Close() }()

	svc := ingest.NewService(producer, ingest.NewMemory())
	server := httpapi.New(cfg, log, svc, producer)

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", server.HTTP().Addr, "brokers", cfg.KafkaBrokers)
		if err := server.HTTP().ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		close(errCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Info("shutting down", "signal", sig.String())
	case err := <-errCh:
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return server.HTTP().Shutdown(ctx)
}
