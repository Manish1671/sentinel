# apps/api

Go HTTP control plane for Sentinel.

## Responsibility

Synchronous API for operators: authentication, service catalog reads, incident reads/creates, and incident timelines.

This service does **not** ingest telemetry, run detection, execute remediations, or call the AI investigator.

## Architecture

```
cmd/api            process entrypoint
internal/httpapi   HTTP adapters and DTOs
internal/auth      login, signed sessions, current user
internal/services  service catalog
internal/incidents incident use cases and persistence
internal/database  pool + SQL migration runner
internal/cache     optional Redis ping
```

Handlers call application services. Services own validation and lifecycle. Repositories talk to PostgreSQL.

## Local setup

From the repository root:

```bash
cp .env.example .env
# set AUTH_TOKEN_SECRET to a local value (16+ characters)
docker compose up -d postgres redis
```

Apply schema (from `apps/api`):

```bash
export DATABASE_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable
export AUTH_TOKEN_SECRET=local-dev-token-secret
export MIGRATIONS_PATH=../../database/migrations
go run ./cmd/api migrate
```

Seed (from repo root, after migrations):

```bash
# Windows PowerShell example
Get-ChildItem database/seeds/*.sql | ForEach-Object { psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f $_.FullName }
```

## Environment variables

| Name | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | yes | PostgreSQL URL |
| `AUTH_TOKEN_SECRET` | yes | HMAC secret for signed session tokens (≥16 chars) |
| `AUTH_TOKEN_TTL` | no | Token lifetime (default `12h`) |
| `REDIS_URL` | no | Included in `/ready` when set |
| `KAFKA_BROKERS` | no | Placeholder for later phases |
| `PORT` | no | Listen port (default `8080`) |
| `ENVIRONMENT` | no | `development` auto-runs migrations on start |
| `LOG_LEVEL` | no | `debug` / `info` / `warn` / `error` |
| `MIGRATIONS_PATH` | no | Directory of Phase 1+ SQL migrations |

## Starting the API

```bash
cd apps/api
export DATABASE_URL=postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable
export AUTH_TOKEN_SECRET=local-dev-token-secret
export REDIS_URL=redis://localhost:6379/0
export ENVIRONMENT=development
export MIGRATIONS_PATH=../../database/migrations
go run ./cmd/api
```

## Authentication (local development)

Seed users share the local password `sentinel-dev`. This is development-only.

| Email | Role |
| --- | --- |
| `maya.chen@sentinel.dev` | admin |
| `jordan.hale@sentinel.dev` | approver |
| `sam.okonkwo@sentinel.dev` | responder |
| `riley.park@sentinel.dev` | viewer |

```bash
curl -s http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d "{\"email\":\"sam.okonkwo@sentinel.dev\",\"password\":\"sentinel-dev\"}"
```

Send `Authorization: Bearer <token>` on authenticated routes. Logout revokes the session in PostgreSQL.

## Available endpoints

| Method | Path |
| --- | --- |
| GET | `/health` |
| GET | `/ready` |
| POST | `/api/v1/auth/login` |
| POST | `/api/v1/auth/logout` |
| GET | `/api/v1/me` |
| GET | `/api/v1/services` |
| GET | `/api/v1/services/:id` |
| GET | `/api/v1/incidents` |
| POST | `/api/v1/incidents` |
| GET | `/api/v1/incidents/:id` |
| GET | `/api/v1/incidents/:id/timeline` |

`POST /api/v1/incidents` requires `Idempotency-Key` and role `responder`, `approver`, or `admin`.

## Testing

```bash
cd apps/api
go test ./...
```

Integration tests expect PostgreSQL (default `DATABASE_URL` or `TEST_DATABASE_URL`) and seeded data.

## Docker

Build from the **repository root** so migrations can be copied:

```bash
docker build -f apps/api/Dockerfile .
```

The image runs as a non-root user and reads configuration from the environment.
