# Architecture

Canonical architecture notes live in this folder. Start with [architecture/README.md](./architecture/README.md).

Phase 10 adds Kubernetes and AWS **foundations** without changing product workflows:

- Local: Docker Compose (PostgreSQL, Redis, Kafka, domain services, optional OTel/Prometheus/Grafana).
- Kubernetes: stateless Sentinel workloads + Ingress. Local overlay may run Postgres/Redis/Kafka for kind only.
- AWS: EKS + RDS + ElastiCache + S3 via Terraform. Kafka remains `KAFKA_BROKERS`.

Deployment mechanics: [deployment.md](./deployment.md). Observability: [architecture/observability.md](./architecture/observability.md).
