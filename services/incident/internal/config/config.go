package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ConsumerGroup   = "sentinel-incident-v1"
	TopicAlerts     = "alerts.created"
	TopicCreated    = "incidents.created"
	TopicUpdated    = "incidents.updated"
)

type Config struct {
	Port              int
	Environment       string
	LogLevel          string
	DatabaseURL       string
	MigrationsPath    string
	KafkaBrokers      []string
	KafkaClientID     string
	KafkaGroup        string
	ShutdownTimeout   time.Duration
	PublishTimeout    time.Duration
	CorrelationWindow time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:              8092,
		Environment:       "development",
		LogLevel:          "info",
		KafkaClientID:     "sentinel-incident",
		KafkaGroup:        ConsumerGroup,
		ShutdownTimeout:   20 * time.Second,
		PublishTimeout:    10 * time.Second,
		CorrelationWindow: 300 * time.Second,
	}
	if v := first("PORT", "INCIDENT_PORT"); v != "" {
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
	cfg.DatabaseURL = first("DATABASE_URL")
	cfg.MigrationsPath = first("MIGRATIONS_PATH")
	if v := first("KAFKA_CLIENT_ID"); v != "" {
		cfg.KafkaClientID = v
	}
	if v := first("KAFKA_CONSUMER_GROUP"); v != "" {
		cfg.KafkaGroup = v
	}
	cfg.KafkaBrokers = splitCSV(first("KAFKA_BROKERS"))
	if v := first("INCIDENT_CORRELATION_WINDOW_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("invalid INCIDENT_CORRELATION_WINDOW_SECONDS %q", v)
		}
		cfg.CorrelationWindow = time.Duration(n) * time.Second
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("missing required configuration: DATABASE_URL")
	}
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
