#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
neo4j_root="$repo_root/infra/uat-neo4j"
compose_file="$neo4j_root/compose.yaml"
env_file="$neo4j_root/.env.example"
expected_image='spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-neo4j@sha256:4bc5b7f6b83d333b1d2c8f60ac145c068d77d50bca65b3a07c927f9e2a541eb9'

bash -n \
  "$neo4j_root/adopt-reason-provider.sh" \
  "$neo4j_root/migrate-openspg-project-databases.sh" \
  "$neo4j_root/rollback-host-provider.sh" \
  "$neo4j_root/verify.sh"

docker compose --env-file "$env_file" -f "$compose_file" config --quiet
[ "$(docker compose --env-file "$env_file" -f "$compose_file" config --services)" = neo4j ]
[ "$(docker compose --env-file "$env_file" -f "$compose_file" config --images)" = "$expected_image" ]

COMPOSE_FILE="$compose_file" ENV_FILE="$env_file" python3 - <<'PY'
import json
import os
import subprocess

result = subprocess.run(
    [
        "docker",
        "compose",
        "--env-file",
        os.environ["ENV_FILE"],
        "-f",
        os.environ["COMPOSE_FILE"],
        "config",
        "--format",
        "json",
    ],
    check=True,
    capture_output=True,
    text=True,
)
config = json.loads(result.stdout)
assert config["name"] == "tidewise-neo4j-uat"
assert set(config["services"]) == {"neo4j"}
service = config["services"]["neo4j"]
assert service["container_name"] == "tidewise-uat-openspg-neo4j"
assert set(service["networks"]) == {"tidewise-uat"}
assert "release-openspg-neo4j" in service["networks"]["tidewise-uat"]["aliases"]
assert config["networks"]["tidewise-uat"]["external"] is True
assert service["environment"]["NEO4J_PLUGINS"] == '["apoc"]'
assert service["environment"]["NEO4J_dbms_security_procedures_unrestricted"] == "*"
assert config["volumes"]["neo4j-data"]["name"] == "tidewise-uat-openspg-neo4j-data"
assert config["volumes"]["neo4j-logs"]["name"] == "tidewise-uat-openspg-neo4j-logs"
PY

if grep -REn --include='*.sh' --include='*.yml' --include='*.yaml' \
  'docker compose.* down|--remove-orphans|docker volume rm' "$neo4j_root"; then
  echo 'UAT OpenSPG Neo4j contains an unsafe lifecycle command' >&2
  exit 1
fi

grep -q 'systemctl disable --now neo4j' "$neo4j_root/adopt-reason-provider.sh"
grep -q 'Existing host-native Neo4j contains' "$neo4j_root/adopt-reason-provider.sh"
grep -q 'systemctl enable neo4j' "$neo4j_root/rollback-host-provider.sh"
grep -q 'migrate-openspg-project-databases.sh" restore' "$neo4j_root/rollback-host-provider.sh"
grep -q 'CREATE DATABASE' "$neo4j_root/migrate-openspg-project-databases.sh"
grep -q 'PASS no UAT OpenSPG project databases are required yet' "$neo4j_root/migrate-openspg-project-databases.sh"
grep -q 'DROP DATABASE' "$neo4j_root/verify.sh"
grep -q 'PASS no UAT OpenSPG project databases require verification yet' "$neo4j_root/verify.sh"
if grep -q '\[ -n "\$project_rows" \]' \
  "$neo4j_root/migrate-openspg-project-databases.sh" "$neo4j_root/verify.sh"; then
  echo 'Empty UAT OpenSPG projects must not block provider adoption' >&2
  exit 1
fi
grep -q -- '--network tidewise-uat' "$neo4j_root/verify.sh"
if grep -q -- '--add-host' "$neo4j_root/verify.sh"; then
  echo 'UAT provider verification must use Docker DNS without host injection' >&2
  exit 1
fi

echo 'PASS UAT OpenSPG Neo4j repository contract'
