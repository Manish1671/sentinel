# Observability

Local OpenTelemetry Collector, Prometheus, and Grafana for Sentinel.

Core application Compose services do **not** require this stack. Start it with the `observability` profile:

```bash
docker compose --profile observability up -d
```

To export traces from services to the collector, set:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

in `.env` for containers, or `http://localhost:4318` for processes on the host.

Metrics are always available on each service’s `/metrics` endpoint. Prometheus scrapes those endpoints. Grafana dashboards read Prometheus only — they are not seeded with fake data.

| Component | Port |
| --- | --- |
| OTel Collector OTLP gRPC | 4317 |
| OTel Collector OTLP HTTP | 4318 |
| Prometheus | 9090 |
| Grafana | 3002 |

Default Grafana login: `admin` / `admin`. Anonymous viewer access is enabled for local use.

Details:

- [otel/](otel/README.md)
- [prometheus/](prometheus/README.md)
- [grafana/](grafana/README.md)
- Architecture: [docs/architecture/observability.md](../docs/architecture/observability.md)
