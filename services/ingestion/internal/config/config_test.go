package config

import "testing"

func TestLoadRequiresBrokers(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadValid(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", "localhost:9092,localhost:9093")
	t.Setenv("PORT", "8090")
	t.Setenv("LOG_LEVEL", "info")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.KafkaBrokers) != 2 || cfg.Port != 8090 {
		t.Fatalf("%+v", cfg)
	}
}
