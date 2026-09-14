# Error contract

All error responses use this body (see `packages/contracts/api/error.v1.schema.json`):

```json
{
  "error": {
    "code": "not_found",
    "message": "Incident not found.",
    "details": {},
    "request_id": "b3c1e2d4-4a5f-4c8a-9b0e-1f2a3b4c5d6e"
  }
}
```

`message` is safe to show an operator. `details` may include field errors; it must not include stack traces or secrets.

## Status mapping

| HTTP | `code` | When |
| --- | --- | --- |
| 400 | `validation_error` | Schema/constraint failure |
| 401 | `unauthenticated` | Missing/invalid token |
| 401 | `user_disabled` | User exists but `disabled` |
| 403 | `forbidden` | Authenticated but role cannot perform the action |
| 404 | `not_found` | Unknown resource id |
| 409 | `conflict` | Generic conflict |
| 409 | `idempotency_key_conflict` | Same key, different body |
| 409 | `incident_invalid_transition` | Illegal status change |
| 409 | `approval_already_decided` | Opposite approval decision |
| 409 | `investigation_already_terminal` | New work on a terminal investigation |
| 429 | `rate_limited` | Too many requests |
| 500 | `internal_error` | Unexpected failure |

## Validation errors

HTTP `400`. `details.fields` is an array of `{path, code, message}`.

```json
{
  "error": {
    "code": "validation_error",
    "message": "Request validation failed.",
    "details": {
      "fields": [
        { "path": "title", "code": "required", "message": "title is required" }
      ]
    },
    "request_id": "..."
  }
}
```

## Authentication failures

HTTP `401`. No `WWW-Authenticate` body besides the envelope. Do not reveal whether an email exists on login beyond a generic `invalid_credentials` message (`code`: `unauthenticated`).

## Authorization failures

HTTP `403` `forbidden`. Example: `viewer` posting an investigation.

## Not found

HTTP `404` `not_found`. Same shape for hidden resources (no existence leak beyond id).

## Conflict

HTTP `409`. Use a specific `code` when listed above.

## Rate limiting

HTTP `429` `rate_limited`. `details.retry_after_seconds` is an integer. `Retry-After` header mirrors it.

## Internal errors

HTTP `500` `internal_error`. `message` is generic (`An internal error occurred.`). Correlate with logs via `request_id`.
