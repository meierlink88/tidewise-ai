#!/usr/bin/env bash
# Sourced only by the bounded Data 91 deployment; image preflight precedes stop.
# Uses the owning deploy.sh state/Compose arrays and its forward-recovery trap.

prepare_entity_retirement_image() {
  entity_backup_image="${REPORT_BACKUP_IMAGE:-}"
  if [ -z "${SWR_REGISTRY:-}" ] || [[ "$entity_backup_image" != "${SWR_REGISTRY}/"* ]] || ! [[ "$entity_backup_image" =~ @sha256:[0-9a-f]{64}$ ]]; then
    echo "FAIL entity-retirement-image: expected a digest-pinned backup client in the configured SWR registry" >&2
    return 1
  fi
  docker pull "$entity_backup_image" >/dev/null
  docker run --rm "$entity_backup_image" pg_dump --version
  run_entity_database_dump --schema-only --no-owner --no-acl >/dev/null
  echo "PASS entity-retirement-backup-image-ready"
  echo "PASS entity-retirement-database-backup-preflight"
}

prepare_entity_retirement_backup() {
  entity_backup_dir="${state_dir}/entity-retirement-${release_sha}"
  umask 077
  mkdir -p "$entity_backup_dir"
  chmod 0700 "$entity_backup_dir"
  entity_backup_dump="${entity_backup_dir}/before.dump"
  entity_backup_manifest="${entity_backup_dir}/before.sha256"
  if [ "$cutover_migration_started" = true ] && [ -s "$entity_backup_manifest" ]; then
    (cd "$entity_backup_dir" && sha256sum --check before.sha256)
    echo "PASS entity-retirement-original-backup-reused"
    test -s "${entity_backup_dir}/before.tsv"
    return
  fi
  if [ "$cutover_migration_started" = true ] || [ "$data_current_version" != 000090 ]; then
    echo "FAIL entity-retirement-backup: original pre-migration backup is missing; restore the verified recovery point" >&2
    return 1
  fi
  run_entity_database_dump --format=custom --no-owner --no-acl > "${entity_backup_dump}.partial"
  test -s "${entity_backup_dump}.partial"
  docker run --rm -i "$entity_backup_image" pg_restore --list < "${entity_backup_dump}.partial" > "${entity_backup_dir}/before.contents"
  capture_entity_snapshot > "${entity_backup_dir}/before.tsv.partial"
  python3 "$(dirname "${BASH_SOURCE[0]}")/verify-entity-retirement.py" "${entity_backup_dir}/before.tsv.partial"
  mv "${entity_backup_dir}/before.tsv.partial" "${entity_backup_dir}/before.tsv"
  mv "${entity_backup_dump}.partial" "$entity_backup_dump"
  (cd "$entity_backup_dir" && sha256sum before.dump before.tsv > before.sha256.partial && mv before.sha256.partial before.sha256)
  sync "$entity_backup_dump" "$entity_backup_manifest"
  sync -f "$entity_backup_dir"
  echo "PASS entity-retirement-database-backup"
}

# Shared by pre-stop compatibility check and post-stop frozen backup.
run_entity_database_dump() { run_entity_database_client pg_dump "$@"; }

run_entity_database_client() {
  local backup_host backup_password
  backup_host="$(runtime_value "$runtime_env" TIDEWISE_DB_HOST)"
  backup_password="$(runtime_value "$runtime_env" TIDEWISW_DB_PASSWORD)"
  if [[ "$backup_host" != *.internal.cn-east-3.postgresql.rds.myhuaweicloud.com ]] || [ -z "$backup_password" ]; then
    echo "FAIL entity-retirement-backup: UAT database identity/credential missing" >&2
    return 1
  fi
  PGHOST="$backup_host" PGPASSWORD="$backup_password" docker run --rm -i --network host \
    -e PGHOST -e PGPASSWORD -e PGPORT=5432 -e PGUSER=tidewise_uat \
    -e PGDATABASE=tidewise_uat -e PGSSLMODE=require -e PGCONNECT_TIMEOUT=10 \
    "$entity_backup_image" "$@"
}

capture_entity_snapshot() {
  run_entity_database_client psql -XAtq -v ON_ERROR_STOP=1 < "$(dirname "${BASH_SOURCE[0]}")/entity-retirement-snapshot.sql"
}

verify_entity_retirement() {
  (cd "$entity_backup_dir" && sha256sum --check before.sha256)
  capture_entity_snapshot > "${entity_backup_dir}/after.tsv"
  python3 "$(dirname "${BASH_SOURCE[0]}")/verify-entity-retirement.py" "${entity_backup_dir}/before.tsv" "${entity_backup_dir}/after.tsv" | tee -a "$summary_file"
  echo "PASS entity-retirement-retained-data-verified"
}
