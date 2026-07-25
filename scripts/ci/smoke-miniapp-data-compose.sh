#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
run_suffix="${GITHUB_RUN_ID:-$$}"
project_name="tidewise-kratos-smoke-${run_suffix}"
prod_container=""

export COMPOSE_NETWORK_NAME="${project_name}-network"
export POSTGRES_PORT="${TIDEWISE_SMOKE_POSTGRES_PORT:-55432}"
export DATA_SERVICE_PORT="${TIDEWISE_SMOKE_DATA_PORT:-19011}"
export MINIAPP_SERVICE_PORT="${TIDEWISE_SMOKE_MINIAPP_PORT:-19012}"
export DATA_SERVICE_IMAGE="tidewise-data:ci"
export MINIAPP_SERVICE_IMAGE="tidewise-miniapp:ci"
export POSTGRES_PASSWORD="tidewise-compose-smoke-password"
export DATA_SERVICE_AGENT_TOKEN="compose-smoke-agent-token"
export DATA_SERVICE_RESEARCH_PUBLISHER_TOKEN="compose-smoke-research-token"
export DATA_SERVICE_MINIAPP_TOKEN="compose-smoke-miniapp-token"
export DATA_SERVICE_ADMIN_TOKEN="compose-smoke-admin-token"

compose=(
  docker compose
  --project-name "$project_name"
  --env-file "$repo_root/infra/local/.env.example"
  -f "$repo_root/infra/local/docker-compose.yaml"
)

cleanup() {
  set +e
  if [[ -n "$prod_container" ]]; then
    docker rm -f "$prod_container" >/dev/null 2>&1
  fi
  "${compose[@]}" down --volumes --remove-orphans >/dev/null 2>&1
}
trap cleanup EXIT

"${compose[@]}" up -d --wait postgres
"${compose[@]}" run --rm --no-deps \
  -e "TIDEWISE_DATABASE_URL=postgres://tidewise:${POSTGRES_PASSWORD}@postgres:5432/tidewise_local?sslmode=disable&tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified&tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified&tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
  data /usr/local/bin/dbmigrate -apply
"${compose[@]}" up -d --wait --no-build --no-deps data
"${compose[@]}" up -d --wait --no-build --no-deps miniapp

curl --fail --silent --show-error \
  "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/api/miniapp/v1/research/themes" \
  >/dev/null
curl --fail --silent --show-error \
  "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/docs/" \
  >/dev/null

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
