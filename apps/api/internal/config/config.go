package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	KafkaBrokers    []string
	Port            int
	Environment     string
	LogLevel        string
	MigrationsPath  string
	AuthTokenSecret string
	AuthTokenTTL    time.Duration
	RemediationURL  string
	CookieSecure    bool
	RequestTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:     firstEnv("DATABASE_URL"),
		RedisURL:        firstEnv("REDIS_URL"),
		KafkaBrokers:    splitCSV(firstEnv("KAFKA_BROKERS")),
		Port:            8080,
		Environment:     "development",
		LogLevel:        "info",
		MigrationsPath:  firstEnv("MIGRATIONS_PATH"),
		AuthTokenSecret: firstEnv("AUTH_TOKEN_SECRET"),
		AuthTokenTTL:    12 * time.Hour,
		RemediationURL:  firstEnv("REMEDIATION_URL"),
		RequestTimeout:  30 * time.Second,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	}

	if v := firstEnv("ENVIRONMENT", "APP_ENV"); v != "" {
		cfg.Environment = v
	}
	if v := firstEnv("LOG_LEVEL", "APP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := firstEnv("PORT", "API_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("invalid PORT %q", v)
		}
		cfg.Port = port
	}
	cfg.CookieSecure = cfg.Environment == "production" || cfg.Environment == "staging"
	if v := firstEnv("AUTH_TOKEN_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("invalid AUTH_TOKEN_TTL %q", v)
		}
		cfg.AuthTokenTTL = d
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.DatabaseURL) == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if strings.TrimSpace(c.AuthTokenSecret) == "" {
		missing = append(missing, "AUTH_TOKEN_SECRET")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q (want debug, info, warn, or error)", c.LogLevel)
	}
	if len(c.AuthTokenSecret) < 16 {
		return fmt.Errorf("AUTH_TOKEN_SECRET must be at least 16 characters")
	}
	return nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v, ok := os.LookupEnv(key); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
