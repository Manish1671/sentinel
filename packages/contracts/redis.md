# Redis key contract

Not a source of record. Prefix every key with `sentinel:v1:`. Values are strings or small JSON. Always set a TTL.

| Purpose | Pattern | TTL | Value |
| --- | --- | --- | --- |
| Idempotency | `sentinel:v1:idempotency:{scope}:{key}` | HTTP 24h; events 7d | JSON `{status, resource_type, resource_id, request_hash}` |
| Cache | `sentinel:v1:cache:{resource}:{id}` | 30–120s | JSON snapshot (health, runbook search) |
| Lock | `sentinel:v1:lock:{resource}:{id}` | 15–60s | owner token |
| Processing state | `sentinel:v1:state:{process}:{id}` | ≤ 15m | JSON lease (`investigation`, `remediation`) |
| Rate limit | `sentinel:v1:ratelimit:{subject}:{window}` | window length | integer counter |

Scopes for idempotency: `event`, `incident.create`, `investigation.request`, `remediation.create`, `remediation.approve`, `remediation.reject`.

Examples:

```
sentinel:v1:idempotency:event:9d2c1a4e-4b0a-4c3f-9f1e-2b7c8d0e1f2a
sentinel:v1:idempotency:incident.create:pay-capture-2026-09-14
sentinel:v1:lock:incident-correlate:22222222-2222-4222-8222-222222222221
sentinel:v1:state:investigation:44444444-4444-4444-8444-444444444441
sentinel:v1:ratelimit:ip:127.0.0.1:login:2026-09-14T04:21
sentinel:v1:cache:service-health:22222222-2222-4222-8222-222222222221
```

Lock names: `incident-correlate:{service_id}`, `remediation-exec:{remediation_id}`.

Do not cache incident aggregates as the source of truth. Redis loss must leave PostgreSQL correct.
