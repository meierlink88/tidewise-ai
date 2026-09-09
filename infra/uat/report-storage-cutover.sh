#!/usr/bin/env bash
# Sourced only by the bounded Data 88 deployment; image preflight precedes stop.
# Uses the owning deploy.sh state/Compose arrays and its forward-recovery trap.

prepare_report_storage_image() {
  report_backup_image="${REPORT_BACKUP_IMAGE:-}"
  if [ -z "${SWR_REGISTRY:-}" ] || [[ "$report_backup_image" != "${SWR_REGISTRY}/"* ]] || ! [[ "$report_backup_image" =~ @sha256:[0-9a-f]{64}$ ]]; then
    echo "FAIL report-storage-image: expected a digest-pinned backup client in the configured SWR registry" >&2
    return 1
  fi
  docker pull "$report_backup_image" >/dev/null
  docker run --rm "$report_backup_image" pg_dump --version
  run_report_database_dump --schema-only --no-owner --no-acl >/dev/null
  echo "PASS report-storage-backup-image-ready"
  echo "PASS report-storage-database-backup-preflight"
}

prepare_report_storage_backup() {
  report_backup_dir="${state_dir}/report-storage-${release_sha}"
  umask 077
  mkdir -p "$report_backup_dir"
  chmod 0700 "$report_backup_dir"
  report_backup_dump="${report_backup_dir}/before.dump"
  report_backup_manifest="${report_backup_dir}/before.sha256"
  if [ "$cutover_migration_started" = true ] && [ -s "$report_backup_manifest" ]; then
    (cd "$report_backup_dir" && sha256sum --check before.sha256)
    echo "PASS report-storage-original-backup-reused"
    return
  fi
  if [ "$cutover_migration_started" = true ] || [ "$data_current_version" != 000087 ]; then
    echo "FAIL report-storage-backup: original pre-migration backup is missing; restore the verified recovery point" >&2
    return 1
  fi
  run_report_database_dump --format=custom --no-owner --no-acl > "${report_backup_dump}.partial"
  test -s "${report_backup_dump}.partial"
  docker run --rm -i "$report_backup_image" pg_restore --list < "${report_backup_dump}.partial" > "${report_backup_dir}/before.contents"
  mv "${report_backup_dump}.partial" "$report_backup_dump"
  (cd "$report_backup_dir" && sha256sum before.dump > before.sha256.partial && mv before.sha256.partial before.sha256)
  sync "$report_backup_dump" "$report_backup_manifest"
  sync -f "$report_backup_dir"
  echo "PASS report-storage-database-backup"
}

apply_report_storage_cutover() {
  (cd "$report_backup_dir" && sha256sum --check before.sha256)
  local backup_hash plan_path
  backup_hash="$(cut -d ' ' -f 1 "$report_backup_manifest")"
  plan_path="report-storage-plan-${GITHUB_RUN_ID:-manual}-${GITHUB_RUN_ATTEMPT:-1}.json"
  local -a maintenance=("${candidate_compose[@]}" run --rm --no-deps \
    --user "$(id -u):$(id -g)" -v "${report_backup_dir}:/maintenance" \
    data /usr/local/bin/report-storage)
  "${maintenance[@]}" --retain-from 2026-09-08T16:00:00Z --plan "/maintenance/${plan_path}"
  # This operator-reviewed cutover must preserve both known Sep9 reports. Newer
  # reports are retained by the fixed boundary; an empty/wrong database fails.
  python3 - "${report_backup_dir}/${plan_path}" <<'PY'
import datetime
import json
import sys
plan = json.load(open(sys.argv[1]))
cutoff = datetime.datetime.fromisoformat('2026-09-08T16:00:00+00:00')
if datetime.datetime.fromisoformat(plan['retain_from'].replace('Z', '+00:00')) != cutoff:
    raise SystemExit('Report storage cutoff differs from the reviewed scope')
keep = plan.get('keep') or []
delete = plan.get('delete') or []
expected = {'RPT4587d39b-fc02-4a4d-bbc6-dbeb64f35081', 'RPT74b0c274-67f8-4e25-978e-d7876291ece5'}
if not expected.issubset({r['id'] for r in keep}):
    raise SystemExit('Known Sep9 reports missing from retained inventory')
for rows, retained in ((keep, True), (delete, False)):
    for row in rows:
        if (datetime.datetime.fromisoformat(row['published_at'].replace('Z', '+00:00')) >= cutoff) != retained:
            raise SystemExit('Report retention boundary mismatch')
print(f'Reviewed report inventory: retain={len(keep)} delete={len(delete)}')
PY
  "${maintenance[@]}" --apply --plan "/maintenance/${plan_path}" \
    --backup-reference "${report_backup_dump}#sha256=${backup_hash}"
  "${maintenance[@]}" --verify
  echo "PASS report-storage-projections-verified"
}

# Shared by pre-stop compatibility check and post-stop frozen backup.
run_report_database_dump() {
  local backup_host backup_password
  backup_host="$(runtime_value "$runtime_env" TIDEWISE_DB_HOST)"
  backup_password="$(runtime_value "$runtime_env" TIDEWISW_DB_PASSWORD)"
  if [[ "$backup_host" != *.internal.cn-east-3.postgresql.rds.myhuaweicloud.com ]] || [ -z "$backup_password" ]; then
    echo "FAIL report-storage-backup: UAT database identity/credential missing" >&2
    return 1
  fi
  PGHOST="$backup_host" PGPASSWORD="$backup_password" docker run --rm --network host \
    -e PGHOST -e PGPASSWORD -e PGPORT=5432 -e PGUSER=tidewise_uat \
    -e PGDATABASE=tidewise_uat -e PGSSLMODE=require -e PGCONNECT_TIMEOUT=10 \
    "$report_backup_image" pg_dump "$@"
}
