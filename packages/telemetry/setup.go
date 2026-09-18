package telemetry

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Config struct {
	ServiceName    string
	Environment    string
	OTLPEndpoint   string
	MetricInterval time.Duration
}

type ShutdownFunc func(context.Context) error

var (
	mu          sync.Mutex
	initialized bool
	promHandle  http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# sentinel telemetry not initialized\n"))
	})
	serviceName = "sentinel"
	environment = "development"
)

func ServiceName() string { return serviceName }
func Environment() string { return environment }

func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promHandle.ServeHTTP(w, r)
	})
}

func FromEnv(defaultService string) Config {
	name := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME"))
	if name == "" {
		name = defaultService
	}
	env := strings.TrimSpace(firstEnv("ENVIRONMENT", "APP_ENV"))
	if env == "" {
		env = "development"
	}
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	return Config{
		ServiceName:    name,
		Environment:    env,
		OTLPEndpoint:   endpoint,
		MetricInterval: 15 * time.Second,
	}
}

func Init(ctx context.Context, cfg Config) (ShutdownFunc, error) {
	mu.Lock()
	defer mu.Unlock()
	if cfg.ServiceName == "" {
		cfg.ServiceName = "sentinel"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.MetricInterval <= 0 {
		cfg.MetricInterval = 15 * time.Second
	}
	serviceName = cfg.ServiceName
	environment = cfg.Environment

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	registry := prometheus.NewRegistry()
	promExp, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}
	promHandle = promhttp.HandlerFor(registry, promhttp.HandlerOpts{EnableOpenMetrics: true})

	metricOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(promExp),
	}
	var shutdowns []ShutdownFunc

	if cfg.OTLPEndpoint != "" && os.Getenv("OTEL_SDK_DISABLED") != "true" {
		texp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(texp),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, texp.Shutdown, tp.Shutdown)
	} else {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		otel.SetTracerProvider(tp)
		shutdowns = append(shutdowns, tp.Shutdown)
	}

	mp := sdkmetric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	initialized = true
	return func(ctx context.Context) error {
		var first error
		for i := len(shutdowns) - 1; i >= 0; i-- {
			if err := shutdowns[i](ctx); err != nil && first == nil {
				first = err
			}
		}
		return first
	}, nil
}

func Initialized() bool {
	mu.Lock()
	defer mu.Unlock()
	return initialized
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}
