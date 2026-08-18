#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
run_suffix="${GITHUB_RUN_ID:-$$}"
project_name="tidewise-kratos-smoke-${run_suffix}"
prod_container=""
postgres_fixture="${project_name}-postgres-fixture"
failure_body="/tmp/${project_name}-miniapp-data-unavailable.json"

export COMPOSE_NETWORK_NAME="${project_name}-network"
export DATA_SERVICE_PORT="${TIDEWISE_SMOKE_DATA_PORT:-19011}"
export MINIAPP_SERVICE_PORT="${TIDEWISE_SMOKE_MINIAPP_PORT:-19012}"
export DATA_SERVICE_IMAGE="tidewise-data:ci"
export MINIAPP_SERVICE_IMAGE="tidewise-miniapp:ci"
export TIDEWISW_DB_PASSWORD="tidewise-compose-smoke-password"
export TIDEWISE_DB_HOST="$postgres_fixture"
export DATA_SERVICE_TOKEN="compose-smoke-data-service-token"

compose=(
  docker compose
  --project-name "$project_name"
  --env-file "$repo_root/infra/local/.env.example"
  -f "$repo_root/infra/local/docker-compose.yaml"
)

cleanup() {
  set +e
  rm -f -- "$failure_body"
  if [[ -n "$prod_container" ]]; then
    docker rm -f "$prod_container" >/dev/null 2>&1
  fi
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1
  docker rm -f "$postgres_fixture" >/dev/null 2>&1
  docker network rm "$COMPOSE_NETWORK_NAME" >/dev/null 2>&1
}
trap cleanup EXIT

docker network create "$COMPOSE_NETWORK_NAME" >/dev/null
docker run -d --name "$postgres_fixture" --network "$COMPOSE_NETWORK_NAME" \
  -e POSTGRES_USER=tidewise \
  -e "POSTGRES_PASSWORD=${TIDEWISW_DB_PASSWORD}" \
  -e POSTGRES_DB=tidewise_local \
  postgres:16 >/dev/null
for _ in $(seq 1 60); do
  if docker exec "$postgres_fixture" pg_isready -U tidewise -d tidewise_local >/dev/null; then
    break
  fi
  sleep 1
done
docker exec "$postgres_fixture" pg_isready -U tidewise -d tidewise_local >/dev/null
"${compose[@]}" run --rm --no-deps \
  -e "PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified -c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified -c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
  data-migrate >/dev/null
if [[ -n "$("${compose[@]}" ps -a -q data-migrate)" ]]; then
  echo "Data migration container was retained after the ephemeral run" >&2
  exit 1
fi
"${compose[@]}" up -d --wait --no-build --no-deps data
"${compose[@]}" up -d --wait --no-build --no-deps miniapp

curl --fail --silent --show-error \
  "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/api/miniapp/v1/research/themes" \
  >/dev/null
curl --fail --silent --show-error \
  "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/docs/" \
  >/dev/null

"${compose[@]}" stop data >/dev/null
failure_status="$(
  curl --silent --show-error \
    --output "$failure_body" \
    --write-out "%{http_code}" \
    "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/api/miniapp/v1/research/themes/11111111-1111-4111-8111-111111111111/reasoning-trees"
)"
if [[ "$failure_status" != "502" ]]; then
  echo "Miniapp returned ${failure_status}, want 502 while Data Service is unavailable" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi
if ! grep -Fq '"code":"RESEARCH_DATA_UNAVAILABLE"' "$failure_body"; then
  echo "Miniapp did not return the stable Data-unavailable error code" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi
if grep -Eiq 'postgres|password|compose-smoke|data:9011' "$failure_body"; then
  echo "Miniapp leaked upstream implementation details while Data Service was unavailable" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi

prod_container="$(
  "${compose[@]}" run -d --no-deps \
    --name "${project_name}-miniapp-prod" \
    -e APP_ENV=prod \
    miniapp
)"

for _ in $(seq 1 30); do
  if docker exec "$prod_container" wget -qO- http://127.0.0.1:9012/healthz >/dev/null; then
    break
  fi
  sleep 1
done
docker exec "$prod_container" wget -qO- http://127.0.0.1:9012/healthz >/dev/null
if docker exec "$prod_container" wget -qO- http://127.0.0.1:9012/docs/ >/dev/null; then
  echo "prod Miniapp unexpectedly exposes Swagger UI" >&2
  exit 1
fi

miniapp_container="$("${compose[@]}" ps -q miniapp)"
docker kill --signal=SIGTERM "$miniapp_container" >/dev/null
for _ in $(seq 1 30); do
  if [[ "$(docker inspect --format '{{.State.Running}}' "$miniapp_container")" == "false" ]]; then
    break
  fi
  sleep 0.5
done
if [[ "$(docker inspect --format '{{.State.Running}}' "$miniapp_container")" != "false" ]]; then
  echo "Miniapp did not stop within 15 seconds after SIGTERM" >&2
  exit 1
fi
miniapp_exit_code="$(docker inspect --format '{{.State.ExitCode}}' "$miniapp_container")"
if [[ "$miniapp_exit_code" != "0" ]]; then
  echo "Miniapp exited with code ${miniapp_exit_code} after SIGTERM" >&2
  exit 1
fi

echo "Miniapp Kratos Compose smoke passed"
