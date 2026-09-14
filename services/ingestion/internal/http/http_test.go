package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sentinel-dev/sentinel/services/ingestion/internal/config"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/ingest"
	"github.com/sentinel-dev/sentinel/services/ingestion/internal/kafka"
)

type silentPub struct{}

func (silentPub) Publish(context.Context, kafka.Message) error { return nil }

type readyOK struct{}

func (readyOK) Ready(context.Context) error { return nil }

func TestHealthAndValidationHTTP(t *testing.T) {
	cfg := config.Config{
		Port:           8090,
		RequestTimeout: 5 * time.Second,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		IdleTimeout:    5 * time.Second,
	}
	svc := ingest.NewService(silentPub{}, ingest.NewMemory())
	s := New(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), svc, readyOK{})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/metric", bytes.NewBufferString(`{"service_slug":"payments-api"}`))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("want 400 got %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "validation_error" {
		t.Fatalf("%s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/metric", bytes.NewBufferString(`{
		"service_slug":"payments-api",
		"name":"payments.capture.latency_p99",
		"value":180,
		"unit":"ms",
		"occurred_at":"2026-09-14T04:18:00Z"
	}`))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("want 202 got %d %s", rec.Code, rec.Body.String())
	}
	var okBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &okBody); err != nil {
		t.Fatal(err)
	}
	data := okBody["data"].(map[string]any)
	event := data["event"].(map[string]any)
	if event["event_type"] != "telemetry.metric" || event["source"] != "services.ingestion" {
		t.Fatalf("envelope %+v", event)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/metric", bytes.NewBufferString(`{"service_slug":"payments-api","name":"n","value":1,"occurred_at":"2026-09-14T04:18:00Z","source":"evil"}`))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("unknown field want 400 got %d", rec.Code)
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := config.Config{
		Port:            0,
		RequestTimeout:  time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
	}
	svc := ingest.NewService(silentPub{}, ingest.NewMemory())
	s := New(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), svc, readyOK{})
	s.HTTP().Addr = "127.0.0.1:0"
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.HTTP().ListenAndServe()
	}()
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.HTTP().Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil && err != http.ErrServerClosed {
		t.Fatal(err)
	}
}
