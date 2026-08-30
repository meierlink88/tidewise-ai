#!/usr/bin/env bash
set -euo pipefail

volumes=(
  local_tidewise_postgres_data
  tidewise-reason_minio-data
)

for volume in "${volumes[@]}"; do
  if ! docker volume inspect "$volume" >/dev/null 2>&1; then
    docker volume create "$volume" >/dev/null
  fi
done
