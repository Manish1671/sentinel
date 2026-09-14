# Incidents

## GET /api/v1/incidents

**Purpose.** List incidents for the dashboard.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `status`, `service_id`, `severity`, `limit`, `cursor`.

**Response `200`.**

```json
{
  "data": [
    {
      "id": "33333333-3333-4333-8333-333333333334",
      "reference": "INC-2026-0004",
      "service_id": "22222222-2222-4222-8222-222222222221",
      "title": "Payments capture failures after 1.18.0",
      "severity": "critical",
      "status": "investigating",
      "commander_user_id": "11111111-1111-4111-8111-111111111113",
      "detected_at": "2026-09-14T04:19:00Z",
      "version": 3
    }
  ],
  "page": { "next_cursor": null, "limit": 20 }
}
```

**Status.** `200` · `400` · `401`.

## POST /api/v1/incidents

**Purpose.** Manually open an incident.

**Auth.** Bearer. Roles: `responder`, `approver`, `admin`.

**Headers.** `Idempotency-Key` required.

**Request.**

```json
{
  "service_id": "22222222-2222-4222-8222-222222222225",
  "title": "Notifications lag impacting password-reset email",
  "summary": "Optional",
  "severity": "medium",
  "alert_ids": ["eeeeeeee-0000-4000-8000-000000000003"]
}
```

**Validation.** `service_id` UUID existing; `title` 1–200 chars; `severity` enum; `alert_ids` optional UUIDs for that service.

**Response `201`.** Incident object (same fields as GET by id, without timeline). Duplicate key + same body: `200` with existing incident.

**Status.** `201` / `200` · `400` · `401` · `403` · `404` (service) · `409` idempotency mismatch.

## GET /api/v1/incidents/:id

**Purpose.** Incident detail for the operator view.

**Auth.** Bearer. Roles: any authenticated.

**Response `200`.**

```json
{
  "data": {
    "id": "33333333-3333-4333-8333-333333333334",
    "reference": "INC-2026-0004",
    "service_id": "22222222-2222-4222-8222-222222222221",
    "title": "Payments capture failures after 1.18.0",
    "summary": "Capture error rate 8.3% and p99 1640ms following synchronous inventory reservation.",
    "severity": "critical",
    "status": "investigating",
    "created_by_user_id": "11111111-1111-4111-8111-111111111113",
    "commander_user_id": "11111111-1111-4111-8111-111111111113",
    "detected_at": "2026-09-14T04:19:00Z",
    "resolved_at": null,
    "closed_at": null,
    "version": 3,
    "alert_ids": [
      "eeeeeeee-0000-4000-8000-000000000004",
      "eeeeeeee-0000-4000-8000-000000000005"
    ]
  }
}
```

**Status.** `200` · `401` · `404`.

## GET /api/v1/incidents/:id/timeline

**Purpose.** Ordered `IncidentEvent` list.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `limit`, `cursor` (ascending `occurred_at`).

**Response `200`.**

```json
{
  "data": [
    {
      "id": "ffffeee1-0000-4000-8000-000000000030",
      "kind": "created",
      "actor_user_id": null,
      "occurred_at": "2026-09-14T04:20:00Z",
      "payload": {
        "source": "alert",
        "alert_id": "eeeeeeee-0000-4000-8000-000000000004"
      }
    }
  ],
  "page": { "next_cursor": null, "limit": 20 }
}
```

**Status.** `200` · `401` · `404`.
