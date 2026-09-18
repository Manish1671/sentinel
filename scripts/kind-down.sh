#!/usr/bin/env bash
set -euo pipefail
kind delete cluster --name sentinel
echo "Deleted kind cluster 'sentinel'."
