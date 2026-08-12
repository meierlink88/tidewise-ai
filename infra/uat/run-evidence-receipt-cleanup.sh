#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
target_data_image="${TARGET_DATA_IMAGE:?TARGET_DATA_IMAGE is required}"
recovery_point_confirmed="${RECOVERY_POINT_CONFIRMED:?RECOVERY_POINT_CONFIRMED is required}"
summary_file="${GITHUB_STEP_SUMMARY:-/dev/null}"
state_dir="${deployment_root}/state"
runtime_env="${deployment_root}/runtime.env"
current_images="${state_dir}/current.images.env"
current_compose="${state_dir}/current.compose.yaml"
current_sha="${state_dir}/current.sha"
phase="initialize"
outcome="failed"
verification_status="not-run"
summary_written=false
work_dir=""
operation_id="${GITHUB_RUN_ID:-manual}"
if ! [[ "$operation_id" =~ ^[A-Za-z0-9_.-]+$ ]]; then
  echo "FAIL operation-id-gate: invalid GitHub run ID" >&2
  exit 1
fi
active_containers=()

write_summary() {
  local release="unknown"
  if [ -s "$current_sha" ]; then
    release="$(sed -n '1p' "$current_sha")"
  fi
  {
    echo "### UAT Evidence receipt cleanup"
    echo
    echo "- Current UAT release: \`${release}\`"
    echo "- Final phase: \`${phase}\`"
    echo "- Operation outcome: \`${outcome}\`"
    echo "- Data migration ledger verification: \`${verification_status}\`"
    echo "- Service containers and release state: \`unchanged\`"
  } >> "$summary_file" 2>/dev/null || true
  summary_written=true
}

finish() {
  local status="$1"
  set +e
  set +u
  for container in "${active_containers[@]}"; do
    docker rm -f "$container" >/dev/null 2>&1 || true
  done
  if [ -n "$work_dir" ] && [ -d "$work_dir" ]; then
    find "$work_dir" -type f -delete 2>/dev/null || true
    rmdir "$work_dir" 2>/dev/null || true
  fi
  if [ "$summary_written" != true ]; then
    write_summary
  fi
}
trap 'status=$?; trap - EXIT; finish "$status"; exit "$status"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [ "$recovery_point_confirmed" != true ]; then
  echo "FAIL recovery-point-gate: confirm_recovery_point=true is required" >&2
  exit 1
fi
phase="validate-release-state"
if ! [[ "$target_data_image" =~ @sha256:[0-9a-f]{64}$ ]]; then
  echo "FAIL image-gate: target Data image must be pinned by digest" >&2
  exit 1
fi
for path in "$runtime_env" "$current_images" "$current_compose" "$current_sha"; do
  if [ ! -s "$path" ]; then
    echo "FAIL release-state-gate: missing current UAT release file $path" >&2
    exit 1
  fi
done

exec 9>"${deployment_root}/deploy.lock"
if ! flock -n 9; then
  echo "FAIL deployment-lock: another UAT operation holds ${deployment_root}/deploy.lock" >&2
  exit 1
fi
echo "PASS deployment-lock"

work_dir="$(mktemp -d "${RUNNER_TEMP:-/tmp}/tidewise-evidence-receipt-cleanup.XXXXXX")"

images_env="${work_dir}/images.env"
awk -v image="$target_data_image" '
  BEGIN { replaced = 0 }
  /^DATA_IMAGE=/ { print "DATA_IMAGE=" image; replaced = 1; next }
  { print }
  END { if (!replaced) exit 2 }
' "$current_images" > "$images_env"
chmod 0600 "$images_env"

compose=(env -u DATA_IMAGE -u MINIAPP_IMAGE -u ADMINPORTAL_IMAGE -u ADMIN_IMAGE -u AGENTRUN_IMAGE docker compose --env-file "$runtime_env" --env-file "$images_env" -f "$current_compose")
run_compose_container() {
  local container="$1"
  shift
  active_containers+=("$container")
  timeout --signal=TERM --kill-after=30s 5m "${compose[@]}" run --rm --name "$container" "$@"
}
preflight_report="${work_dir}/preflight.json"
before_audit="${work_dir}/before.json"
apply_report="${work_dir}/apply.json"
verification_report="${work_dir}/verification.json"
after_audit="${work_dir}/after.json"

phase="preflight"
run_compose_container "tidewise-receipt-cleanup-${operation_id}-preflight" --no-deps data /usr/local/bin/dbmigrate > "$preflight_report"
run_compose_container "tidewise-receipt-cleanup-${operation_id}-before-audit" --no-deps --entrypoint /usr/local/bin/uat-evidence-receipt-cleanup-audit data > "$before_audit"

mode="$(python3 - "$preflight_report" "$before_audit" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
audit = json.loads(pathlib.Path(sys.argv[2]).read_text())
if audit.get("contract_version") != "uat-evidence-receipt-cleanup-audit.v1":
    raise SystemExit("cleanup audit contract mismatch")
objects = audit.get("objects") or {}
expected = {
    "raw_evidence_publication_receipts",
    "evidence_publication_receipts",
    "prevent_evidence_publication_receipt_mutation",
}
if set(objects) != expected:
    raise SystemExit("cleanup audit object set mismatch")
current = str(report.get("current_version") or "").zfill(6)
pending = [str(item.get("Version", item.get("version", ""))).zfill(6) for item in (report.get("pending") or [])]
if current == "000043" and pending == ["000044"] and all(objects.values()):
    print("apply")
elif current == "000044" and not pending and not any(objects.values()):
    print("verified-noop")
else:
    raise SystemExit(f"unexpected cleanup state: current={current} pending={pending} objects={objects}")
PY
)"

apply_status=0
verification_command_status=0
audit_command_status=0
if [ "$mode" = apply ]; then
  phase="apply"
  if ! run_compose_container "tidewise-receipt-cleanup-${operation_id}-apply" --no-deps data /usr/local/bin/dbmigrate -apply -target-version 44 > "$apply_report"; then
    apply_status=1
  fi
  phase="verify"
  if ! run_compose_container "tidewise-receipt-cleanup-${operation_id}-verification" --no-deps data /usr/local/bin/dbmigrate > "$verification_report"; then
    verification_command_status=1
  fi
  if ! run_compose_container "tidewise-receipt-cleanup-${operation_id}-after-audit" --no-deps --entrypoint /usr/local/bin/uat-evidence-receipt-cleanup-audit data > "$after_audit"; then
    audit_command_status=1
  fi
else
  cp "$preflight_report" "$verification_report"
  cp "$before_audit" "$after_audit"
fi

verification_status=failed
if [ "$verification_command_status" -eq 0 ] && [ "$audit_command_status" -eq 0 ] && python3 - "$verification_report" "$before_audit" "$after_audit" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
before = json.loads(pathlib.Path(sys.argv[2]).read_text())
after = json.loads(pathlib.Path(sys.argv[3]).read_text())
current = str(report.get("current_version") or "").zfill(6)
remaining = report.get("remaining")
if remaining is None:
    remaining = report.get("pending") or []
if current != "000044" or remaining:
    raise SystemExit(f"cleanup did not reach ledger 000044: current={current} remaining={remaining}")
if any((after.get("objects") or {}).values()):
    raise SystemExit("retired Evidence receipt objects remain after cleanup")
for table in ("raw_evidences", "evidences"):
    before_value = (before.get("protected_tables") or {}).get(table)
    after_value = (after.get("protected_tables") or {}).get(table)
    if not before_value or not after_value:
        raise SystemExit(f"missing protected table audit for {table}")
    if after_value["row_count"] < before_value["row_count"]:
        raise SystemExit(f"protected table row loss detected for {table}")
    before_rows = before_value.get("row_fingerprints") or {}
    after_rows = after_value.get("row_fingerprints") or {}
    if len(before_rows) != before_value["row_count"] or len(after_rows) != after_value["row_count"]:
        raise SystemExit(f"protected table identity audit is incomplete for {table}")
    for identity, fingerprint in before_rows.items():
        if after_rows.get(identity) != fingerprint:
            raise SystemExit(f"protected table fact drift detected for {table} identity {identity}")
    if after_value["row_count"] == before_value["row_count"] and after_value["fingerprint"] != before_value["fingerprint"]:
        raise SystemExit(f"protected table identity drift detected for {table}")
PY
then
  verification_status=passed
fi

if [ "$apply_status" -ne 0 ]; then
  if [ "$verification_status" = passed ]; then
    outcome="applied-but-command-failed"
  else
    outcome="not-verified-after-command-failure"
  fi
else
  outcome="$mode"
fi
phase="complete"
write_summary

if [ "$apply_status" -ne 0 ]; then
  echo "FAIL evidence-receipt-cleanup: ${outcome}; reviewed forward repair or RDS recovery may be required" >&2
  exit 1
fi
if [ "$verification_status" != passed ]; then
  echo "FAIL evidence-receipt-cleanup: post-operation verification failed" >&2
  exit 1
fi
echo "PASS evidence-receipt-cleanup: ${mode}"
