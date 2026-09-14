# services/detection

Anomaly and rule detection service (Go).

## Responsibility

Consume normalized telemetry, evaluate rules and detectors, and emit detection outcomes.

Planned work:

- consume `telemetry.events`
- evaluate thresholds and rules
- identify suspicious or anomalous behavior
- emit `alerts.created`
- remain stateless where possible, with durable config in PostgreSQL

## Boundaries

- Owns alert generation, not incident lifecycle.
- Does not open incidents directly; `services/incident` correlates alerts into incidents.
- Does not call the AI investigator.

## Status

Not implemented. Detection algorithms are deferred until the event pipeline exists.
