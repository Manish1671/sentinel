# Architecture

Phase 1 contracts are specified in `docs/` and `packages/contracts/`. Operator console visual foundation: [frontend.md](./frontend.md).

## 1. Product overview

Sentinel is an AI-native platform for detecting, investigating, and remediating production incidents.

Operators see a console. Backend services ingest telemetry, detect anomalies, open incidents, run bounded AI investigations, and execute only approved remediations. The loop closes with health verification and incident resolution.

## 2. Architecture overview

Sentinel is a **modular monolith-friendly set of services**, not a mesh of tiny microservices. Boundaries exist so each domain can scale and fail independently, but we will not split further until a boundary is proven.

```
                 ┌──────────── apps/web ────────────┐
                 │         operator console         │
                 └───────────────┬──────────────────┘
                                 │ HTTP
                 ┌───────────────▼──────────────────┐
                 │            apps/api              │
                 │     auth, catalog, orchestration │
                 └───────┬───────────┬──────────────┘
                         │           │
           ┌─────────────▼──┐   ┌────▼─────────────┐
           │ services/*     │   │ services/ai      │
           │ ingestion      │   │ investigation    │
           │ detection      │   └──────────────────┘
           │ incident       │
           │ remediation    │
           └───────┬────────┘
                   │
     Kafka events  │  PostgreSQL (system of record)
                   │  Redis (ephemeral)
                   │  S3-compatible (artifacts)
```

## 3. Service responsibilities

| Component | Role |
| --- | --- |
| `apps/web` | Operator UI |
| `apps/api` | Authn/z, HTTP API, orchestration |
| `services/ingestion` | Normalize and publish telemetry |
| `services/detection` | Rules/anomalies → alerts |
| `services/incident` | Correlate alerts → incidents and timelines |
| `services/ai` | Tool-bounded investigation and recommendations |
| `services/remediation` | Approval, execution, verification, audit |

Details live in each service README.

## 4. Communication patterns

- **Synchronous HTTP** between web and API, and for operator-triggered commands (acknowledge, approve, comment).
- **Asynchronous Kafka** for the detection → incident → investigation → remediation pipeline.
- **Internal service calls** only where a request/response is required (for example API asking incident service for a consistent read). Prefer events for fan-out.
- **No UI-to-Kafka** and **no AI-to-production-shell**.

## 5. Data ownership

| Data | Owner | Store |
| --- | --- | --- |
| Users, roles, sessions | `apps/api` | PostgreSQL / Redis |
| Service catalog, deployments | `apps/api` (catalog) | PostgreSQL |
| Raw/normalized telemetry | `services/ingestion` | Kafka; overflow in object storage |
| Alerts | `services/detection` | PostgreSQL + `alerts.created` |
| Incidents, timelines | `services/incident` | PostgreSQL + incident topics |
| Investigations, evidence, recommendations | `services/ai` + incident records | PostgreSQL + object storage |
| Approvals, remediations | `services/remediation` | PostgreSQL + remediation topics |
| Runbooks, historical incidents | incident/catalog domains | PostgreSQL |
| Evaluation runs | `services/ai` | PostgreSQL |

Other services may **read** owned data through APIs or events. They must not write another service’s aggregate directly.

## 6. Event flow

```
telemetry.events
      → detection → alerts.created
      → incident  → incidents.created / incidents.updated
                  → investigations.requested
      → ai        → investigations.completed
                  → (operator approval via API)
                  → remediation.requested
      → remediation → remediation.completed
system.health is a cross-cutting health signal topic
```

See [docs/events/README.md](../events/README.md). Idempotency: [idempotency.md](./idempotency.md). Observability: [observability.md](./observability.md).

## 7. AI investigation flow

An incident (or an operator) requests an investigation. The AI service loads incident context, calls only registered tools, writes evidence, scores recommendations, and emits `investigations.completed`. Humans approve any remediation. See [docs/ai/README.md](../ai/README.md).

## 8. Security boundaries

Least privilege, service identity, RBAC, secrets outside source control, restricted AI tools, auditable remediations, explicit approval for destructive actions. See [security.md](./security.md).

## 9. Local development architecture

Developers run infrastructure with Docker Compose:

- PostgreSQL on `:5432`
- Redis on `:6379`
- Kafka (KRaft, single node) on `:9092`
- Ingestion, detection, incident, AI, and remediation as Compose services
- Optional profile `observability`: OTel Collector (`:4317`/`:4318`), Prometheus (`:9090`), Grafana (`:3002`)

`apps/api` (`:8080`) and `apps/web` (`:3000`) typically run on the host.

## 10. Eventual production architecture

On AWS (Terraform + EKS overlay; **apply is operator-driven and not assumed done**):

- Kubernetes (EKS) for apps and services
- RDS PostgreSQL, ElastiCache Redis, S3
- Kafka via existing `KAFKA_BROKERS` (no MSK module in this phase)
- OpenTelemetry → Prometheus/Grafana (Compose locally; same contracts in-cluster)
- GitHub Actions for CI; Terraform for infrastructure
- IAM roles for workloads (IRSA); no embedded production credentials

See [deployment.md](../deployment.md).

## 11. Kubernetes communication

Pods talk over ClusterIP Services, not localhost. The operator console proxies to `API_URL` (`http://api:8080` in-cluster). The control plane calls `REMEDIATION_URL` (`http://remediation:8093`). Kafka consumers use `KAFKA_BROKERS`.
