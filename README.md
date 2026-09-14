# Sentinel

AI-native production incident detection, investigation, and remediation.

## Problem

Production incidents are still too slow to detect, too expensive to investigate, and too risky to remediate. Telemetry is fragmented across logs, metrics, traces, and deployments. Engineers reconstruct timelines by hand. Remediation is either delayed by uncertainty or executed without a clear audit trail.

## Product objective

Sentinel is a platform that turns system telemetry into an end-to-end incident workflow:

Telemetry and system events → ingestion → streaming → anomaly detection → incident creation → AI investigation → root-cause analysis → remediation recommendation → human approval → remediation → health verification → incident resolution.

The product is designed as a serious production system, not a CRUD app or a chatbot wrapper. AI investigation is bounded by explicit tools. Remediation is authorized, auditable, and never unrestricted.

## High-level architecture

```
[Sources]          [Platform]                         [Operators]
telemetry    →     ingestion  → Kafka  → detection
deployments        api (Go)            → incident     → web (Next.js)
health checks      ai (Python)         → remediation
```

- **apps/web** is the operator console.
- **apps/api** is the primary HTTP API and orchestration layer.
- **services/** own domain workflows (ingestion, detection, incident, remediation, AI).
- **PostgreSQL** is the system of record.
- **Kafka** carries domain events between services.
- **Redis** holds short-lived operational state.
- **S3-compatible storage** holds large artifacts (logs dumps, traces, investigation evidence).

See [docs/architecture/README.md](docs/architecture/README.md) for the full architecture.

## Technology stack

| Area | Choice |
| --- | --- |
| Frontend | Next.js, TypeScript, Tailwind CSS, TanStack Query, Recharts |
| API | Go |
| AI service | Python, FastAPI |
| Data | PostgreSQL, Redis, S3-compatible object storage |
| Streaming | Apache Kafka |
| Observability | OpenTelemetry, Prometheus, Grafana |
| Infrastructure | Docker, Kubernetes, Terraform |
| Cloud | AWS |
| CI/CD | GitHub Actions |

## Repository structure

```
apps/             Operator UI and primary API
services/         Domain services (ingestion, detection, incident, remediation, AI)
packages/         Shared contracts, config, and utilities
infrastructure/   Docker, Kubernetes, Terraform, AWS
observability/    Prometheus, Grafana, OpenTelemetry
database/         Migrations and seeds
docs/             Architecture, APIs, events, ADRs, AI
scripts/          Local and operational scripts
```

## Development roadmap

1. **Phase 0 — Architecture & repository initialization**
2. **Phase 1 — Contracts & core domain**
3. **Phase 2 — Go API control plane foundation**
4. **Phase 3 — Telemetry ingestion + Kafka** (current)
5. **Phase 4 — Detection and incident lifecycle**
6. **Phase 5 — AI investigation with restricted tools**
7. **Phase 6 — Remediation with human approval**
8. **Phase 7 — Operator UI**
9. **Phase 8 — Observability, hardening, and Kubernetes/AWS delivery**

## Current project status

**Phase 3 — Telemetry Ingestion + Kafka**

The repository includes architecture docs, PostgreSQL schema/seeds, a Go control-plane API, and a Go ingestion service that publishes canonical telemetry and deployment events to Kafka.

It does **not** yet include detection, the operator UI, Kafka consumers as product services, remediation execution, or the AI agent.

## Local infrastructure

```bash
cp .env.example .env
docker compose up -d
```

This starts PostgreSQL, Redis, Kafka, and the ingestion service (`:8090`).

## Documentation

- [Development progress](docs/progress.md)
- [Architecture](docs/architecture/README.md)
- [Domain model](docs/architecture/domain.md)
- [Kafka events](docs/events/README.md)
- [Data stores](docs/architecture/data.md)
- [AI investigation](docs/ai/README.md)
- [Security](docs/architecture/security.md)
- [Architecture decisions](docs/decisions/README.md)
- [HTTP API](docs/api/README.md)
- [Idempotency](docs/architecture/idempotency.md)
