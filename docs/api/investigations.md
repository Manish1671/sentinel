# Investigations

## POST /api/v1/incidents/:id/investigations

**Purpose.** Request an AI investigation. Creates an `investigations` row in `requested` and is intended to emit `investigation.requested`.

**Auth.** Bearer. Roles: `responder`, `approver`, `admin`.

**Headers.** `Idempotency-Key` required.

**Request.**

```json
{
  "time_window": {
    "from": "2026-09-14T03:19:00Z",
    "to": "2026-09-14T04:19:00Z"
  }
}
```

`time_window` optional; default is one hour before `detected_at` through now. `allowed_tools` is not client-specified; the API always sends the allowlist in `packages/contracts/enums.v1.json`.

**Validation.** Incident must exist and not be `closed`. Illegal: requesting on `closed` → `409` `incident_invalid_transition`.

**Response `201`.** Duplicate key: `200` existing investigation.

```json
{
  "data": {
    "id": "44444444-4444-4444-8444-444444444441",
    "incident_id": "33333333-3333-4333-8333-333333333334",
    "status": "requested",
    "requested_by_user_id": "11111111-1111-4111-8111-111111111113",
    "requested_at": "2026-09-14T04:21:00Z"
  }
}
```

**Status.** `201` / `200` · `400` · `401` · `403` · `404` · `409`.

## GET /api/v1/investigations

**Purpose.** List investigations for the operator console.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `incident_id`, `status`, `limit`, `cursor`.

**Response `200`.** `{ "data": [ investigation summaries ], "page": { "next_cursor", "limit" } }`

## GET /api/v1/incidents/:id/investigations

**Purpose.** Investigations for one incident. `:id` may be UUID or reference.

**Auth.** Bearer. Roles: any authenticated.

## GET /api/v1/investigations/:id

**Purpose.** Investigation status and result.

**Auth.** Bearer. Roles: any authenticated.

**Response `200`.**

```json
{
  "data": {
    "id": "44444444-4444-4444-8444-444444444441",
    "incident_id": "33333333-3333-4333-8333-333333333334",
    "status": "completed",
    "model_name": "sentinel-investigator",
    "model_version": "2026.09.01",
    "root_cause_hypothesis": "payments-api 1.18.0 added a synchronous inventory.reserve call with a 150ms timeout. Inventory p99 exceeds that budget, so capture fails and checkout degrades.",
    "reasoning_summary": "Error rate and latency stepped at 04:17 after 1.18.0. Traces show inventory.reserve status=error at 150ms.",
    "confidence": 0.86,
    "risk_level": "high",
    "tool_usage": [
      { "tool": "get_trace", "call_count": 1, "status": "ok" }
    ],
    "requested_at": "2026-09-14T04:21:00Z",
        "completed_at": "2026-09-14T04:27:00Z",
    "evidence": [
      { "id": "77777777-0000-4000-8000-000000000001", "tool_name": "get_recent_deployments", "source_type": "deployment", "summary": "payments-api 1.18.0 completed 04:17 UTC" }
    ]
  }
}
```

**Status.** `200` · `401` · `404`.
