#!/usr/bin/env bash

set -Eeuo pipefail

deployment_root="${DEPLOY_ROOT:-/opt/tidewise/uat}"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=legacy-runtime-manifest.sh
source "${script_directory}/legacy-runtime-manifest.sh"
expected_runner="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
confirmation="${RETIREMENT_CONFIRMATION:?RETIREMENT_CONFIRMATION is required}"
rds_audit_binary="${RDS_AUDIT_BINARY:?RDS_AUDIT_BINARY is required}"
root_retirement_binary="${ROOT_RETIREMENT_BINARY:?ROOT_RETIREMENT_BINARY is required}"
admin_service_token="${ADMIN_SERVICE_TOKEN:?ADMIN_SERVICE_TOKEN is required}"
expected_confirmation='retire-agentos-reason-openspg-qdrant-391'

fail() {
  echo "FAIL $1: $2" >&2
  exit 1
}

pass() {
  echo "PASS $1"
}

[ "$confirmation" = "$expected_confirmation" ] \
  || fail confirmation "expected the exact approved retirement phrase"
[ "${RUNNER_NAME:-}" = "$expected_runner" ] \
  || fail runner-identity "expected ${expected_runner}, got ${RUNNER_NAME:-unset}"
[ -x "$rds_audit_binary" ] || fail rds-audit "audit binary is not executable"
[ -x "$root_retirement_binary" ] || fail root-retirement "root retirement binary is not executable"

for command in docker flock python3 ss; do
  command -v "$command" >/dev/null || fail tooling "${command} is required"
done

exec 8>"${deployment_root}/deploy.lock"
flock -n 8 || fail deployment-lock "another UAT operation is running"

retained_containers=("${UAT_RETAINED_CONTAINERS[@]}")
retired_containers=("${UAT_RETIRED_CONTAINERS[@]}")
retired_volumes=("${UAT_RETIRED_VOLUMES[@]}")
retired_ports=("${UAT_RETIRED_PORTS[@]}")

container_exists() {
  docker inspect "$1" >/dev/null 2>&1
}

container_fingerprint() {
  docker inspect --format '{{.Id}}|{{.State.StartedAt}}|{{.State.Status}}|{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$1"
}

for container in "${retained_containers[@]}"; do
  container_exists "$container" || fail retained-container "${container} is absent"
  state="$(docker inspect --format '{{.State.Status}}' "$container")"
  health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container")"
  [ "$state" = running ] || fail retained-container "${container} state is ${state}"
  [ "$health" = healthy ] || fail retained-container "${container} health is ${health}"
done
pass retained-containers-healthy

dependency_pattern='AGENTOS|AGENTRUN|QDRANT|MYSQL|NEO4J|REASON|OPENSPG'
for container in tidewise-uat-data-1 tidewise-uat-miniapp-1 tidewise-uat-adminportal-1 tidewise-uat-admin-1; do
  if docker inspect --format '{{json .Config.Env}}' "$container" \
    | python3 -c 'import json,sys; print("\n".join(item.split("=",1)[0] for item in (json.load(sys.stdin) or [])))' \
    | grep -Eiq "$dependency_pattern"; then
    fail retained-dependency "${container} still declares a retired runtime key"
  fi
done
pass retained-runtime-has-no-legacy-keys

assert_mounts() {
  local container="$1"
  shift
  local mount
  while IFS= read -r mount; do
    [ -z "$mount" ] && continue
    allowed=false
    for expected in "$@"; do
      [ "$mount" = "$expected" ] && allowed=true
    done
    [ "$allowed" = true ] || fail candidate-mount "${container} has unexpected mount ${mount}"
  done < <(docker inspect --format '{{range .Mounts}}{{println .Source "|" .Destination}}{{end}}' "$container")
}

if container_exists tidewise-agentos-uat-agentos-1; then
  assert_mounts tidewise-agentos-uat-agentos-1 \
    '/opt/tidewise/agentos-uat/data | /app/data' \
    '/opt/tidewise/agentos-uat/jwt-jwks.json | /run/secrets/agentos-jwks.json'
fi
for container in tidewise-uat-agentrun-1 agentrun-service agentrun-migrate agentrun-agent-version; do
  if container_exists "$container"; then
    assert_mounts "$container"
  fi
done
if container_exists reason-server-uat; then
  assert_mounts reason-server-uat
fi
if container_exists tidewise-uat-qdrant; then
  assert_mounts tidewise-uat-qdrant \
    '/var/lib/docker/volumes/tidewise-uat-qdrant-data/_data | /qdrant/storage'
fi
if container_exists tidewise-infra-uat-mysql-1; then
  assert_mounts tidewise-infra-uat-mysql-1 \
    '/var/lib/docker/volumes/tidewise-infra-uat-mysql-data/_data | /var/lib/mysql'
fi
if container_exists tidewise-uat-openspg-neo4j; then
  assert_mounts tidewise-uat-openspg-neo4j \
    '/var/lib/docker/volumes/tidewise-uat-openspg-neo4j-data/_data | /data' \
    '/var/lib/docker/volumes/tidewise-uat-openspg-neo4j-logs/_data | /logs'
fi
pass candidate-mounts-exact

migration_report="${RUNNER_TEMP:-/tmp}/uat-retirement-migration-${GITHUB_RUN_ID:-manual}.json"
rds_report="${RUNNER_TEMP:-/tmp}/uat-retirement-rds-${GITHUB_RUN_ID:-manual}.json"
cleanup_reports() {
  rm -f -- "$migration_report" "$rds_report"
}
trap cleanup_reports EXIT

docker exec tidewise-uat-data-1 /usr/local/bin/dbmigrate >"$migration_report"
python3 "${script_directory}/verify-retirement-migration.py" "$migration_report"
"$rds_audit_binary" >"$rds_report"
python3 - "$rds_report" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    report = json.load(handle)
expected = {
    "current_database": "tidewise_uat",
    "current_role": "tidewise_uat",
    "retired_database": "tidewise_ai_server",
    "retired_role": "agentrun_uat",
}
for key, value in expected.items():
    if report.get(key) != value:
        raise SystemExit(f"FAIL retired-rds-contract: unexpected {key}")
if report.get("retired_database_present"):
    raise SystemExit("FAIL retired-rds-contract: tidewise_ai_server still exists")
print(f"RDS retired database absent; residual role present={str(bool(report.get('retired_role_present'))).lower()}")
PY
pass database-preflight

is_retired_container() {
  local candidate="$1"
  local retired
  for retired in "${retired_containers[@]}"; do
    [ "$candidate" = "$retired" ] && return 0
  done
  return 1
}

for volume in "${retired_volumes[@]}"; do
  while IFS= read -r reference; do
    [ -z "$reference" ] && continue
    is_retired_container "$reference" \
      || fail volume-reference "${volume} is referenced by unapproved container ${reference}"
  done < <(docker ps -a --filter "volume=${volume}" --format '{{.Names}}')
done
pass candidate-volume-references-exact

root_retirement_image="$(docker inspect --format '{{.Image}}' tidewise-uat-data-1)"
[[ "$root_retirement_image" =~ ^sha256:[0-9a-f]{64}$ ]] \
  || fail root-retirement "retained Data image must resolve to a local content ID"

run_root_retirement() {
  local action="$1"
  docker run --rm \
    --user 0:0 \
    --privileged \
    --pid host \
    --network none \
    --read-only \
    --mount type=bind,source=/,target=/host,readonly \
    --mount type=bind,source=/etc/systemd/system,target=/host/etc/systemd/system \
    --mount type=bind,source=/opt/tidewise,target=/host/opt/tidewise \
    --mount "type=bind,source=${root_retirement_binary},target=/uat-root-retirement,readonly" \
    --entrypoint /uat-root-retirement \
    "$root_retirement_image" \
    "$action"
}

run_root_retirement preflight
pass destructive-targets-preflight

declare -A retained_before
for container in "${retained_containers[@]}"; do
  retained_before["$container"]="$(container_fingerprint "$container")"
done

for container in "${retired_containers[@]}"; do
  if container_exists "$container"; then
    docker rm --force "$container" >/dev/null
    echo "REMOVED container ${container}"
  else
    echo "ABSENT container ${container}"
  fi
done

run_root_retirement apply

for volume in "${retired_volumes[@]}"; do
  references="$(docker ps -a --filter "volume=${volume}" --format '{{.Names}}' | paste -sd, -)"
  [ -z "$references" ] || fail volume-reference "${volume} is still referenced by ${references}"
  if docker volume inspect "$volume" >/dev/null 2>&1; then
    docker volume rm "$volume" >/dev/null
    echo "REMOVED volume ${volume}"
  else
    echo "ABSENT volume ${volume}"
  fi
done

for container in "${retired_containers[@]}"; do
  container_exists "$container" && fail retired-container "${container} still exists"
done
for volume in "${retired_volumes[@]}"; do
  docker volume inspect "$volume" >/dev/null 2>&1 \
    && fail retired-volume "${volume} still exists"
done
listeners="$(ss -lntH | awk '{print $4}' | sed 's/.*://' | LC_ALL=C sort -nu)"
for port in "${retired_ports[@]}"; do
  grep -qx "$port" <<<"$listeners" && fail retired-listener "tcp/${port} is still listening"
done
pass retired-runtime-absent

for container in "${retained_containers[@]}"; do
  after="$(container_fingerprint "$container")"
  [ "$after" = "${retained_before[$container]}" ] \
    || fail retained-fingerprint "${container} changed during retirement"
done
ADMIN_SERVICE_TOKEN="$admin_service_token" \
  EXPECTED_INDUSTRY_CHAIN_COUNT=54 \
  python3 "${script_directory}/verify-retained-runtime.py"
pass retained-runtime-unchanged-and-healthy
pass uat-legacy-runtime-retired
