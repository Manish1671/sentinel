# services/ai

Consumes `investigation.requested`, gathers bounded PostgreSQL-backed evidence through allowlisted tools, synthesizes a structured `InvestigationResult`, and publishes `investigation.completed`. It never executes remediation.

## Architecture

```
investigations.requested
        →  load incident context from PostgreSQL
        →  bounded tool calls (allowlist only)
        →  retrieve runbooks / historical incidents
        →  LLM or heuristic synthesis
        →  validate InvestigationResult
        →  persist investigations / evidence / recommendations
        →  investigations.completed
```

`source` for published events is `services.ai`. `causation_id` is the triggering `investigation.requested` `event_id`. Kafka key is `investigation_id`.

## Kafka

| Item | Value |
| --- | --- |
| Consume | `investigations.requested` (`investigation.requested`) |
| Group | `sentinel-ai-v1` |
| Commit | Manual after successful handling (including poison skip) |
| Produce | `investigations.completed` (`investigation.completed` with status `completed` or `failed`) |

At-least-once. Duplicate `event_id`s are recorded in `ai_processed_events`. Duplicate investigation rows are blocked by `investigations.idempotency_key` / `id`. Evidence and recommendation inserts use deterministic UUIDs (`ON CONFLICT DO NOTHING`).

Local development may also **produce** `investigation.requested` via `POST /api/v1/incidents/{id}/investigations` so the Kafka path works without `apps/api` publishing. The public control-plane producer remains `apps/api`.

## Tools

Read-only. No shell, kubectl, arbitrary SQL, or outbound HTTP except the optional LLM provider.

| Tool | Data | On empty |
| --- | --- | --- |
| `get_recent_logs` | `telemetry_events` kind=log | `no_evidence` |
| `get_metrics` | `telemetry_events` kind=metric | `no_evidence` |
| `get_service_health` | `services.health_status` | `no_evidence` |
| `get_recent_deployments` | `deployments` | `no_evidence` |
| `get_trace` | `telemetry_events` kind=trace | `no_evidence` |
| `search_runbooks` | published `runbooks` (tsvector + token overlap) | `no_evidence` |
| `search_previous_incidents` | `historical_incidents` view | `no_evidence` |
| `get_deployment_diff` | changelog/version metadata | `no_evidence` |

Unauthorized tool names are rejected. Arguments are schema-checked. Limits cap result size. A tool failure does not abort the investigation.

Tools read **retained PostgreSQL samples** (`telemetry_events`, catalog, runbooks). Live simulator traffic is processed by detection via Kafka; it is not required to land in `telemetry_events`. Missing samples return `no_evidence` instead of failing the investigation. The seeded payments-api 1.18.0 incident is the primary local demonstration case for logs/metrics/traces.

## Agent limits

| Limit | Default |
| --- | --- |
| Planned tool sequence | ≤ 8 tool types, ≤ 16 calls (`AI_MAX_TOOL_CALLS`) |
| Context size | `AI_MAX_CONTEXT_CHARS` (12000) |
| LLM timeout | `AI_LLM_TIMEOUT_SECONDS` (30) |
| Recursion | None — tools run in a fixed sequence, then **one** synthesis call |

## Retrieval / RAG

PostgreSQL full-text (`ts_rank` / `plainto_tsquery`) plus simple token overlap. No vector database. `app/retrieval/` is the seam for a later embedding backend.

Historical hits are **supporting evidence only**. They do not prove root cause.

## Evidence

Phase 1 `evidence` table. Each tool result becomes one or more rows with `source_type` from the contract enum. Metadata marks `observed` vs `no_evidence`. Large raw payloads are not copied; summaries and ids are stored.

## Confidence

`confidence` is in `[0, 1]`. Labels: **high** ≥ 0.75, **medium** ≥ 0.45, **low** otherwise. Weak/missing evidence should stay in the low band. The model/heuristic is instructed not to imply certainty.

Reasoning text must include **OBSERVED EVIDENCE**, **INFERENCE / HYPOTHESIS**, and **RECOMMENDATION**.

## LLM provider

`app/models/llm.py`:

- Empty `AI_API_KEY` → `HeuristicProvider` (deterministic, used in tests and local default)
- Set `AI_API_KEY` → OpenAI-compatible `POST {AI_BASE_URL}/chat/completions` via `httpx` (no vendor SDK)

Invalid JSON is rejected; synthesis falls back to the heuristic provider.

## Lifecycle

`requested` → `running` → `completed` | `failed`

Cancellation is in the enum but not implemented as an API in this phase.

## HTTP (local / internal)

No auth. `apps/api` remains the external control plane.

| Path | Purpose |
| --- | --- |
| `GET /health` | Liveness |
| `GET /ready` | Postgres + Kafka producer |
| `POST /api/v1/incidents/{id}/investigations` | Insert `requested` + publish Kafka |
| `POST /api/v1/investigations/{id}/run` | Same workflow in-process |
| `GET /api/v1/investigations/{id}` | Status, result, evidence |

Port **8000**.

## Configuration

| Variable | Default |
| --- | --- |
| `DATABASE_URL` | required |
| `KAFKA_BROKERS` | required |
| `PORT` | `8000` |
| `AI_API_KEY` | empty (heuristic) |
| `AI_MODEL` | `sentinel-investigator` |
| `AI_BASE_URL` | `https://api.openai.com/v1` |
| `KAFKA_CONSUMER_GROUP` | `sentinel-ai-v1` |

## Local development

```bash
docker compose up -d postgres kafka ai
curl http://localhost:8000/health
# after an incident exists:
curl -X POST http://localhost:8000/api/v1/incidents/{incident_id}/investigations
```

## Tests

```bash
cd services/ai && PYTHONPATH=. pytest
```

Database tests skip without `DATABASE_URL`. The suite never calls a live LLM API.

## Safety

The service does not: run production commands, write Kubernetes, deploy, rollback, open arbitrary HTTP (except the optional LLM endpoint), or execute recommendations.
