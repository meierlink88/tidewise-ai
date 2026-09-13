#!/usr/bin/env bash
# Sourced only by the bounded Data 92-93 deployment; image preflight precedes stop.
# Uses the owning deploy.sh state/Compose arrays and its forward-recovery trap.

prepare_storyline_domain_image() {
  storyline_backup_image="${REPORT_BACKUP_IMAGE:-}"
  if [ -z "${SWR_REGISTRY:-}" ] || [[ "$storyline_backup_image" != "${SWR_REGISTRY}/"* ]] || ! [[ "$storyline_backup_image" =~ @sha256:[0-9a-f]{64}$ ]]; then
    echo "FAIL storyline-domain-image: expected a digest-pinned backup client in the configured SWR registry" >&2
    return 1
  fi
  docker pull "$storyline_backup_image" >/dev/null
  docker run --rm "$storyline_backup_image" pg_dump --version
  run_storyline_database_dump --schema-only --no-owner --no-acl >/dev/null
  echo "PASS storyline-domain-backup-image-ready"
  echo "PASS storyline-domain-database-backup-preflight"
}

prepare_storyline_domain_backup() {
  storyline_backup_dir="${state_dir}/storyline-domain-${release_sha}"
  umask 077
  mkdir -p "$storyline_backup_dir"
  chmod 0700 "$storyline_backup_dir"
  storyline_backup_dump="${storyline_backup_dir}/before.dump"
  storyline_backup_manifest="${storyline_backup_dir}/before.sha256"
  if [ "$cutover_migration_started" = true ] && [ -s "$storyline_backup_manifest" ]; then
    (cd "$storyline_backup_dir" && sha256sum --check before.sha256)
    echo "PASS storyline-domain-original-backup-reused"
    test -s "${storyline_backup_dir}/before.tsv"
    return
  fi
  if [ "$cutover_migration_started" = true ] || [ "$data_current_version" != 000091 ]; then
    echo "FAIL storyline-domain-backup: original pre-migration backup is missing; restore the verified recovery point" >&2
    return 1
  fi
  run_storyline_database_dump --format=custom --no-owner --no-acl > "${storyline_backup_dump}.partial"
  test -s "${storyline_backup_dump}.partial"
  docker run --rm -i "$storyline_backup_image" pg_restore --list < "${storyline_backup_dump}.partial" > "${storyline_backup_dir}/before.contents"
  capture_storyline_snapshot > "${storyline_backup_dir}/before.tsv.partial"
  python3 "$(dirname "${BASH_SOURCE[0]}")/verify-storyline-domain.py" "${storyline_backup_dir}/before.tsv.partial"
  mv "${storyline_backup_dir}/before.tsv.partial" "${storyline_backup_dir}/before.tsv"
  mv "${storyline_backup_dump}.partial" "$storyline_backup_dump"
  (cd "$storyline_backup_dir" && sha256sum before.dump before.tsv > before.sha256.partial && mv before.sha256.partial before.sha256)
  sync "$storyline_backup_dump" "$storyline_backup_manifest"
  sync -f "$storyline_backup_dir"
  echo "PASS storyline-domain-database-backup"
}

# Shared by pre-stop compatibility check and post-stop frozen backup.
run_storyline_database_dump() { run_storyline_database_client pg_dump "$@"; }

run_storyline_database_client() {
  local backup_host backup_password
  backup_host="$(runtime_value "$runtime_env" TIDEWISE_DB_HOST)"
  backup_password="$(runtime_value "$runtime_env" TIDEWISW_DB_PASSWORD)"
  if [[ "$backup_host" != *.internal.cn-east-3.postgresql.rds.myhuaweicloud.com ]] || [ -z "$backup_password" ]; then
    echo "FAIL storyline-domain-backup: UAT database identity/credential missing" >&2
    return 1
  fi
  PGHOST="$backup_host" PGPASSWORD="$backup_password" docker run --rm -i --network host \
    -e PGHOST -e PGPASSWORD -e PGPORT=5432 -e PGUSER=tidewise_uat \
    -e PGDATABASE=tidewise_uat -e PGSSLMODE=require -e PGCONNECT_TIMEOUT=10 \
    "$storyline_backup_image" "$@"
}

capture_storyline_snapshot() {
  run_storyline_database_client psql -XAtq -v ON_ERROR_STOP=1 < "$(dirname "${BASH_SOURCE[0]}")/storyline-domain-snapshot.sql"
}

verify_storyline_domains() {
  (cd "$storyline_backup_dir" && sha256sum --check before.sha256)
  capture_storyline_snapshot > "${storyline_backup_dir}/after.tsv"
  python3 "$(dirname "${BASH_SOURCE[0]}")/verify-storyline-domain.py" "${storyline_backup_dir}/before.tsv" "${storyline_backup_dir}/after.tsv" | tee -a "$summary_file"
  echo "PASS storyline-domain-retained-data-verified"
}

apply_storyline_domain_cutover() {
  # At 93 only verification remains; the backfill deliberately only accepts 92.
  if [ "$data_current_version" -lt 93 ]; then
    "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply -target-version 92 > "$data2_apply_report_file"
    "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/storyline-domain-backfill -apply
    "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply -target-version 93 > "$data2_apply_report_file"
  fi
}
