# services/incident

Consumes `alert.created`, correlates related alerts into operational Incidents, and publishes `incident.created` / `incident.updated`. It does not run AI investigation or remediation.

Detection answers “is this alert-worthy?”. This service answers “which alerts belong to the same operational incident?”.

## Architecture

```
alerts.created
        →  validate envelope (event_id / alert_id)
        →  correlate (deterministic, same service + env + window + compatible evidence)
        →  PostgreSQL incidents / incident_alerts / incident_events / incident_processed_events
        →  incidents.created  (new incident)
        →  incidents.updated  (new alert attached)
```

`source` for published events is always `services.incident`. `causation_id` is the triggering `alert.created` `event_id`. Kafka message key is `incident_id`.

## Correlation rules

An incoming alert attaches to an **active** incident (`open`, `investigating`, `remediating`, `verifying`) when all of the following hold:

1. Same `service_id`
2. Same environment (from the service catalog; a service row is one environment)
3. Alert `started_at` within `INCIDENT_CORRELATION_WINDOW_SECONDS` of the incident’s last alert `started_at` (or `detected_at` if none)
4. Compatible evidence: shared **degradation family** **or** shared `deployment_id`

Degradation family: `high_latency`, `high_error_rate`, `db_connection_saturation`, `error_log_burst`.

`deployment_associated` is a Detection **label**, not a detector. When present, deployment id/version are stored on the timeline. This service does **not** treat a deployment as root cause.

Alerts on the same service are **not** merged only because they share a service. An unrelated detector (for example TLS expiry) in the same window becomes a **new** incident unless it shares deployment context.

If no candidate matches, a new incident is created with status `open`. Correlation never moves status to `investigating` or `resolved`.

## Correlation window

Default: **300 seconds** (`INCIDENT_CORRELATION_WINDOW_SECONDS`).

Seeded historical incidents (for example `INC-2026-0004`) stay outside a 5-minute window of a live simulator burst, so they are not reused.

## Incident lifecycle (this phase)

`open` → `investigating` → `remediating` → `verifying` → `resolved` → `closed`

New correlated incidents start at **`open`**. Severity may change. Status does not auto-advance.

## Alert association

Alerts are never deleted or merged. The join table is `incident_alerts`. Unique `alert_id` means an alert belongs to at most one incident. The composite primary key still prevents the same pair twice.

## Severity aggregation

`critical > high > medium > low`

`incident.severity = max(current, newly attached alert)`. Severity is never decreased automatically.

Title is deterministic, for example `payments-api production degradation`. Title stays stable after create; summary is updated as alerts attach.

## Idempotency

Kafka is at-least-once. Source of truth is PostgreSQL, not process memory.

| Identity | Effect |
| --- | --- |
| `event_id` in `incident_processed_events` | Same `alert.created` delivery is a no-op after success |
| `alert_id` unique on `incident_alerts` | No second association |
| Partial unique index on `alert_attached` timeline `(incident_id, payload.alert_id)` | No duplicate timeline attach |

Offsets commit only after the correlation transaction succeeds. If publish fails after commit, `published=false` and a retry republishes the same deterministic Kafka `event_id`. Downstream consumers must still be idempotent.

## Transaction boundaries

One transaction: lock service → find or create incident → attach alert → timeline rows → severity/summary/version → processed-event row. All commit or all roll back. Kafka is not acknowledged before that commit.

## Kafka

| Item | Value |
| --- | --- |
| Consume | `alerts.created` (`alert.created`) |
| Group | `sentinel-incident-v1` |
| Commit | Manual |
| Produce | `incidents.created`, `incidents.updated` |
| Acks | `all` (at-least-once) |

## HTTP (local / internal)

No auth. `apps/api` remains the external control plane.

| Path | Purpose |
| --- | --- |
| `GET /health` | Liveness |
| `GET /ready` | PostgreSQL + Kafka TCP |
| `GET /api/v1/incidents` | List (`status`, `limit`) |
| `GET /api/v1/incidents/{id}` | Detail (same fields as the Go API) |
| `GET /api/v1/incidents/{id}/timeline` | Append-only events (`created`, `alert_attached`, …) |

Port **8092**.

## Configuration

| Variable | Default |
| --- | --- |
| `DATABASE_URL` | required |
| `KAFKA_BROKERS` | required |
| `PORT` / `INCIDENT_PORT` | `8092` |
| `KAFKA_CONSUMER_GROUP` | `sentinel-incident-v1` |
| `INCIDENT_CORRELATION_WINDOW_SECONDS` | `300` |
| `SEED_ON_START` | Compose sets `true` if the catalog is empty |

## Local development

```bash
docker compose up -d postgres kafka incident
curl http://localhost:8092/health
curl http://localhost:8092/ready
```

Full path: ingestion `:8090` → detection `:8091` → incident `:8092`. The Phase 4 abnormal simulator should produce three `payments-api` alerts that become **one** incident.

## Tests

```bash
go test ./...
```

Database and Kafka tests skip when those services are unreachable. Set `DATABASE_URL` and `KAFKA_BROKERS` to match Compose (host Postgres may be `localhost:5433`).
