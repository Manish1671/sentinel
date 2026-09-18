# Evaluation and chaos harness

Phase 11 proves Sentinel’s detect → correlate → investigate → recommend → approve → remediate → verify path with **controlled, deterministic** failures.

This is not a published model-quality benchmark. Durations are recorded only when timestamps exist. Missing times are omitted.

## Commands

Offline (no live stack):

```bash
PYTHONPATH=. pytest -q evaluation/tests
```

Live local stack (Compose pipeline + `apps/api` on `:8080`):

```bash
docker compose up -d
# start apps/api on :8080 if it is not already running
PYTHONPATH=. python -m evaluation.chaos.run
```

Outputs: `evaluation/results/<run-id>.json` and `.md` (gitignored).

Keyword case from Phase 6 (unchanged):

```bash
python evaluation/runner.py
```

## Fault injection

Disabled unless `SENTINEL_FAULT_INJECTION=true` **and** a specific flag:

| Flag | Effect |
| --- | --- |
| `REMEDIATION_FAULT_VERIFY_FAIL` | Simulator leaves health/latency above verification thresholds |
| `REMEDIATION_FAULT_EXECUTE_FAIL` | Simulator returns an execute error |
| `AI_FAULT_FAIL_INVESTIGATION` | AI processor fails the investigation explicitly |

Default Compose does **not** set these. Do not enable in production.

## Scenarios

| ID | Injects | Expected | Typical execution |
| --- | --- | --- | --- |
| CHAOS-001 | High latency + deployment | `high_latency` alert, correlation, investigation | live HTTP |
| CHAOS-002 | High error rate | `high_error_rate` | live HTTP |
| CHAOS-003 | DB utilization | `db_connection_saturation` | live HTTP |
| CHAOS-004 | Replay same `event_id` | ingest replay; no extra alert row | live HTTP |
| CHAOS-005 | Viewer approve | HTTP 403; pending remediation unchanged | live HTTP |
| CHAOS-006 | Verify-fail flag | remediation failed; incident stays active | Go integration (`TestVerificationFailureKeepsIncidentActive`) unless flags are on the remediation process |
| CHAOS-007 | AI fail flag | investigation `failed`; no success recs | pytest `test_fault_injection_fails_investigation` unless flags are on the AI process |
| CHAOS-008 | Same `Idempotency-Key` | same incident id | live HTTP |

New alerts use detector ids `high_latency` / `high_error_rate` / `db_connection_saturation`. Seeded alerts use different detector names, so they are not confused with chaos alerts.

## Categories

Detection, correlation, investigation completion, evidence grounding, recommendation validity, approval enforcement, remediation correctness, recovery verification, idempotency, failure handling.

AI checks are schema, evidence ids, allowlisted actions, OBSERVED/INFERENCE/RECOMMENDATION separation, confidence in `[0,1]`.

## Pass / fail

A live scenario passes only if its assertions pass after talking to real services. `not_executed` means the stack or required fault flags were missing — that is **not** a pass.

## What was executed locally (2026-09-18)

Live Compose + `apps/api` (`python -m evaluation.chaos.run`, run `20260918T031540Z-f3cdcea8`):

| Scenario | Status |
| --- | --- |
| CHAOS-001 latency | **LOCALLY EXECUTED, PASSED** (open `high_latency` fingerprint reused; investigation completed on INC-2026-0012) |
| CHAOS-002 error rate | **LOCALLY EXECUTED, PASSED** |
| CHAOS-003 DB saturation | **LOCALLY EXECUTED, PASSED** |
| CHAOS-004 duplicate telemetry | **LOCALLY EXECUTED, PASSED** (same alert id; summary updated to 1600ms) |
| CHAOS-005 unauthorized approve | **LOCALLY EXECUTED, PASSED** (HTTP 403, remediation still `pending_approval`) |
| CHAOS-006 verify-fail | **IMPLEMENTED**; live Compose **NOT EXECUTED** (flags off). **PASSED** via Go `TestVerificationFailureKeepsIncidentActive` |
| CHAOS-007 AI fail | **IMPLEMENTED**; live Compose **NOT EXECUTED** (flags off). **PASSED** via pytest `test_fault_injection_fails_investigation` |
| CHAOS-008 idempotency key | **LOCALLY EXECUTED, PASSED** (201 then 200, same incident id) |

Detection/correlation durations were omitted when the open alert `started_at` predated this emit (fingerprint cooldown). That is not a fabricated latency.

- `apps/api` does not POST investigations; the runner uses `services/ai` `POST /api/v1/incidents/{id}/investigations`, which publishes Kafka (documented local producer).
- Live telemetry is Kafka-backed for detection. AI tools read PostgreSQL samples; live metrics may appear as `no_evidence` while alerts/deployments on the incident still ground the heuristic.
- New payments-api alerts usually **attach** to seeded `INC-2026-0004` (still active). That is correct correlation, not a new incident every run.
- Approving remediations mutates local demo state; CHAOS-001/002 request investigation but do not approve.
- CHAOS-006/007 need process-level env flags for a live Compose path; in-process tests are the default executed proof.
