# Kubernetes

Manifests live under `infrastructure/kubernetes/`.

| Path | Purpose |
| --- | --- |
| `base/` | Namespace, service accounts, RBAC, ConfigMap, Deployments, Services, Ingress, migrate Job |
| `overlays/local/` | kind/minikube: in-cluster Postgres, Redis, Kafka + local secret generator |
| `overlays/aws/` | EKS overlay: HPA, PDB, ALB ingress class, IRSA annotation placeholders |

Stateful **production** data stores are **not** in the AWS overlay. RDS, ElastiCache, and S3 are Terraform. Kafka remains an external `KAFKA_BROKERS` setting.

## Images

Build from the repository root:

```bash
docker build -f apps/api/Dockerfile -t sentinel/api:local .
docker build -f services/ingestion/Dockerfile -t sentinel/ingestion:local .
docker build -f services/detection/Dockerfile -t sentinel/detection:local .
docker build -f services/incident/Dockerfile -t sentinel/incident:local .
docker build -f services/ai/Dockerfile -t sentinel/ai:local .
docker build -f services/remediation/Dockerfile -t sentinel/remediation:local .
docker build -f apps/web/Dockerfile -t sentinel/web:local .
```

## Validate without a cluster

```bash
kubectl kustomize infrastructure/kubernetes/overlays/local > /tmp/sentinel-local.yaml
kubectl kustomize infrastructure/kubernetes/overlays/aws > /tmp/sentinel-aws.yaml
kubeconform -strict -summary /tmp/sentinel-local.yaml /tmp/sentinel-aws.yaml
```

Or `scripts/k8s-validate.sh` (kustomize render, then kubeconform if installed, else kubectl dry-run).

## Local kind (optional)

```bash
kind create cluster --name sentinel
kind load docker-image sentinel/api:local --name sentinel
# repeat load for other images
kubectl apply -k infrastructure/kubernetes/overlays/local
kubectl delete -k infrastructure/kubernetes/overlays/local
kind delete cluster --name sentinel
```

`overlays/local/secrets.env` uses the same **local** Compose credentials. It is not a production secret.

## AWS overlay

1. Apply Terraform (RDS/Redis/S3/EKS).
2. Copy `overlays/aws/secret.example.yaml` to `secret.yaml` (gitignored) and fill values from Terraform outputs / Secrets Manager.
3. `kubectl apply -f secret.yaml`
4. Patch `KAFKA_BROKERS` and `S3_BUCKET` in the ConfigMap overlay.
5. Replace `ACCOUNT_ID` in the IRSA annotation with the workload role ARN from Terraform.
6. `kubectl apply -k infrastructure/kubernetes/overlays/aws`

Do not put long-lived AWS access keys in Kubernetes Secrets.

## Probes

Existing application endpoints only:

| Workload | Liveness | Readiness |
| --- | --- | --- |
| api, ingestion, detection, incident, ai, remediation | `GET /health` | `GET /ready` |
| web | `GET /login` | `GET /api/ready` |

## Resources

CPU/memory requests and limits in the manifests are **starting values**, not load-tested capacity.

Kafka consumers (detection, incident, AI, remediation) stay at one replica by default to avoid unplanned consumer-group scaling.
