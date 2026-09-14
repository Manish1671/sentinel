# Domain model

Phase 1 canonical model. Identifiers are UUID v4 unless noted. Timestamps are UTC (`timestamptz`).

Ownership is the service allowed to **write** the aggregate. Other components may read via API or events.

Shared constrained values live in [`packages/contracts/enums.v1.json`](../../packages/contracts/enums.v1.json).

---

## User

**Purpose.** A human operator of Sentinel. Every mutating incident, investigation request, approval, and remediation is attributed to a user.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `apps/api` |
| Audit | Create/disable are API audit concerns; domain actions reference `actor_user_id` on timeline and approval rows |

**Fields.** `email` (unique), `display_name`, `role` (`viewer` \| `responder` \| `approver` \| `admin`), `status` (`active` \| `disabled`), `password_hash` (local/dev credential store; not used until auth is implemented), `created_at`, `updated_at`.

**Relationships.** Creates and commands incidents; requests investigations; decides approvals; requests remediations; authors runbooks.

**Lifecycle.** `active` → `disabled`. Disabled users cannot authenticate. Historical rows keep the original `id`.

---

## Service

**Purpose.** A production system Sentinel watches (for example `payments-api` in `production`).

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `apps/api` (catalog) |
| Audit | Catalog changes later; not on the incident timeline |

**Fields.** `slug`, `name`, `environment` (`production` \| `staging` \| `development`), `description`, `owner_user_id`, `health_status` (`healthy` \| `degraded` \| `unhealthy` \| `unknown`), `created_at`, `updated_at`. Unique `(slug, environment)`.

**Relationships.** Has many deployments, telemetry samples, alerts, incidents, runbooks.

---

## Deployment

**Purpose.** A release of a service. Used to correlate incidents with change and to support `get_deployment_diff`.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `apps/api` (catalog); ingestion may emit `deployment.created` |
| Audit | Immutable after insert except `status` / `completed_at` |

**Fields.** `service_id`, `version`, `git_sha`, `status` (`in_progress` \| `succeeded` \| `failed` \| `rolled_back`), `changelog`, `deployed_by_user_id`, `started_at`, `completed_at`, `metadata` (JSON).

**Relationships.** Belongs to one service. Evidence and recommendations may reference a deployment id.

**Lifecycle.** `in_progress` → `succeeded` \| `failed`; `succeeded` → `rolled_back`.

---

## TelemetryEvent

**Purpose.** A normalized observation (metric, log, trace, or health sample). Kafka (`telemetry.events`) is the hot path. PostgreSQL stores a **retained subset** for seeds, tools, and evidence pointers — not the full firehose.

| | |
| --- | --- |
| Primary key | `id` (UUID), equal to Kafka `event_id` when persisted |
| Ownership | `services/ingestion` |
| Audit | Events are append-only |

**Fields.** `service_id`, `kind` (`metric` \| `log` \| `trace` \| `health`), `source` (collector/agent name), `occurred_at`, `ingested_at`, `correlation_id`, `payload` (JSON, kind-specific).

**Relationships.** Belongs to a service. Alerts and evidence may point at a telemetry id.

---

## Alert

**Purpose.** A detector fired. An alert is not an incident.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/detection` |
| Audit | Status changes may attach to an incident timeline once correlated |

**Fields.** `service_id`, `detector_id`, `severity` (`critical` \| `high` \| `medium` \| `low`), `status` (`open` \| `acknowledged` \| `resolved` \| `suppressed`), `title`, `summary`, `fingerprint` (dedup token), `labels` (JSON), `triggering_event_id`, `started_at`, `ended_at`, `created_at`, `updated_at`.

**Uniqueness.** At most one `open` or `acknowledged` alert per `(service_id, fingerprint)`.

**Relationships.** Belongs to a service. Many-to-many with incidents via `incident_alerts`. Optional link to a telemetry event.

**Lifecycle.** `open` → `acknowledged` → `resolved`; `open` → `resolved`; `open` \| `acknowledged` → `suppressed`.

---

## Incident

**Purpose.** The operator aggregate: correlated alerts, investigations, recommendations, remediations, and a timeline.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Natural key | `reference` (`INC-YYYY-NNNN`) |
| Ownership | `services/incident` (`apps/api` orchestrates writes through this domain) |
| Audit | All significant changes append `IncidentEvent`. `version` is optimistic concurrency |

**Fields.** `reference`, `service_id`, `title`, `summary`, `severity`, `status`, `created_by_user_id`, `commander_user_id`, `detected_at`, `resolved_at`, `closed_at`, `dedup_key` (nullable unique; incident-creation idempotency), `version` (integer, starts at 1), `created_at`, `updated_at`.

**Relationships.** Belongs to one primary service. Many alerts. Many incident events, investigations, recommendations, remediations.

**Lifecycle.** See [Incident state machine](#incident-state-machine).

---

## IncidentEvent

**Purpose.** Immutable timeline and audit record for an incident.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/incident` |
| Audit | Append-only. Never update `payload` or `occurred_at` |

**Fields.** `incident_id`, `kind` (see below), `actor_user_id` (null for system), `occurred_at`, `payload` (JSON).

**Kinds.** `created`, `status_changed`, `alert_attached`, `comment`, `commander_changed`, `investigation_requested`, `investigation_completed`, `recommendation_added`, `approval_recorded`, `remediation_requested`, `remediation_completed`, `remediation_failed`.

**Relationships.** Belongs to one incident.

---

## Investigation

**Purpose.** A bounded AI (or later, human) investigation of an incident.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/ai` writes results; `services/incident` / `apps/api` create the request row |
| Audit | Request and completion are timeline events. Tool calls are summarized on the row and in evidence |

**Fields.** `incident_id`, `status`, `requested_by_user_id`, `model_name`, `model_version`, `root_cause_hypothesis`, `reasoning_summary`, `confidence` (0–1), `risk_level`, `tool_usage` (JSON), `error_message`, `idempotency_key` (unique), `requested_at`, `started_at`, `completed_at`.

**Relationships.** Belongs to one incident. Has many evidence rows and recommendations.

**Lifecycle.** See [Investigation state machine](#investigation-state-machine).

---

## Evidence

**Purpose.** A retrieved artifact used during investigation. Metadata in PostgreSQL; bulky bodies in object storage.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/ai` |
| Audit | Append-only |

**Fields.** `investigation_id`, `tool_name`, `source_type` (`logs` \| `metrics` \| `trace` \| `deployment` \| `runbook` \| `previous_incident` \| `health`), `summary`, `artifact_uri` (nullable), `source_ref` (id of log/metric/trace/deployment/runbook/incident), `metadata` (JSON), `captured_at`.

**Relationships.** Belongs to one investigation. Recommendations refer to evidence by id in JSON, not a mandatory FK, to keep the model small.

---

## Recommendation

**Purpose.** A proposed next action (usually a remediation) with confidence and risk.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/ai` (proposed); operators change status via `apps/api` |
| Audit | `recommendation_added` on the timeline; later status via `IncidentEvent` and remediations |

**Fields.** `incident_id`, `investigation_id` (nullable for a future human-authored rec), `action_type`, `title`, `rationale`, `target_service_id`, `parameters` (JSON), `confidence`, `risk_level` (`low` \| `medium` \| `high`), `required_approval_role` (`approver` \| `admin`), `status`, `created_at`, `updated_at`.

**Relationships.** Belongs to incident and optional investigation. May spawn remediations.

**Lifecycle.** See [Recommendation state machine](#recommendation-state-machine).

---

## Remediation

**Purpose.** An execution record for an approved (or pending) recommended action, including verification.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/remediation` |
| Audit | Timeline events plus this row. Never delete. Failed attempts remain |

**Fields.** `incident_id`, `recommendation_id`, `service_id`, `status`, `action_type`, `parameters` (JSON), `requested_by_user_id`, `attempt_number`, `idempotency_key` (unique), `result_summary`, `verification_status` (`pending` \| `passed` \| `failed` \| `skipped`), `verification_details` (JSON), `error_message`, `started_at`, `completed_at`, `created_at`, `updated_at`.

**Relationships.** Belongs to incident, recommendation, service. Has one approval.

**Lifecycle.** See [Remediation state machine](#remediation-state-machine).

---

## Approval

**Purpose.** A recorded human decision on a remediation. Destructive and high-impact actions require an approved row before execution.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/remediation` (written through `apps/api`) |
| Audit | The row is the audit record (`actor_user_id`, `decision`, `comment`, `decided_at`) plus `approval_recorded` on the timeline |

**Fields.** `remediation_id` (unique), `incident_id`, `recommendation_id`, `decision` (`pending` \| `approved` \| `rejected`), `actor_user_id`, `comment`, `created_at`, `decided_at`.

**Relationships.** One-to-one with a remediation.

**Lifecycle.** See [Approval state machine](#approval-state-machine).

---

## Runbook

**Purpose.** Human-owned operational guidance. Searchable via `search_runbooks`. Not generated as source of truth.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `apps/api` (catalog) |
| Audit | `updated_at` and `created_by_user_id`; body history is out of scope for Phase 1 |

**Fields.** `service_id` (nullable = platform-wide), `slug` (unique), `title`, `failure_class`, `body`, `status` (`draft` \| `published`), `created_by_user_id`, `created_at`, `updated_at`.

---

## HistoricalIncident

**Purpose.** A resolved or closed incident retained for similarity search and learning.

**Not a separate write table.** It is a PostgreSQL view over `incidents` where `status IN ('resolved', 'closed')`, exposing search-friendly columns (`id`, `reference`, `service_id`, `title`, `severity`, `summary`, `resolved_at`). Live rows are never copied. Investigations, evidence, and remediations remain on the original incident.

| | |
| --- | --- |
| Primary key | `id` (same as `incidents.id`) |
| Ownership | Derived from `services/incident` |
| Audit | Same as the underlying incident |

---

## EvaluationRun

**Purpose.** A scored replay of the investigator against a labeled (usually historical) incident. Does not change production incident state.

| | |
| --- | --- |
| Primary key | `id` (UUID) |
| Ownership | `services/ai` |
| Audit | Row is the record of the run |

**Fields.** `name`, `status` (`pending` \| `running` \| `completed` \| `failed`), `labeled_incident_id`, `model_name`, `model_version`, `metrics` (JSON: scores, notes), `error_message`, `started_at`, `completed_at`, `created_at`.

**Lifecycle.** `pending` → `running` → `completed` \| `failed`.

---

## Incident state machine

```
open ──► investigating ──► remediating ──► verifying ──► resolved ──► closed
  │            │                │              │            │
  │            ├─► resolved     ├─► investigating          └─► open (reopen)
  │            └─► open         └─► failed path via investigating
  └─► resolved (no action / false positive)

verifying ──► remediating (verification failed, retry)
verifying ──► investigating (verification failed, new analysis)
```

| From | To | Trigger |
| --- | --- | --- |
| `open` | `investigating` | Investigation requested |
| `open` | `resolved` | False positive / no action |
| `investigating` | `remediating` | Remediation starts running |
| `investigating` | `resolved` | Investigation finds no action |
| `investigating` | `open` | Investigation cancelled with no other work |
| `remediating` | `verifying` | Action finished, health check begins |
| `remediating` | `investigating` | Action failed; new analysis |
| `verifying` | `resolved` | Health check passed |
| `verifying` | `remediating` | Retry same action |
| `verifying` | `investigating` | Health check failed |
| `resolved` | `closed` | Operator closes |
| `resolved` | `open` | Reopen |
| `closed` | — | Terminal in Phase 1 |

Illegal skips (for example `open` → `verifying`) are API `409` / `incident_invalid_transition`.

---

## Investigation state machine

```
requested ──► running ──► completed
                 │
                 ├─► failed
requested/running ──► cancelled
```

| From | To | Trigger |
| --- | --- | --- |
| `requested` | `running` | AI worker claims the job |
| `running` | `completed` | `investigation.completed` with success |
| `running` | `failed` | Worker or model error |
| `requested` \| `running` | `cancelled` | Operator or incident close |

A completed investigation does not auto-execute remediation.

---

## Recommendation state machine

```
proposed ──► accepted
proposed ──► rejected
proposed ──► superseded
proposed ──► expired
```

| From | To | Trigger |
| --- | --- | --- |
| `proposed` | `accepted` | Remediation for this rec is approved (or operator accepts) |
| `proposed` | `rejected` | Remediation rejected, or operator rejects the rec |
| `proposed` | `superseded` | A newer investigation replaces it |
| `proposed` | `expired` | TTL / incident closed without decision |

Terminal: `accepted`, `rejected`, `superseded`, `expired`.

---

## Remediation state machine

```
pending_approval ──► approved ──► running ──► verifying ──► succeeded
       │                 │            │
       └─► rejected      └─► cancelled ├─► failed
                         approved ──► cancelled
```

| From | To | Trigger |
| --- | --- | --- |
| `pending_approval` | `approved` | `POST .../approve` by `approver` or `admin` |
| `pending_approval` | `rejected` | `POST .../reject` |
| `approved` | `running` | Executor claims work |
| `approved` | `cancelled` | Operator cancel before start |
| `running` | `verifying` | Action reported success |
| `running` | `failed` | Action error (`remediation.failed`) |
| `verifying` | `succeeded` | Health verification passed |
| `verifying` | `failed` | Health verification failed |

High-impact `action_type` values (`rollback_deployment`, `restart_service`, `scale_replicas`, `disable_feature_flag`) **must** pass through `pending_approval` → `approved` before `running`.

---

## Approval state machine

```
pending ──► approved
pending ──► rejected
```

Terminal after `approved` or `rejected`. Duplicate submit of the **same** decision is idempotent. A different decision against a terminal approval is `409`.

---

## Entities not added

No separate Tenant, Comment, Detector, or Timeline projection tables. Comments are `IncidentEvent` rows with `kind = comment`. Detectors are `alerts.detector_id` strings until detection is implemented.
