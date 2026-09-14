# services/incident

Incident lifecycle and correlation service (Go).

## Responsibility

Turn related alerts and events into a durable incident record with a timeline.

Planned work:

- create incidents from correlated alerts
- update incident state
- maintain incident timelines (`IncidentEvent`)
- publish `incidents.created` and `incidents.updated`
- request investigations via `investigations.requested`

## Boundaries

- System of record for incident aggregates lives in PostgreSQL, owned by this service’s domain.
- `apps/api` orchestrates operator actions against this domain; it does not invent a second incident store.
- Does not execute remediation.

## Status

Not implemented. Domain documentation lives in `docs/architecture/domain.md`.
