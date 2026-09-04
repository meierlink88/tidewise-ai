#!/usr/bin/env bash

set -Eeuo pipefail

deployment_root="${DEPLOY_ROOT:-/opt/tidewise/uat}"
script_directory="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=legacy-runtime-manifest.sh
source "${script_directory}/legacy-runtime-manifest.sh"
expected_runner="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
actual_runner="${RUNNER_NAME:-}"
rds_audit_binary="${RDS_AUDIT_BINARY:?RDS_AUDIT_BINARY is required}"
temporary_files=()

cleanup() {
  local path
  for path in "${temporary_files[@]}"; do
    rm -f -- "$path"
  done
}
trap cleanup EXIT

if [ "$actual_runner" != "$expected_runner" ]; then
  echo "FAIL runner-identity: expected ${expected_runner}, got ${actual_runner:-unset}" >&2
  exit 1
fi

for command in curl docker flock getent python3 ss systemctl; do
  command -v "$command" >/dev/null || {
    echo "FAIL audit-tooling: ${command} is required" >&2
    exit 1
  }
done
[ -x "$rds_audit_binary" ] || {
  echo "FAIL audit-tooling: RDS audit binary is not executable" >&2
  exit 1
}

exec 8>"${deployment_root}/deploy.lock"
flock -n 8 || {
  echo "FAIL deployment-lock: another UAT operation is running" >&2
  exit 1
}

retained_required=("${UAT_RETAINED_CONTAINERS[@]}")
retirement_candidates=("${UAT_RETIRED_CONTAINERS[@]}")
candidate_volumes=("${UAT_RETIRED_VOLUMES[@]}")
candidate_ports=("${UAT_RETIRED_PORTS[@]}")
dependency_key_pattern='AGENTOS|AGENTRUN|QDRANT|MYSQL|NEO4J|REASON(_SERVER|_SERVICE)?|OPENSPG|KAG'

container_exists() {
  docker inspect "$1" >/dev/null 2>&1
}

container_health() {
  docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$1"
}

container_state() {
  docker inspect --format '{{.State.Status}}' "$1"
}

environment_keys() {
  docker inspect --format '{{json .Config.Env}}' "$1" | python3 -c '
import json
import sys

keys = {item.split("=", 1)[0] for item in (json.load(sys.stdin) or [])}
for key in sorted(keys):
    print(key)
'
}

print_container() {
  local name="$1"
  if ! container_exists "$name"; then
    echo "ABSENT container ${name}"
    return
  fi

  echo "PRESENT container ${name}"
  docker inspect --format \
    '  image={{.Config.Image}} state={{.State.Status}} health={{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}} restart={{.HostConfig.RestartPolicy.Name}}' \
    "$name"
  docker inspect --format \
    '  networks={{range $name, $_ := .NetworkSettings.Networks}}{{$name}} {{end}}' \
    "$name"
  echo "  mounts:"
  docker inspect --format \
    '{{range .Mounts}}{{println "   " .Type "|" .Name "|" .Source "|" .Destination}}{{end}}' \
    "$name" | sed '/^[[:space:]]*$/d'
  docker inspect --format '  log-path={{.LogPath}}' "$name"
  echo "  environment-keys:"
  environment_keys "$name" | sed 's/^/    /'
  echo "  legacy-dependency-keys:"
  environment_keys "$name" \
    | grep -Ei "$dependency_key_pattern" \
    | sed 's/^/    /' || true
}

echo "## UAT runtime inventory"
docker ps -a --format \
  'container={{.Names}} | image={{.Image}} | state={{.State}} | status={{.Status}} | ports={{.Ports}} | networks={{.Networks}}' \
  | LC_ALL=C sort

echo
echo "## Required retained containers"
retained_failure=false
for container in "${retained_required[@]}"; do
  print_container "$container"
  if ! container_exists "$container" \
    || [ "$(container_state "$container")" != running ] \
    || [ "$(container_health "$container")" != healthy ]; then
    retained_failure=true
  fi
done

echo
echo "## Retirement candidates"
for container in "${retirement_candidates[@]}"; do
  print_container "$container"
done

echo
echo "## Candidate persistent volumes"
for volume in "${candidate_volumes[@]}"; do
  if docker volume inspect "$volume" >/dev/null 2>&1; then
    references="$(docker ps -a --filter "volume=${volume}" --format '{{.Names}}' | LC_ALL=C sort | paste -sd, -)"
    echo "PRESENT volume ${volume} references=${references:-none}"
  else
    echo "ABSENT volume ${volume}"
  fi
done

echo
echo "## Candidate host paths"
for path in "${UAT_RETIRED_PATHS[@]}"; do
  if [ -e "$path" ]; then
    size_bytes="$(du -sb -- "$path" 2>/dev/null | awk '{print $1}' || true)"
    echo "PRESENT path ${path} bytes=${size_bytes:-unreadable}"
  else
    echo "ABSENT path ${path}"
  fi
done

echo
echo "## Candidate AgentRun host state"
if getent group tidewise-agentrun >/dev/null; then
  echo "PRESENT group tidewise-agentrun"
else
  echo "ABSENT group tidewise-agentrun"
fi
host_matches="$(find /opt/tidewise/uat -maxdepth 2 \
  \( -iname '*agentrun*' -o -iname '*agent-run*' \) -printf '%y|%p\n' 2>/dev/null \
  | LC_ALL=C sort || true)"
if [ -z "$host_matches" ]; then
  echo "ABSENT matching AgentRun host paths"
else
  echo "PRESENT matching AgentRun host paths:"
  sed 's/^/  /' <<<"$host_matches"
fi

echo
echo "## Legacy keys in bounded UAT runtime state"
runtime_state_files="$(find "$deployment_root" -maxdepth 2 -type f \
  \( -name 'runtime.env' -o -name '*.runtime.env' \) -print 2>/dev/null \
  | LC_ALL=C sort || true)"
legacy_state_found=false
while IFS= read -r state_file; do
  [ -n "$state_file" ] || continue
  keys="$(awk '
    /^[[:space:]]*(export[[:space:]]+)?[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=/ {
      line=$0
      sub(/^[[:space:]]*(export[[:space:]]+)?/, "", line)
      sub(/[[:space:]]*=.*/, "", line)
      print line
    }
  ' "$state_file" | LC_ALL=C sort -u | grep -Ei "$dependency_key_pattern" || true)"
  if [ -n "$keys" ]; then
    legacy_state_found=true
    echo "PRESENT legacy runtime keys file=${state_file}"
    sed 's/^/  /' <<<"$keys"
  fi
done <<<"$runtime_state_files"
if [ "$legacy_state_found" = false ]; then
  echo "ABSENT legacy keys in bounded runtime state"
fi

echo
echo "## Candidate listeners"
listeners="$(ss -lntH | awk '{print $4}' | sed 's/.*://' | LC_ALL=C sort -nu)"
for port in "${candidate_ports[@]}"; do
  if grep -qx "$port" <<<"$listeners"; then
    echo "PRESENT listener tcp/${port}"
  else
    echo "ABSENT listener tcp/${port}"
  fi
done

echo
echo "## Candidate systemd units"
units="$(systemctl list-unit-files --type=service --no-legend --no-pager 2>/dev/null \
  | awk '{print $1}' \
  | grep -Ei 'agentos|agentrun|agent-run|qdrant|mysql|neo4j|reason|openspg|kag' \
  | LC_ALL=C sort -u || true)"
if [ -z "$units" ]; then
  echo "ABSENT matching systemd units"
else
  while IFS= read -r unit; do
    [ -n "$unit" ] || continue
    active="$(systemctl is-active "$unit" 2>/dev/null || true)"
    enabled="$(systemctl is-enabled "$unit" 2>/dev/null || true)"
    echo "PRESENT unit ${unit} active=${active:-unknown} enabled=${enabled:-unknown}"
  done <<<"$units"
fi

retired_unit_failure=false
for unit in "${UAT_RETIRED_PROJECT_UNITS[@]}"; do
  load_state="$(systemctl show --property=LoadState --value "$unit" 2>/dev/null || true)"
  if [ "$load_state" = loaded ]; then
    echo "FAIL retired-systemd-unit: ${unit} is still loaded" >&2
    retired_unit_failure=true
  else
    echo "ABSENT retired project unit ${unit}"
  fi
done

echo
echo "## Deployment state"
if [ -s "${deployment_root}/state/current.sha" ]; then
  echo "current-release=$(sed -n '1p' "${deployment_root}/state/current.sha")"
else
  echo "current-release=missing"
fi
markers="$(find "${deployment_root}/state" -maxdepth 1 -type f \
  \( -name '*in-progress*' -o -name '*recovery*' -o -name '*rollback-required*' \) \
  -printf '%f\n' 2>/dev/null | LC_ALL=C sort || true)"
if [ -z "$markers" ]; then
  echo "cutover-markers=none"
else
  echo "cutover-markers:"
  sed 's/^/  /' <<<"$markers"
fi

if container_exists tidewise-uat-data-1; then
  migration_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-runtime-audit-migration-${GITHUB_RUN_ID:-manual}.json"
  temporary_files+=("$migration_report")
  docker exec tidewise-uat-data-1 /usr/local/bin/dbmigrate >"$migration_report"
  python3 - "$migration_report" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    report = json.load(handle)
print(f"data-migration-current={report.get('current_version')}")
pending = report.get("pending") or report.get("pending_migrations") or []
print(f"data-migration-pending={len(pending)}")
PY
fi

echo
echo "## Retired RDS objects"
rds_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-runtime-audit-rds-${GITHUB_RUN_ID:-manual}.json"
temporary_files+=("$rds_report")
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
print(f"current-database={report['current_database']}")
print(f"current-role={report['current_role']}")
print(f"retired-database={report['retired_database']} present={str(bool(report.get('retired_database_present'))).lower()}")
print(f"retired-role={report['retired_role']} present={str(bool(report.get('retired_role_present'))).lower()}")
PY

echo
echo "## Retained public entry checks"
curl --fail --silent --show-error --max-time 5 http://127.0.0.1:9012/healthz >/dev/null
echo "PASS miniapp-health"
curl --fail --silent --show-error --max-time 5 http://127.0.0.1:9014/healthz >/dev/null
echo "PASS admin-health"
curl --fail --silent --show-error --max-time 5 http://127.0.0.1:9000/minio/health/live >/dev/null
echo "PASS minio-health"
home_status="$(curl --silent --show-error --max-time 10 -o /dev/null -w '%{http_code}' \
  http://127.0.0.1:9012/api/miniapp/v1/reports/home)"
[ "$home_status" = 200 ] || {
  echo "FAIL miniapp-report-read: HTTP ${home_status}" >&2
  exit 1
}
echo "PASS miniapp-report-read"

if [ "$retained_failure" = true ]; then
  echo "FAIL retained-runtime: one or more required containers are absent or unhealthy" >&2
  exit 1
fi
if [ "$retired_unit_failure" = true ]; then
  echo "FAIL retired-runtime: one or more retired project units remain loaded" >&2
  exit 1
fi
echo "PASS retained-runtime"
