# API conventions

## Authentication

Authenticated endpoints require `Authorization: Bearer <token>` **or** the HttpOnly cookie `sentinel_session`. Tokens are issued by `POST /api/v1/auth/login`. Unauthenticated calls return `401` `unauthenticated`. Disabled users return `401` `user_disabled`.

`GET /health` and `GET /ready` are unauthenticated.

## Roles

| Role | Read catalog/incidents | Create incident / request investigation | Approve/reject remediation |
| --- | --- | --- | --- |
| `viewer` | yes | no | no |
| `responder` | yes | yes | no |
| `approver` | yes | yes | yes |
| `admin` | yes | yes | yes |

## Idempotency

Mutating endpoints listed in [../architecture/idempotency.md](../architecture/idempotency.md) require:

```
Idempotency-Key: <8-128 chars>
```

## Pagination

List endpoints use cursor pagination:

```
GET /api/v1/incidents?limit=20&cursor=<opaque>
```

Response:

```json
{
  "data": [],
  "page": { "next_cursor": null, "limit": 20 }
}
```

`limit` default 20, max 100.

## Optimistic concurrency

Updates to an incident that exist later must send `If-Match` with `incidents.version`. Phase 1 POST create does not require it.

## Request ids

Every response includes `X-Request-Id`. The same value is `error.request_id`.
