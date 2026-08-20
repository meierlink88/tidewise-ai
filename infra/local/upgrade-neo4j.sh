#!/usr/bin/env bash

set -Eeuo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
env_file="$script_dir/.env.local"
target_started=false
unowned_containers=(
  tidewise-infra-postgres-1
  tidewise-infra-mysql-1
  tidewise-infra-minio-1
  tidewise-infra-qdrant-1
)

compose=(docker compose --env-file "$env_file" -f "$compose_file")

fingerprint_unowned() {
  local container
  for container in "${unowned_containers[@]}"; do
    if docker inspect "$container" >/dev/null 2>&1; then
      docker inspect "$container" --format '{{.Name}}|{{.Id}}|{{.State.StartedAt}}'
    fi
  done
}

rollback_on_error() {
  local code="$1"
  trap - ERR
  if [ "$target_started" = true ]; then
    echo 'Target Neo4j failed acceptance; restoring the retained 5.25.1 provider' >&2
    if bash "$script_dir/rollback-neo4j.sh"; then
      echo 'PASS restored local Neo4j 5.25.1 from the retained legacy volume' >&2
    else
      echo 'FAIL automatic Neo4j rollback; run npm run infra:rollback:neo4j' >&2
    fi
  fi
  exit "$code"
}
trap 'rollback_on_error $?' ERR

docker volume inspect tidewise-reason_neo4j-data >/dev/null
unowned_before="$(fingerprint_unowned)"

bash "$script_dir/ensure-volumes.sh"
bash "$script_dir/prepare-neo4j-plugins.sh"
"${compose[@]}" config >/dev/null

target_started=true
"${compose[@]}" up -d --no-deps --force-recreate --wait neo4j
bash "$script_dir/verify-neo4j.sh"
bash "$script_dir/migrate-openspg-project-graph-store.sh" apply
bash "$script_dir/verify-openspg-neo4j-consumer.sh"

unowned_after="$(fingerprint_unowned)"
[ "$unowned_before" = "$unowned_after" ] || {
  echo 'One or more unrelated local infrastructure containers changed during the Neo4j upgrade' >&2
  exit 1
}
docker volume inspect tidewise-reason_neo4j-data >/dev/null

target_started=false
trap - ERR
echo 'PASS upgraded only local Neo4j; the 5.25.1 data volume remains available for rollback'
