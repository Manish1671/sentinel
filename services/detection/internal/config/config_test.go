package config

import (
	"os"
	"testing"
)

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
	if cfg.Rules.LatencyThresholdMS != 1000 || cfg.KafkaGroup != ConsumerGroup {
		t.Fatalf("%+v", cfg)
	}
}

func TestRuleEnvOverrides(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("LATENCY_THRESHOLD_MS", "250")
	t.Setenv("ERROR_BURST_COUNT", "9")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Rules.LatencyThresholdMS != 250 || cfg.Rules.ErrorBurstCount != 9 {
		t.Fatalf("%+v", cfg.Rules)
	}
	_ = os.Unsetenv
}
