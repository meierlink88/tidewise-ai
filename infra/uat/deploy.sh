#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
candidate_images="${CANDIDATE_IMAGES:?CANDIDATE_IMAGES is required}"
release_sha="${COMMIT_SHA:?COMMIT_SHA is required}"
expected_current_available="${EXPECTED_CURRENT_RELEASE_AVAILABLE:?EXPECTED_CURRENT_RELEASE_AVAILABLE is required}"
expected_current_state_fingerprint="${EXPECTED_CURRENT_RELEASE_STATE_FINGERPRINT:?EXPECTED_CURRENT_RELEASE_STATE_FINGERPRINT is required}"
expected_current_sha="${EXPECTED_CURRENT_RELEASE_SHA:-}"
expected_current_data_image="${EXPECTED_CURRENT_DATA_IMAGE:-}"
expected_current_miniapp_image="${EXPECTED_CURRENT_MINIAPP_IMAGE:-}"
expected_current_adminportal_image="${EXPECTED_CURRENT_ADMINPORTAL_IMAGE:-}"
expected_current_admin_image="${EXPECTED_CURRENT_ADMIN_IMAGE:-}"
backup_confirmed="${HIGH_RISK_BACKUP_CONFIRMED:-false}"
deployment_mode="${DEPLOYMENT_MODE:-normal}"
destructive_data_change_confirmed="${DESTRUCTIVE_DATA_CHANGE_CONFIRMED:-false}"
empty_data_schema_rebuild_requested="${EMPTY_DATA_SCHEMA_REBUILD_REQUESTED:-false}"
compose_file="${COMPOSE_FILE:-infra/uat/docker-compose.yaml}"
migration_risk_manifest="${MIGRATION_RISK_MANIFEST:-infra/uat/migration-risk.tsv}"
summary_file="${GITHUB_STEP_SUMMARY:-/dev/null}"
state_dir="${deployment_root}/state"
current_runtime="${deployment_root}/runtime.env"
current_images="${state_dir}/current.images.env"
current_compose="${state_dir}/current.compose.yaml"
current_sha="${state_dir}/current.sha"
previous_runtime="${deployment_root}/previous.runtime.env"
previous_images="${state_dir}/previous.images.env"
previous_compose="${state_dir}/previous.compose.yaml"
previous_sha="${state_dir}/previous.sha"
report_file="${RUNNER_TEMP:-/tmp}/tidewise-uat-migration-${GITHUB_RUN_ID:-manual}.json"
data2_apply_report_file="${RUNNER_TEMP:-/tmp}/tidewise-uat-data2-apply-${GITHUB_RUN_ID:-manual}.json"
release_state_write_marker="${state_dir}/release-state-write-in-progress"
data2_cutover_marker="${state_dir}/tidewise-2-cutover-in-progress"
pre_data2_runtime="${deployment_root}/pre-data2.runtime.env"
pre_data2_images="${state_dir}/pre-data2.images.env"
pre_data2_compose="${state_dir}/pre-data2.compose.yaml"
pre_data2_sha="${state_dir}/pre-data2.sha"
pre_data59_runtime="${deployment_root}/pre-data59.runtime.env"
pre_data59_images="${state_dir}/pre-data59.images.env"
pre_data59_compose="${state_dir}/pre-data59.compose.yaml"
pre_data59_sha="${state_dir}/pre-data59.sha"
pre_data60_runtime="${deployment_root}/pre-data60.runtime.env"
pre_data60_images="${state_dir}/pre-data60.images.env"
pre_data60_compose="${state_dir}/pre-data60.compose.yaml"
pre_data60_sha="${state_dir}/pre-data60.sha"
pre_data78_runtime="${deployment_root}/pre-data78.runtime.env"
pre_data78_images="${state_dir}/pre-data78.images.env"
pre_data78_compose="${state_dir}/pre-data78.compose.yaml"
pre_data78_sha="${state_dir}/pre-data78.sha"
pre_data80_runtime="${deployment_root}/pre-data80.runtime.env"
pre_data80_images="${state_dir}/pre-data80.images.env"
pre_data80_compose="${state_dir}/pre-data80.compose.yaml"
pre_data80_sha="${state_dir}/pre-data80.sha"
agentrun_rollback_marker="${state_dir}/agentrun-010-rollback-required"
agentrun_version_publication="${state_dir}/agentrun-agent-version-publication.json"
candidate_services_started=false
rollback_snapshot_ready=false
cutover_migration_started=false
cutover_recovery_phase=""
bounded_data_cutover=false
cutover_target_version=""
cutover_target_version_padded=""
cutover_initial_current_version=""
cutover_initial_pending_versions=""
cutover_recovery_minimum_version=""
cutover_gate_name=""
cutover_release_state_mode=""
cutover_checkpoint_runtime=""
cutover_checkpoint_images=""
cutover_checkpoint_compose=""
cutover_checkpoint_sha=""
interrupted_state_recovery_mode=""
committed_cutover_recovery=false
host_base_url="${UAT_HOST_BASE_URL:-http://127.0.0.1}"
application_services=(data miniapp adminportal admin)

test -d "$state_dir"
test -w "$state_dir"

case "$deployment_mode" in
  normal) ;;
  tidewise_2_cutover)
    bounded_data_cutover=true
    cutover_target_version=58
    cutover_target_version_padded=000058
    cutover_initial_current_version=000044
    cutover_initial_pending_versions=000045,000046,000047,000048,000049,000050,000051,000052,000053,000054,000055,000056,000057,000058
    cutover_recovery_minimum_version=44
    cutover_gate_name=data2
    cutover_release_state_mode=pre-data2
    cutover_checkpoint_runtime="$pre_data2_runtime"
    cutover_checkpoint_images="$pre_data2_images"
    cutover_checkpoint_compose="$pre_data2_compose"
    cutover_checkpoint_sha="$pre_data2_sha"
    ;;
  data_59_cutover)
    bounded_data_cutover=true
    cutover_target_version=59
    cutover_target_version_padded=000059
    cutover_initial_current_version=000058
    cutover_initial_pending_versions=000059
    cutover_recovery_minimum_version=58
    cutover_gate_name=data59
    cutover_release_state_mode=pre-data59
    cutover_checkpoint_runtime="$pre_data59_runtime"
    cutover_checkpoint_images="$pre_data59_images"
    cutover_checkpoint_compose="$pre_data59_compose"
    cutover_checkpoint_sha="$pre_data59_sha"
    ;;
  data_60_cutover)
    bounded_data_cutover=true
    cutover_target_version=60
    cutover_target_version_padded=000060
    cutover_initial_current_version=000059
    cutover_initial_pending_versions=000060
    cutover_recovery_minimum_version=59
    cutover_gate_name=data60
    cutover_release_state_mode=pre-data60
    cutover_checkpoint_runtime="$pre_data60_runtime"
    cutover_checkpoint_images="$pre_data60_images"
    cutover_checkpoint_compose="$pre_data60_compose"
    cutover_checkpoint_sha="$pre_data60_sha"
    ;;
  data_78_79_cutover)
    bounded_data_cutover=true
    cutover_target_version=79
    cutover_target_version_padded=000079
    cutover_initial_current_version=000077
    cutover_initial_pending_versions=000078,000079
    cutover_recovery_minimum_version=77
    cutover_gate_name=data78-79
    cutover_release_state_mode=pre-data78
    cutover_checkpoint_runtime="$pre_data78_runtime"
    cutover_checkpoint_images="$pre_data78_images"
    cutover_checkpoint_compose="$pre_data78_compose"
    cutover_checkpoint_sha="$pre_data78_sha"
    ;;
  data_80_cutover)
    bounded_data_cutover=true
    cutover_target_version=80
    cutover_target_version_padded=000080
    cutover_initial_current_version=000079
    cutover_initial_pending_versions=000080
    cutover_recovery_minimum_version=79
    cutover_gate_name=data80
    cutover_release_state_mode=pre-data80
    cutover_checkpoint_runtime="$pre_data80_runtime"
    cutover_checkpoint_images="$pre_data80_images"
    cutover_checkpoint_compose="$pre_data80_compose"
    cutover_checkpoint_sha="$pre_data80_sha"
    ;;
  *)
    echo "FAIL deployment-mode-gate: DEPLOYMENT_MODE must be normal, tidewise_2_cutover, data_59_cutover, data_60_cutover, data_78_79_cutover, or data_80_cutover" >&2
    exit 1
    ;;
esac
if [ "$empty_data_schema_rebuild_requested" = true ] && [ "$deployment_mode" != tidewise_2_cutover ]; then
  echo "FAIL deployment-mode-gate: empty Data schema rebuild is only available in tidewise_2_cutover" >&2
  exit 1
fi

write_data2_cutover_marker() {
  local phase="$1"
  local temporary_marker
  temporary_marker="$(mktemp "${state_dir}/tidewise-2-cutover.XXXXXX")"
  {
    printf 'release_sha=%s\n' "$release_sha"
    printf 'phase=%s\n' "$phase"
    printf 'target_version=%s\n' "$cutover_target_version"
  } > "$temporary_marker"
  chmod 0640 "$temporary_marker"
  sync "$temporary_marker"
  mv -f "$temporary_marker" "$data2_cutover_marker"
  sync -f "$state_dir"
}

data2_cutover_marker_value() {
  local key="$1"
  sed -n "s/^${key}=//p" "$data2_cutover_marker" | tail -n 1
}

data2_cutover_marker_target_version() {
  local target_version
  target_version="$(data2_cutover_marker_value target_version)"
  printf '%s\n' "${target_version:-58}"
}

application_container_ids() {
  local include_stopped="$1"
  local service="$2"
  local -a docker_ps=(docker ps)
  if [ "$include_stopped" = true ]; then
    docker_ps+=(--all)
  fi
  "${docker_ps[@]}" \
    --filter label=com.docker.compose.project=tidewise-uat \
    --filter "label=com.docker.compose.service=${service}" \
    --quiet
}

write_release_state_marker() {
  local phase="$1"
  local temporary_marker
  temporary_marker="$(mktemp "${state_dir}/release-state-write.XXXXXX")"
  printf '%s\n' "$phase" > "$temporary_marker"
  chmod 0640 "$temporary_marker"
  sync "$temporary_marker"
  mv -f "$temporary_marker" "$release_state_write_marker"
  sync -f "$state_dir"
}

restore_interrupted_release_state() {
  local recovery_mode
  recovery_mode="$(sed -n '1p' "$release_state_write_marker")"
  case "$recovery_mode" in
    committed)
      if [ ! -s "$current_runtime" ] || [ ! -s "$current_images" ] || [ ! -s "$current_compose" ] || [ ! -s "$current_sha" ]; then
        echo "FAIL release-state-recovery: committed state is incomplete" >&2
        return 1
      fi
      ;;
    previous)
      if [ ! -s "$previous_runtime" ] || [ ! -s "$previous_images" ] || [ ! -s "$previous_compose" ] || [ ! -s "$previous_sha" ]; then
        echo "FAIL release-state-recovery: previous snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$previous_runtime" "$current_runtime"
      install -m 0640 "$previous_images" "$current_images"
      install -m 0640 "$previous_compose" "$current_compose"
      install -m 0640 "$previous_sha" "$current_sha"
      ;;
    pre-data2)
      if [ ! -s "$pre_data2_runtime" ] || [ ! -s "$pre_data2_images" ] || [ ! -s "$pre_data2_compose" ] || [ ! -s "$pre_data2_sha" ]; then
        echo "FAIL release-state-recovery: pre-Data-2.0 snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$pre_data2_runtime" "$current_runtime"
      install -m 0640 "$pre_data2_images" "$current_images"
      install -m 0640 "$pre_data2_compose" "$current_compose"
      install -m 0640 "$pre_data2_sha" "$current_sha"
      ;;
    pre-data59)
      if [ ! -s "$pre_data59_runtime" ] || [ ! -s "$pre_data59_images" ] || [ ! -s "$pre_data59_compose" ] || [ ! -s "$pre_data59_sha" ]; then
        echo "FAIL release-state-recovery: pre-Data-59 snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$pre_data59_runtime" "$current_runtime"
      install -m 0640 "$pre_data59_images" "$current_images"
      install -m 0640 "$pre_data59_compose" "$current_compose"
      install -m 0640 "$pre_data59_sha" "$current_sha"
      ;;
    pre-data60)
      if [ ! -s "$pre_data60_runtime" ] || [ ! -s "$pre_data60_images" ] || [ ! -s "$pre_data60_compose" ] || [ ! -s "$pre_data60_sha" ]; then
        echo "FAIL release-state-recovery: pre-Data-60 snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$pre_data60_runtime" "$current_runtime"
      install -m 0640 "$pre_data60_images" "$current_images"
      install -m 0640 "$pre_data60_compose" "$current_compose"
      install -m 0640 "$pre_data60_sha" "$current_sha"
      ;;
    pre-data78)
      if [ ! -s "$pre_data78_runtime" ] || [ ! -s "$pre_data78_images" ] || [ ! -s "$pre_data78_compose" ] || [ ! -s "$pre_data78_sha" ]; then
        echo "FAIL release-state-recovery: pre-Data-78 snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$pre_data78_runtime" "$current_runtime"
      install -m 0640 "$pre_data78_images" "$current_images"
      install -m 0640 "$pre_data78_compose" "$current_compose"
      install -m 0640 "$pre_data78_sha" "$current_sha"
      ;;
    pre-data80)
      if [ ! -s "$pre_data80_runtime" ] || [ ! -s "$pre_data80_images" ] || [ ! -s "$pre_data80_compose" ] || [ ! -s "$pre_data80_sha" ]; then
        echo "FAIL release-state-recovery: pre-Data-80 snapshot is incomplete" >&2
        return 1
      fi
      install -m 0600 "$pre_data80_runtime" "$current_runtime"
      install -m 0640 "$pre_data80_images" "$current_images"
      install -m 0640 "$pre_data80_compose" "$current_compose"
      install -m 0640 "$pre_data80_sha" "$current_sha"
      ;;
    none)
      rm -f "$current_runtime" "$current_images" "$current_compose" "$current_sha"
      ;;
    *)
      echo "FAIL release-state-recovery: invalid write marker" >&2
      return 1
      ;;
  esac
  sync "$current_runtime" "$current_images" "$current_compose" "$current_sha" 2>/dev/null || true
  sync -f "$deployment_root"
  rm -f "$release_state_write_marker"
  sync -f "$state_dir"
  echo "PASS recovered-interrupted-release-state"
}

current_image_value() {
  local key="$1"
  sed -n "s/^${key}=//p" "$current_images" | tail -n 1
}

current_release_state_fingerprint() {
  local path
  for path in "$current_runtime" "$current_sha" "$current_images" "$current_compose" "$release_state_write_marker" "$data2_cutover_marker"; do
    if [ -e "$path" ]; then
      sha256sum "$path"
    else
      printf 'missing  %s\n' "$path"
    fi
  done | sha256sum | cut -d ' ' -f 1
}

verify_planned_release_state() {
  local recovered_cutover_state=false
  if [ "$bounded_data_cutover" = true ] && [[ "$interrupted_state_recovery_mode" =~ ^(pre-data2|pre-data59|pre-data60|pre-data78|pre-data80|committed)$ ]]; then
    recovered_cutover_state=true
  fi
  if [ "$recovered_cutover_state" != true ] && [ "$(current_release_state_fingerprint)" != "$expected_current_state_fingerprint" ]; then
    echo "FAIL release-state-plan-gate: current release state changed after planning; rerun deployment" >&2
    return 1
  fi
  if [ "$expected_current_available" = false ]; then
    echo "PASS release-state-plan-gate"
    return 0
  fi
  if [ "$expected_current_available" != true ]; then
    echo "FAIL release-state-plan-gate: expected current release availability must be true or false" >&2
    return 1
  fi
  if [ -e "$release_state_write_marker" ] || [ ! -s "$current_runtime" ] || [ ! -s "$current_images" ] || [ ! -s "$current_compose" ] || [ ! -s "$current_sha" ]; then
    echo "FAIL release-state-plan-gate: current release became incomplete after planning; rerun deployment" >&2
    return 1
  fi
  if [ "$(sed -n '1p' "$current_sha")" != "$expected_current_sha" ] || \
    [ "$(current_image_value DATA_IMAGE)" != "$expected_current_data_image" ] || \
    [ "$(current_image_value MINIAPP_IMAGE)" != "$expected_current_miniapp_image" ] || \
    [ "$(current_image_value ADMINPORTAL_IMAGE)" != "$expected_current_adminportal_image" ] || \
    [ "$(current_image_value ADMIN_IMAGE)" != "$expected_current_admin_image" ]; then
    echo "FAIL release-state-plan-gate: current release changed after planning; rerun deployment" >&2
    return 1
  fi
  echo "PASS release-state-plan-gate"
}

exec 9>"${deployment_root}/deploy.lock"
if ! flock -n 9; then
  echo "FAIL deployment-lock: another UAT deployment holds ${deployment_root}/deploy.lock" >&2
  exit 1
fi
echo "PASS deployment-lock"
if [ -f "$release_state_write_marker" ]; then
  interrupted_state_recovery_mode="$(sed -n '1p' "$release_state_write_marker")"
  restore_interrupted_release_state
fi
verify_planned_release_state
if [ "$bounded_data_cutover" = true ] && [ "$interrupted_state_recovery_mode" = committed ]; then
  if [ ! -s "$data2_cutover_marker" ] || \
    [ "$(data2_cutover_marker_value release_sha)" != "$release_sha" ] || \
    [ "$(data2_cutover_marker_value phase)" != data-migrated ] || \
    [ "$(data2_cutover_marker_target_version)" != "$cutover_target_version" ]; then
    echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: matching data-migrated cutover marker is required" >&2
    exit 1
  fi
  if [ "$(sed -n '1p' "$current_sha")" != "$release_sha" ]; then
    echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: committed release SHA does not match the requested release" >&2
    exit 1
  fi
  for image_key in DATA_IMAGE MINIAPP_IMAGE ADMINPORTAL_IMAGE ADMIN_IMAGE; do
    if [[ "$(current_image_value "$image_key")" != *":${release_sha}" ]]; then
      echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: ${image_key} is not the committed cutover image" >&2
      exit 1
    fi
  done
  committed_cutover_recovery=true
fi

# Process environment variables have higher precedence than Compose --env-file.
# The workflow exposes candidate image names at job scope, so clear them before
# every Compose invocation and let the selected release image file be authoritative.
compose_command=(env -u DATA_IMAGE -u MINIAPP_IMAGE -u ADMINPORTAL_IMAGE -u ADMIN_IMAGE docker compose)
candidate_compose=("${compose_command[@]}" --env-file "$runtime_env" --env-file "$candidate_images" -f "$compose_file")

runtime_value() {
  local file="$1"
  local key="$2"
  sed -n "s/^${key}=//p" "$file" | tail -n 1
}

validate_application_only_release() {
  local validation_runtime="$1"
  local validation_images="$2"
  local validation_compose_file="$3"
  local validation_label="$4"
  local validation_services
  local -a validation_compose=(
    "${compose_command[@]}"
    --env-file "$validation_runtime"
    --env-file "$validation_images"
    -f "$validation_compose_file"
  )

  if grep -q '^QDRANT_IMAGE=' "$validation_images"; then
    echo "FAIL ${validation_label}-qdrant-ownership: application release state must not manage Qdrant" >&2
    return 1
  fi
  if ! validation_services="$("${validation_compose[@]}" config --services)"; then
    echo "FAIL ${validation_label}-compose-contract: application release state is invalid" >&2
    return 1
  fi
  if printf '%s\n' "$validation_services" | grep -qx qdrant; then
    echo "FAIL ${validation_label}-qdrant-ownership: application release state must not manage Qdrant" >&2
    return 1
  fi
  local expected_services
  expected_services="$(printf '%s\n' admin adminportal data miniapp)"
  if [ "$(printf '%s\n' "$validation_services" | LC_ALL=C sort -u)" != "$expected_services" ]; then
    echo "FAIL ${validation_label}-compose-contract: release must contain exactly data, miniapp, adminportal, and admin" >&2
    return 1
  fi
}

verify_services() {
  local verification_runtime="$1"
  shift
  local -a compose_command=("$@")
  local verification_admin_token
  verification_admin_token="$(runtime_value "$verification_runtime" ADMIN_SERVICE_TOKEN)"

  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T admin wget -qO- http://127.0.0.1:9014/healthz >/dev/null || return 1
  echo "PASS container-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9012/healthz" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9014/healthz" >/dev/null || return 1
  echo "PASS host-entry-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9012/api/miniapp/v1/reports/home" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${host_base_url}:9014/api/admin/v1/events?page=1&page_size=1" >/dev/null || return 1
  echo "PASS bff-to-service-read-paths"
}

rollback_current_release() {
  local rollback_runtime="$current_runtime"
  local rollback_images="$current_images"
  local rollback_compose_file="$current_compose"
  local rollback_sha="$current_sha"
  if [ "$rollback_snapshot_ready" = true ]; then
    rollback_runtime="$previous_runtime"
    rollback_images="$previous_images"
    rollback_compose_file="$previous_compose"
    rollback_sha="$previous_sha"
  elif [ "$expected_current_available" != true ]; then
    echo "FAIL rollback: no trusted previous repository-managed UAT release is available" >&2
    return 1
  fi
  if [ ! -s "$rollback_runtime" ] || [ ! -s "$rollback_images" ] || [ ! -s "$rollback_compose_file" ] || [ ! -s "$rollback_sha" ]; then
    echo "FAIL rollback: no previous repository-managed UAT release is available" >&2
    return 1
  fi
  echo "Candidate verification failed; restoring release $(sed -n '1p' "$rollback_sha")" >&2
  validate_application_only_release \
    "$rollback_runtime" "$rollback_images" "$rollback_compose_file" rollback || return 1
  local -a rollback_compose=("${compose_command[@]}" --env-file "$rollback_runtime" --env-file "$rollback_images" -f "$rollback_compose_file")
  if ! "${rollback_compose[@]}" up -d --wait --wait-timeout 120; then
    echo "FAIL rollback-start: previous application release did not start" >&2
    return 1
  fi
  if ! verify_services "$rollback_runtime" "${rollback_compose[@]}"; then
    echo "FAIL rollback-health: previous application release is not healthy" >&2
    return 1
  fi
  if [ -f "$release_state_write_marker" ]; then
    restore_interrupted_release_state
  fi
  echo "PASS rollback: previous complete release restored" >&2
}

recover_failed_deployment() {
  local exit_status=$?
  trap - EXIT
  if [ "$exit_status" -ne 0 ]; then
    if [ "$bounded_data_cutover" = true ]; then
      if [ "$cutover_migration_started" = true ]; then
        if ! "${candidate_compose[@]}" stop; then
          echo "FAIL ${cutover_gate_name}-cutover-recovery: candidate application services could not be stopped" >&2
        fi
        echo "FAIL ${cutover_gate_name}-cutover-recovery: database may be mutated; old application images remain stopped and marker is retained" >&2
      elif [ -e "$data2_cutover_marker" ]; then
        if rollback_current_release; then
          rm -f "$data2_cutover_marker"
          sync -f "$state_dir"
          echo "PASS ${cutover_gate_name}-cutover-pre-migration-rollback" >&2
        else
          echo "FAIL ${cutover_gate_name}-cutover-pre-migration-rollback: marker retained" >&2
        fi
      fi
      exit "$exit_status"
    fi
    if [ "$candidate_services_started" = true ]; then
      rollback_current_release || true
    fi
  fi
  exit "$exit_status"
}

validate_application_only_release "$runtime_env" "$candidate_images" "$compose_file" candidate
if [ "$expected_current_available" = true ]; then
  validate_application_only_release \
    "$current_runtime" "$current_images" "$current_compose" rollback
fi
"${candidate_compose[@]}" config --quiet
echo "PASS compose-contract"

# Check-only dbmigrate establishes a real TLS PostgreSQL connection and reports
# current/pending migration state without taking the migration lock or writing.
"${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate > "$report_file"
echo "PASS rds-tls-readonly"

migration_risk_summary="$(python3 - "$report_file" "$migration_risk_manifest" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
risk = {}
for line in pathlib.Path(sys.argv[2]).read_text().splitlines():
    if not line.strip() or line.lstrip().startswith("#"):
        continue
    fields = line.split("\t", 3)
    if len(fields) != 4:
        raise SystemExit(f"invalid migration manifest row: {line}")
    version, classification, scope, reason = fields
    if classification not in {"normal", "high", "blocked"}:
        raise SystemExit(f"invalid migration risk classification for {version}: {classification}")
    if scope not in {"schema", "data", "mixed"}:
        raise SystemExit(f"invalid migration scope for {version}: {scope}")
    if not reason.strip():
        raise SystemExit(f"migration reason is required for {version}")
    risk[version] = (classification, scope)
pending = report.get("pending") or []
versions = [str(item.get("Version", item.get("version", ""))).zfill(6) for item in pending]
unclassified = [version for version in versions if version not in risk]
if unclassified:
    raise SystemExit("pending migrations lack risk classification: " + ",".join(unclassified))
print(",".join(version for version in versions if risk[version][0] == "high"))
print(",".join(version for version in versions if risk[version][0] == "blocked"))
print(",".join(f"{version}:{risk[version][1]}" for version in versions if risk[version][1] != "schema"))
print(str(report.get("current_version") or "").zfill(6))
print(",".join(versions))
PY
)"
high_risk_pending="$(printf '%s\n' "$migration_risk_summary" | sed -n '1p')"
blocked_pending="$(printf '%s\n' "$migration_risk_summary" | sed -n '2p')"
non_schema_pending="$(printf '%s\n' "$migration_risk_summary" | sed -n '3p')"
data_current_version="$(printf '%s\n' "$migration_risk_summary" | sed -n '4p')"
data_pending_versions="$(printf '%s\n' "$migration_risk_summary" | sed -n '5p')"

database_identity="tidewise_uat@config.uat.yaml/tidewise_uat"

{
  echo "### UAT migration preflight"
  echo
  echo "- Release: \`${release_sha}\`"
  echo "- Database: \`${database_identity}\`"
  echo "- TLS database check: passed"
  echo "- High-risk pending migrations: \`${high_risk_pending:-none}\`"
  echo "- Release-blocked pending migrations: \`${blocked_pending:-none}\`"
  echo "- Non-schema pending migrations: \`${non_schema_pending:-none}\`"
  echo
  echo '<details><summary>Migration state before apply</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$report_file"
  echo '```'
  echo '</details>'
} >> "$summary_file"

if [ -n "$blocked_pending" ]; then
  echo "FAIL migration-release-gate: pending Data migration is not release-compatible: ${blocked_pending}" >&2
  exit 1
fi
echo "PASS migration-release-gate"

if [ "$bounded_data_cutover" = true ]; then
  if [ "$expected_current_available" != true ]; then
    echo "FAIL ${cutover_gate_name}-cutover-gate: a complete repository-managed current UAT release is required" >&2
    exit 1
  fi
  if [ "$destructive_data_change_confirmed" != true ]; then
    echo "FAIL ${cutover_gate_name}-cutover-gate: confirm_destructive_data_change=true is required" >&2
    exit 1
  fi
  if [ "$backup_confirmed" != true ]; then
    echo "FAIL ${cutover_gate_name}-cutover-gate: confirm_high_risk_backup=true is required" >&2
    exit 1
  fi
  if [ -e "$data2_cutover_marker" ]; then
    if [ "$(data2_cutover_marker_value release_sha)" != "$release_sha" ]; then
      echo "FAIL ${cutover_gate_name}-cutover-gate: recovery marker belongs to another release" >&2
      exit 1
    fi
    if [ "$(data2_cutover_marker_target_version)" != "$cutover_target_version" ]; then
      echo "FAIL ${cutover_gate_name}-cutover-gate: recovery marker belongs to another target version" >&2
      exit 1
    fi
    cutover_recovery_phase="$(data2_cutover_marker_value phase)"
    case "$cutover_recovery_phase" in
      prepared|services-stopped) ;;
      migration-started|data-migrated)
        cutover_migration_started=true
        ;;
      *)
        echo "FAIL ${cutover_gate_name}-cutover-gate: recovery marker has an invalid phase" >&2
        exit 1
        ;;
    esac
    recovery_minimum_version="$cutover_recovery_minimum_version"
    if [ "$empty_data_schema_rebuild_requested" = true ]; then
      recovery_minimum_version=0
    fi
    if ! python3 - "$data_current_version" "$data_pending_versions" "$recovery_minimum_version" "$cutover_target_version" <<'PY'
import sys

current = int(sys.argv[1])
pending = [int(version) for version in sys.argv[2].split(",") if version]
minimum = int(sys.argv[3])
target = int(sys.argv[4])
if current < minimum or current > target or pending != list(range(current + 1, target + 1)):
    raise SystemExit(1)
PY
    then
      echo "FAIL ${cutover_gate_name}-cutover-gate: recovery state is not a contiguous migration suffix ending at ${cutover_target_version}" >&2
      exit 1
    fi
    if [[ "$cutover_recovery_phase" =~ ^(prepared|services-stopped)$ ]] && \
      { [ "$data_current_version" != "$cutover_initial_current_version" ] || [ "$data_pending_versions" != "$cutover_initial_pending_versions" ]; }; then
      cutover_migration_started=true
      cutover_recovery_phase=migration-started
      trap recover_failed_deployment EXIT
      write_data2_cutover_marker migration-started
      echo "PASS ${cutover_gate_name}-cutover-recovery-phase-reconciled"
    fi
    echo "PASS ${cutover_gate_name}-cutover-recovery-gate"
  elif [ "$empty_data_schema_rebuild_requested" = true ]; then
    echo "FAIL ${cutover_gate_name}-cutover-gate: empty Data schema rebuild requires an existing cutover recovery marker" >&2
    exit 1
  elif [ "$data_current_version" != "$cutover_initial_current_version" ] || [ "$data_pending_versions" != "$cutover_initial_pending_versions" ]; then
    echo "FAIL ${cutover_gate_name}-cutover-gate: initial cutover requires Data migration ${cutover_initial_current_version#000} with the exact pending range ${cutover_initial_pending_versions}" >&2
    exit 1
  fi
  echo "PASS ${cutover_gate_name}-cutover-gate"
else
  if [ -e "$data2_cutover_marker" ]; then
    echo "FAIL data2-cutover-recovery: ordinary deployment is blocked while a cutover marker exists" >&2
    exit 1
  fi
  if [ -n "$non_schema_pending" ]; then
    echo "FAIL migration-scope-gate: UAT system deploy accepts schema-only Data migrations: ${non_schema_pending}" >&2
    exit 1
  fi
  echo "PASS migration-scope-gate"
fi

if [ -n "$high_risk_pending" ] && [ "$backup_confirmed" != true ]; then
  echo "FAIL migration-risk-gate: confirm_high_risk_backup=true is required for Data=${high_risk_pending}" >&2
  exit 1
fi
echo "PASS migration-risk-gate"

if [ "$committed_cutover_recovery" = true ]; then
  if [ "$data_current_version" != "$cutover_target_version_padded" ] || [ -n "$data_pending_versions" ]; then
    echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: Data must be at migration ${cutover_target_version} with no pending migrations" >&2
    exit 1
  fi
  if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
    if ! "${candidate_compose[@]}" stop; then
      echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: candidate application services could not be stopped" >&2
    fi
    echo "FAIL ${cutover_gate_name}-cutover-committed-recovery: committed candidate is not healthy; cutover marker retained" >&2
    exit 1
  fi
  rm -f "$data2_cutover_marker"
  sync -f "$state_dir"
  echo "PASS ${cutover_gate_name}-cutover-committed-recovery"
  exit 0
fi

if [ "$bounded_data_cutover" = true ]; then
  trap recover_failed_deployment EXIT
  if [ "$cutover_migration_started" != true ]; then
    write_data2_cutover_marker prepared
  fi
  current_release_compose=("${compose_command[@]}" --env-file "$current_runtime" --env-file "$current_images" -f "$current_compose")
  "${current_release_compose[@]}" stop
  for service in "${application_services[@]}"; do
    if ! application_container_ids="$(application_container_ids true "$service")"; then
      echo "FAIL application-write-stop: unable to inspect ${service} containers before enforced stop" >&2
      exit 1
    fi
    if [ -n "$application_container_ids" ]; then
      while IFS= read -r application_container_id; do
        if [ -n "$application_container_id" ] && ! docker stop "$application_container_id" >/dev/null; then
          echo "FAIL application-write-stop: unable to stop ${service} container ${application_container_id}" >&2
          exit 1
        fi
      done <<< "$application_container_ids"
    fi
  done
  if ! running_services="$("${current_release_compose[@]}" ps --status running --services)"; then
    echo "FAIL application-write-stop: unable to inspect current Compose services" >&2
    exit 1
  fi
  if [ -n "$running_services" ]; then
    echo "FAIL application-write-stop: one or more current UAT services are still running" >&2
    exit 1
  fi
  for service in "${application_services[@]}"; do
    if ! running_container_ids="$(application_container_ids false "$service")"; then
      echo "FAIL application-write-stop: unable to inspect ${service} containers" >&2
      exit 1
    fi
    if [ -n "$running_container_ids" ]; then
      echo "FAIL application-write-stop: ${service} still has a running tidewise-uat container" >&2
      exit 1
    fi
  done
  if [ "$cutover_migration_started" != true ]; then
    write_data2_cutover_marker services-stopped
  fi
  echo "PASS application-write-stop"
  if [ "$cutover_migration_started" != true ]; then
    cutover_migration_started=true
    write_data2_cutover_marker migration-started
  fi
  if [ "$empty_data_schema_rebuild_requested" = true ]; then
    "${candidate_compose[@]}" run --rm --no-deps \
      -e TIDEWISE_EMPTY_DATA_SCHEMA_REBUILD_CONFIRMED=issue-266-data-only \
      -e "PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified -c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified -c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
      data /usr/local/bin/dbmigrate -apply -target-version "$cutover_target_version" -rebuild-empty-schema > "$data2_apply_report_file"
  else
    "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply -target-version "$cutover_target_version" > "$data2_apply_report_file"
  fi
  "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate > "$report_file"
  if ! python3 - "$report_file" "$cutover_target_version_padded" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
current = str(report.get("current_version") or "").zfill(6)
if current != sys.argv[2] or report.get("pending") or report.get("remaining"):
    raise SystemExit(1)
PY
  then
    echo "FAIL ${cutover_gate_name}-target-version: Data did not reach migration ${cutover_target_version} with no pending migrations" >&2
    exit 1
  fi
  write_data2_cutover_marker data-migrated
  echo "PASS ${cutover_gate_name}-target-version"
else
  "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply > "$report_file"
fi
{
  echo
  echo '<details><summary>Migration apply result</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$report_file"
  echo '```'
  echo '</details>'
} >> "$summary_file"
echo "PASS migration-apply"

if ! "${candidate_compose[@]}" up -d --remove-orphans --wait --wait-timeout 120; then
  if [ "$bounded_data_cutover" != true ] && [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
  if [ "$bounded_data_cutover" != true ] && [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
candidate_services_started=true
trap recover_failed_deployment EXIT

if [ "$bounded_data_cutover" = true ]; then
  install -m 0600 "$current_runtime" "$cutover_checkpoint_runtime"
  install -m 0640 "$current_images" "$cutover_checkpoint_images"
  install -m 0640 "$current_compose" "$cutover_checkpoint_compose"
  install -m 0640 "$current_sha" "$cutover_checkpoint_sha"
  rm -f "$previous_runtime" "$previous_images" "$previous_compose" "$previous_sha"
elif [ "$expected_current_available" = true ]; then
  install -m 0600 "$current_runtime" "$previous_runtime"
  install -m 0640 "$current_images" "$previous_images"
  install -m 0640 "$current_compose" "$previous_compose"
  install -m 0640 "$current_sha" "$previous_sha"
  rollback_snapshot_ready=true
else
  # A missing or invalid prior application release cannot be a rollback target.
  # Purge all old snapshots so the first successful four-service release becomes
  # the only baseline and no retired runtime values survive in persisted state.
  rm -f \
    "$previous_runtime" "$previous_images" "$previous_compose" "$previous_sha" \
    "$pre_data2_runtime" "$pre_data2_images" "$pre_data2_compose" "$pre_data2_sha" \
    "$pre_data59_runtime" "$pre_data59_images" "$pre_data59_compose" "$pre_data59_sha" \
    "$pre_data60_runtime" "$pre_data60_images" "$pre_data60_compose" "$pre_data60_sha" \
    "$pre_data78_runtime" "$pre_data78_images" "$pre_data78_compose" "$pre_data78_sha" \
    "$pre_data80_runtime" "$pre_data80_images" "$pre_data80_compose" "$pre_data80_sha" \
    "$agentrun_rollback_marker" "$agentrun_version_publication"
fi
if [ "$bounded_data_cutover" = true ]; then
  write_release_state_marker "$cutover_release_state_mode"
elif [ "$rollback_snapshot_ready" = true ]; then
  write_release_state_marker previous
else
  write_release_state_marker none
fi
install -m 0600 "$runtime_env" "$current_runtime"
install -m 0640 "$candidate_images" "$current_images"
install -m 0640 "$compose_file" "$current_compose"
printf '%s\n' "$release_sha" > "$current_sha"
chmod 0640 "$current_sha"
sync "$current_runtime" "$current_images" "$current_compose" "$current_sha"
sync -f "$deployment_root"
write_release_state_marker committed
sync -f "$state_dir"
rm -f "$release_state_write_marker"
sync -f "$state_dir"
if [ "$bounded_data_cutover" = true ]; then
  rm -f "$data2_cutover_marker"
  sync -f "$state_dir"
fi
trap - EXIT
echo "PASS release-state-recorded"

{
  echo
  echo "### UAT deployment"
  echo
  echo "Deployed \`${release_sha}\` with a complete four-service immutable image state; Qdrant remains independently operated."
  if [ -s "$previous_sha" ]; then
    echo "Previous successful release: \`$(sed -n '1p' "$previous_sha")\`."
  fi
} >> "$summary_file"
