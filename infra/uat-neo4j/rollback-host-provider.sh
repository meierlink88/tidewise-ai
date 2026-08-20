#!/usr/bin/env bash

set -euo pipefail

[ "$(id -u)" -eq 0 ] || {
  echo 'UAT Neo4j rollback must run as root' >&2
  exit 1
}

script_dir="$(cd "$(dirname "$0")" && pwd)"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
compose=(docker compose --env-file "$runtime_env" -f "$script_dir/compose.yaml")

set -a
# shellcheck disable=SC1090
. "$runtime_env"
set +a

wait_for_host_neo4j_ready() {
  local attempt state
  for attempt in $(seq 1 36); do
    state="$(systemctl is-active neo4j 2>/dev/null || true)"
    if [ "$state" = active ] && \
      curl --fail --silent --show-error http://127.0.0.1:7474/ >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  echo "Host-native Neo4j did not become HTTP-ready within 180 seconds; state=$state" >&2
  return 1
}

"${compose[@]}" stop --timeout 30 neo4j || true
"${compose[@]}" rm -f neo4j || true
RUNTIME_ENV="$runtime_env" bash "$script_dir/migrate-openspg-project-databases.sh" restore
systemctl enable neo4j >/dev/null
systemctl start neo4j
wait_for_host_neo4j_ready
echo 'PASS restored the previous host-native UAT Neo4j provider; candidate volumes were retained'
