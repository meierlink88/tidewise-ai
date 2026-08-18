#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
# The runtime-relative library is checked separately in CI.
# shellcheck disable=SC1091
source "$script_dir/lib.sh"

validate_neo4j_backup_target /opt/tidewise/neo4j-uat/backups/20260818T141151Z
validate_neo4j_access_backup_target \
  /opt/tidewise/neo4j-uat/access-backups/20260818T141151Z
if validate_neo4j_backup_target /tmp/neo4j-backup; then
  echo "Unsafe backup target was accepted" >&2
  exit 1
fi
if validate_neo4j_access_backup_target /tmp/neo4j-access-backup; then
  echo "Unsafe access backup target was accepted" >&2
  exit 1
fi
normalized_listeners="$(
  printf '%s\n' '[::]:7687' '*:7474' | normalize_neo4j_listener_endpoints
)"
expected_listeners="$(printf '%s\n' '0.0.0.0:7474' '0.0.0.0:7687')"
[ "$normalized_listeners" = "$expected_listeners" ]

test_root="$(mktemp -d)"
test_parent="$(cd "$(dirname "$test_root")" && pwd)"
expected_test_parent="$(cd "${TMPDIR:-/tmp}" && pwd)"
[ "$test_parent" = "$expected_test_parent" ] || {
  echo "Unexpected test root: $test_root" >&2
  exit 1
}
trap 'rm -rf -- "$test_root"' EXIT

render_config="$test_root/neo4j.conf"
render_fragment="$test_root/neo4j.conf.fragment"
printf '%s\n' \
  'unmanaged.setting=preserved' \
  'server.http.listen_address=127.0.0.1:7474' \
  '# BEGIN TIDEWISE UAT NEO4J' \
  'server.bolt.listen_address=172.17.0.1:7687' \
  '# END TIDEWISE UAT NEO4J' >"$render_config"
printf '%s\n' \
  '# BEGIN TIDEWISE UAT NEO4J' \
  'server.http.listen_address=0.0.0.0:7474' \
  'server.bolt.listen_address=0.0.0.0:7687' \
  '# END TIDEWISE UAT NEO4J' >"$render_fragment"
apply_neo4j_config_fragment "$render_config" "$render_fragment"
grep -qx 'unmanaged.setting=preserved' "$render_config"
[ "$(grep -c '^server.http.listen_address=' "$render_config")" -eq 1 ]
grep -qx 'server.http.listen_address=0.0.0.0:7474' "$render_config"
grep -qx 'server.bolt.listen_address=0.0.0.0:7687' "$render_config"

data_dir="$test_root/data"
config_dir="$test_root/etc-neo4j"
plugins_dir="$test_root/plugins"
backup_dir="$test_root/backup"
mkdir -p "$data_dir" "$config_dir" "$plugins_dir" \
  "$backup_dir/data" "$backup_dir/etc-neo4j" "$backup_dir/plugins"
printf 'new-data\n' >"$data_dir/state"
printf 'new-config\n' >"$config_dir/state"
printf 'new-plugin\n' >"$plugins_dir/state"
printf 'old-data\n' >"$backup_dir/data/state"
printf 'old-config\n' >"$backup_dir/etc-neo4j/state"
printf 'old-plugin\n' >"$backup_dir/plugins/state"

restore_neo4j_files "$data_dir" "$config_dir" "$plugins_dir" "$backup_dir"
grep -qx old-data "$data_dir/state"
grep -qx old-config "$config_dir/state"
grep -qx old-plugin "$plugins_dir/state"
grep -qx new-data "$backup_dir/failed-data/state"
grep -qx new-config "$backup_dir/failed-etc-neo4j/state"
grep -qx new-plugin "$backup_dir/failed-plugins/state"

missing_backup="$test_root/missing-backup"
untouched_data="$test_root/untouched-data"
mkdir -p "$missing_backup/data" "$missing_backup/etc-neo4j" "$untouched_data"
printf 'untouched\n' >"$untouched_data/state"
if restore_neo4j_files "$untouched_data" "$test_root/missing-config" \
  "$test_root/missing-plugins" "$missing_backup" 2>/dev/null; then
  echo "Incomplete backup was accepted" >&2
  exit 1
fi
grep -qx untouched "$untouched_data/state"

mock_bin="$test_root/bin"
service_log="$test_root/systemctl.log"
mkdir -p "$mock_bin"
# Write a mock whose variables expand when the mock executes, not while building it.
# shellcheck disable=SC2016
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'printf "%s\\n" "$*" >>"$NEO4J_TEST_SERVICE_LOG"' \
  'if [ "${NEO4J_TEST_FAIL_STOP:-false}" = true ] && [ "$1" = stop ]; then exit 1; fi' \
  >"$mock_bin/systemctl"
printf '#!/usr/bin/env bash\nexit 0\n' >"$mock_bin/chown"
chmod 0755 "$mock_bin/systemctl" "$mock_bin/chown"

restore_config="$test_root/restore-neo4j.conf"
restore_backup="$test_root/restore-neo4j.conf.backup"
printf 'new-config\n' >"$restore_config"
printf 'old-config\n' >"$restore_backup"
: >"$service_log"
PATH="$mock_bin:/usr/bin:/bin" NEO4J_TEST_SERVICE_LOG="$service_log" \
  restore_neo4j_config "$restore_config" "$restore_backup"
grep -qx old-config "$restore_config"
printf 'stop neo4j\nstart neo4j\n' >"$test_root/expected-config-systemctl.log"
cmp "$test_root/expected-config-systemctl.log" "$service_log"

orchestration_root="$test_root/orchestration"
mkdir -p "$orchestration_root/current-data" "$orchestration_root/current-config" \
  "$orchestration_root/current-plugins" "$orchestration_root/backup/data" \
  "$orchestration_root/backup/etc-neo4j" "$orchestration_root/backup/plugins"
printf 'new\n' >"$orchestration_root/current-data/state"
printf 'old\n' >"$orchestration_root/backup/data/state"
: >"$service_log"
PATH="$mock_bin:/usr/bin:/bin" NEO4J_TEST_SERVICE_LOG="$service_log" \
  recover_neo4j_after_failure true true \
  "$orchestration_root/current-data" "$orchestration_root/current-config" \
  "$orchestration_root/current-plugins" "$orchestration_root/backup"
grep -qx old "$orchestration_root/current-data/state"
printf 'stop neo4j\nstart neo4j\n' >"$test_root/expected-systemctl.log"
cmp "$test_root/expected-systemctl.log" "$service_log"

pre_mutation_data="$test_root/pre-mutation-data"
mkdir -p "$pre_mutation_data"
printf 'untouched\n' >"$pre_mutation_data/state"
: >"$service_log"
PATH="$mock_bin:/usr/bin:/bin" NEO4J_TEST_SERVICE_LOG="$service_log" \
  recover_neo4j_after_failure false true \
  "$pre_mutation_data" "$test_root/unused-config" "$test_root/unused-plugins" \
  "$test_root/unused-backup"
grep -qx untouched "$pre_mutation_data/state"
grep -qx 'start neo4j' "$service_log"

stop_failure_root="$test_root/stop-failure"
mkdir -p "$stop_failure_root/current-data" "$stop_failure_root/current-config" \
  "$stop_failure_root/current-plugins" "$stop_failure_root/backup/data" \
  "$stop_failure_root/backup/etc-neo4j" "$stop_failure_root/backup/plugins"
printf 'live\n' >"$stop_failure_root/current-data/state"
set +e
PATH="$mock_bin:/usr/bin:/bin" NEO4J_TEST_SERVICE_LOG="$service_log" \
  NEO4J_TEST_FAIL_STOP=true recover_neo4j_after_failure true true \
  "$stop_failure_root/current-data" "$stop_failure_root/current-config" \
  "$stop_failure_root/current-plugins" "$stop_failure_root/backup"
stop_failure_code="$?"
set -e
[ "$stop_failure_code" -ne 0 ]
grep -qx live "$stop_failure_root/current-data/state"

printf '#!/usr/bin/env bash\nexit 1\n' >"$mock_bin/systemctl"
chmod 0755 "$mock_bin/systemctl"
diagnostic_secret='diagnostic-secret-must-not-appear'
set +e
diagnostics="$(PATH="$mock_bin:/usr/bin:/bin" \
  NEO4J_ADMIN_PASSWORD="$diagnostic_secret" \
  bash "$script_dir/verify.sh" 2>&1)"
verify_code="$?"
set -e
[ "$verify_code" -ne 0 ]
if grep -Fq "$diagnostic_secret" <<<"$diagnostics"; then
  echo "Neo4j diagnostics exposed the credential" >&2
  exit 1
fi

echo "PASS UAT Neo4j rollback, target and diagnostic contracts"
