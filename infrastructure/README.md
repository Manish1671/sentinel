# infrastructure

| Path | Role |
| --- | --- |
| Docker Compose at repo root | Local inner loop (unchanged) |
| `docker/` | Notes on application images |
| `kubernetes/` | Workload manifests (base + local/aws overlays) |
| `terraform/` | AWS VPC, EKS, RDS, ElastiCache, S3, IAM |
| `aws/` | Mapping notes only |

See [docs/deployment.md](../docs/deployment.md).
