# Health

Unauthenticated liveness and readiness for the API process. These are not Kafka `system.health` events.

## GET /health

**Purpose.** Process is up.

**Auth.** None.

**Response `200`.**

```json
{ "status": "ok" }
```

Does not check PostgreSQL, Redis, or Kafka.

## GET /ready

**Purpose.** Process can serve traffic.

**Auth.** None.

**Response `200`** when PostgreSQL can be pinged (and, once wired, Redis). **`503`** otherwise:

```json
{
  "status": "unavailable",
  "checks": {
    "postgres": "ok",
    "redis": "unavailable"
  }
}
```

`503` uses the error envelope only when an unexpected exception occurs; a known failed dependency may use the body above. Phase 1 documents both; implementation should prefer the structured `checks` object for operators and still set `X-Request-Id`.
