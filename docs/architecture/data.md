# Data stores

Phase 1 physical design. Schema lives in `database/migrations/`. Seeds in `database/seeds/`.

## PostgreSQL

System of record for transactional and auditable state.

### Relationships

```
User ──< Service.owner_user_id
User ──< Deployment.deployed_by_user_id
User ──< Incident.created_by / commander
User ──< Investigation.requested_by
User ──< Approval.actor
User ──< Remediation.requested_by
User ──< Runbook.created_by

Service ──< Deployment
Service ──< TelemetryEvent          (retained samples only)
Service ──< Alert
Service ──< Incident
Service ──< Runbook                 (nullable service = platform-wide)
Service ──< Recommendation.target
Service ──< Remediation

Alert >──< Incident                 (incident_alerts)

Incident ──< IncidentEvent          (append-only)
Incident ──< Investigation
Incident ──< Recommendation
Incident ──< Remediation
Incident ──< Approval
Incident ══ HistoricalIncident      (VIEW of resolved/closed incidents)

Investigation ──< Evidence
Investigation ──< Recommendation

Recommendation ──< Remediation
Remediation ── Approval             (1:1)

EvaluationRun ──> Incident          (labeled_incident_id, typically historical)
```

### Integrity and query patterns

| Concern | Approach |
| --- | --- |
| Identifiers | UUID primary keys (`gen_random_uuid()` default) |
| Incident concurrency | `incidents.version` incremented on status/commander/title updates |
| Dedup / idempotent create | `incidents.dedup_key` UNIQUE; investigation/remediation `idempotency_key` UNIQUE |
| Open alert dedup | Partial unique index on `(service_id, fingerprint)` where status is `open` or `acknowledged` |
| Timeline | `incident_events(incident_id, occurred_at)` |
| Dashboard lists | `incidents(status, detected_at DESC)`, `incidents(service_id, status)` |
| Tooling samples | `telemetry_events(service_id, occurred_at DESC)` |
| Audit | No updates to `incident_events` payload; remediations and approvals are never deleted |
| History | `historical_incidents` view; investigations remain on the incident row |

### What this supports

- Transactional incident state (one row, versioned)
- Auditability (timeline + approval + remediation history)
- Historical investigations (rows stay after resolve/close)
- Remediation history (including `failed`)
- AI evaluation results (`evaluation_runs`)

Complete ORM repositories are not in this phase.

## Redis

Not a source of record. Loss must not corrupt incidents. Canonical key layout: [`packages/contracts/redis.md`](../../packages/contracts/redis.md).

| Use | Pattern |
| --- | --- |
| Idempotency | `sentinel:v1:idempotency:{scope}:{key}` |
| Cache | `sentinel:v1:cache:{resource}:{id}` |
| Locks | `sentinel:v1:lock:{resource}:{id}` |
| Short-lived processing | `sentinel:v1:state:{process}:{id}` |
| Rate limiting | `sentinel:v1:ratelimit:{subject}:{window}` |

## Object storage (S3-compatible)

Evidence bodies, tool transcripts, evaluation fixtures. PostgreSQL stores `artifact_uri` and hashes in `metadata`, not multi-megabyte blobs.
