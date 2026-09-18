# Deployment

This document covers **how** Sentinel is meant to run. It does not claim that AWS has been applied from this repository.

## 1. Local Docker Compose (inner loop)

Validated daily for application development:

```bash
cp .env.example .env
docker compose up -d
docker compose --profile observability up -d   # optional
```

Then run `apps/api` on `:8080` and `apps/web` on `:3000` (or use images if you add them to Compose). Pipeline services already run in Compose.

Do not change Compose networking for Kubernetes. Kubernetes uses Service DNS (`api`, `kafka`, `postgres`) instead of `localhost`.

## 2. Kubernetes

Workloads: `api`, `ingestion`, `detection`, `incident`, `ai`, `remediation`, `web`.

Internal DNS examples (`namespace` `sentinel`):

- `http://api:8080`
- `http://ingestion:8090`
- `http://remediation:8093`
- `kafka:9092`

Configuration: ConfigMap `sentinel-config` (non-secret). Secrets: Secret `sentinel-secrets` (`DATABASE_URL`, `REDIS_URL`, `AUTH_TOKEN_SECRET`, `AI_API_KEY`).

Migrations: `kubectl apply -f infrastructure/kubernetes/base/migrate-job.yaml` after secrets exist (`/app/api migrate`).

## 3. AWS (design)

Terraform creates VPC, EKS, RDS PostgreSQL, ElastiCache Redis, S3, and IAM (IRSA). Kafka is **not** created.

Suggested sequence (operator-controlled):

1. `terraform init` / `plan` / `apply` in `infrastructure/terraform`
2. Install AWS Load Balancer Controller on EKS (not bundled)
3. Create Kubernetes secret from RDS Secrets Manager + Redis endpoint
4. Set `KAFKA_BROKERS` to the operator-managed brokers
5. `kubectl apply -k infrastructure/kubernetes/overlays/aws`
6. Run the migrate Job

GitHub workflow `.github/workflows/deploy.yml` is **manual** (`workflow_dispatch`). Terraform apply and kubectl apply are separate flags (apply defaults off). They skip if `AWS_DEPLOY_ROLE_ARN` is unset. The optional Terraform GitHub OIDC role is scoped to ECR + EKS describe, not full infrastructure apply.

## 4. CI/CD

| Workflow | Role |
| --- | --- |
| `ci.yml` | Tests, kustomize dry-run, terraform validate, compose config |
| `images.yml` | Build images tagged with git SHA; push only if `PUSH_IMAGES=true` |
| `deploy.yml` | Manual AWS/K8s; OIDC role |

## 5. Validation status

| Item | Status |
| --- | --- |
| Application tests (Go/Python/frontend) | Implemented; run in CI and locally |
| `docker compose config` | Implemented |
| Kubernetes render / kubeconform | Implemented; no cluster required |
| Terraform fmt/validate | Implemented; **no apply** |
| kind/minikube full stack | Scripts provided; optional |
| AWS `terraform apply` / live EKS | **Not performed** unless an operator runs it |
| Chaos/evaluation (`evaluation/chaos`) | Implemented; run `python -m evaluation.chaos.run` against Compose + API |

## 6. Cost-sensitive demo notes

A small EKS cluster (two `t3.medium` nodes), one NAT gateway, `db.t4g.micro`, and `cache.t4g.micro` is cheaper than multi-AZ production, but the EKS control plane and NAT still bill while running. Tear down unused demo accounts. Exact USD amounts change; do not treat tfvars sizes as quotes.
