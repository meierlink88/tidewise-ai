#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
expected_image='spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9'
consumer_image="${REASON_CONSUMER_IMAGE:-spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-server@sha256:fe6708deef9ebb8da8da7b1cb643e83b827769a5be8811961311639aa1f2cb88}"

set -a
# shellcheck disable=SC1090
. "$runtime_env"
set +a

[[ "$consumer_image" =~ @sha256:[a-f0-9]{64}$ ]] || {
  echo 'REASON_CONSUMER_IMAGE must be digest-addressed' >&2
  exit 1
}

compose=(docker compose --env-file "$runtime_env" -f "$script_dir/compose.yaml")
container_id="$(${compose[@]} ps -q neo4j)"
[ -n "$container_id" ]
[ "$(docker inspect "$container_id" --format '{{.Config.Image}}')" = "$expected_image" ]
[ "$(docker inspect "$container_id" --format '{{.State.Health.Status}}')" = healthy ]
[ "$(docker exec "$container_id" neo4j --version)" = 5.25.1 ]
[ "$(systemctl is-active neo4j)" = inactive ]
curl --fail --silent --show-error http://127.0.0.1:7474/ >/dev/null

for port in 7474 7687; do
  owners="$(docker ps --filter "publish=$port" --format '{{.Names}}')"
  [ "$owners" = tidewise-uat-openspg-neo4j ] || {
    echo "Unexpected UAT Neo4j port $port owner: $owners" >&2
    exit 1
  }
done

component="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "CALL dbms.components() YIELD versions RETURN versions[0] AS version;" |
    tail -n 1 | tr -d "\""
')"
[ "$component" = 5.25.1 ]
gds_version="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "RETURN gds.version() AS version;" | tail -n 1 | tr -d "\""
')"
[ "$gds_version" = 2.12.0 ]
apoc_version="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "RETURN apoc.version() AS version;" | tail -n 1 | tr -d "\""
')"
[ "$apoc_version" = 5.25.1 ]

"${compose[@]}" exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -d system -u "$user" -p "$password" \
    "CREATE DATABASE \`tidewiseuatprovidersmoke\` IF NOT EXISTS"
  cypher-shell -d system -u "$user" -p "$password" \
    "DROP DATABASE \`tidewiseuatprovidersmoke\` IF EXISTS"
' >/dev/null

project_rows="$(docker exec -e MYSQL_PWD="$OPENSPG_MYSQL_ROOT_PASSWORD" \
  tidewise-infra-uat-mysql-1 mysql --batch --raw --skip-column-names -uroot openspg -e "
    SELECT id, namespace, JSON_UNQUOTE(JSON_EXTRACT(config, '\$.graph_store.database'))
    FROM kg_project_info
    WHERE JSON_VALID(config) = 1
      AND JSON_UNQUOTE(JSON_EXTRACT(config, '\$.graph_store.uri')) IN (
        'neo4j://neo4j:7687',
        'neo4j://release-openspg-neo4j:7687'
      )
    ORDER BY id;
  ")"
[ -n "$project_rows" ]

databases="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -d system -u "$user" -p "$password" --format plain \
    "SHOW DATABASES YIELD name, currentStatus RETURN name, currentStatus ORDER BY name;"
')"
while IFS=$'\t' read -r project_id namespace database; do
  expected_database="$(tr '[:upper:]' '[:lower:]' <<<"$namespace")"
  [ "$database" = "$expected_database" ]
  grep -Fq "\"$database\", \"online\"" <<<"$databases"
  if docker inspect reason-server-uat >/dev/null 2>&1; then
    response="$(curl --fail-with-body --silent --show-error \
      "http://127.0.0.1:8887/public/v1/graph/allLabels?projectId=$project_id")"
    RESPONSE="$response" python3 - <<'PY'
import json
import os

value = json.loads(os.environ["RESPONSE"])
if not isinstance(value, list):
    raise SystemExit("OpenSPG graph/allLabels did not return an array")
PY
  fi
  echo "PASS UAT OpenSPG project $project_id ($namespace) database $database"
done <<<"$project_rows"

docker run --rm -i \
  --network tidewise-uat \
  -e OPENSPG_NEO4J_USER \
  -e OPENSPG_NEO4J_PASSWORD \
  --entrypoint python \
  "$consumer_image" - <<'PY'
import os

from neo4j import GraphDatabase

uri = "neo4j://release-openspg-neo4j:7687"
auth = (os.environ["OPENSPG_NEO4J_USER"], os.environ["OPENSPG_NEO4J_PASSWORD"])
with GraphDatabase.driver(uri, auth=auth) as driver:
    driver.verify_connectivity()
    with driver.session(database="neo4j") as session:
        version = session.run("RETURN gds.version() AS version").single()["version"]
        if version != "2.12.0":
            raise RuntimeError(f"unexpected GDS version: {version}")
print(f"Reason consumer reached OpenSPG Neo4j with GDS {version}")
PY

printf 'PASS UAT OpenSPG Neo4j 5.25.1 GDS=%s APOC=%s\n' "$gds_version" "$apoc_version"
