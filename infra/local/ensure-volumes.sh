#!/usr/bin/env bash
set -euo pipefail

volumes=(
  local_tidewise_postgres_data
  tidewise-reason_mysql-data
  tidewise-reason_neo4j-5.26-data
  tidewise-reason_neo4j-5.26-logs
  tidewise-reason_neo4j-5.26-plugins
  tidewise-reason_minio-data
  tidewise-qdrant-local-storage
)

for volume in "${volumes[@]}"; do
  if ! docker volume inspect "$volume" >/dev/null 2>&1; then
    docker volume create "$volume" >/dev/null
  fi
done
