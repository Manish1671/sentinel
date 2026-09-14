# services/ingestion

Telemetry and deployment intake. Validates JSON, wraps it in the Phase 1 event envelope, and publishes to Kafka.

This service does not detect incidents, run AI, or execute remediations.

## Architecture

```
HTTP POST  →  validate  →  canonical envelope  →  Kafka producer
                 │
                 └─ process-local idempotency index (optional keys only)
```

`source` is always `services.ingestion`. Clients cannot override it.

## HTTP endpoints

| Method | Path | Kafka topic |
| --- | --- | --- |
| GET | `/health` | — |
| GET | `/ready` | — (Kafka TCP check) |
| POST | `/api/v1/telemetry/metric` | `telemetry.events` |
| POST | `/api/v1/telemetry/log` | `telemetry.events` |
| POST | `/api/v1/telemetry/trace` | `telemetry.events` |
| POST | `/api/v1/deployments` | `deployments.created` |

Success: **202** for a new publish, **200** for an idempotent replay. Errors use the Phase 1 envelope (`error.code`, `message`, `details`, `request_id`).

### Metric

```json
{
  "service_slug": "payments-api",
  "environment": "production",
  "name": "payments.capture.latency_p99",
  "value": 180,
  "unit": "ms",
  "occurred_at": "2026-09-14T04:18:00Z",
  "labels": { "route": "POST /capture" }
}
```

`service_id` (UUID) may be sent instead of or with `service_slug`. `timestamp` is accepted as an alias of `occurred_at`.

### Log

Required: service identity, `severity` (`debug|info|warn|error|fatal`), `message`, `occurred_at`.

### Trace

Required: service identity, `trace_id`, `span_id`, `name` (or `operation`), `duration_ms` (or `duration`), `status` (`ok|error|unset`), `occurred_at`.

### Deployment

Required: service identity, `version`, `started_at`. `status` defaults to `succeeded` (`in_progress|succeeded|failed|rolled_back`).

## Kafka

| Item | Value |
| --- | --- |
| Topics | `telemetry.events`, `deployments.created` (3 partitions locally via Compose `KAFKA_NUM_PARTITIONS`) |
| Message key | `service_id` UUID |
| Why | Events for one service hash to the same partition, so per-service order is preserved. Cross-service order is not. |
| Delivery | **At-least-once** to Kafka. Producer uses `acks=all`, retries, and is not async. Not exactly-once. |
| Envelope | Phase 1 v1: `event_id`, `event_type`, `event_version`, `occurred_at`, `source`, `correlation_id`, `causation_id` (null for origin), `payload` |

Create topics automatically on first publish (broker auto-create). If a first write races topic creation, the producer retries a few times. You can also pre-create:

```bash
docker exec sentinel-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic telemetry.events --partitions 3 --replication-factor 1
docker exec sentinel-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic deployments.created --partitions 3 --replication-factor 1
```

## Idempotency

- Caller may send `Idempotency-Key`, `idempotency_key`, or `event_id`.
- Same key + same canonical payload → **200**, no second Kafka write (process-local memory, TTL 24h).
- Same key + different payload → **409** `idempotency_key_conflict`.
- No Postgres write on the ingest path. Consumers must still treat `event_id` as the durable dedup key.

## Local development

```bash
docker compose up -d postgres redis kafka ingestion
# or run the binary against Compose Kafka:
export KAFKA_BROKERS=localhost:9092
export PORT=8090
cd services/ingestion && go run ./cmd/ingestion
```

Send a metric:

```bash
curl -s http://localhost:8090/api/v1/telemetry/metric -H "Content-Type: application/json" -d "{\"service_slug\":\"payments-api\",\"name\":\"payments.capture.latency_p99\",\"value\":180,\"unit\":\"ms\",\"occurred_at\":\"2026-09-14T04:18:00Z\"}"
```

Verify Kafka:

```bash
docker exec sentinel-kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic telemetry.events --from-beginning --timeout-ms 5000 --max-messages 1
```

Simulator (payments-api, orders-api, auth-service):

```bash
python scripts/telemetry_simulator.py --base-url http://localhost:8090
python scripts/telemetry_simulator.py --abnormal
```

## Environment

| Name | Required | Default |
| --- | --- | --- |
| `KAFKA_BROKERS` | yes | — |
| `PORT` | no | `8090` |
| `LOG_LEVEL` | no | `info` |
| `ENVIRONMENT` | no | `development` |
| `KAFKA_CLIENT_ID` | no | `sentinel-ingestion` |

## Testing

```bash
cd services/ingestion
go test ./...
```

A Kafka integration test publishes a metric and consumes it when `localhost:9092` (or `KAFKA_BROKERS`) is reachable; otherwise it skips.

## Docker

From the repository root:

```bash
docker build -f services/ingestion/Dockerfile .
```
