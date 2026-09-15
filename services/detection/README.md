# services/detection

Consumes canonical Kafka telemetry, evaluates deterministic rules, persists Alerts in PostgreSQL, and publishes `alert.created`. It does not create incidents, run AI, or change production systems.

## Architecture

```
telemetry.events / deployments.created
        →  validate envelope
        →  rule engine (in-memory windows)
        →  PostgreSQL alerts + detection_processed_events
        →  alerts.created  (new logical alerts only)
```

`source` for published events is always `services.detection`.

## Kafka consumer

| Item | Value |
| --- | --- |
| Topics | `telemetry.events`, `deployments.created` |
| Group | `sentinel-detection-v1` |
| Commit | Manual after successful processing (or after a logged poison skip) |

The consumer group is the durable offset cursor for Detection. Multiple Detection replicas in the same group share partitions; each message is processed by one member.

## Detection rules (defaults)

| Rule | Trigger | Severity |
| --- | --- | --- |
| `high_latency` | metric name contains `latency` and value > **1000** ms | high |
| `high_error_rate` | metric name contains `error_rate`; value > **0.05** (values > 1 treated as percent) | high |
| `db_connection_saturation` | DB connection metric > **0.90** | critical |
| `error_log_burst` | ≥ **5** error/fatal logs per service in **60s** | high |
| `deployment_associated` | annotates other hits if a `deployment.created` for the service is within **15m** | (label only) |

Traces are accepted and ignored by rules (no silent drop of the Kafka message).

## Configuration

| Variable | Default |
| --- | --- |
| `DATABASE_URL` | required |
| `KAFKA_BROKERS` | required |
| `PORT` | `8091` |
| `KAFKA_CONSUMER_GROUP` | `sentinel-detection-v1` |
| `LATENCY_THRESHOLD_MS` | `1000` |
| `ERROR_RATE_THRESHOLD` | `0.05` |
| `DB_CONNECTION_THRESHOLD` | `0.90` |
| `ERROR_BURST_COUNT` | `5` |
| `ERROR_BURST_WINDOW_SECONDS` | `60` |
| `DEPLOYMENT_CORRELATION_WINDOW_SECONDS` | `900` |
| `ALERT_COOLDOWN_SECONDS` | `300` |
| `SEED_ON_START` | unset; Compose sets `true` if the catalog is empty |

## Stateful detection vs durable alerts

Detection windows (error bursts, recent deployments) live **in memory** with time-based cleanup. A process restart forgets them; that may delay a burst/correlation until enough new events arrive.

Alert identity and source-event processing live in **PostgreSQL**. That is the source of truth.

## Alert fingerprint and cooldown

Fingerprint: `{detector_id}:{service_id}:{metric_or_subject}`

- Same `event_id` → no second alert (`detection_processed_events`).
- Open/acknowledged row with the same fingerprint → update summary, no `alert.created`.
- Resolved fingerprint inside cooldown → no new logical alert.
- Unique index `alerts_open_fingerprint_uidx` enforces one active alert per fingerprint.

## At-least-once / failures

Kafka is at-least-once. Offsets commit after DB write and (when needed) publish.

| Case | Behavior |
| --- | --- |
| A: DB ok, publish fails | Offset not committed. Retry republishes. `published=false` until success. |
| B: publish ok, crash before commit | Re-delivery sees `published=true` (or republishes the same deterministic `alert.created` `event_id`). Downstream must treat `alert_id` as idempotent. |
| C: same `event_id` again | No new logical alert. |
| D: crash during window processing | In-memory windows lost. Durable alerts unchanged. |

Poison / malformed JSON: logged, offset committed so the partition is not blocked. Invalid envelopes with a usable `event_id` are recorded as processed.

## HTTP

- `GET /health`
- `GET /ready` (Postgres + Kafka TCP)
- `GET /api/v1/rules` (read-only)

## Local development

```bash
docker compose up -d postgres redis kafka ingestion detection
curl -s http://localhost:8091/health
curl -s http://localhost:8091/ready
python scripts/telemetry_simulator.py
python scripts/telemetry_simulator.py --abnormal
```

Verify:

```bash
docker exec sentinel-kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic alerts.created --from-beginning --timeout-ms 5000 --max-messages 1
```

## Testing

```bash
cd services/detection
go test ./...
```

Postgres/Kafka integration tests skip when those services are not reachable.

## Docker

From the repository root:

```bash
docker build -f services/detection/Dockerfile .
```
