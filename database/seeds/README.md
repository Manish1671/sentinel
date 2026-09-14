# Database seeds

Local development data only. Apply **after** all files in `database/migrations/`, in filename order.

```text
001_users.sql
002_catalog.sql
003_telemetry_and_alerts.sql
004_incidents.sql
005_investigations.sql
006_remediation_and_evaluation.sql
```

There is no seed runner in Phase 1. Example:

```bash
for f in database/migrations/*.sql database/seeds/*.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
done
```

## Narrative

| Reference | Service | Status | Notes |
| --- | --- | --- | --- |
| INC-2026-0001 | auth-service | closed | TLS expiry; historical |
| INC-2026-0002 | inventory-worker | resolved | OOM after 2.4.1; historical; evaluation replay |
| INC-2026-0003 | notifications-worker | open | Consumer lag; no investigation yet |
| INC-2026-0004 | payments-api | investigating | Capture failures after 1.18.0; recommendation pending approval |

Stable ids used in API examples:

| Entity | Id |
| --- | --- |
| Sam Okonkwo (responder) | `11111111-1111-4111-8111-111111111113` |
| Jordan Hale (approver) | `11111111-1111-4111-8111-111111111112` |
| payments-api | `22222222-2222-4222-8222-222222222221` |
| INC-2026-0004 | `33333333-3333-4333-8333-333333333334` |
| Investigation | `44444444-4444-4444-8444-444444444441` |
| Rollback recommendation | `55555555-5555-4555-8555-555555555551` |
| Pending remediation | `66666666-6666-4666-8666-666666666661` |

Auth is implemented in `apps/api`. Every seed user uses the local-only password `sentinel-dev`.
