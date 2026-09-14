# packages/contracts

Language-neutral, versioned contracts (`*.v1`). No generated clients in Phase 1.

| Path | Contents |
| --- | --- |
| `enums.v1.json` | Shared constrained values |
| `events/envelope.v1.schema.json` | Kafka envelope |
| `events/*.v1.schema.json` | Event payloads |
| `ai/` | InvestigationRequest, InvestigationResult, tools |
| `api/error.v1.schema.json` | HTTP error envelope |
| `redis.md` | Key naming |

JSON Schema is draft 2020-12. Extra properties on payloads are forbidden for v1 required events; consumers must still ignore unknown fields if a producer on a newer patch added an additive property after a minor documentation update — treat unknown fields as ignorable at runtime even if schemas say `additionalProperties: false` for authoring.

HTTP narrative docs: `docs/api/`.
