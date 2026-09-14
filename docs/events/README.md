# Kafka event strategy

JSON payloads. Ingestion (`services/ingestion`) publishes `telemetry.*` and `deployment.created`. Other event types are still contract-only.

Contracts are versioned (`event_version`). Additive fields only within a version; breaking changes increment `event_version` or introduce a new `event_type`.

## Envelope

Canonical field names (`packages/contracts/events/envelope.v1.schema.json`):

| Field | Meaning |
| --- | --- |
| `event_id` | UUID of this message. Retries reuse it. |
| `event_type` | Canonical name (`incident.created`). |
| `event_version` | Integer payload version, starting at 1. |
| `occurred_at` | When the domain fact happened (UTC). |
| `source` | Producer identity (`services.ingestion`, `apps.api`, …). Replaces Phase 0 `producer`. |
| `correlation_id` | UUID tying alert → incident → investigation → remediation. |
| `causation_id` | `event_id` of the causing message; `null` for origin events. |
| `payload` | Event-specific object. |
| `tenant_id` | Optional; local default `default`. |
| `idempotency_key` | Optional; defaults to `event_id`. |

Example:

```json
{
  "event_id": "9d2c1a4e-4b0a-4c3f-9f1e-2b7c8d0e1f2a",
  "event_type": "incident.created",
  "event_version": 1,
  "occurred_at": "2026-09-14T04:20:00Z",
  "source": "services.incident",
  "correlation_id": "dddddddd-0000-4000-8000-0000000000d1",
  "causation_id": "eeeeeeee-0000-4000-8000-000000000004",
  "tenant_id": "default",
  "idempotency_key": "alert:eeeeeeee-0000-4000-8000-000000000004",
  "payload": {
    "incident_id": "33333333-3333-4333-8333-333333333334",
    "reference": "INC-2026-0004",
    "service_id": "22222222-2222-4222-8222-222222222221",
    "title": "Payments capture failures after 1.18.0",
    "severity": "critical",
    "status": "open",
    "alert_ids": ["eeeeeeee-0000-4000-8000-000000000004"],
    "detected_at": "2026-09-14T04:19:00Z",
    "dedup_key": "alert:eeeeeeee-0000-4000-8000-000000000004"
  }
}
```

The **producer named in `source` owns** the event type. Consumers ignore unknown payload fields.

## Topics and event types (v1)

| event_type | topic | producer | consumers |
| --- | --- | --- | --- |
| `telemetry.metric` | `telemetry.events` | `services.ingestion` | `services.detection` |
| `telemetry.log` | `telemetry.events` | `services.ingestion` | `services.detection` |
| `telemetry.trace` | `telemetry.events` | `services.ingestion` | `services.detection` |
| `deployment.created` | `deployments.created` | `services.ingestion` | `apps.api`, `services.detection` |
| `alert.created` | `alerts.created` | `services.detection` | `services.incident` |
| `incident.created` | `incidents.created` | `services.incident` | `apps.api`, `services.ai` |
| `incident.updated` | `incidents.updated` | `services.incident` | `apps.api`, `services.ai`, `services.remediation` |
| `investigation.requested` | `investigations.requested` | `apps.api` (also `services.incident` for auto-investigate) | `services.ai` |
| `investigation.completed` | `investigations.completed` | `services.ai` | `services.incident`, `apps.api` |
| `remediation.requested` | `remediation.requested` | `apps.api` after approval | `services.remediation` |
| `remediation.completed` | `remediation.completed` | `services.remediation` | `services.incident`, `apps.api` |
| `remediation.failed` | `remediation.failed` | `services.remediation` | `services.incident`, `apps.api` |

`system.health` remains reserved; it is not a Phase 1 contract.

Payload schemas live beside this document in `packages/contracts/events/*.v1.schema.json`. `investigation.requested` payload is `InvestigationRequest`. `investigation.completed.result` is `InvestigationResult`.

## Ingestion (Phase 3)

`services/ingestion` is the producer for `telemetry.metric`, `telemetry.log`, `telemetry.trace`, and `deployment.created`.

- Kafka message key: `service_id` (UUID), so a service’s events stay ordered on one partition.
- Delivery to Kafka is **at-least-once** (`acks=all`, synchronous writes). Exactly-once is not claimed.
- `source` is always `services.ingestion`. `causation_id` is `null` for these origin events.
- Caller `correlation_id` is kept when it is a UUID; otherwise ingestion generates one.
- Process-local idempotency (header/`event_id`) avoids duplicate publishes in the same process. Consumers must still deduplicate on `event_id`.

See [services/ingestion/README.md](../../services/ingestion/README.md).

## Naming

- Topics: lowercase dotted plural domain (`incidents.created`).
- Event types: singular entity + past tense (`incident.created`).
- Compatibility: additive JSON fields only within `event_version`.
