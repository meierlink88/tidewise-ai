#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
candidate_images="${CANDIDATE_IMAGES:?CANDIDATE_IMAGES is required}"
release_sha="${COMMIT_SHA:?COMMIT_SHA is required}"
public_base_url="${UAT_PUBLIC_BASE_URL:?UAT_PUBLIC_BASE_URL is required}"
backup_confirmed="${HIGH_RISK_BACKUP_CONFIRMED:-false}"
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

test -d "$state_dir"
test -w "$state_dir"

exec 9>"${deployment_root}/deploy.lock"
if ! flock -n 9; then
  echo "FAIL deployment-lock: another UAT deployment holds ${deployment_root}/deploy.lock" >&2
  exit 1
fi
echo "PASS deployment-lock"

candidate_compose=(docker compose --env-file "$runtime_env" --env-file "$candidate_images" -f "$compose_file")

runtime_value() {
  local file="$1"
  local key="$2"
  sed -n "s/^${key}=//p" "$file" | tail -n 1
}

verify_services() {
  local verification_runtime="$1"
  shift
  local -a compose_command=("$@")
  local verification_admin_token
  verification_admin_token="$(runtime_value "$verification_runtime" ADMIN_API_TOKEN)"

  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/healthz >/dev/null
  "${compose_command[@]}" exec -T data wget -qO- http://127.0.0.1:9011/readyz >/dev/null
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/healthz >/dev/null
  "${compose_command[@]}" exec -T miniapp wget -qO- http://127.0.0.1:9012/readyz >/dev/null
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/healthz >/dev/null
  "${compose_command[@]}" exec -T adminportal wget -qO- http://127.0.0.1:9013/readyz >/dev/null
  "${compose_command[@]}" exec -T admin wget -qO- http://127.0.0.1:9014/healthz >/dev/null
  echo "PASS container-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${public_base_url}:9012/healthz" >/dev/null
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${public_base_url}:9013/healthz" >/dev/null
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${public_base_url}:9014/healthz" >/dev/null
  echo "PASS public-entry-health"

  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 "${public_base_url}:9012/api/v1/research/themes?limit=1" >/dev/null
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 --retry 2 --header "Authorization: Bearer ${verification_admin_token}" "${public_base_url}:9013/admin/source-catalogs?limit=1" >/dev/null
  echo "PASS bff-to-data-read-paths"
}

rollback_current_release() {
  if [ ! -s "$current_runtime" ] || [ ! -s "$current_images" ] || [ ! -s "$current_compose" ] || [ ! -s "$current_sha" ]; then
    echo "FAIL rollback: no previous repository-managed UAT release is available" >&2
    return 1
  fi
  echo "Candidate verification failed; restoring release $(sed -n '1p' "$current_sha")" >&2
  local -a rollback_compose=(docker compose --env-file "$current_runtime" --env-file "$current_images" -f "$current_compose")
  "${rollback_compose[@]}" up -d --wait --wait-timeout 120 --remove-orphans
  verify_services "$current_runtime" "${rollback_compose[@]}"
  echo "PASS rollback: previous complete release restored" >&2
}

"${candidate_compose[@]}" config --quiet
echo "PASS compose-contract"

# Check-only dbmigrate establishes a real TLS PostgreSQL connection and reports
# current/pending migration state without taking the migration lock or writing.
"${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate > "$report_file"
echo "PASS rds-tls-readonly"

high_risk_pending="$(python3 - "$report_file" "$migration_risk_manifest" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
risk = {}
for line in pathlib.Path(sys.argv[2]).read_text().splitlines():
    if not line.strip() or line.lstrip().startswith("#"):
        continue
    version, classification, *_ = line.split("\t")
    if classification not in {"normal", "high"}:
        raise SystemExit(f"invalid migration risk classification for {version}: {classification}")
    risk[version] = classification
pending = report.get("pending", [])
versions = [str(item.get("Version", item.get("version", ""))).zfill(6) for item in pending]
unclassified = [version for version in versions if version not in risk]
if unclassified:
    raise SystemExit("pending migrations lack risk classification: " + ",".join(unclassified))
print(",".join(version for version in versions if risk[version] == "high"))
PY
)"

database_identity="$(python3 - <<'PY'
import os
from urllib.parse import urlparse

endpoint = urlparse(os.environ["UAT_DATABASE_URL"])
print(f"{endpoint.username or '<unknown>'}@{endpoint.hostname or '<unknown>'}:{endpoint.port or 5432}{endpoint.path}")
PY
)"

{
  echo "### UAT migration preflight"
  echo
  echo "- Release: \`${release_sha}\`"
  echo "- Database: \`${database_identity}\`"
  echo "- TLS database check: passed"
  echo "- High-risk pending migrations: \`${high_risk_pending:-none}\`"
  echo
  echo '<details><summary>Migration state before apply</summary>'
  echo
  echo '```json'
  sed -n '1,200p' "$report_file"
  echo '```'
  echo '</details>'
} >> "$summary_file"

if [ -n "$high_risk_pending" ] && [ "$backup_confirmed" != true ]; then
  echo "FAIL migration-risk-gate: confirm_high_risk_backup=true is required for $high_risk_pending" >&2
  exit 1
fi
echo "PASS migration-risk-gate"

"${candidate_compose[@]}" run --rm --no-deps data /usr/local/bin/dbmigrate -apply > "$report_file"
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

if ! "${candidate_compose[@]}" up -d --wait --wait-timeout 120 --remove-orphans; then
  rollback_current_release
  exit 1
fi
if ! verify_services "$runtime_env" "${candidate_compose[@]}"; then
  rollback_current_release
  exit 1
fi

if [ -s "$current_runtime" ] && [ -s "$current_images" ] && [ -s "$current_compose" ] && [ -s "$current_sha" ]; then
  install -m 0600 "$current_runtime" "$previous_runtime"
  install -m 0640 "$current_images" "$previous_images"
  install -m 0640 "$current_compose" "$previous_compose"
  install -m 0640 "$current_sha" "$previous_sha"
fi
install -m 0600 "$runtime_env" "$current_runtime"
install -m 0640 "$candidate_images" "$current_images"
install -m 0640 "$compose_file" "$current_compose"
printf '%s\n' "$release_sha" > "$current_sha"
chmod 0640 "$current_sha"
echo "PASS release-state-recorded"

{
  echo
  echo "### UAT deployment"
  echo
  echo "Deployed \`${release_sha}\` as one four-image release unit."
  if [ -s "$previous_sha" ]; then
    echo "Previous successful release: \`$(sed -n '1p' "$previous_sha")\`."
  fi
} >> "$summary_file"
