# OpenTelemetry Collector

Config: `otel-collector.yaml`.

Receives OTLP on:

- gRPC `:4317`
- HTTP `:4318`

Pipelines:

- **traces** → `debug` exporter (stdout of the collector container). There is no Tempo/Jaeger in this phase.
- **metrics** → Prometheus exporter on collector `:8889` (prefixed `otelcol_`). Application metrics are **not** dual-exported here; services expose `/metrics` for Prometheus scrape.

Health: the collector process is distroless (no `wget`). Compose runs `otelcol-contrib validate --config=...` as a liveness-style check that the binary and config are valid. The live extension still listens on **:13133** (published to the host) for `GET /`.

Leave `OTEL_EXPORTER_OTLP_ENDPOINT` empty if the collector is not running. The SDKs still create in-process spans and Prometheus metrics.
