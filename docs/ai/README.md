# AI investigation

`services/ai` is a **tool-using** FastAPI service. It never receives a shell, Kubernetes client, or ad-hoc SQL. It investigates and recommends; it does not remediate.

Schemas:

- `packages/contracts/ai/investigation-request.v1.schema.json`
- `packages/contracts/ai/investigation-result.v1.schema.json`
- `packages/contracts/ai/tools.v1.schema.json`

Transport: Kafka `investigation.requested` / `investigation.completed`. Local HTTP on `:8000` uses the same workflow.

## InvestigationRequest

Enough context to investigate **without** unrestricted reads:

- `investigation_id`, `incident_id`, `correlation_id`
- `service`, `incident`, `alerts[]`, `recent_deployments[]`, `time_window`
- `allowed_tools` (server-defined allowlist)

## InvestigationResult

Must include root-cause hypothesis, confidence (0–1), evidence refs, reasoning, recommendations, risk, tool usage, and model identity. Invalid model JSON is rejected.

Reasoning distinguishes **OBSERVED EVIDENCE**, **INFERENCE / HYPOTHESIS**, and **RECOMMENDATION**.

Confidence bands: high ≥ 0.75, medium ≥ 0.45, low otherwise.

## Flow

1. Operator (`apps/api`) or local `POST /api/v1/incidents/:id/investigations` creates `investigations.status=requested` and emits `investigation.requested`.
2. `services/ai` claims the job (`running`), calls **only** allowlisted tools against PostgreSQL, stores `evidence`.
3. One bounded synthesis step produces `InvestigationResult`.
4. Emits `investigation.completed` (`status=completed|failed`).
5. Recommendations are **proposed** only. Humans approve later via the API. AI cannot approve or execute.

## Tools

Read-only, argument-validated, size-limited. Implemented against Sentinel PostgreSQL (catalog, telemetry samples, alerts, runbooks, historical incidents). Missing data returns structured `no_evidence`; it does not fail the whole run.

Retrieval is PostgreSQL full-text + token overlap. A vector backend can replace `app/retrieval/` later.

## LLM

Configurable `AI_API_KEY`, `AI_MODEL`, `AI_BASE_URL`. Empty key uses a deterministic heuristic provider so local/CI never requires a vendor.

## Limits

Max 16 tool calls, ~12k context characters, no recursive agent loop, one synthesis call. Timeouts on the LLM HTTP client.

See [services/ai/README.md](../../services/ai/README.md).
