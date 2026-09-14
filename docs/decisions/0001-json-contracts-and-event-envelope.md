# ADR 0001: JSON contracts and event envelope field `source`

- Status: accepted
- Date: 2026-09-14

## Context

Phase 0 documented a Kafka envelope with a `producer` field. Phase 1 requires a canonical envelope using `source`, plus language-neutral schemas the Go API and Python AI service can share without generating code yet.

## Decision

- Store versioned JSON Schema under `packages/contracts/` (`*.v1`).
- Name the producer field `source` on every event.
- Keep HTTP narrative docs in `docs/api/` rather than a second OpenAPI source of truth until the API is implemented.

## Consequences

- Phase 0 `producer` is not part of the v1 contract.
- Adding fields inside a version must be additive; breaking changes bump `event_version` or `v1` → `v2` in the filename.
