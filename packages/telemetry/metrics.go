package telemetry

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const MeterName = "sentinel"

var (
	counterMu sync.Mutex
	counters  = map[string]metric.Int64Counter{}
	histos    = map[string]metric.Float64Histogram{}
	updowns   = map[string]metric.Int64UpDownCounter{}
	gauges    = map[string]metric.Float64Gauge{}
)

func Attrs(kv ...string) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, 2+len(kv)/2)
	out = append(out,
		attribute.String("service", ServiceName()),
		attribute.String("environment", Environment()),
	)
	for i := 0; i+1 < len(kv); i += 2 {
		out = append(out, attribute.String(kv[i], kv[i+1]))
	}
	return out
}

func Count(ctx context.Context, name string, kv ...string) {
	c := counter(name)
	c.Add(ctx, 1, metric.WithAttributes(Attrs(kv...)...))
}

func Add(ctx context.Context, name string, n int64, kv ...string) {
	c := counter(name)
	c.Add(ctx, n, metric.WithAttributes(Attrs(kv...)...))
}

func Observe(ctx context.Context, name string, seconds float64, kv ...string) {
	h := histogram(name)
	h.Record(ctx, seconds, metric.WithAttributes(Attrs(kv...)...))
}

func IncrActive(ctx context.Context, name string, delta int64, kv ...string) {
	u := updown(name)
	u.Add(ctx, delta, metric.WithAttributes(Attrs(kv...)...))
}

func Gauge(ctx context.Context, name string, value float64, kv ...string) {
	g := gauge(name)
	g.Record(ctx, value, metric.WithAttributes(Attrs(kv...)...))
}

func counter(name string) metric.Int64Counter {
	counterMu.Lock()
	defer counterMu.Unlock()
	if c, ok := counters[name]; ok {
		return c
	}
	c, _ := otel.Meter(MeterName).Int64Counter(name)
	counters[name] = c
	return c
}

func histogram(name string) metric.Float64Histogram {
	counterMu.Lock()
	defer counterMu.Unlock()
	if h, ok := histos[name]; ok {
		return h
	}
	h, _ := otel.Meter(MeterName).Float64Histogram(name, metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(
		0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
	))
	histos[name] = h
	return h
}

func updown(name string) metric.Int64UpDownCounter {
	counterMu.Lock()
	defer counterMu.Unlock()
	if u, ok := updowns[name]; ok {
		return u
	}
	u, _ := otel.Meter(MeterName).Int64UpDownCounter(name)
	updowns[name] = u
	return u
}

func gauge(name string) metric.Float64Gauge {
	counterMu.Lock()
	defer counterMu.Unlock()
	if g, ok := gauges[name]; ok {
		return g
	}
	g, _ := otel.Meter(MeterName).Float64Gauge(name)
	gauges[name] = g
	return g
}

const (
	HTTPRequests        = "sentinel.http.requests"
	HTTPDuration        = "sentinel.http.request.duration"
	HTTPActive          = "sentinel.http.requests.active"
	KafkaPublished      = "sentinel.kafka.messages_published"
	KafkaConsumed       = "sentinel.kafka.messages_consumed"
	KafkaFailures       = "sentinel.kafka.processing_failures"
	KafkaLag            = "sentinel.kafka.consumer.lag"
	KafkaDuration       = "sentinel.kafka.processing.duration"
	DetectionEvents     = "sentinel.detection.events_processed"
	DetectionCreated    = "sentinel.detection.alerts_created"
	DetectionSuppressed = "sentinel.detection.alerts_suppressed"
	DetectionDuration   = "sentinel.detection.duration"
	DetectionRules      = "sentinel.detection.rule_triggers"
	IncidentCorrelated  = "sentinel.incident.alerts_correlated"
	IncidentCreated     = "sentinel.incident.incidents_created"
	IncidentAttached    = "sentinel.incident.alerts_attached"
	IncidentFailures    = "sentinel.incident.correlation_failures"
	IncidentDuration    = "sentinel.incident.duration"
	AIStarted           = "sentinel.ai.investigations_started"
	AICompleted         = "sentinel.ai.investigations_completed"
	AIFailed            = "sentinel.ai.investigations_failed"
	AIDuration          = "sentinel.ai.investigation.duration"
	AIToolCalls         = "sentinel.ai.tool_calls"
	AIToolFailures      = "sentinel.ai.tool_failures"
	AIModelRequests     = "sentinel.ai.model_requests"
	AIModelFailures     = "sentinel.ai.model_failures"
	AITokens            = "sentinel.ai.tokens"
	RemRequested        = "sentinel.remediation.requested"
	RemApproved         = "sentinel.remediation.approved"
	RemRejected         = "sentinel.remediation.rejected"
	RemStarted          = "sentinel.remediation.started"
	RemSucceeded        = "sentinel.remediation.succeeded"
	RemFailed           = "sentinel.remediation.failed"
	RemVerifyPassed     = "sentinel.remediation.verification_passed"
	RemVerifyFailed     = "sentinel.remediation.verification_failed"
	RemDuration         = "sentinel.remediation.duration"
	DBDuration          = "sentinel.db.operation.duration"
	DBErrors            = "sentinel.db.errors"
	DBPoolAcquired      = "sentinel.db.pool.acquired"
	IncidentActive      = "sentinel.incident.active"
)

func RecordPool(ctx context.Context, acquired float64) {
	Gauge(ctx, DBPoolAcquired, acquired, "operation", "pool")
}
