#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
run_suffix="${GITHUB_RUN_ID:-$$}"
project_name="tidewise-agentrun-smoke-${run_suffix}"
success_body="/tmp/${project_name}-success.json"
failure_body="/tmp/${project_name}-unavailable.json"

export COMPOSE_NETWORK_NAME="${project_name}-network"
export POSTGRES_PORT="${TIDEWISE_SMOKE_POSTGRES_PORT:-55433}"
export DATA_SERVICE_PORT="${TIDEWISE_SMOKE_AGENTRUN_DATA_PORT:-19015}"
export ADMIN_SERVICE_PORT="${TIDEWISE_SMOKE_ADMIN_PORT:-19013}"
export AGENTRUN_SERVICE_PORT="${TIDEWISE_SMOKE_AGENTRUN_PORT:-19080}"
export DATA_SERVICE_IMAGE="tidewise-data:ci"
export ADMIN_SERVICE_IMAGE="tidewise-adminportal:ci"
export AGENTRUN_SERVICE_IMAGE="tidewise-agentrun:ci"
export POSTGRES_USER="tidewise"
export TIDEWISW_DB_PASSWORD="compose-smoke-postgres-password"
export POSTGRES_DB="tidewise_local"
export AGENTRUN_DATABASE_USER="agentrun"
export AGENTRUN_DB_PASSWORD="compose-smoke-agentrun-database-password"
export AGENTRUN_DATABASE_NAME="tidewise_ai_server"
export AGENTRUN_SERVICE_TOKEN="compose-smoke-agentrun-service-token"
export ADMIN_SERVICE_TOKEN="compose-smoke-admin-browser-token"
export DATA_SERVICE_TOKEN="compose-smoke-data-service-token"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="compose-smoke-neo4j-password"

compose=(
  docker compose
  --project-name "$project_name"
  --env-file "$repo_root/infra/local/.env.example"
  -f "$repo_root/infra/local/docker-compose.yaml"
)

cleanup() {
  set +e
  rm -f -- "$success_body" "$failure_body"
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1
}
trap cleanup EXIT

"${compose[@]}" up -d --wait postgres
"${compose[@]}" run --rm --no-deps \
  -e "PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified -c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified -c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
  data /usr/local/bin/dbmigrate -apply >/dev/null
"${compose[@]}" up -d --wait --no-build agentrun
"${compose[@]}" up -d --wait --no-build --no-deps adminportal

success_status="$(
  curl --silent --show-error \
    --header "Authorization: Bearer ${ADMIN_SERVICE_TOKEN}" \
    --output "$success_body" \
    --write-out "%{http_code}" \
    "http://127.0.0.1:${ADMIN_SERVICE_PORT}/api/admin/v1/model-providers"
)"
if [[ "$success_status" != "200" ]]; then
  echo "Admin Portal returned ${success_status}, want 200 while AgentRun is available" >&2
  sed -n '1,20p' "$success_body" >&2
  echo "Direct AgentRun response:" >&2
  curl --silent --show-error \
    --header "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}" \
    "http://127.0.0.1:${AGENTRUN_SERVICE_PORT}/api/admin/v1/model-providers" >&2 || true
  exit 1
fi
if ! grep -Fq '"result"' "$success_body"; then
  echo "Admin Portal did not return its stable success envelope" >&2
  sed -n '1,20p' "$success_body" >&2
  exit 1
fi
if ! grep -Fq '"provider_key":"deepseek"' "$success_body"; then
  echo "Admin Portal did not decode the registered DeepSeek provider from AgentRun" >&2
  sed -n '1,20p' "$success_body" >&2
  exit 1
fi

"${compose[@]}" stop agentrun >/dev/null
failure_status="$(
  curl --silent --show-error \
    --header "Authorization: Bearer ${ADMIN_SERVICE_TOKEN}" \
    --output "$failure_body" \
    --write-out "%{http_code}" \
    "http://127.0.0.1:${ADMIN_SERVICE_PORT}/api/admin/v1/model-providers"
)"
if [[ "$failure_status" != "503" ]]; then
  echo "Admin Portal returned ${failure_status}, want 503 while AgentRun is unavailable" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi
if ! grep -Fq '"code":"AGENTRUN_UNAVAILABLE"' "$failure_body"; then
  echo "Admin Portal did not return the stable AgentRun-unavailable error code" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi
if grep -Eiq 'postgres|password|compose-smoke|agentrun:9080' "$failure_body"; then
  echo "Admin Portal leaked upstream implementation details" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi

echo "Admin Portal to AgentRun Compose smoke passed"
