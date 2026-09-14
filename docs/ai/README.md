# AI investigation contract

The AI investigator is a **tool-using service**. It never receives database credentials, cloud keys, or a shell. Phase 1 defines the contract only.

Schemas:

- `packages/contracts/ai/investigation-request.v1.schema.json`
- `packages/contracts/ai/investigation-result.v1.schema.json`
- `packages/contracts/ai/tools.v1.schema.json`

Transport (later): Kafka `investigation.requested` / `investigation.completed`, and optionally HTTP between `apps.api` and `services.ai`. Both use the same JSON bodies.

## InvestigationRequest

Enough context to investigate **without** unrestricted reads:

- `investigation_id`, `incident_id`, `correlation_id`
- `service` (id, slug, environment, health)
- `incident` (reference, title, summary, severity, status, detected_at)
- `alerts[]` summaries
- `recent_deployments[]` ids/versions (not full diffs)
- `time_window`
- `allowed_tools` (server-defined allowlist)

No SQL, no kube configs, no log firehose.

## InvestigationResult

Must include:

- `root_cause_hypothesis`
- `confidence` (0–1)
- `evidence_refs[]`
- `reasoning_summary`
- `recommendations[]` (action, rationale, risk, confidence, parameters)
- `risk_level`
- `tool_usage[]`
- `model.name` / `model.version`

## Flow

1. Operator or correlator requests investigation (`POST /api/v1/incidents/:id/investigations`).
2. API writes `investigations.status=requested` and emits `investigation.requested`.
3. `services.ai` claims the job (`running`), calls **only** allowlisted tools, stores `evidence`.
4. Emits `investigation.completed` with `InvestigationResult` (or `status=failed`).
5. Incident domain appends timeline rows and `recommendations`. High-impact recs also get a `remediations` row in `pending_approval`.
6. Humans approve via the API. AI cannot approve or execute.

## Tools

Read-only, argument-validated, size-limited, audited. Implemented later as internal APIs, not as model-side plugins to production.

| Tool | Request (required) | Returns |
| --- | --- | --- |
| `get_recent_logs` | `service_id`, `from`, `to` | Bounded log entries |
| `get_metrics` | `service_id`, `name`, `from`, `to` | Time series points |
| `get_service_health` | `service_id` | Health snapshot |
| `get_recent_deployments` | `service_id` | Recent deployment summaries |
| `get_trace` | `trace_id` or error example | Spans |
| `search_runbooks` | `query` | Ranked excerpts |
| `search_previous_incidents` | `query` | Historical incident hits |
| `get_deployment_diff` | `deployment_id` | Change summary |

Tool errors use `{code, message}`. Timeouts count as `tool_usage.status=timeout` and must not crash the investigation (it may `failed` if too little evidence).
