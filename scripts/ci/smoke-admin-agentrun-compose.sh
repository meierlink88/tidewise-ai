#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
run_suffix="${GITHUB_RUN_ID:-$$}"
project_name="tidewise-agentrun-smoke-${run_suffix}"
success_body="/tmp/${project_name}-success.json"
failure_body="/tmp/${project_name}-unavailable.json"
data_postgres_fixture="${project_name}-data-postgres-fixture"
agentrun_postgres_fixture="${project_name}-agentrun-postgres-fixture"
qdrant_fixture="${project_name}-qdrant-fixture"

export COMPOSE_NETWORK_NAME="${project_name}-network"
export DATA_SERVICE_PORT="${TIDEWISE_SMOKE_AGENTRUN_DATA_PORT:-19015}"
export ADMIN_SERVICE_PORT="${TIDEWISE_SMOKE_ADMIN_PORT:-19013}"
export ADMIN_WEB_PORT="${TIDEWISE_SMOKE_ADMIN_WEB_PORT:-19014}"
export AGENTRUN_SERVICE_PORT="${TIDEWISE_SMOKE_AGENTRUN_PORT:-19080}"
qdrant_fixture_port="${TIDEWISE_SMOKE_QDRANT_PORT:-56333}"
export DATA_SERVICE_IMAGE="tidewise-data:ci"
export ADMIN_SERVICE_IMAGE="tidewise-adminportal:ci"
export ADMIN_WEB_IMAGE="tidewise-admin:ci"
export AGENTRUN_SERVICE_IMAGE="tidewise-agentrun:ci"
export TIDEWISW_DB_PASSWORD="compose-smoke-postgres-password"
export TIDEWISE_DB_HOST="$data_postgres_fixture"
export AGENTRUN_DB_PASSWORD="compose-smoke-agentrun-database-password"
export AGENTRUN_DB_HOST="$agentrun_postgres_fixture"
export AGENTRUN_QDRANT_URL="http://${qdrant_fixture}:6333"
export AGENTRUN_SERVICE_TOKEN="compose-smoke-agentrun-service-token"
export ADMIN_SERVICE_TOKEN="compose-smoke-admin-browser-token"
export DATA_SERVICE_TOKEN="compose-smoke-data-service-token"

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
  docker rm -f \
    "$data_postgres_fixture" "$agentrun_postgres_fixture" \
    "$qdrant_fixture" >/dev/null 2>&1
  docker network rm "$COMPOSE_NETWORK_NAME" >/dev/null 2>&1
}
trap cleanup EXIT

"${compose[@]}" create --no-build data-migrate >/dev/null
docker run -d --name "$data_postgres_fixture" --network "$COMPOSE_NETWORK_NAME" \
  -e POSTGRES_USER=tidewise \
  -e "POSTGRES_PASSWORD=${TIDEWISW_DB_PASSWORD}" \
  -e POSTGRES_DB=tidewise_local \
  postgres:16 >/dev/null
docker run -d --name "$agentrun_postgres_fixture" --network "$COMPOSE_NETWORK_NAME" \
  -e POSTGRES_USER=agentrun \
  -e "POSTGRES_PASSWORD=${AGENTRUN_DB_PASSWORD}" \
  -e POSTGRES_DB=tidewise_ai_server \
  postgres:16 >/dev/null
docker run -d --name "$qdrant_fixture" --network "$COMPOSE_NETWORK_NAME" \
  -p "127.0.0.1:${qdrant_fixture_port}:6333" \
  qdrant/qdrant:v1.15.5 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$data_postgres_fixture" pg_isready -U tidewise -d tidewise_local >/dev/null &&
    docker exec "$agentrun_postgres_fixture" pg_isready -U agentrun -d tidewise_ai_server >/dev/null &&
    curl --fail --silent "http://127.0.0.1:${qdrant_fixture_port}/readyz" >/dev/null; then
    break
  fi
  sleep 1
done
docker exec "$data_postgres_fixture" pg_isready -U tidewise -d tidewise_local >/dev/null
docker exec "$agentrun_postgres_fixture" pg_isready -U agentrun -d tidewise_ai_server >/dev/null
curl --fail --silent "http://127.0.0.1:${qdrant_fixture_port}/readyz" >/dev/null
"${compose[@]}" run --rm --no-deps \
  -e "PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified -c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified -c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
  data-migrate >/dev/null
"${compose[@]}" run --rm --no-deps agentrun-migrate >/dev/null
printf '%s' 'compose-smoke-deepseek-key' | "${compose[@]}" run --rm --no-deps -T \
  --entrypoint /app/agentrun-config agentrun \
  model set --provider deepseek --base-url https://api.deepseek.com \
  --model deepseek-chat --api-key-stdin >/dev/null
"${compose[@]}" up -d --wait --no-build agentrun
"${compose[@]}" up -d --wait --no-build --no-deps adminportal
"${compose[@]}" up -d --wait --no-build --no-deps admin

success_status="$(
  curl --silent --show-error \
    --header "Authorization: Bearer ${ADMIN_SERVICE_TOKEN}" \
    --output "$success_body" \
    --write-out "%{http_code}" \
    "http://127.0.0.1:${ADMIN_WEB_PORT}/api/admin/v1/model-providers"
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
    "http://127.0.0.1:${ADMIN_WEB_PORT}/api/admin/v1/model-providers"
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
