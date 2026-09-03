#!/usr/bin/env sh

set -eu

readonly approved_snapshot_sha256="cb178f849357d71c2490638ad69b56d8dbb268082370903e3b652a7fbdd142ef"
readonly expected_database="tidewise_uat"
readonly expected_user="tidewise_uat"
readonly expected_current_migration="80"
readonly expected_restored_migration="81"
readonly expected_table_count="51"
readonly expected_report_count="2"
readonly expected_source_count="27"
readonly expected_raw_evidence_count="93"
readonly encrypted_archive="/snapshot/snapshot.dump.enc"
readonly plaintext_archive="/work/snapshot.dump"
readonly toc_file="/work/snapshot.toc"
readonly function_config_sql="/work/function-config.sql"
readonly snapshot_key_file="${SNAPSHOT_KEY_FILE:-/run/secrets/snapshot_key}"
readonly database_password_file="${DATABASE_PASSWORD_FILE:-/run/secrets/database_password}"

fail() {
  echo "FAIL $1" >&2
  exit 1
}

scalar() {
  psql -X --quiet --tuples-only --no-align --set ON_ERROR_STOP=1 --command "$1"
}

require_exact() {
  actual="$1"
  expected="$2"
  label="$3"
  [ "$actual" = "$expected" ] || fail "$label expected $expected, got $actual"
}

cleanup() {
  rm -f "$plaintext_archive" "$toc_file" "$function_config_sql"
  unset PGPASSWORD
}
trap cleanup EXIT HUP INT TERM

action="${1:-}"
case "$action" in
  check | apply) ;;
  *) fail "action must be check or apply" ;;
esac

for command in openssl pg_restore psql sha256sum; do
  command -v "$command" >/dev/null 2>&1 || fail "required command is unavailable: $command"
done

require_exact "${PGHOST:-}" "775b3ecf9c934ae185c0b8eda157c50din03.internal.cn-east-3.postgresql.rds.myhuaweicloud.com" "database host"
require_exact "${PGPORT:-}" "5432" "database port"
require_exact "${PGDATABASE:-}" "$expected_database" "database name"
require_exact "${PGUSER:-}" "$expected_user" "database user"
require_exact "${PGSSLMODE:-}" "require" "database SSL mode"
require_exact "${SNAPSHOT_SHA256:-}" "$approved_snapshot_sha256" "approved snapshot SHA-256"
require_exact "${EXPECTED_ENCRYPTED_SHA256:-}" "$(sha256sum "$encrypted_archive" | cut -d ' ' -f 1)" "encrypted snapshot SHA-256"

[ -s "$snapshot_key_file" ] || fail "snapshot decryption key file is missing"
[ -s "$database_password_file" ] || fail "database password file is missing"
PGPASSWORD="$(cat "$database_password_file")"
export PGPASSWORD
export PGOPTIONS="-c client_min_messages=warning"

openssl enc -d -aes-256-cbc -pbkdf2 -iter 200000 \
  -in "$encrypted_archive" \
  -out "$plaintext_archive" \
  -pass "file:$snapshot_key_file"
chmod 0600 "$plaintext_archive"
require_exact "$(sha256sum "$plaintext_archive" | cut -d ' ' -f 1)" "$approved_snapshot_sha256" "decrypted snapshot SHA-256"

pg_restore --list "$plaintext_archive" > "$toc_file"
grep -Fq "Dumped from database version: 16.14" "$toc_file" || fail "snapshot was not produced by the approved PostgreSQL 16.14 source"
if grep -Eq '(^|[[:space:]])(DATABASE|DATABASE PROPERTIES|ROLE|ACL|DEFAULT ACL)([[:space:]]|$)' "$toc_file"; then
  fail "snapshot contains database-, role-, or ACL-level objects"
fi
if grep -E '^[0-9]+;.* SCHEMA ' "$toc_file" | grep -Fv ' SCHEMA public ' >/dev/null; then
  fail "snapshot contains a schema other than public"
fi

require_exact "$(scalar 'SELECT current_database()')" "$expected_database" "connected database"
require_exact "$(scalar 'SELECT current_user')" "$expected_user" "connected role"
server_version_num="$(scalar 'SHOW server_version_num')"
case "$server_version_num" in
  '' | *[!0-9]*) fail "target PostgreSQL server_version_num is invalid" ;;
esac
[ "$server_version_num" -ge 160000 ] || fail "target PostgreSQL must be version 16 or newer"
require_exact "$(scalar "SELECT COALESCE(MAX(version_id) FILTER (WHERE is_applied), 0) FROM public.goose_db_version")" "$expected_current_migration" "current migration"

if [ "$action" = check ]; then
  printf '{"action":"check","database":"%s","server_version_num":%s,"current_migration":%s,"snapshot_migration":%s,"snapshot_sha256":"%s"}\n' \
    "$expected_database" "$server_version_num" "$expected_current_migration" "$expected_restored_migration" "$approved_snapshot_sha256"
  exit 0
fi

require_exact "${TIDEWISE_UAT_PUBLIC_REPLACEMENT_CONFIRMED:-}" "issue-389-tidewise-uat-public-replacement" "destructive replacement confirmation"
require_exact "$(scalar "
SELECT COUNT(*)
FROM pg_stat_activity
WHERE datname = current_database()
  AND usename = current_user
  AND pid <> pg_backend_pid()
")" "0" "other tidewise_uat client connection count"

psql -X --quiet --set ON_ERROR_STOP=1 <<'SQL'
DROP SCHEMA public CASCADE;
CREATE SCHEMA public AUTHORIZATION pg_database_owner;
GRANT USAGE ON SCHEMA public TO PUBLIC;
SQL

pg_restore --exit-on-error --no-owner --no-acl --section=pre-data --dbname "$PGDATABASE" "$plaintext_archive"

scalar "
SELECT format(
  'ALTER FUNCTION %I.%I(%s) SET search_path TO public, pg_catalog;',
  n.nspname,
  p.proname,
  pg_get_function_identity_arguments(p.oid)
)
FROM pg_proc AS p
JOIN pg_namespace AS n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
  AND NOT EXISTS (
    SELECT 1
    FROM pg_depend AS d
    WHERE d.classid = 'pg_proc'::regclass
      AND d.objid = p.oid
      AND d.deptype = 'e'
  )
ORDER BY p.oid
" > "$function_config_sql"
psql -X --quiet --set ON_ERROR_STOP=1 --file "$function_config_sql"

pg_restore --exit-on-error --no-owner --no-acl --section=data --dbname "$PGDATABASE" "$plaintext_archive"
pg_restore --exit-on-error --no-owner --no-acl --section=post-data --dbname "$PGDATABASE" "$plaintext_archive"

scalar "
SELECT format(
  'ALTER FUNCTION %I.%I(%s) RESET search_path;',
  n.nspname,
  p.proname,
  pg_get_function_identity_arguments(p.oid)
)
FROM pg_proc AS p
JOIN pg_namespace AS n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
  AND NOT EXISTS (
    SELECT 1
    FROM pg_depend AS d
    WHERE d.classid = 'pg_proc'::regclass
      AND d.objid = p.oid
      AND d.deptype = 'e'
  )
ORDER BY p.oid
" > "$function_config_sql"
psql -X --quiet --set ON_ERROR_STOP=1 --file "$function_config_sql"

restored_migration="$(scalar "SELECT COALESCE(MAX(version_id) FILTER (WHERE is_applied), 0) FROM public.goose_db_version")"
table_count="$(scalar "SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public'")"
report_count="$(scalar 'SELECT COUNT(*) FROM public.reports')"
source_count="$(scalar 'SELECT COUNT(*) FROM public.sources')"
raw_evidence_count="$(scalar 'SELECT COUNT(*) FROM public.raw_evidences')"
configured_function_count="$(scalar "
SELECT COUNT(*)
FROM pg_proc AS p
JOIN pg_namespace AS n ON n.oid = p.pronamespace
WHERE n.nspname = 'public'
  AND p.proconfig IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM unnest(p.proconfig) AS setting WHERE setting LIKE 'search_path=%'
  )
")"

require_exact "$restored_migration" "$expected_restored_migration" "restored migration"
require_exact "$table_count" "$expected_table_count" "restored public table count"
require_exact "$report_count" "$expected_report_count" "restored report count"
require_exact "$source_count" "$expected_source_count" "restored source count"
require_exact "$raw_evidence_count" "$expected_raw_evidence_count" "restored raw evidence count"
require_exact "$configured_function_count" "0" "temporary function search_path count"

printf '{"action":"apply","database":"%s","migration":%s,"public_tables":%s,"reports":%s,"sources":%s,"raw_evidences":%s,"temporary_function_search_paths":%s,"snapshot_sha256":"%s"}\n' \
  "$expected_database" "$restored_migration" "$table_count" "$report_count" "$source_count" "$raw_evidence_count" "$configured_function_count" "$approved_snapshot_sha256"
