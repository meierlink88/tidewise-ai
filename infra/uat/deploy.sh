#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
candidate_images="${CANDIDATE_IMAGES:?CANDIDATE_IMAGES is required}"
release_sha="${COMMIT_SHA:?COMMIT_SHA is required}"
backup_confirmed="${HIGH_RISK_BACKUP_CONFIRMED:-false}"
industry_import_enabled="${INDUSTRY_RELATIONSHIP_IMPORT_ENABLED:-false}"
industry_package_sha="${INDUSTRY_RELATIONSHIP_PACKAGE_SHA:-}"
industry_graph_projection_enabled="${INDUSTRY_GRAPH_PROJECTION_ENABLED:-false}"
industry_graph_package_sha="${INDUSTRY_GRAPH_PACKAGE_SHA:-}"
event_semantic_projection_enabled="${EVENT_SEMANTIC_PROJECTION_ENABLED:-false}"
agentrun_recovery_target_version="${AGENTRUN_RECOVERY_TARGET_VERSION:-}"
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
agentrun_report_file="${RUNNER_TEMP:-/tmp}/tidewise-uat-agentrun-migration-${GITHUB_RUN_ID:-manual}.json"
agentrun_rollback_compatibility_required=false
agentrun_rollback_target_version=""
# Retain the legacy filename so an interrupted zero-byte 010-era marker can be
# recovered by the explicit operator-supplied target instead of being orphaned.
agentrun_rollback_marker="${state_dir}/agentrun-010-rollback-required"
release_state_write_marker="${state_dir}/release-state-write-in-progress"
candidate_services_started=false
rollback_snapshot_ready=false
industry_import_dry_run_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-relationships-dry-run-${GITHUB_RUN_ID:-manual}.json"
industry_import_apply_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-relationships-apply-${GITHUB_RUN_ID:-manual}.json"
industry_import_replay_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-relationships-replay-${GITHUB_RUN_ID:-manual}.json"
industry_graph_dry_run_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-graph-dry-run-${GITHUB_RUN_ID:-manual}.json"
industry_graph_apply_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-graph-apply-${GITHUB_RUN_ID:-manual}.json"
industry_graph_replay_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-industry-graph-replay-${GITHUB_RUN_ID:-manual}.json"
event_semantic_projection_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-event-semantic-projection-${GITHUB_RUN_ID:-manual}.json"
event_semantic_entity_collection_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-event-semantic-entity-collection-${GITHUB_RUN_ID:-manual}.json"
event_semantic_variable_collection_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-event-semantic-variable-collection-${GITHUB_RUN_ID:-manual}.json"
excluded_fact_before_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-excluded-facts-before-${GITHUB_RUN_ID:-manual}.json"
excluded_fact_after_report="${RUNNER_TEMP:-/tmp}/tidewise-uat-excluded-facts-after-${GITHUB_RUN_ID:-manual}.json"
host_base_url="${UAT_HOST_BASE_URL:-http://127.0.0.1}"
industry_package_path="/app/data/industry_relationships/2026-07-27-v1"
event_semantic_qdrant_url="http://qdrant:6333"
event_semantic_embedding_base_url="https://dashscope.aliyuncs.com/compatible-mode/v1"

if [ -n "$agentrun_recovery_target_version" ]; then
  if ! [[ "$agentrun_recovery_target_version" =~ ^01[0-3]$ ]]; then
    echo "FAIL agentrun-recovery-gate: recovery target must be 010 through 013" >&2
    exit 1
  fi
  if [ "$backup_confirmed" != true ]; then
    echo "FAIL agentrun-recovery-gate: confirm_high_risk_backup=true is required" >&2
    exit 1
  fi
fi

if [ "$industry_import_enabled" != true ] && [ "$industry_import_enabled" != false ]; then
  echo "FAIL industry-relationship-import-gate: enable flag must be true or false" >&2
  exit 1
fi
if [ "$industry_import_enabled" = true ]; then
  if ! [[ "$industry_package_sha" =~ ^[0-9a-f]{64}$ ]]; then
    echo "FAIL industry-relationship-import-gate: a 64-character lowercase package SHA is required" >&2
    exit 1
  fi
  if [ "$backup_confirmed" != true ]; then
    echo "FAIL industry-relationship-import-gate: confirm_high_risk_backup=true is required" >&2
    exit 1
  fi
elif [ -n "$industry_package_sha" ]; then
  echo "FAIL industry-relationship-import-gate: package SHA was supplied while import is disabled" >&2
  exit 1
fi
if [ "$industry_graph_projection_enabled" != true ] && [ "$industry_graph_projection_enabled" != false ]; then
  echo "FAIL industry-graph-projection-gate: enable flag must be true or false" >&2
  exit 1
fi
if [ "$industry_graph_projection_enabled" = true ]; then
  if ! [[ "$industry_graph_package_sha" =~ ^[0-9a-f]{64}$ ]]; then
    echo "FAIL industry-graph-projection-gate: a 64-character lowercase package SHA is required" >&2
    exit 1
  fi
  for required_name in NEO4J_URI NEO4J_USERNAME NEO4J_PASSWORD NEO4J_DATABASE; do
    if [ -z "${!required_name:-}" ]; then
      echo "FAIL industry-graph-projection-gate: ${required_name} is required" >&2
      exit 1
    fi
  done
elif [ -n "$industry_graph_package_sha" ]; then
  echo "FAIL industry-graph-projection-gate: package SHA was supplied while projection is disabled" >&2
  exit 1
fi
if [ "$event_semantic_projection_enabled" != true ] && [ "$event_semantic_projection_enabled" != false ]; then
  echo "FAIL event-semantic-projection-gate: enable flag must be true or false" >&2
  exit 1
fi
if [ "$event_semantic_projection_enabled" = true ] && [ -z "${EMBEDDING_API_KEY:-}" ]; then
  echo "FAIL event-semantic-projection-gate: EMBEDDING_API_KEY is required" >&2
  exit 1
fi

test -d "$state_dir"
test -w "$state_dir"

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

exec 9>"${deployment_root}/deploy.lock"
if ! flock -n 9; then
  echo "FAIL deployment-lock: another UAT deployment holds ${deployment_root}/deploy.lock" >&2
  exit 1
fi
echo "PASS deployment-lock"
if [ -f "$release_state_write_marker" ]; then
  restore_interrupted_release_state
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
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9013/healthz" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9014/healthz" >/dev/null || return 1
  echo "PASS host-entry-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${host_base_url}:9012/api/miniapp/v1/research/themes?limit=1" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${host_base_url}:9013/api/admin/v1/events?page=1&page_size=1" >/dev/null || return 1
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${host_base_url}:9013/api/admin/v1/model-providers" >/dev/null || return 1
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

cleanup_unfinished_agentrun_migration() {
  local exit_status=$?
  trap - EXIT
  if [ "$exit_status" -ne 0 ]; then
    if [ "$candidate_services_started" = true ]; then
      rollback_current_release || true
    elif [ -f "$agentrun_rollback_marker" ] && prepare_previous_release_agentrun_rollback; then
      rm -f "$agentrun_rollback_marker"
    elif [ -f "$agentrun_rollback_marker" ]; then
      echo "FAIL interrupted AgentRun migration cleanup: marker retained" >&2
    fi
  fi
  exit "$exit_status"
}

validate_application_only_release "$runtime_env" "$candidate_images" "$compose_file" candidate
if [ -s "$current_runtime" ] && [ -s "$current_images" ] && [ -s "$current_compose" ] && [ -s "$current_sha" ]; then
  validate_application_only_release \
    "$current_runtime" "$current_images" "$current_compose" rollback
fi
"${candidate_compose[@]}" config --quiet
echo "PASS compose-contract"

if [ -n "$agentrun_recovery_target_version" ]; then
  if [ -s "$agentrun_rollback_marker" ] && [ "$(sed -n '1p' "$agentrun_rollback_marker")" != "$agentrun_recovery_target_version" ]; then
    echo "FAIL agentrun-recovery-gate: recovery input conflicts with persisted rollback target" >&2
    exit 1
  fi
  printf '%s\n' "$agentrun_recovery_target_version" > "$agentrun_rollback_marker"
  chmod 0640 "$agentrun_rollback_marker"
  sync "$agentrun_rollback_marker"
  sync -f "$state_dir"
  prepare_previous_release_agentrun_rollback
  rm -f "$agentrun_rollback_marker"
  sync -f "$state_dir"
  echo "PASS recovered-explicit-agentrun-migration-target"
fi

verify_external_qdrant "${candidate_compose[@]}"
echo "PASS external-qdrant-ready"

# The host runner owning the bind-mount directory is not enough: the
# unprivileged AgentRun image user must be able to create durable Artifacts.
"${candidate_compose[@]}" run --rm --no-deps --entrypoint /bin/sh agentrun \
  -c 'probe="$(mktemp /app/data/.uat-write-probe.XXXXXX)" && rm -f "$probe"'
echo "PASS agentrun-artifact-write"

"${candidate_compose[@]}" run --rm --no-deps \
  --entrypoint /usr/local/bin/uat-excluded-fact-audit data \
  > "$excluded_fact_before_report"
echo "PASS excluded-fact-audit-before"

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
    version, classification, *_ = line.split("\t")
    if classification not in {"normal", "high", "blocked"}:
        raise SystemExit(f"invalid migration risk classification for {version}: {classification}")
    risk[version] = classification
pending = report.get("pending") or []
versions = [str(item.get("Version", item.get("version", ""))).zfill(6) for item in pending]
unclassified = [version for version in versions if version not in risk]
if unclassified:
    raise SystemExit("pending migrations lack risk classification: " + ",".join(unclassified))
print(",".join(version for version in versions if risk[version] == "high"))
print(",".join(version for version in versions if risk[version] == "blocked"))
PY
)"
high_risk_pending="$(printf '%s\n' "$migration_risk_summary" | sed -n '1p')"
blocked_pending="$(printf '%s\n' "$migration_risk_summary" | sed -n '2p')"

agentrun_migration_risk_summary="$(python3 - "$agentrun_report_file" "$agentrun_migration_risk_manifest" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
risk = {}
for line in pathlib.Path(sys.argv[2]).read_text().splitlines():
    if not line.strip() or line.lstrip().startswith("#"):
        continue
    version, classification, *_ = line.split("\t")
    if classification not in {"normal", "high", "blocked"}:
        raise SystemExit(f"invalid AgentRun migration risk classification for {version}: {classification}")
    risk[version] = classification
pending = report.get("pending") or []
versions = [str(item.get("version", item.get("Version", ""))).zfill(3) for item in pending]
unclassified = [version for version in versions if version not in risk]
if unclassified:
    raise SystemExit("pending AgentRun migrations lack risk classification: " + ",".join(unclassified))
print(",".join(version for version in versions if risk[version] == "high"))
print(",".join(version for version in versions if risk[version] == "blocked"))
rollback_versions = {"011", "012", "013", "014", "015"}
print("true" if rollback_versions.intersection(versions) else "false")
print(str(report.get("current_version") or "").zfill(3))
PY
)"
agentrun_high_risk_pending="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '1p')"
agentrun_blocked_pending="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '2p')"
agentrun_rollback_compatibility_required="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '3p')"
agentrun_rollback_target_version="$(printf '%s\n' "$agentrun_migration_risk_summary" | sed -n '4p')"

database_identity="tidewise_uat@config.uat.yaml/tidewise_uat"

{
  echo "### UAT migration preflight"
  echo
  echo "- Release: \`${release_sha}\`"
  echo "- Database: \`${database_identity}\`"
  echo "- TLS database check: passed"
  echo "- High-risk pending migrations: \`${high_risk_pending:-none}\`"
  echo "- Release-blocked pending migrations: \`${blocked_pending:-none}\`"
  echo "- AgentRun high-risk pending migrations: \`${agentrun_high_risk_pending:-none}\`"
  echo "- AgentRun release-blocked pending migrations: \`${agentrun_blocked_pending:-none}\`"
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

if { [ -n "$high_risk_pending" ] || [ -n "$agentrun_high_risk_pending" ]; } && [ "$backup_confirmed" != true ]; then
  echo "FAIL migration-risk-gate: confirm_high_risk_backup=true is required for data=${high_risk_pending:-none} agentrun=${agentrun_high_risk_pending:-none}" >&2
  exit 1
fi
echo "PASS migration-risk-gate"

if [ "$agentrun_rollback_compatibility_required" = true ] && ! [[ "$agentrun_rollback_target_version" =~ ^01[0-4]$ ]]; then
  echo "FAIL agentrun-rollback-target-gate: unsupported previous migration version ${agentrun_rollback_target_version:-none}" >&2
  exit 1
fi

"${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply > "$report_file"
if [ "$agentrun_rollback_compatibility_required" = true ]; then
  printf '%s\n' "$agentrun_rollback_target_version" > "$agentrun_rollback_marker"
  chmod 0640 "$agentrun_rollback_marker"
  sync "$agentrun_rollback_marker"
  sync -f "$state_dir"
  trap cleanup_unfinished_agentrun_migration EXIT
fi
"${candidate_compose[@]}" run --rm --no-deps --entrypoint /app/agentrun-migrate agentrun > "$agentrun_report_file"
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

if [ "$industry_import_enabled" = true ]; then
  industry_import_command=(
    /usr/local/bin/industry-relationship-import
    -package "$industry_package_path"
    -expected-sha256 "$industry_package_sha"
    -caller-subject tidewise-uat-industry-relationship-import-v1
  )
  "${candidate_compose[@]}" run --rm --no-deps data \
    "${industry_import_command[@]}" -dry-run > "$industry_import_dry_run_report"
  echo "PASS industry-relationship-import-dry-run"
  "${candidate_compose[@]}" run --rm --no-deps data \
    "${industry_import_command[@]}" -apply -allow-env uat > "$industry_import_apply_report"
  echo "PASS industry-relationship-import-apply"
  "${candidate_compose[@]}" run --rm --no-deps data \
    "${industry_import_command[@]}" -apply -allow-env uat > "$industry_import_replay_report"

  python3 - \
    "$industry_package_sha" \
    "$industry_import_dry_run_report" \
    "$industry_import_apply_report" \
    "$industry_import_replay_report" <<'PY'
import json
import pathlib
import sys

expected_sha = sys.argv[1]
reports = [json.loads(pathlib.Path(path).read_text()) for path in sys.argv[2:]]
labels = ["dry-run", "apply", "replay"]
for label, report in zip(labels, reports):
    if report.get("package_sha256") != expected_sha:
        raise SystemExit(f"{label} package SHA does not match the approved input")
if reports[0].get("dry_run") is not True:
    raise SystemExit("preflight result is not marked dry_run")
if reports[1].get("dry_run") is not False or reports[2].get("dry_run") is not False:
    raise SystemExit("apply/replay result unexpectedly reports dry_run")
if reports[2].get("unchanged") is not True:
    raise SystemExit("replay did not verify the persisted package as unchanged")
counts = [report.get("package_counts") for report in reports]
if counts[0] != counts[1] or counts[1] != counts[2]:
    raise SystemExit("dry-run/apply/replay package counts differ")
PY
  echo "PASS industry-relationship-import-replay"
  {
    echo
    echo "### UAT Industry relationship import"
    echo
    echo "- Package SHA: \`${industry_package_sha}\`"
    echo "- Database preflight: passed"
    echo "- Transactional apply: passed"
    echo "- Same-package replay: unchanged"
    echo
    echo '<details><summary>Package counts</summary>'
    echo
    echo '```json'
    sed -n '1,200p' "$industry_import_replay_report"
    echo '```'
    echo '</details>'
  } >> "$summary_file"
fi

if [ "$industry_graph_projection_enabled" = true ]; then
  industry_graph_command=(
    /usr/local/bin/industry-graph-projector
    -package "$industry_package_path"
    -expected-sha256 "$industry_graph_package_sha"
  )
  neo4j_environment=(
    -e NEO4J_URI
    -e NEO4J_USERNAME
    -e NEO4J_PASSWORD
    -e NEO4J_DATABASE
  )
  "${candidate_compose[@]}" run --rm --no-deps \
    "${neo4j_environment[@]}" data \
    "${industry_graph_command[@]}" -dry-run > "$industry_graph_dry_run_report"
  echo "PASS industry-graph-projection-dry-run"
  "${candidate_compose[@]}" run --rm --no-deps \
    "${neo4j_environment[@]}" data \
    "${industry_graph_command[@]}" -apply -allow-env uat > "$industry_graph_apply_report"
  echo "PASS industry-graph-projection-apply"
  "${candidate_compose[@]}" run --rm --no-deps \
    "${neo4j_environment[@]}" data \
    "${industry_graph_command[@]}" -apply -allow-env uat > "$industry_graph_replay_report"

  python3 - \
    "$industry_graph_package_sha" \
    "$industry_graph_dry_run_report" \
    "$industry_graph_apply_report" \
    "$industry_graph_replay_report" <<'PY'
import json
import pathlib
import sys

expected_sha = sys.argv[1]
reports = [json.loads(pathlib.Path(path).read_text()) for path in sys.argv[2:]]
labels = ["dry-run", "apply", "replay"]
expected_node_types = {
    "industry": 512,
    "concept": 180,
    "industry_chain": 708,
    "chain_node": 3049,
}
expected_relationship_types = {
    "MAPPED_TO_INDUSTRY": 716,
    "MAPPED_TO_CONCEPT": 521,
    "HAS_NODE": 3350,
    "INPUT_TO": 1537,
    "IS_COMPONENT_OF": 704,
    "DEPENDS_ON": 404,
    "IS_SUBCATEGORY_OF": 635,
}
expected_node_fingerprint = "4229146e37ee554cd58377843743f93dc753bdfd92bbe7f2c9afac61c2003d63"
expected_relationship_fingerprint = "aba6be387c0dad1b93c6fd14a4f9216b77a625d206cae9e7b977854f0cacec94"
for label, report in zip(labels, reports):
    if report.get("namespace") != "tidewise-industry-v1":
        raise SystemExit(f"{label} namespace does not match the frozen V1 contract")
    if report.get("contract_version") != "industry-graph-projection-v1":
        raise SystemExit(f"{label} contract version does not match the frozen V1 contract")
    if report.get("package_sha256") != expected_sha:
        raise SystemExit(f"{label} package SHA does not match the approved input")
    if report.get("node_count") != 4449 or report.get("relationship_count") != 7867:
        raise SystemExit(f"{label} top-level graph counts do not match the frozen V1 contract")
    source = report.get("source") or {}
    if source.get("node_count") != 4449 or source.get("relationship_count") != 7867:
        raise SystemExit(f"{label} source counts do not match the frozen V1 contract")
    if source.get("node_type_counts") != expected_node_types:
        raise SystemExit(f"{label} source node type counts do not match the frozen V1 contract")
    if source.get("relationship_type_counts") != expected_relationship_types:
        raise SystemExit(f"{label} source relationship type counts do not match the frozen V1 contract")
    if source.get("node_fingerprint") != expected_node_fingerprint:
        raise SystemExit(f"{label} source node fingerprint does not match the frozen V1 contract")
    if source.get("relationship_fingerprint") != expected_relationship_fingerprint:
        raise SystemExit(f"{label} source relationship fingerprint does not match the frozen V1 contract")
    for defect in (
        "orphan_count",
        "duplicate_node_count",
        "duplicate_relationship_count",
        "self_loop_count",
        "missing_chain_identity_count",
    ):
        if source.get(defect) != 0:
            raise SystemExit(f"{label} source integrity check {defect} is not zero")
if reports[0].get("dry_run") is not True or reports[0].get("applied") is not False:
    raise SystemExit("preflight result has invalid dry-run/apply flags")
for label, report in zip(labels[1:], reports[1:]):
    if report.get("dry_run") is not False:
        raise SystemExit(f"{label} unexpectedly reports dry_run")
    if report.get("final_integrity_violation_count") != 0:
        raise SystemExit(f"{label} final Neo4j integrity violations are not zero")
    if report.get("final_neo4j") != report.get("source"):
        raise SystemExit(f"{label} final Neo4j projection differs from the approved source")
if reports[1].get("applied") is not True and reports[1].get("unchanged") is not True:
    raise SystemExit("apply neither changed nor confirmed the approved graph")
if reports[2].get("unchanged") is not True:
    raise SystemExit("replay did not verify the persisted graph as unchanged")
PY
  echo "PASS industry-graph-projection-replay"
  {
    echo
    echo "### UAT Industry graph projection"
    echo
    echo "- Package SHA: \`${industry_graph_package_sha}\`"
    echo "- PostgreSQL/Neo4j preflight: passed"
    echo "- Transactional projection: passed"
    echo "- Same-package replay: unchanged"
    echo
    echo '<details><summary>Projection counts and fingerprints</summary>'
    echo
    echo '```json'
    sed -n '1,240p' "$industry_graph_replay_report"
    echo '```'
    echo '</details>'
  } >> "$summary_file"
fi

if [ "$event_semantic_projection_enabled" = true ]; then
  if ! "${candidate_compose[@]}" stop agentrun; then
    echo "FAIL event-semantic-projection-pause: AgentRun could not be stopped" >&2
    exit 1
  fi
  candidate_services_started=true
  trap cleanup_unfinished_agentrun_migration EXIT

  if ! verify_external_qdrant "${candidate_compose[@]}"; then
    exit 1
  fi
  echo "PASS event-semantic-projection-qdrant-ready"

  event_semantic_projection_environment=(
    -e "QDRANT_URL=${event_semantic_qdrant_url}"
    -e "EMBEDDING_BASE_URL=${event_semantic_embedding_base_url}"
    -e EMBEDDING_API_KEY
  )
  if ! "${candidate_compose[@]}" run --rm --no-deps \
    "${event_semantic_projection_environment[@]}" data \
    /usr/local/bin/event-semantic-projector -apply -allow-env uat > "$event_semantic_projection_report"; then
    exit 1
  fi
  echo "PASS event-semantic-projection-apply"

  "${candidate_compose[@]}" run --rm --no-deps --entrypoint /bin/sh data \
    -ec "wget -q -T 10 -t 2 -O- ${event_semantic_qdrant_url}/collections/entity_semantic_v1" \
    > "$event_semantic_entity_collection_report"
  "${candidate_compose[@]}" run --rm --no-deps --entrypoint /bin/sh data \
    -ec "wget -q -T 10 -t 2 -O- ${event_semantic_qdrant_url}/collections/variable_definition_semantic_v1" \
    > "$event_semantic_variable_collection_report"

  python3 - \
    "$event_semantic_projection_report" \
    "$event_semantic_entity_collection_report" \
    "$event_semantic_variable_collection_report" <<'PY'
import json
import pathlib
import sys

projection = json.loads(pathlib.Path(sys.argv[1]).read_text())
entity = json.loads(pathlib.Path(sys.argv[2]).read_text()).get("result") or {}
variable = json.loads(pathlib.Path(sys.argv[3]).read_text()).get("result") or {}

if projection.get("projection_version") != "event-semantic-projection.v1":
    raise SystemExit("Event Semantic projection version does not match the frozen contract")
if projection.get("embedding_model") != "text-embedding-v4":
    raise SystemExit("Event Semantic embedding model does not match the frozen contract")
if projection.get("entity_count") != 4973:
    raise SystemExit("Event Semantic Entity projection count does not match the UAT production baseline")
if projection.get("variable_definition_count") != 12:
    raise SystemExit("Event Semantic Variable Definition projection count does not match the UAT production baseline")

def verify_collection(label, value, expected_count):
    if value.get("status") != "green":
        raise SystemExit(f"{label} collection is not green")
    if value.get("points_count") != expected_count:
        raise SystemExit(f"{label} collection point count does not match the UAT production baseline")
    vectors = (((value.get("config") or {}).get("params") or {}).get("vectors") or {})
    if vectors.get("size") != 1024 or vectors.get("distance") != "Cosine":
        raise SystemExit(f"{label} collection vector contract does not match 1024/Cosine")

verify_collection("Entity", entity, 4973)
verify_collection("Variable Definition", variable, 12)
PY
  echo "PASS event-semantic-projection-verify"
  {
    echo
    echo "### UAT Event Semantic Qdrant projection"
    echo
    echo "- Projection: \`event-semantic-projection.v1\`"
    echo "- Embedding model: \`text-embedding-v4\`"
    echo "- Entity points: \`4973\`"
    echo "- Variable Definition points: \`12\`"
    echo "- Vector contract: \`1024 / Cosine\`"
  } >> "$summary_file"
fi

"${candidate_compose[@]}" run --rm --no-deps \
  --entrypoint /usr/local/bin/uat-excluded-fact-audit data \
  > "$excluded_fact_after_report"
python3 - "$excluded_fact_before_report" "$excluded_fact_after_report" <<'PY'
import json
import pathlib
import sys

before = json.loads(pathlib.Path(sys.argv[1]).read_text())
after = json.loads(pathlib.Path(sys.argv[2]).read_text())
expected_contract = "uat-excluded-fact-audit.v1"
if before.get("contract_version") != expected_contract or after.get("contract_version") != expected_contract:
    raise SystemExit("excluded fact audit contract version does not match")
if before.get("tables") != after.get("tables"):
    before_tables = before.get("tables") or {}
    after_tables = after.get("tables") or {}
    changed = sorted(
        table for table in set(before_tables) | set(after_tables)
        if before_tables.get(table) != after_tables.get(table)
    )
    raise SystemExit("excluded PostgreSQL facts changed during deployment: " + ",".join(changed))
PY
echo "PASS excluded-fact-audit-unchanged"
{
  echo
  echo "### UAT excluded PostgreSQL fact audit"
  echo
  echo "Event, RawDocument, Theme, and Reason Tree table counts and schema-normalized content fingerprints are unchanged."
} >> "$summary_file"

if ! "${candidate_compose[@]}" up -d --wait --wait-timeout 120; then
  if [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
  if [ "$candidate_services_started" != true ]; then
    rollback_current_release
  fi
  exit 1
fi
candidate_services_started=true
trap cleanup_unfinished_agentrun_migration EXIT

if [ -s "$current_runtime" ] && [ -s "$current_images" ] && [ -s "$current_compose" ] && [ -s "$current_sha" ]; then
  install -m 0600 "$current_runtime" "$previous_runtime"
  install -m 0640 "$current_images" "$previous_images"
  install -m 0640 "$current_compose" "$previous_compose"
  install -m 0640 "$current_sha" "$previous_sha"
  rollback_snapshot_ready=true
fi
if [ "$rollback_snapshot_ready" = true ]; then
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
sync -f "$state_dir"
agentrun_rollback_compatibility_required=false
rm -f "$release_state_write_marker"
sync -f "$state_dir"
trap - EXIT
echo "PASS release-state-recorded"

{
  echo
  echo "### UAT deployment"
  echo
  echo "Deployed \`${release_sha}\` as one five-business-image release unit; Qdrant remains independently operated."
  if [ -s "$previous_sha" ]; then
    echo "Previous successful release: \`$(sed -n '1p' "$previous_sha")\`."
  fi
} >> "$summary_file"
