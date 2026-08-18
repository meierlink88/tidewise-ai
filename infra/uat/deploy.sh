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
expected_current_agentrun_image="${EXPECTED_CURRENT_AGENTRUN_IMAGE:-}"
backup_confirmed="${HIGH_RISK_BACKUP_CONFIRMED:-false}"
deployment_mode="${DEPLOYMENT_MODE:-normal}"
destructive_data_change_confirmed="${DESTRUCTIVE_DATA_CHANGE_CONFIRMED:-false}"
empty_data_schema_rebuild_requested="${EMPTY_DATA_SCHEMA_REBUILD_REQUESTED:-false}"
compose_file="${COMPOSE_FILE:-infra/uat/docker-compose.yaml}"
migration_risk_manifest="${MIGRATION_RISK_MANIFEST:-infra/uat/migration-risk.tsv}"
agentrun_migration_risk_manifest="${AGENTRUN_MIGRATION_RISK_MANIFEST:-infra/uat/agentrun-migration-risk.tsv}"
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
agentrun_report_file="${RUNNER_TEMP:-/tmp}/tidewise-uat-agentrun-migration-${GITHUB_RUN_ID:-manual}.json"
agentrun_rollback_compatibility_required=false
agentrun_rollback_target_version=""
# Keep the established state filename so an interrupted migration rollback remains recoverable.
agentrun_rollback_marker="${state_dir}/agentrun-010-rollback-required"
agent_version_publication_marker="${state_dir}/agentrun-agent-version-publication.json"
release_state_write_marker="${state_dir}/release-state-write-in-progress"
data2_cutover_marker="${state_dir}/tidewise-2-cutover-in-progress"
pre_data2_runtime="${deployment_root}/pre-data2.runtime.env"
pre_data2_images="${state_dir}/pre-data2.images.env"
pre_data2_compose="${state_dir}/pre-data2.compose.yaml"
pre_data2_sha="${state_dir}/pre-data2.sha"
candidate_services_started=false
rollback_snapshot_ready=false
cutover_migration_started=false
interrupted_state_recovery_mode=""
committed_cutover_recovery=false
host_base_url="${UAT_HOST_BASE_URL:-http://127.0.0.1}"
event_semantic_qdrant_url="http://qdrant:6333"

test -d "$state_dir"
test -w "$state_dir"

case "$deployment_mode" in
  normal|tidewise_2_cutover) ;;
  *)
    echo "FAIL deployment-mode-gate: DEPLOYMENT_MODE must be normal or tidewise_2_cutover" >&2
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
      rm -f "$agentrun_rollback_marker"
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
  if [ "$deployment_mode" = tidewise_2_cutover ] && [[ "$interrupted_state_recovery_mode" =~ ^(pre-data2|committed)$ ]]; then
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
    [ "$(current_image_value ADMIN_IMAGE)" != "$expected_current_admin_image" ] || \
    [ "$(current_image_value AGENTRUN_IMAGE)" != "$expected_current_agentrun_image" ]; then
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
if [ "$deployment_mode" = tidewise_2_cutover ] && [ "$interrupted_state_recovery_mode" = committed ]; then
  if [ ! -s "$data2_cutover_marker" ] || \
    [ "$(data2_cutover_marker_value release_sha)" != "$release_sha" ] || \
    [ "$(data2_cutover_marker_value phase)" != data-migrated ]; then
    echo "FAIL data2-cutover-committed-recovery: matching data-migrated cutover marker is required" >&2
    exit 1
  fi
  if [ "$(sed -n '1p' "$current_sha")" != "$release_sha" ]; then
    echo "FAIL data2-cutover-committed-recovery: committed release SHA does not match the requested release" >&2
    exit 1
  fi
  for image_key in DATA_IMAGE MINIAPP_IMAGE ADMINPORTAL_IMAGE ADMIN_IMAGE AGENTRUN_IMAGE; do
    if [[ "$(current_image_value "$image_key")" != *":${release_sha}" ]]; then
      echo "FAIL data2-cutover-committed-recovery: ${image_key} is not the committed cutover image" >&2
      exit 1
    fi
  done
  committed_cutover_recovery=true
fi

# Process environment variables have higher precedence than Compose --env-file.
# The workflow exposes candidate image names at job scope, so clear them before
# every Compose invocation and let the selected release image file be authoritative.
compose_command=(env -u DATA_IMAGE -u MINIAPP_IMAGE -u ADMINPORTAL_IMAGE -u ADMIN_IMAGE -u AGENTRUN_IMAGE docker compose)
candidate_compose=("${compose_command[@]}" --env-file "$runtime_env" --env-file "$candidate_images" -f "$compose_file")

runtime_value() {
  local file="$1"
  local key="$2"
  sed -n "s/^${key}=//p" "$file" | tail -n 1
}

verify_external_qdrant() {
  local -a verification_compose=("$@")
  "${verification_compose[@]}" run --rm --no-deps --entrypoint /bin/sh data \
    -ec "wget -q -T 10 -t 2 -O- ${event_semantic_qdrant_url}/collections >/dev/null"
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
  expected_services="$(printf '%s\n' admin adminportal agentrun data miniapp)"
  if [ "$(printf '%s\n' "$validation_services" | LC_ALL=C sort -u)" != "$expected_services" ]; then
    echo "FAIL ${validation_label}-compose-contract: release must contain exactly data, agentrun, miniapp, adminportal, and admin" >&2
    return 1
  fi
}

verify_services() {
  local verification_runtime="$1"
  shift
  local -a compose_command=("$@")
  local verification_admin_token
  verification_admin_token="$(runtime_value "$verification_runtime" ADMIN_SERVICE_TOKEN)"

  verify_external_qdrant "${compose_command[@]}" || return 1
  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T agentrun wget -qO- http://127.0.0.1:9080/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/healthz >/dev/null || return 1
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/readyz >/dev/null || return 1
  "${compose_command[@]}" exec -T admin wget -qO- http://127.0.0.1:9014/healthz >/dev/null || return 1
  echo "PASS container-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9012/healthz" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9014/healthz" >/dev/null || return 1
  echo "PASS host-entry-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9012/api/miniapp/v1/research/themes?limit=1" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${host_base_url}:9014/api/admin/v1/events?page=1&page_size=1" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${host_base_url}:9014/api/admin/v1/model-providers" >/dev/null || return 1
  echo "PASS bff-to-service-read-paths"
}

prepare_previous_release_agentrun_rollback() {
  local previous_release_version
  previous_release_version="$(sed -n '1p' "$agentrun_rollback_marker")"
  if ! [[ "$previous_release_version" =~ ^01[0-4]$ ]]; then
    echo "FAIL agentrun-previous-release-database-compatibility: invalid rollback target marker" >&2
    return 1
  fi
  "${candidate_compose[@]}" run --rm --no-deps \
    --entrypoint /app/agentrun-migrate agentrun \
    --prepare-previous-release-rollback \
    --previous-release-version "$previous_release_version"
}

withdraw_candidate_agent_versions() {
  if [ ! -e "$agent_version_publication_marker" ]; then
    return 0
  fi
  if [ ! -s "$agent_version_publication_marker" ]; then
    echo "FAIL agent-version-withdrawal: publication record is empty" >&2
    return 1
  fi
  if ! "${candidate_compose[@]}" run --rm -T --no-deps \
    --entrypoint /app/agentrun-agent-version agentrun withdraw-publication \
    < "$agent_version_publication_marker"; then
    echo "FAIL agent-version-withdrawal: candidate version is already in use; restore the pre-cutover database snapshot" >&2
    return 1
  fi
  rm -f "$agent_version_publication_marker"
  sync -f "$state_dir"
  echo "PASS agent-version-withdrawal" >&2
}

rollback_current_release() {
  if ! withdraw_candidate_agent_versions; then
    return 1
  fi
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
  if [ "$agentrun_rollback_compatibility_required" = true ] || [ -f "$agentrun_rollback_marker" ]; then
    if ! prepare_previous_release_agentrun_rollback; then
      echo "FAIL agentrun-previous-release-database-compatibility: marker retained" >&2
      return 1
    fi
    rm -f "$agentrun_rollback_marker"
    agentrun_rollback_compatibility_required=false
    echo "PASS agentrun-previous-release-database-compatibility" >&2
  fi
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
    if [ "$deployment_mode" = tidewise_2_cutover ]; then
      if [ -e "$agent_version_publication_marker" ]; then
        withdraw_candidate_agent_versions || true
      fi
      if [ "$cutover_migration_started" = true ]; then
        if ! "${candidate_compose[@]}" stop; then
          echo "FAIL data2-cutover-recovery: candidate application services could not be stopped" >&2
        fi
        echo "FAIL data2-cutover-recovery: database may be mutated; old application images remain stopped and marker is retained" >&2
      elif [ -e "$data2_cutover_marker" ]; then
        if rollback_current_release; then
          rm -f "$data2_cutover_marker"
          sync -f "$state_dir"
          echo "PASS data2-cutover-pre-migration-rollback" >&2
        else
          echo "FAIL data2-cutover-pre-migration-rollback: marker retained" >&2
        fi
      fi
      exit "$exit_status"
    fi
    if [ "$candidate_services_started" = true ]; then
      rollback_current_release || true
    else
      if [ -e "$agent_version_publication_marker" ]; then
        withdraw_candidate_agent_versions || true
      fi
      if [ -f "$agentrun_rollback_marker" ] && prepare_previous_release_agentrun_rollback; then
        rm -f "$agentrun_rollback_marker"
      elif [ -f "$agentrun_rollback_marker" ]; then
        echo "FAIL interrupted AgentRun migration cleanup: marker retained" >&2
      fi
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

if [ -e "$agent_version_publication_marker" ] && [ "$committed_cutover_recovery" != true ]; then
  withdraw_candidate_agent_versions
  echo "PASS recovered-interrupted-agent-version-publication"
fi

verify_external_qdrant "${candidate_compose[@]}"
echo "PASS external-qdrant-ready"

# The host runner owning the bind-mount directory is not enough: the
# unprivileged AgentRun image user must be able to create durable Artifacts.
"${candidate_compose[@]}" run --rm --no-deps --entrypoint /bin/sh agentrun \
  -c 'probe="$(mktemp /app/data/.uat-write-probe.XXXXXX)" && rm -f "$probe"'
echo "PASS agentrun-artifact-write"

if [ -f "$agentrun_rollback_marker" ]; then
  prepare_previous_release_agentrun_rollback
  rm -f "$agentrun_rollback_marker"
  echo "PASS recovered-interrupted-agentrun-migration"
fi

# Check-only dbmigrate establishes a real TLS PostgreSQL connection and reports
# current/pending migration state without taking the migration lock or writing.
"${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate > "$report_file"
echo "PASS rds-tls-readonly"
"${candidate_compose[@]}" run --rm --no-deps --entrypoint /app/agentrun-migrate agentrun --check-only > "$agentrun_report_file"
echo "PASS agentrun-rds-tls-readonly"

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

agentrun_migration_risk_summary="$(python3 - "$agentrun_report_file" "$agentrun_migration_risk_manifest" <<'PY'
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
        raise SystemExit(f"invalid AgentRun migration manifest row: {line}")
    version, classification, scope, reason = fields
    if classification not in {"normal", "high", "blocked"}:
        raise SystemExit(f"invalid AgentRun migration risk classification for {version}: {classification}")
    if scope not in {"schema", "data", "mixed"}:
        raise SystemExit(f"invalid AgentRun migration scope for {version}: {scope}")
    if not reason.strip():
        raise SystemExit(f"AgentRun migration reason is required for {version}")
    risk[version] = (classification, scope)
pending = report.get("pending") or []
versions = [str(item.get("version", item.get("Version", ""))).zfill(3) for item in pending]
unclassified = [version for version in versions if version not in risk]
if unclassified:
    raise SystemExit("pending AgentRun migrations lack risk classification: " + ",".join(unclassified))
print(",".join(version for version in versions if risk[version][0] == "high"))
print(",".join(version for version in versions if risk[version][0] == "blocked"))
print(",".join(f"{version}:{risk[version][1]}" for version in versions if risk[version][1] != "schema"))
rollback_versions = {"011", "012", "013", "014", "015"}
print("true" if rollback_versions.intersection(versions) else "false")
print(str(report.get("current_version") or "").zfill(3))
print(",".join(versions))
PY
)"
agentrun_high_risk_pending="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '1p')"
agentrun_blocked_pending="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '2p')"
agentrun_non_schema_pending="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '3p')"
agentrun_rollback_compatibility_required="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '4p')"
agentrun_rollback_target_version="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '5p')"
agentrun_pending_versions="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '6p')"

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
  echo "- AgentRun high-risk pending migrations: \`${agentrun_high_risk_pending:-none}\`"
  echo "- AgentRun release-blocked pending migrations: \`${agentrun_blocked_pending:-none}\`"
  echo "- AgentRun non-schema pending migrations: \`${agentrun_non_schema_pending:-none}\`"
  echo
  echo '<details><summary>Migration state before apply</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$report_file"
  echo '```'
  echo '</details>'
  echo
  echo '<details><summary>AgentRun migration state before apply</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$agentrun_report_file"
  echo '```'
  echo '</details>'
} >> "$summary_file"

if [ -n "$blocked_pending" ] || [ -n "$agentrun_blocked_pending" ]; then
  echo "FAIL migration-release-gate: pending migration is not release-compatible: data=${blocked_pending:-none} agentrun=${agentrun_blocked_pending:-none}" >&2
  exit 1
fi
echo "PASS migration-release-gate"

if [ "$deployment_mode" = tidewise_2_cutover ]; then
  if [ "$expected_current_available" != true ]; then
    echo "FAIL data2-cutover-gate: a complete repository-managed current UAT release is required" >&2
    exit 1
  fi
  if [ "$destructive_data_change_confirmed" != true ]; then
    echo "FAIL data2-cutover-gate: confirm_destructive_data_change=true is required" >&2
    exit 1
  fi
  if [ "$backup_confirmed" != true ]; then
    echo "FAIL data2-cutover-gate: confirm_high_risk_backup=true is required" >&2
    exit 1
  fi
  if [ "$agentrun_rollback_target_version" != 015 ] || [ -n "$agentrun_pending_versions" ]; then
    echo "FAIL data2-cutover-gate: AgentRun must already be at migration 015 with no pending migrations" >&2
    exit 1
  fi
  if [ -e "$data2_cutover_marker" ]; then
    if [ "$(data2_cutover_marker_value release_sha)" != "$release_sha" ]; then
      echo "FAIL data2-cutover-gate: recovery marker belongs to another release" >&2
      exit 1
    fi
    if ! python3 - "$data_current_version" "$data_pending_versions" "$empty_data_schema_rebuild_requested" <<'PY'
import sys

current = int(sys.argv[1])
pending = [int(version) for version in sys.argv[2].split(",") if version]
minimum = 0 if sys.argv[3] == "true" else 44
if current < minimum or current > 58 or pending != list(range(current + 1, 59)):
    raise SystemExit(1)
PY
    then
      echo "FAIL data2-cutover-gate: recovery state is not a contiguous migration suffix ending at 58" >&2
      exit 1
    fi
    echo "PASS data2-cutover-recovery-gate"
  elif [ "$empty_data_schema_rebuild_requested" = true ]; then
    echo "FAIL data2-cutover-gate: empty Data schema rebuild requires an existing cutover recovery marker" >&2
    exit 1
  elif [ "$data_current_version" != 000044 ] || [ "$data_pending_versions" != 000045,000046,000047,000048,000049,000050,000051,000052,000053,000054,000055,000056,000057,000058 ]; then
    echo "FAIL data2-cutover-gate: initial cutover requires Data migration 44 with the exact pending range 45-58" >&2
    exit 1
  fi
  echo "PASS data2-cutover-gate"
else
  if [ -e "$data2_cutover_marker" ]; then
    echo "FAIL data2-cutover-recovery: ordinary deployment is blocked while a cutover marker exists" >&2
    exit 1
  fi
  if [ -n "$non_schema_pending" ] || [ -n "$agentrun_non_schema_pending" ]; then
    echo "FAIL migration-scope-gate: UAT system deploy accepts schema-only migrations: data=${non_schema_pending:-none} agentrun=${agentrun_non_schema_pending:-none}" >&2
    exit 1
  fi
  echo "PASS migration-scope-gate"
fi

if { [ -n "$high_risk_pending" ] || [ -n "$agentrun_high_risk_pending" ]; } && [ "$backup_confirmed" != true ]; then
  echo "FAIL migration-risk-gate: confirm_high_risk_backup=true is required for data=${high_risk_pending:-none} agentrun=${agentrun_high_risk_pending:-none}" >&2
  exit 1
fi
echo "PASS migration-risk-gate"

if [ "$agentrun_rollback_compatibility_required" = true ] && ! [[ "$agentrun_rollback_target_version" =~ ^01[0-4]$ ]]; then
  echo "FAIL agentrun-rollback-target-gate: unsupported previous migration version ${agentrun_rollback_target_version:-none}" >&2
  exit 1
fi

if [ "$committed_cutover_recovery" = true ]; then
  if [ "$data_current_version" != 000058 ] || [ -n "$data_pending_versions" ]; then
    echo "FAIL data2-cutover-committed-recovery: Data must be at migration 58 with no pending migrations" >&2
    exit 1
  fi
  if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
    if ! "${candidate_compose[@]}" stop; then
      echo "FAIL data2-cutover-committed-recovery: candidate application services could not be stopped" >&2
    fi
    echo "FAIL data2-cutover-committed-recovery: committed candidate is not healthy; cutover marker retained" >&2
    exit 1
  fi
  rm -f "$agentrun_rollback_marker" "$agent_version_publication_marker" "$data2_cutover_marker"
  sync -f "$state_dir"
  echo "PASS data2-cutover-committed-recovery"
  exit 0
fi

if [ "$deployment_mode" = tidewise_2_cutover ]; then
  trap recover_failed_deployment EXIT
  write_data2_cutover_marker prepared
  current_release_compose=("${compose_command[@]}" --env-file "$current_runtime" --env-file "$current_images" -f "$current_compose")
  "${current_release_compose[@]}" stop
  for service in data agentrun miniapp adminportal admin; do
    if ! running_container_ids="$(docker ps \
      --filter label=com.docker.compose.project=tidewise-uat \
      --filter "label=com.docker.compose.service=${service}" \
      --filter status=running \
      --quiet)"; then
      echo "FAIL application-write-stop: unable to inspect ${service} containers before enforced stop" >&2
      exit 1
    fi
    if [ -n "$running_container_ids" ]; then
      while IFS= read -r running_container_id; do
        if [ -n "$running_container_id" ] && ! docker stop "$running_container_id" >/dev/null; then
          echo "FAIL application-write-stop: unable to stop ${service} container ${running_container_id}" >&2
          exit 1
        fi
      done <<< "$running_container_ids"
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
  for service in data agentrun miniapp adminportal admin; do
    if ! running_container_ids="$(docker ps \
      --filter label=com.docker.compose.project=tidewise-uat \
      --filter "label=com.docker.compose.service=${service}" \
      --filter status=running \
      --quiet)"; then
      echo "FAIL application-write-stop: unable to inspect ${service} containers" >&2
      exit 1
    fi
    if [ -n "$running_container_ids" ]; then
      echo "FAIL application-write-stop: ${service} still has a running tidewise-uat container" >&2
      exit 1
    fi
  done
  write_data2_cutover_marker services-stopped
  echo "PASS application-write-stop"
  cutover_migration_started=true
  write_data2_cutover_marker migration-started
  if [ "$empty_data_schema_rebuild_requested" = true ]; then
    "${candidate_compose[@]}" run --rm --no-deps \
      -e TIDEWISE_EMPTY_DATA_SCHEMA_REBUILD_CONFIRMED=issue-266-data-only \
      -e "PGOPTIONS=-c tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified -c tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified -c tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified" \
      data /usr/local/bin/dbmigrate -apply -target-version 58 -rebuild-empty-schema > "$data2_apply_report_file"
  else
    "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply -target-version 58 > "$data2_apply_report_file"
  fi
  "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate > "$report_file"
  if ! python3 - "$report_file" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
current = str(report.get("current_version") or "").zfill(6)
if current != "000058" or report.get("pending") or report.get("remaining"):
    raise SystemExit(1)
PY
  then
    echo "FAIL data2-target-version: Data did not reach migration 58 with no pending migrations" >&2
    exit 1
  fi
  write_data2_cutover_marker data-migrated
  echo "PASS data2-target-version"
else
  "${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply > "$report_file"
fi
if [ "$agentrun_rollback_compatibility_required" = true ]; then
  printf '%s\n' "$agentrun_rollback_target_version" > "$agentrun_rollback_marker"
  chmod 0640 "$agentrun_rollback_marker"
  sync "$agentrun_rollback_marker"
  sync -f "$state_dir"
  trap recover_failed_deployment EXIT
fi
"${candidate_compose[@]}" run --rm --no-deps --entrypoint /app/agentrun-migrate agentrun > "$agentrun_report_file"
agent_version_publication_temp="$(mktemp "${state_dir}/agentrun-agent-version-publication.XXXXXX")"
if ! "${candidate_compose[@]}" run --rm --no-deps \
  --entrypoint /app/agentrun-agent-version agentrun publish-current \
  > "$agent_version_publication_temp"; then
  rm -f "$agent_version_publication_temp"
  exit 1
fi
chmod 0640 "$agent_version_publication_temp"
sync "$agent_version_publication_temp"
mv -f "$agent_version_publication_temp" "$agent_version_publication_marker"
sync -f "$state_dir"
trap recover_failed_deployment EXIT
{
  echo
  echo '<details><summary>Migration apply result</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$report_file"
  echo '```'
  echo '</details>'
  echo
  echo '<details><summary>AgentRun migration apply result</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$agentrun_report_file"
  echo '```'
  echo '</details>'
} >> "$summary_file"
echo "PASS migration-apply"
echo "PASS agent-version-publication"

if ! "${candidate_compose[@]}" up -d --wait --wait-timeout 120; then
  if [ "$deployment_mode" != tidewise_2_cutover ] && [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
  if [ "$deployment_mode" != tidewise_2_cutover ] && [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
candidate_services_started=true
trap recover_failed_deployment EXIT

if [ "$deployment_mode" = tidewise_2_cutover ]; then
  install -m 0600 "$current_runtime" "$pre_data2_runtime"
  install -m 0640 "$current_images" "$pre_data2_images"
  install -m 0640 "$current_compose" "$pre_data2_compose"
  install -m 0640 "$current_sha" "$pre_data2_sha"
  rm -f "$previous_runtime" "$previous_images" "$previous_compose" "$previous_sha"
elif [ "$expected_current_available" = true ]; then
  install -m 0600 "$current_runtime" "$previous_runtime"
  install -m 0640 "$current_images" "$previous_images"
  install -m 0640 "$current_compose" "$previous_compose"
  install -m 0640 "$current_sha" "$previous_sha"
  rollback_snapshot_ready=true
fi
if [ "$deployment_mode" = tidewise_2_cutover ]; then
  write_release_state_marker pre-data2
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
rm -f "$agentrun_rollback_marker"
rm -f "$agent_version_publication_marker"
sync -f "$state_dir"
agentrun_rollback_compatibility_required=false
rm -f "$release_state_write_marker"
sync -f "$state_dir"
if [ "$deployment_mode" = tidewise_2_cutover ]; then
  rm -f "$data2_cutover_marker"
  sync -f "$state_dir"
fi
trap - EXIT
echo "PASS release-state-recorded"

{
  echo
  echo "### UAT deployment"
  echo
  echo "Deployed \`${release_sha}\` with a complete five-service immutable image state; Qdrant remains independently operated."
  if [ -s "$previous_sha" ]; then
    echo "Previous successful release: \`$(sed -n '1p' "$previous_sha")\`."
  fi
} >> "$summary_file"
