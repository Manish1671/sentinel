package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ConsumerGroup    = "sentinel-detection-v1"
	TopicTelemetry   = "telemetry.events"
	TopicDeployments = "deployments.created"
	TopicAlerts      = "alerts.created"
)

type Config struct {
	Port            int
	Environment     string
	LogLevel        string
	DatabaseURL     string
	MigrationsPath  string
	KafkaBrokers    []string
	KafkaClientID   string
	KafkaGroup      string
	ShutdownTimeout time.Duration
	PublishTimeout  time.Duration
	Rules           Rules
}

type Rules struct {
	LatencyThresholdMS          float64
	ErrorRateThreshold          float64
	DBConnectionThreshold       float64
	ErrorBurstCount             int
	ErrorBurstWindow            time.Duration
	DeploymentCorrelationWindow time.Duration
	AlertCooldown               time.Duration
	ErrorBurstSeverity          string
}

func Load() (Config, error) {
	cfg := Config{
		Port:            8091,
		Environment:     "development",
		LogLevel:        "info",
		KafkaClientID:   "sentinel-detection",
		KafkaGroup:      ConsumerGroup,
		ShutdownTimeout: 20 * time.Second,
		PublishTimeout:  10 * time.Second,
		Rules: Rules{
			LatencyThresholdMS:          1000,
			ErrorRateThreshold:          0.05,
			DBConnectionThreshold:       0.90,
			ErrorBurstCount:             5,
			ErrorBurstWindow:            60 * time.Second,
			DeploymentCorrelationWindow: 15 * time.Minute,
			AlertCooldown:               5 * time.Minute,
			ErrorBurstSeverity:          "high",
		},
	}
	if v := first("PORT", "DETECTION_PORT"); v != "" {
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

	if err := applyRuleEnv(&cfg.Rules); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyRuleEnv(r *Rules) error {
	if err := floatEnv("LATENCY_THRESHOLD_MS", &r.LatencyThresholdMS); err != nil {
		return err
	}
	if err := floatEnv("ERROR_RATE_THRESHOLD", &r.ErrorRateThreshold); err != nil {
		return err
	}
	if err := floatEnv("DB_CONNECTION_THRESHOLD", &r.DBConnectionThreshold); err != nil {
		return err
	}
	if v := first("ERROR_BURST_COUNT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid ERROR_BURST_COUNT %q", v)
		}
		r.ErrorBurstCount = n
	}
	if err := secondsEnv("ERROR_BURST_WINDOW_SECONDS", &r.ErrorBurstWindow); err != nil {
		return err
	}
	if err := secondsEnv("DEPLOYMENT_CORRELATION_WINDOW_SECONDS", &r.DeploymentCorrelationWindow); err != nil {
		return err
	}
	if err := secondsEnv("ALERT_COOLDOWN_SECONDS", &r.AlertCooldown); err != nil {
		return err
	}
	return nil
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

func floatEnv(key string, dest *float64) error {
	v := first(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("invalid %s %q", key, v)
	}
	*dest = n
	return nil
}

func secondsEnv(key string, dest *time.Duration) error {
	v := first(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fmt.Errorf("invalid %s %q", key, v)
	}
	*dest = time.Duration(n) * time.Second
	return nil
}
