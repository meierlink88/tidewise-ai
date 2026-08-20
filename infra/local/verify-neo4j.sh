#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
env_file="$script_dir/.env.local"
expected_image='spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9'
expected_data_volume='tidewise-reason_neo4j-data'
expected_log_volume='tidewise-reason_neo4j-logs'

compose=(docker compose --env-file "$env_file" -f "$compose_file")
container_id="$(${compose[@]} ps -q neo4j)"
[ -n "$container_id" ] || {
  echo 'Local OpenSPG Neo4j container is not running' >&2
  exit 1
}

[ "$(docker inspect "$container_id" --format '{{.Config.Image}}')" = "$expected_image" ]
[ "$(docker inspect "$container_id" --format '{{.State.Health.Status}}')" = healthy ]
[ "$(docker exec "$container_id" neo4j --version)" = 5.25.1 ]
http_endpoint="$(${compose[@]} port neo4j 7474)"
[ -n "$http_endpoint" ]
curl --fail --silent --show-error "http://$http_endpoint/" >/dev/null

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

databases="$(${compose[@]} exec -T neo4j bash -c '
  user="${NEO4J_AUTH%%/*}"
  password="${NEO4J_AUTH#*/}"
  cypher-shell -d system -u "$user" -p "$password" --format plain \
    "SHOW DATABASES YIELD name, currentStatus RETURN name, currentStatus ORDER BY name;"
')"
for database in neo4j reasonsmoke system tidewise; do
  grep -Fq "\"$database\", \"online\"" <<<"$databases"
done

mounts="$(docker inspect "$container_id" --format '{{range .Mounts}}{{println .Name .Destination}}{{end}}')"
grep -qx "$expected_data_volume /data" <<<"$mounts"
grep -qx "$expected_log_volume /logs" <<<"$mounts"
if grep -q 'neo4j-5.26' <<<"$mounts"; then
  echo 'OpenSPG Neo4j unexpectedly mounts an abandoned 5.26 volume' >&2
  exit 1
fi

printf 'PASS local OpenSPG Neo4j 5.25.1 GDS=%s APOC=%s with isolated project databases\n' \
  "$gds_version" "$apoc_version"
