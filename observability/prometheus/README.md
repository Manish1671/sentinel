# Prometheus

Config: `prometheus.yml`.

Local flags (Compose): 10s scrape, **24h** TSDB retention, lifecycle API enabled.

## Scrape targets

| Job | Target | Notes |
| --- | --- | --- |
| prometheus | `localhost:9090` | Self |
| otel-collector | `otel-collector:8888` | Collector internal metrics |
| ingestion | `ingestion:8090/metrics` | Compose network |
| detection | `detection:8091/metrics` | |
| incident | `incident:8092/metrics` | |
| ai | `ai:8000/metrics` | |
| remediation | `remediation:8093/metrics` | |
| api | `host.docker.internal:8080/metrics` | Optional; DOWN if the API is not on the host |

## Important series

Names use OpenTelemetry Prometheus mapping (`.` → `_`, counters get `_total`):

- `sentinel_http_requests_total`
- `sentinel_http_request_duration_seconds_*`
- `sentinel_kafka_messages_published_total` / `_consumed_total` / `sentinel_kafka_processing_failures_total`
- `sentinel_kafka_consumer_lag` (from kafka-go `Reader.Stats().Lag` when the consumer is running)
- `sentinel_detection_*`, `sentinel_incident_*`, `sentinel_ai_*`, `sentinel_remediation_*`

Do not scrape Grafana. Tests must not require Prometheus to be up.
