# Terraform

AWS foundation for Sentinel. **No `terraform apply` has been run from this repository by default.**

```
infrastructure/terraform/
  versions.tf providers
  variables.tf
  terraform.tfvars.example
  main.tf
  outputs.tf
  modules/networking
  modules/eks
  modules/rds
  modules/redis
  modules/s3
  modules/iam
```

## What is provisioned (when applied)

- VPC, public/private subnets, NAT, internet gateway
- EKS cluster + managed node group
- RDS PostgreSQL 16 (private)
- ElastiCache Redis 7 (private, single node)
- S3 artifacts bucket (account-scoped name, public access blocked)
- IRSA role for `sentinel-api` and `sentinel-ai` limited to that bucket
- Optional GitHub Actions OIDC role (`enable_github_oidc`)

## What is not provisioned

- **Kafka / MSK** — applications already take `KAFKA_BROKERS`. Operators supply a cluster. This module does not invent an MSK integration.
- In-cluster Prometheus/Grafana — local Compose profile `observability` remains the inner-loop stack. Cloud observability can reuse the same OTLP/`/metrics` contracts.

## Usage

```bash
cd infrastructure/terraform
cp terraform.tfvars.example terraform.tfvars
# edit non-secret sizes/region
terraform init
terraform fmt
terraform validate
terraform plan
# terraform apply   # creates real AWS resources; do not run casually
```

RDS master password is generated and stored in **Secrets Manager**. It is not written to Git or Terraform outputs as plaintext.

Configure a remote S3 backend before any shared environment. Local state files are gitignored.

## Sizing

Instance classes in `terraform.tfvars.example` are **demo starting sizes** (`t3.medium` nodes, `db.t4g.micro`, `cache.t4g.micro`) plus a single NAT gateway. They are cost-conscious, not capacity-tested. A small always-on demo still incurs EKS control plane, NAT, RDS, and Redis charges.

## IAM notes

- Workload role: S3 object access on the artifacts bucket only.
- GitHub role (optional): ECR push to `sentinel-*` repositories plus `ecr:GetAuthorizationToken` on `*` (AWS requires `*` for that action) and EKS describe/list. **It is not an account-admin Terraform apply role.** Operators who use `.github/workflows/deploy.yml` must point `AWS_DEPLOY_ROLE_ARN` at a role they created with the IAM needed for plan/apply, or run Terraform locally.
- Public EKS API endpoint is enabled so operators and GitHub can reach the cluster; private access is also on. Tighten for production.
- Mapping GitHub or human IAM principals into EKS (`aws-auth` / access entries) is not automated; kubectl from CI needs that extra step.
