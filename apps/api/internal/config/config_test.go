package config

import (
	"os"
	"testing"
)

func TestLoadMissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_TOKEN_SECRET", "")
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("AUTH_TOKEN_SECRET")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadValid(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable")
	t.Setenv("AUTH_TOKEN_SECRET", "local-dev-token-secret")
	t.Setenv("PORT", "8080")
	t.Setenv("ENVIRONMENT", "test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("port=%d", cfg.Port)
	}
	if len(cfg.KafkaBrokers) != 1 {
		t.Fatalf("brokers=%v", cfg.KafkaBrokers)
	}
}

func TestValidateSecretLength(t *testing.T) {
	cfg := Config{
		DatabaseURL:     "postgres://localhost/db",
		AuthTokenSecret: "short",
		LogLevel:        "info",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected secret length error")
	}
}
