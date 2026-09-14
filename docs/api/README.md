# HTTP API (v1)

Public contract for `apps/api`. Telemetry intake is a separate service: [services/ingestion/README.md](../../services/ingestion/README.md) (`http://localhost:8090`).

Base URL (local): `http://localhost:8080`

| File | Contents |
| --- | --- |
| [errors.md](./errors.md) | Error envelope and status mapping |
| [conventions.md](./conventions.md) | Auth, pagination, idempotency headers |
| [auth.md](./auth.md) | Login, logout, me |
| [services.md](./services.md) | Service catalog |
| [incidents.md](./incidents.md) | Incidents and timeline |
| [investigations.md](./investigations.md) | Investigations |
| [recommendations.md](./recommendations.md) | Recommendations |
| [remediations.md](./remediations.md) | Remediation + approval |
| [health.md](./health.md) | `/health`, `/ready` |

Machine-readable error shape: `packages/contracts/api/error.v1.schema.json`.
