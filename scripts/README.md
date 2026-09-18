# scripts

- `telemetry_simulator.py` — posts sample metrics, logs, traces, and deployments through `services/ingestion`. `--abnormal` raises payments-api latency/error-rate/DB utilization for Detection (and the local observability dashboards).
- `k8s-validate.sh` — kustomize render + `kubectl apply --dry-run=client` (no cluster required for client dry-run).
- `terraform-validate.sh` — `terraform fmt -check`, `init -backend=false`, `validate` (does not apply).
- `kind-up.sh` / `kind-down.sh` — optional local cluster apply/teardown.
