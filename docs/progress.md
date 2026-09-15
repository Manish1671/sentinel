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
investigation and remediation remain later phases.

## Phase 6 — AI Investigation
Status: ⏳ Not Started

## Phase 7 — Recommendation & Remediation
Status: ⏳ Not Started

## Phase 8 — Operator Dashboard
Status: ⏳ Not Started

## Phase 9 — Observability
Status: ⏳ Not Started

## Phase 10 — Kubernetes, Terraform & AWS
Status: ⏳ Not Started

## Phase 11 — Failure Injection, Evaluation & Final Polish
Status: ⏳ Not Started

## Git Checkpoints

Each completed phase should produce:

- a reviewed commit
- a pushed GitHub commit
- a phase tag
- an updated `docs/progress.md`

Do not mark a phase complete until that work is actually done.
Future phases remain **Not Started** until they are implemented.
