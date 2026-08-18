#!/usr/bin/env bash

set -Eeuo pipefail

[ "$(id -u)" -eq 0 ] || {
  echo "UAT Neo4j office-access configuration must run as root" >&2
  exit 1
}

script_dir="$(cd "$(dirname "$0")" && pwd)"
# The runtime-relative library is checked separately in CI.
# shellcheck disable=SC1091
source "$script_dir/lib.sh"
neo4j_admin_password="${NEO4J_ADMIN_PASSWORD:?NEO4J_ADMIN_PASSWORD is required}"
deploy_root=/opt/tidewise/neo4j-uat
backup_id="$(date -u +%Y%m%dT%H%M%SZ)"
backup_root="$deploy_root/access-backups/$backup_id"
config_path=/etc/neo4j/neo4j.conf
backup_path="$backup_root/neo4j.conf"
mutation_started=false
unowned_containers=(
  tidewise-infra-uat-mysql-1
  tidewise-infra-uat-minio-1
  tidewise-uat-data-1
  tidewise-uat-miniapp-1
  tidewise-uat-adminportal-1
  tidewise-uat-admin-1
  tidewise-agentos-uat-agentos-1
  tidewise-uat-qdrant
)

[[ "$neo4j_admin_password" =~ ^[A-Za-z0-9_-]{24,64}$ ]] || {
  echo "NEO4J_ADMIN_PASSWORD must be 24-64 URL-safe characters" >&2
  exit 1
}
validate_neo4j_access_backup_target "$backup_root" || {
  echo "Refusing unexpected access backup target: $backup_root" >&2
  exit 1
}
[ ! -e "$backup_root" ] || {
  echo "Refusing existing access backup target: $backup_root" >&2
  exit 1
}
for command in curl docker flock neo4j python3 ss systemctl; do
  command -v "$command" >/dev/null || {
    echo "Missing dependency: $command" >&2
    exit 1
  }
done
[ "$(neo4j version)" = "neo4j 5.26.28" ]
[ "$(systemctl is-active neo4j)" = active ]
[ -f "$config_path" ]
[ -f "$script_dir/neo4j.conf.fragment" ]
docker info >/dev/null
docker network inspect tidewise-uat >/dev/null

unowned_fingerprint() {
  local container
  for container in "${unowned_containers[@]}"; do
    docker inspect --format '{{.Name}}|{{.Id}}|{{.State.StartedAt}}' "$container"
  done
}

on_error() {
  local code="$1"
  local recovery_code=0
  trap - ERR
  if [ "$mutation_started" = true ]; then
    set +e
    restore_neo4j_config "$config_path" "$backup_path"
    recovery_code="$?"
    set -e
    if [ "$recovery_code" -eq 0 ]; then
      echo "PASS restored pre-change Neo4j configuration from $backup_root" >&2
    else
      echo "FAIL recovery: manual Neo4j recovery required from $backup_root" >&2
    fi
  fi
  exit "$code"
}
trap 'on_error $?' ERR

install -d -m 0750 -o root -g root "$deploy_root" "$deploy_root/access-backups" "$backup_root"
exec 8>/opt/tidewise/uat/deploy.lock
flock -n 8 || {
  echo "Another UAT deployment holds /opt/tidewise/uat/deploy.lock" >&2
  exit 1
}
exec 9>"$deploy_root/deploy.lock"
flock -n 9 || {
  echo "Another UAT Neo4j operation holds $deploy_root/deploy.lock" >&2
  exit 1
}

unowned_before="$(unowned_fingerprint)"
cp -a "$config_path" "$backup_path"
mutation_started=true
apply_neo4j_config_fragment "$config_path" "$script_dir/neo4j.conf.fragment"
chown root:neo4j "$config_path"

if cmp -s "$backup_path" "$config_path"; then
  echo "PASS Neo4j office-access configuration already current"
else
  systemctl restart neo4j
fi
for _ in $(seq 1 60); do
  if curl --fail --silent --max-time 2 http://127.0.0.1:7474/ >/dev/null; then
    break
  fi
  sleep 2
done
curl --fail --silent --show-error http://127.0.0.1:7474/ >/dev/null
NEO4J_ADMIN_PASSWORD="$neo4j_admin_password" "$script_dir/verify.sh"

unowned_after="$(unowned_fingerprint)"
[ "$unowned_before" = "$unowned_after" ] || {
  echo "One or more unrelated UAT containers restarted" >&2
  exit 1
}
printf '%s\n' "$backup_root" >"$deploy_root/current-access-backup"
chmod 0640 "$deploy_root/current-access-backup"
sync "$deploy_root/current-access-backup"
mutation_started=false
trap - ERR
echo "PASS configured office-allowlisted UAT Neo4j access; backup=$backup_root"
