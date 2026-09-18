# Observability

Sentinel emits **logs**, **metrics**, and **traces** from the control-plane API and domain services. Local backends are OpenTelemetry Collector, Prometheus, and Grafana. The Next.js console does not emit server-side OpenTelemetry in this phase.

## What we can answer

- What is happening (structured logs + span names)
- How much is happening (Prometheus counters)
- How long it takes (histograms)
- Where time is spent (spans on HTTP, Kafka, DB ping, detection, correlation, investigation, remediation)
- Where errors occur (error counters + span status)
- How much Kafka work is pending (consumer lag from kafka-go reader stats, when the consumer is live)

## Components

| Service | HTTP `/metrics` | OTLP traces | JSON logs |
| --- | --- | --- | --- |
| `apps/api` | yes | if `OTEL_EXPORTER_OTLP_ENDPOINT` set | yes |
| `services/ingestion` | yes | same | yes |
| `services/detection` | yes | same | yes |
| `services/incident` | yes | same | yes |
| `services/ai` | yes | same | yes |
| `services/remediation` | yes | same | yes |
| `apps/web` | not instrumented | — | Next.js default |

Shared Go SDK: `packages/telemetry`. Python uses `prometheus_client` + the OTel SDK.

## Logs

JSON to stdout. Common fields:

`timestamp`, `service`, `environment`, `level`, `message`, `request_id`, `trace_id` (when a span is active), plus `event_id` / `incident_id` / `investigation_id` / `remediation_id` when the code attaches them.

Never logged: passwords, tokens, API keys, database credentials, full `Authorization` headers, AI prompts, tool raw dumps.

## Metrics

Prometheus names (dots become underscores; counters add `_total`).

Always-on labels: `service`, `environment`. Additional low-cardinality labels: `method`, `route` (IDs replaced with `{id}`), `status`, `topic`, `operation`, `outcome`, `rule`, `severity`, `tool`, `provider`, `action_type`, `kind`.

**Not used as metric labels:** `incident_id`, `event_id`, `request_id`, `user_id`, `investigation_id`, `remediation_id`. Those belong on logs and span attributes for debugging.

### HTTP

`sentinel_http_requests_total`, `sentinel_http_request_duration_seconds`, `sentinel_http_requests_active`

### Kafka

`sentinel_kafka_messages_published_total`, `sentinel_kafka_messages_consumed_total`, `sentinel_kafka_processing_failures_total`, `sentinel_kafka_processing_duration_seconds` (Go consumers), `sentinel_kafka_consumer_lag`

Lag is `Reader.Stats().Lag` in Go. It is **not** invented. Python AI does not expose lag (aiokafka is not wired to a lag gauge).

### Detection / incident / AI / remediation

See metric constants in `packages/telemetry/metrics.go` and `services/ai/app/observability/metrics.py`. Token counters increment **only** when an OpenAI-compatible response includes `usage.prompt_tokens` / `usage.completion_tokens`. The local heuristic provider reports no tokens.

### Database

High-level `sentinel.db.*` duration/errors around ping and selected reads (`get_processed`). Pool acquired connections from pgx `Stat().AcquiredConns()` on ping. SQL text is not logged.

## Traces

Span names include:

`sentinel.http.request`, `sentinel.kafka.publish`, `sentinel.kafka.consume`, `sentinel.detection.evaluate`, `sentinel.incident.correlate`, `sentinel.ai.investigation`, `sentinel.ai.retrieval`, `sentinel.ai.tool`, `sentinel.ai.model`, `sentinel.ai.validate`, `sentinel.db.persist`, `sentinel.remediation.execute`, `sentinel.remediation.verify`, `sentinel.db.<operation>`

### Propagation

- HTTP: W3C `traceparent` via the global propagator
- Kafka: the same W3C headers are injected/extracted on message headers (`request_id` is also copied)

This **is** real Kafka context propagation. End-to-end traces are visible in collector **debug logs**, not in Grafana (no trace backend). Do not treat Grafana as a trace UI.

## Local setup

Core stack (no observability profile):

```bash
docker compose up -d
```

With collector, Prometheus, Grafana:

```bash
# containers talking to the collector
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
docker compose --profile observability up -d
```

Host-run API (optional scrape job `api`): `http://localhost:8080`. Grafana: `http://localhost:3002`. Prometheus: `http://localhost:9090`.

Observability processes are optional. Empty `OTEL_EXPORTER_OTLP_ENDPOINT` disables OTLP export; `/metrics` still works.

## Dashboards

Provisioned under `observability/grafana/dashboards/`. Values come only from scraped instrumentation.

## Known limitations

- No Tempo/Jaeger/Loki. Traces stay on the collector debug exporter. Logs stay on container stdout.
- Grafana does not show traces.
- Kafka lag is Go-only and follows kafka-go’s partition lag, not a broker admin API.
- Dual-export of metrics over OTLP is disabled to avoid double counting; scrape `/metrics`.
- Frontend RUM is deferred.
- `apps/api` is not a Compose service; Prometheus may show the `api` job down until the API runs on the host.
- Active-incident gauge updates when the incident service correlates an alert (query of non-resolved/closed rows). It is not a background poller.
