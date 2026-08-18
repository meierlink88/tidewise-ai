#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
neo4j_root="$repo_root/infra/uat-neo4j"

bash -n "$neo4j_root/adopt-reason-provider.sh" "$neo4j_root/lib.sh" \
  "$neo4j_root/test-rollback.sh" "$neo4j_root/verify.sh"
bash "$neo4j_root/test-rollback.sh"

grep -qx 'server.bolt.listen_address=172.17.0.1:7687' "$neo4j_root/neo4j.conf.fragment"
grep -qx 'server.http.listen_address=127.0.0.1:7474' "$neo4j_root/neo4j.conf.fragment"
grep -qx 'dbms.security.procedures.unrestricted=apoc.*,gds.*' "$neo4j_root/neo4j.conf.fragment"
grep -qx 'dbms.security.procedures.allowlist=apoc.*,gds.*' "$neo4j_root/neo4j.conf.fragment"

if grep -REn 'REPLACE_|docker compose.* down|--remove-orphans' "$neo4j_root"; then
  echo "UAT Neo4j contains an unresolved value or an unsafe shared lifecycle command" >&2
  exit 1
fi

grep -q 'sha256sum --check' "$neo4j_root/adopt-reason-provider.sh"
# Assert the literal protected source/backup move.
# shellcheck disable=SC2016
grep -q 'mv "$neo4j_data" "$backup_root/data"' "$neo4j_root/adopt-reason-provider.sh"
# shellcheck disable=SC2016
move_line="$(grep -n -F 'mv "$neo4j_data" "$backup_root/data"' \
  "$neo4j_root/adopt-reason-provider.sh" | cut -d: -f1)"
mutation_line="$(grep -n -F 'adoption_started=true' \
  "$neo4j_root/adopt-reason-provider.sh" | head -n 1 | cut -d: -f1)"
[ "$move_line" -lt "$mutation_line" ]
grep -q 'recover_neo4j_after_failure' "$neo4j_root/adopt-reason-provider.sh"
grep -q 'neo4j_stop_attempted' "$neo4j_root/adopt-reason-provider.sh"
grep -q 'unowned_fingerprint' "$neo4j_root/adopt-reason-provider.sh"

echo "PASS UAT Neo4j repository contract"
