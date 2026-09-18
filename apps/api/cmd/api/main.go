package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sentinel-dev/sentinel/apps/api/internal/auth"
	"github.com/sentinel-dev/sentinel/apps/api/internal/cache"
	"github.com/sentinel-dev/sentinel/apps/api/internal/config"
	"github.com/sentinel-dev/sentinel/apps/api/internal/database"
	"github.com/sentinel-dev/sentinel/apps/api/internal/httpapi"
	"github.com/sentinel-dev/sentinel/apps/api/internal/incidents"
	"github.com/sentinel-dev/sentinel/apps/api/internal/observability"
	"github.com/sentinel-dev/sentinel/apps/api/internal/services"
	"github.com/sentinel-dev/sentinel/packages/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "api: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	telShutdown, err := telemetry.Init(ctx, telemetry.FromEnv("sentinel-api"))
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() { _ = telShutdown(context.Background()) }()
	log := observability.NewLogger(cfg.LogLevel)

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsPath := cfg.MigrationsPath
	if migrationsPath == "" {
		migrationsPath = database.DefaultMigrationsPath()
	}
	migrateOnly := len(os.Args) > 1 && os.Args[1] == "migrate"
	if migrateOnly || cfg.Environment == "development" {
		log.Info("running migrations", "path", migrationsPath)
		if err := database.Migrate(ctx, db, migrationsPath); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	if migrateOnly {
		log.Info("migrations complete")
		return nil
	}

	redisClient, err := cache.Connect(cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = redisClient.Close() }()

	authSvc := auth.NewService(auth.NewRepository(db.Pool), auth.NewTokenService(cfg.AuthTokenSecret, cfg.AuthTokenTTL))
	catalogSvc := services.NewService(services.NewRepository(db.Pool))
	incidentSvc := incidents.NewService(incidents.NewRepository(db.Pool))
	server := httpapi.New(cfg, log, db, redisClient, authSvc, catalogSvc, incidentSvc)

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", server.HTTP().Addr, "env", cfg.Environment)
		if err := server.HTTP().ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return server.HTTP().Shutdown(shutdownCtx)
}
