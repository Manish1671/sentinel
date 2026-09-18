# AWS

Production landing zone is defined in Terraform (`../terraform/`) and consumed by Kubernetes (`../kubernetes/overlays/aws`).

Intended mapping:

| Concern | AWS |
| --- | --- |
| Compute | EKS |
| PostgreSQL | RDS |
| Redis | ElastiCache |
| Artifacts | S3 |
| Ingress | AWS Load Balancer Controller + ALB (install the controller separately) |
| Kafka | Operator-managed; `KAFKA_BROKERS` only |
| Identity | IRSA, no long-lived keys in pods |

This directory stays documentation-first. Apply order is in [docs/deployment.md](../../docs/deployment.md).
