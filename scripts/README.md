# scripts

- `telemetry_simulator.py` — posts sample metrics, logs, traces, and deployments through `services/ingestion`. `--abnormal` raises payments-api latency/error-rate/DB utilization for Detection (and the local observability dashboards).
- `run_evaluation.py` — live chaos/evaluation runner (`python -m evaluation.chaos.run`). See [evaluation/README.md](../evaluation/README.md).
- `k8s-validate.sh` — kustomize render + kubeconform/kubectl dry-run.
- `terraform-validate.sh` — `terraform fmt -check`, `init -backend=false`, `validate` (does not apply).
- `kind-up.sh` / `kind-down.sh` — optional local cluster apply/teardown.
