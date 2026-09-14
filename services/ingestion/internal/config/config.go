package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            int
	Environment     string
	LogLevel        string
	KafkaBrokers    []string
	KafkaClientID   string
	RequestTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	PublishTimeout  time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:            8090,
		Environment:     "development",
		LogLevel:        "info",
		KafkaClientID:   "sentinel-ingestion",
		RequestTimeout:  15 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		PublishTimeout:  10 * time.Second,
	}
	if v := first("PORT", "INGESTION_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 65535 {
			return Config{}, fmt.Errorf("invalid PORT %q", v)
		}
		cfg.Port = n
	}
	if v := first("ENVIRONMENT", "APP_ENV"); v != "" {
		cfg.Environment = v
	}
	if v := first("LOG_LEVEL", "APP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := first("KAFKA_CLIENT_ID"); v != "" {
		cfg.KafkaClientID = v
	}
	cfg.KafkaBrokers = splitCSV(first("KAFKA_BROKERS"))
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("missing required configuration: KAFKA_BROKERS")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q", c.LogLevel)
	}
	return nil
}

func first(keys ...string) string {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
