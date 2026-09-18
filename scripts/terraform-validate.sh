#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/infrastructure/terraform"
terraform fmt -check -recursive
terraform init -backend=false
terraform validate
echo "Terraform formatted and validated (no apply)."
