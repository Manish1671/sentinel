package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sentinel-dev/sentinel/services/incident/internal/config"
)

type readyOK struct{}

func (readyOK) Ready(context.Context) error { return nil }

type pingOK struct{}

func (pingOK) Ping(context.Context) error { return nil }

func TestHealthAndReady(t *testing.T) {
	cfg := config.Config{Port: 8092}
	s := New(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), nil, pingOK{}, readyOK{})
	rec := httptest.NewRecorder()
	s.HTTP().Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	s.HTTP().Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if rec.Code != 200 {
		t.Fatalf("ready %d %s", rec.Code, rec.Body.String())
	}
}
