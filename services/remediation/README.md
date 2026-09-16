# services/remediation

Owns **approval, controlled execution, verification, and audit** for Sentinel remediations. It does not decide root cause and does not generate AI recommendations.

The AI layer proposes a `recommendations` row. This service validates the action, waits for a human, runs an allowlisted simulator executor, verifies health signals, and updates the incident.

## Architecture

```
investigation.completed  (bridge; no recommendation.created event in Phase 1)
        → validate allowlisted recommendation
        → remediations.status = pending_approval
        → POST /approve  (approver|admin)
        → remediation.requested
        → executor.Execute (simulator)
        → verifying
        → remediation.completed | remediation.failed
```

A future Kubernetes executor implements the same `executor.Executor` interface (`Validate`, `Execute`, `State`). The workflow in `internal/remediation` does not change.

## Supported actions

Phase 1 `action_type` names. Spec “scale_service” is stored as `scale_replicas`.

| Action | Parameters | Approval | Verify |
| --- | --- | --- | --- |
| `rollback_deployment` | `to_version` or `to_deployment_id` | required | error rate, latency, DB util, health |
| `restart_service` | target service | required | error rate, latency, health |
| `scale_replicas` | `replicas` 1–20 | required | error rate, latency, replica count |

Unknown actions are rejected. No shell, kubectl, or host commands.

## Approval

`POST /api/v1/remediations/{id}/approve` and `/reject`.

- Auth: `Authorization: Bearer` using the same `AUTH_TOKEN_SECRET` JWT as `apps/api`. In `development`, `X-Sentinel-User-Id` may identify a seed user.
- Roles: viewer/responder cannot approve. `approver`/`admin` can; `required_approval_role=admin` requires admin.
- `Idempotency-Key` required. Same key + same body returns the same decision. A contradictory decision is `409`.
- Rejected remediations never execute.

## Safety

1. Unknown actions rejected.
2. Missing parameters rejected.
3. Target must exist and match the incident service.
4. Closed/resolved incidents and expired/superseded/rejected recommendations cannot execute.
5. Duplicate Kafka/HTTP delivery cannot run `Execute` twice (`status=approved` claim).
6. Human approval required for all three actions.
7. Simulator state is PostgreSQL-only.

## Verification

After execute, status is `verifying`. Rollback requires error rate ≤ 0.05, latency ≤ 500ms, DB utilization ≤ 0.80, and health `recovering` or `healthy`. Multiple signals; not a single check. Success sets catalog health to `healthy` and incident to `resolved`. Failure sets remediation `failed` and returns the incident to `investigating`.

## Incident transitions (Phase 1 only)

| When | Incident |
| --- | --- |
| Execution starts | `open`→`investigating` if needed, then `remediating` |
| Verification starts | `remediating`→`verifying` |
| Verification passed | `verifying`→`resolved` |
| Execute/verify failed | `remediating`/`verifying`→`investigating` (not resolved) |

Timeline uses existing `incident_event_kind` values with `payload.audit_event` (`remediation.created`, `.approved`, `.rejected`, `.started`, `.verifying`, `.succeeded`, `.failed`).

## Idempotency

- Remediation identity: `idempotency_key = recommendation:{id}:1` and deterministic UUID.
- Kafka: `remediation_processed_events` by `event_id`.
- Approve/reject: `http_idempotency_keys`.
- Execution: `UPDATE … WHERE status = 'approved'` claims the row once.

## Kafka

| | |
| --- | --- |
| Consume | `investigations.completed`, `remediation.requested` (group `sentinel-remediation-v1`) |
| Produce | `remediation.requested`, `remediation.completed`, `remediation.failed` |
| `source` | `services.remediation` |
| Key | `remediation_id` |

Phase 1 has no `recommendation.created`. Consuming `investigation.completed` and reading `recommendations` is the bridge.

## HTTP

| Path | Notes |
| --- | --- |
| `GET /health` `GET /ready` | liveness / postgres+kafka |
| `GET /api/v1/remediations/{id}` | includes approval |
| `GET /api/v1/remediations/{id}/execution` | simulator snapshot |
| `POST …/approve` `POST …/reject` | idempotent |
| `GET /api/v1/incidents/{id}/remediations` | list |
| `GET /api/v1/incidents/{id}/timeline` | audit |

Port **8093**. External auth product remains `apps/api`.

## Local development

```bash
docker compose up -d --build remediation
curl http://localhost:8093/health
# after a pending remediation exists:
curl -H "X-Sentinel-User-Id: 11111111-1111-4111-8111-111111111114" \
  http://localhost:8093/api/v1/remediations/{id}   # viewer: inspect
curl -X POST -H "X-Sentinel-User-Id: 11111111-1111-4111-8111-111111111112" \
  -H "Idempotency-Key: approve-local-0001" \
  http://localhost:8093/api/v1/remediations/{id}/approve
```

Approver seed user: `jordan.hale@sentinel.dev` (`11111111-1111-4111-8111-111111111112`). Viewer: `11111111-1111-4111-8111-111111111114`.

## Tests

```bash
cd services/remediation && go test ./...
```

Database tests skip if PostgreSQL is unavailable.
