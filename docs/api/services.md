# Services

## GET /api/v1/services

**Purpose.** List watched services for the dashboard.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `environment`, `health_status`, `limit`, `cursor`.

**Response `200`.**

```json
{
  "data": [
    {
      "id": "22222222-2222-4222-8222-222222222221",
      "slug": "payments-api",
      "name": "Payments API",
      "environment": "production",
      "health_status": "unhealthy",
      "owner_user_id": "11111111-1111-4111-8111-111111111113"
    }
  ],
  "page": { "next_cursor": null, "limit": 20 }
}
```

**Status.** `200` · `400` · `401`.

## GET /api/v1/services/:id

**Purpose.** Service detail including recent deployments.

**Auth.** Bearer. Roles: any authenticated.

**Response `200`.**

```json
{
  "data": {
    "id": "22222222-2222-4222-8222-222222222221",
    "slug": "payments-api",
    "name": "Payments API",
    "environment": "production",
    "description": "Authorization, capture, and refunds for checkout.",
    "health_status": "unhealthy",
    "owner_user_id": "11111111-1111-4111-8111-111111111113",
    "recent_deployments": [
      {
        "id": "aaaaaaa1-0000-4000-8000-000000000005",
        "version": "1.18.0",
        "git_sha": "e4b21aa",
        "status": "succeeded",
        "started_at": "2026-09-14T04:12:00Z",
        "completed_at": "2026-09-14T04:17:00Z"
      }
    ]
  }
}
```

**Status.** `200` · `401` · `404`.

## GET /api/v1/deployments

**Purpose.** Recent catalog deployments.

**Auth.** Bearer. Roles: any authenticated.

**Query.** `service_id`, `limit`, `cursor`.

**Response `200`.** `{ "data": [ { "id", "service_id", "service_slug", "version", "git_sha", "status", "started_at", "completed_at" } ], "page": { "next_cursor", "limit" } }`
