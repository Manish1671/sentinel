# Idempotency and auditability

Contract only. No Redis or worker implementation in Phase 1.

Clients and producers **should** send an idempotency key on every command that can create or transition state. Consumers **must** treat Kafka delivery as at-least-once.

## Key behavior

| Rule | Detail |
| --- | --- |
| Format | Opaque string, 8–128 chars (`[A-Za-z0-9._:-]+`). HTTP header: `Idempotency-Key`. Events: envelope `idempotency_key` or `event_id` when the key is omitted. |
| Scope | Keys are namespaced by operation, not global. `incident:create:abc` and `investigation:request:abc` are different. |
| TTL (Redis) | 24 hours for HTTP commands; 7 days for processed `event_id`s. Durable uniqueness still comes from PostgreSQL unique columns where listed below. |
| Hit (same key, same payload) | Return the original result (`200` or `201` as documented). Do not create a second aggregate. |
| Hit (same key, different payload) | `409` `idempotency_key_conflict`. |
| Missing key | HTTP mutating endpoints listed below require the header (`400` `validation_error`). Kafka origin events may use `event_id`. |

## Where it is required

### Event processing

- Dedup key: `event_id`.
- Duplicate: acknowledge and skip side effects. Do not emit a second downstream event with a new `event_id` for the same cause unless compensating.
- Store: Redis `sentinel:v1:idempotency:event:{event_id}` plus, for incident opens, `incidents.dedup_key`.
- Audit: skipped duplicates are log metrics only, not timeline rows.

### Incident creation

- HTTP `POST /api/v1/incidents` requires `Idempotency-Key`.
- Pipeline opens from an alert use `dedup_key = "alert:{alert_id}"` or `"fingerprint:{service_id}:{fingerprint}:{bucket}"`.
- Duplicate: return the existing incident. Attach a new alert with `incident_alerts` if needed; write `alert_attached` once per alert id (unique on `(incident_id, alert_id)`).

### Investigation requests

- HTTP and `investigation.requested` require a key. Persist `investigations.idempotency_key`.
- Duplicate while `requested` or `running`: return the existing investigation.
- Duplicate after `completed` / `failed` / `cancelled`: `409` `investigation_already_terminal` unless the client is retrying the **read** of the original request (same key → original investigation, not a new one).

### Remediation requests

- Creating a remediation (from a proposed recommendation, internally) uses `remediations.idempotency_key` typically `{recommendation_id}:{attempt_number}`.
- Duplicate: return the existing remediation id.

### Approval actions

- `POST /api/v1/remediations/:id/approve` and `/reject` require `Idempotency-Key`.
- Same decision already recorded: `200` with the existing approval (no second timeline event).
- Opposite decision after terminal approval: `409` `approval_already_decided`.
- One approval row per remediation (`remediations` 1:1 `approvals`).

## Duplicate Kafka events

| Incoming | Behavior |
| --- | --- |
| Same `event_id` | No-op |
| New `event_id`, same `idempotency_key` / `dedup_key` | Treat as retry of the command; no second aggregate |
| New `event_id`, different key, same business facts | Allowed only if the domain permits (for example a second alert fingerprint). Incident correlation may attach rather than open |

`causation_id` must be the `event_id` of the message that caused this one. Retries keep the original `event_id`; they must not mint a new id for the same envelope.

## Audit expectations

Every **accepted** state change writes:

1. The aggregate row (`incidents.version` bump when the incident itself changes).
2. An append-only `incident_events` row for operator-visible history.
3. For approvals: the `approvals` row (`actor_user_id`, `decision`, `decided_at`, `comment`).
4. For remediations: status, parameters, result, verification, timestamps. Failures remain.

Retries and duplicate events must **not** insert a second timeline row for the same logical action. Use a deterministic timeline fingerprint in payload (`{"dedup":"investigation:{id}:requested"}`) when implementing, unique per `(incident_id, kind, dedup)` if needed later. Phase 1 does not add that extra table.
