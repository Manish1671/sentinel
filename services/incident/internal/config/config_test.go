package config

import "testing"

func TestLoadRequiresBrokersAndDatabase(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("DATABASE_URL", "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8092 || cfg.KafkaGroup != ConsumerGroup || cfg.CorrelationWindow.Seconds() != 300 {
		t.Fatalf("%+v", cfg)
	}
}

func TestCorrelationWindowOverride(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("INCIDENT_CORRELATION_WINDOW_SECONDS", "120")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CorrelationWindow.Seconds() != 120 {
		t.Fatalf("%v", cfg.CorrelationWindow)
	}
}
