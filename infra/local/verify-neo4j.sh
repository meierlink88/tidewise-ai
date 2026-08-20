#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
env_file="$script_dir/.env.local"
expected_image='neo4j:5.26.28-community@sha256:ff32db30b2baff97971e441b46bfd9c832c1b62c970398ef579244c06b21d357'
old_data_volume='tidewise-reason_neo4j-data'
target_data_volume='tidewise-reason_neo4j-5.26-data'
target_plugin_volume='tidewise-reason_neo4j-5.26-plugins'

compose=(docker compose --env-file "$env_file" -f "$compose_file")
container_id="$(${compose[@]} ps -q neo4j)"
[ -n "$container_id" ] || {
  echo 'Local Neo4j container is not running' >&2
  exit 1
}

[ "$(docker inspect "$container_id" --format '{{.Config.Image}}')" = "$expected_image" ]
[ "$(docker inspect "$container_id" --format '{{.State.Health.Status}}')" = healthy ]
[ "$(docker exec "$container_id" neo4j --version)" = 5.26.28 ]
http_endpoint="$(${compose[@]} port neo4j 7474)"
[ -n "$http_endpoint" ]
curl --fail --silent --show-error "http://$http_endpoint/" >/dev/null

component="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "CALL dbms.components() YIELD versions, edition RETURN versions[0] AS version, edition;" |
    tail -n 1 | tr -d "\""
')"
[ "$component" = '5.26.28, community' ] || {
  echo "Unexpected Neo4j component: $component" >&2
  exit 1
}

gds_version="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "RETURN gds.version() AS version;" | tail -n 1 | tr -d "\""
')"
[ "$gds_version" = 2.13.4 ]

apoc_version="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "RETURN apoc.version() AS version;" | tail -n 1 | tr -d "\""
')"
[ "$apoc_version" = 5.26.28 ]

write_smoke="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -u "$user" -p "$password" --format plain \
    "CREATE (n:LocalNeo4jUpgradeSmoke {id: randomUUID()}) WITH n DELETE n RETURN 1 AS deleted;" |
    tail -n 1
')"
[ "$write_smoke" = 1 ]

mounts="$(docker inspect "$container_id" --format '{{range .Mounts}}{{println .Name .Destination}}{{end}}')"
grep -qx "$target_data_volume /data" <<<"$mounts"
grep -qx "$target_plugin_volume /plugins" <<<"$mounts"
if grep -q "^$old_data_volume " <<<"$mounts"; then
  echo "Target Neo4j unexpectedly mounts legacy data volume $old_data_volume" >&2
  exit 1
fi
docker volume inspect "$old_data_volume" >/dev/null

printf 'PASS local Neo4j 5.26.28 Community GDS=%s APOC=%s; legacy volume retained\n' \
  "$gds_version" "$apoc_version"
