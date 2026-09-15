package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sentinel-dev/sentinel/services/detection/internal/config"
)

type readyOK struct{}

func (readyOK) Ready(context.Context) error { return nil }

type pingOK struct{}

func (pingOK) Ping(context.Context) error { return nil }

func TestHealthAndRules(t *testing.T) {
	cfg := config.Config{Port: 8091, Rules: config.Rules{LatencyThresholdMS: 1000, ErrorBurstSeverity: "high"}}
	s := New(cfg, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})), pingOK{}, readyOK{})
	rec := httptest.NewRecorder()
	s.HTTP().Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	s.HTTP().Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil))
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	data := body["data"].(map[string]any)
	if data["consumer_group"] != config.ConsumerGroup {
		t.Fatalf("%v", data)
	}
}
