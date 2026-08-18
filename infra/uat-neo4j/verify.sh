#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
source "$script_dir/lib.sh"
neo4j_admin_password="${NEO4J_ADMIN_PASSWORD:?NEO4J_ADMIN_PASSWORD is required}"
consumer_image="${REASON_CONSUMER_IMAGE:-spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-server@sha256:fe6708deef9ebb8da8da7b1cb643e83b827769a5be8811961311639aa1f2cb88}"

[[ "$consumer_image" =~ @sha256:[a-f0-9]{64}$ ]] || {
  echo "REASON_CONSUMER_IMAGE must be digest-addressed" >&2
  exit 1
}

[ "$(systemctl is-active neo4j)" = active ]
[ "$(neo4j version)" = "neo4j 5.26.28" ]
curl --fail --silent --show-error http://127.0.0.1:7474/ >/dev/null
echo "PASS Neo4j service, version and HTTP"

listener_endpoints="$(
  ss -H -ltn '( sport = :7474 or sport = :7687 )' |
    awk '{print $4}' |
    normalize_neo4j_listener_endpoints
)"
expected_endpoints="$(printf '%s\n' '0.0.0.0:7474' '0.0.0.0:7687' | LC_ALL=C sort)"
if [ "$listener_endpoints" != "$expected_endpoints" ]; then
  echo "Unexpected Neo4j listeners:" >&2
  printf '%s\n' "$listener_endpoints" >&2
  exit 1
fi
printf '%s\n' "$listener_endpoints"
echo "PASS Neo4j office-allowlisted listeners"

gds_version="$(NEO4J_ADDRESS=bolt://172.17.0.1:7687 \
  NEO4J_USERNAME=neo4j \
  NEO4J_PASSWORD="$neo4j_admin_password" \
  cypher-shell \
  --format plain \
  'RETURN gds.version() AS version;' | tail -n 1 | tr -d '"')"
[ "$gds_version" = 2.13.4 ] || {
  echo "Expected GDS 2.13.4, got $gds_version" >&2
  exit 1
}
echo "PASS GDS $gds_version"

apoc_version="$(NEO4J_ADDRESS=bolt://172.17.0.1:7687 \
  NEO4J_USERNAME=neo4j \
  NEO4J_PASSWORD="$neo4j_admin_password" \
  cypher-shell \
  --format plain \
  'RETURN apoc.version() AS version;' | tail -n 1 | tr -d '"')"
[ "$apoc_version" = 5.26.28 ]
echo "PASS APOC $apoc_version"

OPENSPG_NEO4J_USER=neo4j \
OPENSPG_NEO4J_PASSWORD="$neo4j_admin_password" \
docker run --rm -i \
  --network tidewise-uat \
  --add-host release-openspg-neo4j:host-gateway \
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
        if version != "2.13.4":
            raise RuntimeError(f"unexpected GDS version: {version}")
print(f"Reason consumer reached Neo4j with GDS {version}")
PY

printf 'PASS UAT Neo4j 5.26.28 GDS=%s APOC=%s\n' "$gds_version" "$apoc_version"
