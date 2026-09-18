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
packages/         Shared contracts, config, telemetry, and utilities
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
4. **Phase 3 — Telemetry ingestion + Kafka**
5. **Phase 4 — Detection engine**
6. **Phase 5 — Incident correlation**
7. **Phase 6 — AI investigation with restricted tools**
8. **Phase 7 — Recommendation & remediation**
9. **Phase 8 — Operator UI**
10. **Phase 9 — Observability**
11. **Phase 10 — Kubernetes, Terraform & AWS**
12. **Phase 11 — Failure injection, evaluation & final polish** (current)

## Current project status

**Phase 10 — Kubernetes, Terraform & AWS** is complete (manifests, Terraform, CI/CD foundations). AWS has not been applied from this repository. Next: Phase 11.

## Local infrastructure

```bash
cp .env.example .env
docker compose up -d
```

This starts PostgreSQL, Redis, Kafka, ingestion (`:8090`), detection (`:8091`), incident (`:8092`), AI (`:8000`), and remediation (`:8093`).

Optional observability stack (OTel `:4317`/`:4318`, Prometheus `:9090`, Grafana `:3002`):

```bash
docker compose --profile observability up -d
```

Run `apps/api` (`:8080`) and `apps/web` (`:3000`) on the host. See [observability/README.md](observability/README.md).

## Documentation

- [Development progress](docs/progress.md)
- [Architecture](docs/architecture/README.md)
- [Observability](docs/architecture/observability.md)
- [Domain model](docs/architecture/domain.md)
- [Kafka events](docs/events/README.md)
- [Data stores](docs/architecture/data.md)
- [AI investigation](docs/ai/README.md)
- [Security](docs/architecture/security.md)
- [Architecture decisions](docs/decisions/README.md)
- [HTTP API](docs/api/README.md)
- [Idempotency](docs/architecture/idempotency.md)
- [Deployment](docs/deployment.md)
