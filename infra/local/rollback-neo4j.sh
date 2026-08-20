#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
rollback_file="$script_dir/docker-compose.neo4j-5.25-rollback.yaml"
env_file="$script_dir/.env.local"

docker compose \
  --env-file "$env_file" \
  -f "$compose_file" \
  -f "$rollback_file" \
  up -d --no-deps --force-recreate --wait neo4j

bash "$script_dir/migrate-openspg-project-graph-store.sh" restore
echo 'PASS restored Neo4j 5.25.1 and the backed-up OpenSPG project graph-store configs'
