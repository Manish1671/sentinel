#!/usr/bin/env bash
# Optional local cluster. Requires kind and kubectl. Does not use AWS.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
kind create cluster --name sentinel || true
kubectl apply -k infrastructure/kubernetes/overlays/local
echo "Applied local overlay to kind cluster 'sentinel'."
echo "Load images with: kind load docker-image sentinel/api:local --name sentinel"
echo "Tear down with: scripts/kind-down.sh"
