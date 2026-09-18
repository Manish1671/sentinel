# Sentinel Development Progress

## Project Objective

Sentinel is an AI-native production incident detection,
investigation, and remediation platform.

Core workflow:

Detect → Investigate → Recommend → Fix → Verify

## Phase 0 — Architecture & Repository Initialization
Status: ✅ Complete

Summary:
Established the monorepo structure, technology stack,
local PostgreSQL/Redis/Kafka infrastructure, documentation
structure, and high-level service boundaries.

## Phase 1 — Contracts & Core Domain
Status: ✅ Complete

Summary:
Defined the domain model, PostgreSQL schema and migrations,
Kafka event contracts, API contracts, lifecycle states,
idempotency behavior, and AI service contracts.

## Phase 2 — Go API Control Plane
Status: ✅ Complete

Summary:
Implemented the Go control-plane API with PostgreSQL,
authentication, authorization, service and incident APIs,
health/readiness endpoints, and database-backed idempotency.

## Phase 3 — Telemetry Ingestion + Kafka
Status: ✅ Complete

Summary:
Implemented the standalone Go ingestion service that validates
telemetry, creates canonical event envelopes, and publishes
telemetry and deployment events to Kafka using at-least-once delivery.

## Phase 4 — Detection Engine
Status: ✅ Complete

Summary:
Implemented the standalone Go detection service that consumes Kafka telemetry,
evaluates deterministic rules, persists alerts in PostgreSQL, and publishes
`alert.created` with at-least-once, idempotent processing.

## Phase 5 — Incident Correlation
Status: ✅ Complete

Summary:
Implemented the standalone Go incident service that consumes `alert.created`,
correlates related alerts into operational incidents, persists associations and
an append-only timeline, and publishes `incident.created` / `incident.updated`
with at-least-once, idempotent processing. New incidents start `open`; AI
investigation is Phase 6; remediation execution is Phase 7.

## Phase 6 — AI Investigation
Status: ✅ Complete

Summary:
Implemented the standalone Python FastAPI AI service that consumes
`investigation.requested`, gathers bounded PostgreSQL-backed evidence through
allowlisted tools, synthesizes a validated `InvestigationResult`, persists
investigations/evidence/recommendations, and publishes `investigation.completed`.
Remediation execution is implemented in Phase 7.

## Phase 7 — Recommendation & Remediation
Status: ✅ Complete

Summary:
Implemented the standalone Go remediation service that validates AI
recommendations, requires human approval, executes only allowlisted actions
against a local simulator, verifies multiple health signals, and updates
incident lifecycle and timeline. Kubernetes/AWS execution remains later phases.

## Phase 8 — Operator Dashboard
Status: ✅ Complete

Phase 8A added the operator console design system, application shell,
reusable components, and isolated mock fixtures in `apps/web`.

## Phase 8B — Real API Integration
Status: ✅ Complete

Wired the Phase 8A console to `apps/api` through a same-origin Next.js
proxy and HttpOnly session cookie: real auth, catalog, incidents,
investigations, recommendations, remediations, and approval delegation
to `services/remediation`. Historical telemetry APIs are still
unavailable and render as explicit empty states.

## Phase 8C — Flagship Incident Workspace
Status: ✅ Complete

Polished `/incidents/[id]` as the primary SRE workspace using real
`apps/api` data: header and summary strip, correlation signals from
incident-service metadata, timeline backbone, correlated alerts,
investigation / evidence / confidence, recommendation approval UX,
and Phase 7 remediation lifecycle. Historical telemetry is still not
exposed by the control plane and stays an explicit unavailable state.

## Phase 8D — Final Operator Experience
Status: ✅ Complete

Finished the operator product surface without changing the Phase 8A
visual language: `/overview` as a live production snapshot, `/services/[id]`
from catalog data, polished incident / investigation / remediation /
deployment lists, command navigation, and control-plane readiness in
the top bar. Live updates remain polling (`src/lib/refresh.ts`); SSE is
deferred because `apps/api` has no event stream. Observability and
Kubernetes/AWS are later phases.

## Phase 9 — Observability
Status: ✅ Complete

Summary:
Instrumented the Go API, ingestion, detection, incident, and remediation
services plus the Python AI service with structured JSON logs, Prometheus
`/metrics`, and OpenTelemetry traces (W3C HTTP and Kafka headers). Local
Collector, Prometheus, and Grafana run under Compose profile `observability`.
Kubernetes/AWS remain Phase 10.

## Phase 10 — Kubernetes, Terraform & AWS
Status: ✅ Complete

Summary:
Added Kubernetes manifests (base + local/aws overlays) for the seven
stateless workloads, Terraform modules for VPC/EKS/RDS/ElastiCache/S3/IAM,
and GitHub Actions CI (tests + kustomize/kubeconform + terraform validate),
image build (SHA tags; push gated), and a manual deploy workflow.
Local Docker Compose is unchanged. Kafka is still `KAFKA_BROKERS` (no MSK).
AWS `terraform apply` was **not** run; no cloud resources were created.
CPU/memory values are starting limits, not load-tested capacity.

## Phase 11 — Failure Injection, Evaluation & Final Polish
Status: ✅ Complete

Summary:
Added a deterministic chaos/evaluation harness (`evaluation/chaos`) that injects
telemetry through ingestion and asserts detection, correlation, investigation,
approval, and idempotency on the live local stack. Fault injection is off unless
`SENTINEL_FAULT_INJECTION` plus a specific flag is set. Live run `20260918T031540Z-f3cdcea8`
passed CHAOS-001..005 and CHAOS-008. CHAOS-006/007 were executed in-process
(Go `TestVerificationFailureKeepsIncidentActive`, pytest
`test_fault_injection_fails_investigation`), not by mutating Compose env.
Detection latency was not reported for reused open fingerprints (alerts from
2026-09-15). No AWS resources were created.

## Git Checkpoints

Each completed phase should produce:

- a reviewed commit
- a pushed GitHub commit
- a phase tag
- an updated `docs/progress.md`

Do not mark a phase complete until that work is actually done.
Future phases remain **Not Started** until they are implemented.
