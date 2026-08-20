#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
env_file="$script_dir/.env.local"
openspg_url="${OPENSPG_WEB_URL:-http://127.0.0.1:8887}"
compose=(docker compose --env-file "$env_file" -f "$compose_file")

docker inspect reason-server >/dev/null
curl --fail --silent --show-error "$openspg_url/actuator/health" >/dev/null
docker exec \
  --env KAG_PROJECT_HOST_ADDR=http://127.0.0.1:8887 \
  reason-server kag --help >/dev/null
docker exec \
  --env KAG_PROJECT_HOST_ADDR=http://127.0.0.1:8887 \
  reason-server knext --help >/dev/null

project_rows="$(${compose[@]} exec -T mysql bash -c '
  mysql --batch --raw --skip-column-names -uroot -p"$MYSQL_ROOT_PASSWORD" openspg -e "
    SELECT
      id,
      namespace,
      JSON_UNQUOTE(JSON_EXTRACT(config, '\''$.graph_store.database'\''))
    FROM kg_project_info
    WHERE JSON_VALID(config) = 1
      AND JSON_UNQUOTE(JSON_EXTRACT(config, '\''$.graph_store.uri'\'')) IN (
        '\''neo4j://neo4j:7687'\'',
        '\''neo4j://release-openspg-neo4j:7687'\''
      )
    ORDER BY id;
  "
')"
[ -n "$project_rows" ] || {
  echo 'No local OpenSPG projects use the Neo4j provider' >&2
  exit 1
}

while IFS=$'\t' read -r project_id namespace database; do
  expected_database="$(tr '[:upper:]' '[:lower:]' <<<"$namespace")"
  [ "$database" = "$expected_database" ] || {
    echo "OpenSPG project $project_id ($namespace) uses database $database; expected $expected_database" >&2
    exit 1
  }
  response="$(curl --fail-with-body --silent --show-error \
    "$openspg_url/public/v1/graph/allLabels?projectId=$project_id")"
  jq -e 'type == "array"' <<<"$response" >/dev/null || {
    echo "OpenSPG graph API failed for project $project_id ($namespace)" >&2
    exit 1
  }
  echo "PASS OpenSPG project $project_id ($namespace) reached Neo4j database $database"
done <<<"$project_rows"

echo 'PASS reason-server graph API and bundled KAG/KNEXT consumer acceptance'
