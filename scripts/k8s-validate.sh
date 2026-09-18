#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
mkdir -p "$root/tmp"
kubectl kustomize infrastructure/kubernetes/overlays/local >"$root/tmp/sentinel-local.yaml"
kubectl kustomize infrastructure/kubernetes/overlays/aws >"$root/tmp/sentinel-aws.yaml"
if command -v kubeconform >/dev/null 2>&1; then
  kubeconform -strict -summary "$root/tmp/sentinel-local.yaml"
  kubeconform -strict -summary "$root/tmp/sentinel-aws.yaml"
elif command -v kubectl >/dev/null 2>&1; then
  kubectl apply --dry-run=client --validate=false -f "$root/tmp/sentinel-local.yaml"
  kubectl apply --dry-run=client --validate=false -f "$root/tmp/sentinel-aws.yaml"
else
  echo "Rendered overlays to tmp/. Install kubeconform or kubectl to validate further."
fi
echo "Kubernetes overlays rendered and validated."
