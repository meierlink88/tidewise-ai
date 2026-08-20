#!/usr/bin/env bash

set -Eeuo pipefail

[ "$(id -u)" -eq 0 ] || {
  echo 'UAT OpenSPG Neo4j adoption must run as root' >&2
  exit 1
}

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/compose.yaml"
deploy_root=/opt/tidewise/neo4j-uat
runtime_env="$deploy_root/runtime.env"
neo4j_user="${OPENSPG_NEO4J_USER:-neo4j}"
neo4j_password="${OPENSPG_NEO4J_PASSWORD:?OPENSPG_NEO4J_PASSWORD is required}"
mysql_password="${OPENSPG_MYSQL_ROOT_PASSWORD:?OPENSPG_MYSQL_ROOT_PASSWORD is required}"
expected_image='spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9'
adoption_started=false
unowned_containers=(
  tidewise-infra-uat-mysql-1
  tidewise-infra-uat-minio-1
  tidewise-uat-data-1
  tidewise-uat-miniapp-1
  tidewise-uat-adminportal-1
  tidewise-uat-admin-1
  tidewise-uat-qdrant
  tidewise-agentos-uat-agentos-1
)

[[ "$neo4j_user" =~ ^[A-Za-z0-9_-]{1,64}$ ]] || {
  echo 'OPENSPG_NEO4J_USER must be 1-64 URL-safe characters' >&2
  exit 1
}
[[ "$neo4j_password" =~ ^[A-Za-z0-9_-]{24,64}$ ]] || {
  echo 'OPENSPG_NEO4J_PASSWORD must be 24-64 URL-safe characters' >&2
  exit 1
}
[[ "$mysql_password" =~ ^[A-Za-z0-9_-]{24,64}$ ]] || {
  echo 'OPENSPG_MYSQL_ROOT_PASSWORD must be 24-64 URL-safe characters' >&2
  exit 1
}

for command in curl cypher-shell docker flock install systemctl; do
  command -v "$command" >/dev/null || {
    echo "Missing dependency: $command" >&2
    exit 1
  }
done
[ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ]
docker info >/dev/null
docker compose version >/dev/null
docker network inspect tidewise-uat >/dev/null
[ "$(systemctl is-active neo4j)" = active ]
curl --fail --silent --show-error http://127.0.0.1:7474/ >/dev/null

existing_graph_counts="$(cypher-shell \
  -a bolt://127.0.0.1:7687 \
  -d neo4j \
  -u "$neo4j_user" \
  -p "$neo4j_password" \
  --format plain \
  'MATCH (node) WITH count(node) AS nodes OPTIONAL MATCH ()-[relationship]->() RETURN nodes, count(relationship) AS relationships;' |
  tail -n 1 | tr -d '"\r ')"
IFS=, read -r existing_nodes existing_relationships <<<"$existing_graph_counts"
[[ "$existing_nodes" =~ ^[0-9]+$ && "$existing_relationships" =~ ^[0-9]+$ ]] || {
  echo "Could not inventory the existing host-native Neo4j graph: $existing_graph_counts" >&2
  exit 1
}
if [ "$existing_nodes" -ne 0 ] || [ "$existing_relationships" -ne 0 ]; then
  echo "Existing host-native Neo4j contains $existing_nodes nodes and $existing_relationships relationships." >&2
  echo 'Adoption stopped before mutation because a backward-compatible 5.26 -> 5.25 graph migration is not defined.' >&2
  exit 1
fi

unowned_fingerprint() {
  local container
  for container in "${unowned_containers[@]}"; do
    docker inspect --format '{{.Name}}|{{.Id}}|{{.State.StartedAt}}' "$container"
  done
}

rollback_on_error() {
  local code="$1"
  trap - ERR
  if [ "$adoption_started" = true ]; then
    set +e
    RUNTIME_ENV="$runtime_env" \
      OPENSPG_MYSQL_ROOT_PASSWORD="$mysql_password" \
      bash "$script_dir/rollback-host-provider.sh"
    rollback_code="$?"
    set -e
    [ "$rollback_code" -eq 0 ] || echo 'FAIL recovery: manual UAT Neo4j recovery required' >&2
  fi
  exit "$code"
}
trap 'rollback_on_error $?' ERR

install -d -m 0750 -o root -g root "$deploy_root"
exec 8>/opt/tidewise/uat/deploy.lock
flock -n 8 || {
  echo 'Another UAT deployment holds /opt/tidewise/uat/deploy.lock' >&2
  exit 1
}
exec 9>"$deploy_root/deploy.lock"
flock -n 9 || {
  echo "Another UAT Neo4j operation holds $deploy_root/deploy.lock" >&2
  exit 1
}

umask 077
runtime_candidate="$(mktemp "$deploy_root/runtime.env.XXXXXX")"
printf '%s\n' \
  "OPENSPG_NEO4J_USER=$neo4j_user" \
  "OPENSPG_NEO4J_PASSWORD=$neo4j_password" \
  "OPENSPG_MYSQL_ROOT_PASSWORD=$mysql_password" >"$runtime_candidate"
install -m 0600 "$runtime_candidate" "$runtime_env"
rm -f "$runtime_candidate"

compose=(docker compose --env-file "$runtime_env" -f "$compose_file")
"${compose[@]}" config --quiet
docker pull "$expected_image"
for volume in tidewise-uat-openspg-neo4j-data tidewise-uat-openspg-neo4j-logs; do
  docker volume inspect "$volume" >/dev/null 2>&1 || docker volume create "$volume" >/dev/null
done

unowned_before="$(unowned_fingerprint)"
adoption_started=true
systemctl disable --now neo4j >/dev/null
[ "$(systemctl is-active neo4j)" = inactive ]
"${compose[@]}" up -d --no-deps --force-recreate --wait --wait-timeout 180 neo4j
RUNTIME_ENV="$runtime_env" \
  OPENSPG_MYSQL_ROOT_PASSWORD="$mysql_password" \
  bash "$script_dir/migrate-openspg-project-databases.sh" apply
RUNTIME_ENV="$runtime_env" bash "$script_dir/verify.sh"

"${compose[@]}" restart neo4j
"${compose[@]}" up -d --no-deps --wait --wait-timeout 180 neo4j
RUNTIME_ENV="$runtime_env" bash "$script_dir/verify.sh"

unowned_after="$(unowned_fingerprint)"
[ "$unowned_before" = "$unowned_after" ] || {
  echo 'One or more unrelated UAT containers restarted during Neo4j adoption' >&2
  exit 1
}

printf '%s\n' "$expected_image" >"$deploy_root/current-image"
chmod 0640 "$deploy_root/current-image"
sync "$runtime_env" "$deploy_root/current-image"
adoption_started=false
trap - ERR
echo 'PASS adopted the OpenSPG-specialized UAT Neo4j provider; host-native Neo4j remains disabled for rollback'
