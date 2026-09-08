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
export DATA_SERVICE_CONTAINER_NAME="${project_name}-data"
export MINIAPP_SERVICE_CONTAINER_NAME="${project_name}-miniapp"
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

data_api="http://127.0.0.1:${DATA_SERVICE_PORT}/api/data/v1"
miniapp_api="http://127.0.0.1:${MINIAPP_SERVICE_PORT}/api/miniapp/v1"
auth_header="Authorization: Bearer ${DATA_SERVICE_TOKEN}"

echo "Publishing smoke Raw Evidence fixture"
if ! raw_response="$(
  jq --arg publication_key "report-smoke-${run_suffix}" '
    .raw_evidence.publication_key = $publication_key
  ' "$repo_root/data-service/backend/api/data/v1/evidence/testdata/raw-evidence-publication.json" | \
    curl --fail-with-body --silent --show-error \
      -H "$auth_header" -H 'Content-Type: application/json' \
      --data-binary @- "$data_api/raw-evidence-publications"
)"; then
  echo "Raw Evidence fixture publication failed" >&2
  printf '%s\n' "$raw_response" >&2
  exit 1
fi
raw_evidence_id="$(jq -er '.result.id' <<<"$raw_response")"

echo "Publishing smoke Atomic Evidence fixture"
evidence_response="$(
  jq --arg raw_evidence_id "$raw_evidence_id" '.raw_evidence_id = $raw_evidence_id' \
    "$repo_root/data-service/backend/api/data/v1/evidence/testdata/evidence-publication.json" | \
    curl --fail-with-body --silent --show-error \
      -H "$auth_header" -H 'Content-Type: application/json' \
      --data-binary @- "$data_api/evidence-publications"
)"
evidence_one="$(jq -er '.result.items | map(select(.input_index == 0)) | .[0].id' <<<"$evidence_response")"
evidence_two="$(jq -er '.result.items | map(select(.input_index == 1)) | .[0].id' <<<"$evidence_response")"

echo "Publishing smoke Report fixture"
if ! report_response="$(
  jq --arg evidence_one "$evidence_one" --arg evidence_two "$evidence_two" \
    --arg publisher_report_id "report-smoke-${run_suffix}" '
      .publisher_report_id = $publisher_report_id
      | walk(
          if type == "string" and . == "EVD11111111-1111-4111-8111-111111111111" then
            $evidence_one
          elif type == "string" and . == "EVD22222222-2222-4222-8222-222222222222" then
            $evidence_two
          else . end
        )
    ' "$repo_root/data-service/backend/api/data/v1/report/testdata/normalized-publication-request.json" | \
    curl --fail-with-body --silent --show-error \
      -H "$auth_header" -H 'Content-Type: application/json' \
      --data-binary @- "$data_api/report-publications"
)"; then
  echo "Report fixture publication failed" >&2
  printf '%s\n' "$report_response" >&2
  exit 1
fi
report_id="$(jq -er '.result.report_id' <<<"$report_response")"

echo "Reading smoke Miniapp Report homepage"
if ! home_response="$(curl --fail-with-body --silent --show-error "$miniapp_api/reports/home")"; then
  echo "Miniapp Report homepage read failed" >&2
  printf '%s\n' "$home_response" >&2
  exit 1
fi
jq -e --arg report_id "$report_id" '
  .result.selection.timezone == "Asia/Shanghai"
  and (.result.reports | length == 1)
  and .result.reports[0].report.id == $report_id
  and .result.reports[0].report.schema_version == "report-publication/v4"
  and (.result.reports[0].analysis_groups | map(.kind) ==
    ["geopolitical_stories", "macroeconomic_stories", "concept_analyses"])
  and all(.result.reports[0].analysis_groups[]; (.items | length == 1) and .next_cursor == null)
' <<<"$home_response" >/dev/null

for kind in geopolitical_stories macroeconomic_stories concept_analyses; do
  echo "Reading smoke Miniapp ${kind} cards and details"
  page_response="$(curl --fail --silent --show-error "$miniapp_api/reports/$report_id/analyses/$kind?limit=1")"
  expected_items="$(jq -c --arg kind "$kind" '.result.reports[0].analysis_groups[] | select(.kind == $kind) | .items' <<<"$home_response")"
  jq -e --argjson expected "$expected_items" '
    .result.items == $expected and .result.next_cursor == null
  ' <<<"$page_response" >/dev/null
  analysis_key="$(jq -er '.result.items[0].local_key' <<<"$page_response")"
  detail_response="$(curl --fail --silent --show-error "$miniapp_api/reports/$report_id/analyses/$kind/$analysis_key")"
  jq -e --argjson expected "$expected_items" '
    .result.summary == $expected[0] and (.result.industry_chains | length > 0)
  ' <<<"$detail_response" >/dev/null
  chain_keys="$(jq -er '.result.industry_chains[].local_key' <<<"$detail_response")"
  while IFS= read -r chain_key; do
    chain_response="$(curl --fail --silent --show-error "$miniapp_api/reports/$report_id/analyses/$kind/$analysis_key/industry-chains/$chain_key")"
    expected_chain="$(jq -c --arg kind "$kind" --arg analysis_key "$analysis_key" --arg chain_key "$chain_key" '
      .report[$kind][] | select(.local_key == $analysis_key)
      | .detail.industry_chains[] | select(.local_key == $chain_key)
    ' "$repo_root/data-service/backend/api/data/v1/report/testdata/normalized-publication-request.json")"
    jq -e --argjson expected "$expected_chain" '
      .result.local_key == $expected.local_key
      and .result.graph == $expected.graph
      and (.result.affected_nodes | map(.local_key)) == ($expected.affected_nodes | map(.local_key))
      and .result.empty_state == $expected.empty_state
    ' <<<"$chain_response" >/dev/null
  done <<<"$chain_keys"
done
scope_token="$(jq -er '.result.reports[0].analysis_groups[0].items[0].summary.evidence_scope_token' <<<"$home_response")"

evidence_list_response="$(curl --fail --silent --show-error \
  "$miniapp_api/reports/$report_id/evidences?scope_token=$scope_token")"
jq -e --arg scope_token "$scope_token" '
  .result.scope_token == $scope_token
  and (.result.items | length == 1)
  and (.result.items[0].summary | length > 0)
  and (.result.items[0] | has("evidence_id") | not)
' <<<"$evidence_list_response" >/dev/null

for retired_path in "industry-chains" "layers/geopolitics" "industry-chains/chain-01"; do
  retired_status="$(curl --silent --show-error --output /dev/null --write-out "%{http_code}" "$miniapp_api/reports/$report_id/$retired_path")"
  if [[ "$retired_status" != "404" ]]; then
    echo "Retired Report route ${retired_path} returned ${retired_status}, want 404" >&2
    exit 1
  fi
done

curl --fail --silent --show-error \
  "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/docs/" \
  >/dev/null

"${compose[@]}" stop data >/dev/null
failure_status="$(
  curl --silent --show-error \
    --output "$failure_body" \
    --write-out "%{http_code}" \
    "http://127.0.0.1:${MINIAPP_SERVICE_PORT}/api/miniapp/v1/reports/home"
)"
if [[ "$failure_status" != "503" ]]; then
  echo "Miniapp returned ${failure_status}, want 503 while Data Service is unavailable" >&2
  sed -n '1,20p' "$failure_body" >&2
  exit 1
fi
if ! grep -Fq '"code":"REPORT_SERVICE_UNAVAILABLE"' "$failure_body"; then
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
