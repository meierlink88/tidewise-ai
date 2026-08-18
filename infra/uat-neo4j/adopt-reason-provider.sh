#!/usr/bin/env bash

set -Eeuo pipefail

[ "$(id -u)" -eq 0 ] || {
  echo "UAT Neo4j adoption must run as root" >&2
  exit 1
}

script_dir="$(cd "$(dirname "$0")" && pwd)"
# The runtime-relative library is checked separately in CI.
# shellcheck disable=SC1091
source "$script_dir/lib.sh"
neo4j_admin_password="${NEO4J_ADMIN_PASSWORD:?NEO4J_ADMIN_PASSWORD is required}"
gds_version=2.13.4
gds_url="https://graphdatascience.ninja/neo4j-graph-data-science-${gds_version}.jar"
gds_sha256="10e072f73992224f1159f246c9d6a89da5f3b3434aeffa5be42647edda13a8d8"
gds_source="${GDS_JAR_PATH:-/tmp/neo4j-graph-data-science-${gds_version}.jar}"
deploy_root=/opt/tidewise/neo4j-uat
backup_id="$(date -u +%Y%m%dT%H%M%SZ)"
backup_root="$deploy_root/backups/$backup_id"
neo4j_data=/var/lib/neo4j/data
neo4j_plugins=/var/lib/neo4j/plugins
neo4j_labs=/var/lib/neo4j/labs
neo4j_config=/etc/neo4j
adoption_started=false
neo4j_stop_attempted=false
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

for command in curl docker flock install ip neo4j neo4j-admin python3 runuser sha256sum ss systemctl; do
  command -v "$command" >/dev/null || {
    echo "Missing dependency: $command" >&2
    exit 1
  }
done
[ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ]
[ "$(neo4j version)" = "neo4j 5.26.28" ]
[ "$(systemctl is-active neo4j)" = active ]
docker info >/dev/null
docker network inspect tidewise-uat >/dev/null
ip -4 address show docker0 | grep -Eq 'inet 172\.17\.0\.1/16([[:space:]]|$)'
[ -f "$neo4j_labs/apoc-5.26.28-core.jar" ]
[ -f "$script_dir/neo4j.conf.fragment" ]

validate_neo4j_backup_target "$backup_root" || {
  echo "Refusing unexpected backup target: $backup_root" >&2
  exit 1
}

unowned_fingerprint() {
  local container
  for container in "${unowned_containers[@]}"; do
    docker inspect --format '{{.Name}}|{{.Id}}|{{.State.StartedAt}}' "$container"
  done
}

on_error() {
  local code="$1"
  local recovery_code
  trap - ERR
  set +e
  recover_neo4j_after_failure \
    "$adoption_started" "$neo4j_stop_attempted" \
    "$neo4j_data" "$neo4j_config" "$neo4j_plugins" "$backup_root"
  recovery_code="$?"
  set -e
  if [ "$recovery_code" -ne 0 ]; then
    echo "FAIL recovery: manual Neo4j recovery required from $backup_root" >&2
  elif [ "$adoption_started" = true ]; then
    echo "PASS restored pre-adoption Neo4j from $backup_root" >&2
  fi
  exit "$code"
}
trap 'on_error $?' ERR

install -d -m 0750 -o root -g root "$deploy_root" "$deploy_root/backups"
exec 8>/opt/tidewise/uat/deploy.lock
flock -n 8 || {
  echo "Another UAT deployment holds /opt/tidewise/uat/deploy.lock" >&2
  exit 1
}
exec 9>"$deploy_root/deploy.lock"
flock -n 9 || {
  echo "Another UAT Neo4j adoption holds $deploy_root/deploy.lock" >&2
  exit 1
}

if [ ! -f "$gds_source" ]; then
  curl --fail --location --retry 8 --retry-delay 2 --output "$gds_source" "$gds_url"
fi
printf '%s  %s\n' "$gds_sha256" "$gds_source" | sha256sum --check --status

unowned_before="$(unowned_fingerprint)"
install -d -m 0750 -o root -g root "$backup_root"
cp -a "$neo4j_config" "$backup_root/etc-neo4j"
cp -a "$neo4j_plugins" "$backup_root/plugins"
cp -a "$neo4j_labs" "$backup_root/labs"

neo4j_stop_attempted=true
systemctl stop neo4j
[ "$(systemctl is-active neo4j)" = inactive ]
mv "$neo4j_data" "$backup_root/data"
adoption_started=true
install -d -m 0750 -o neo4j -g neo4j "$neo4j_data"

install -m 0644 -o root -g neo4j \
  "$neo4j_labs/apoc-5.26.28-core.jar" \
  "$neo4j_plugins/apoc-5.26.28-core.jar"
install -m 0644 -o root -g neo4j \
  "$gds_source" \
  "$neo4j_plugins/neo4j-graph-data-science-${gds_version}.jar"

CONFIG_PATH="$neo4j_config/neo4j.conf" \
FRAGMENT_PATH="$script_dir/neo4j.conf.fragment" \
python3 - <<'PY'
import os
from pathlib import Path

config_path = Path(os.environ["CONFIG_PATH"])
fragment_path = Path(os.environ["FRAGMENT_PATH"])
managed_keys = {
    line.split("=", 1)[0]
    for line in fragment_path.read_text().splitlines()
    if line and not line.startswith("#") and "=" in line
}
kept = []
inside_managed_block = False
for line in config_path.read_text().splitlines():
    if line == "# BEGIN TIDEWISE UAT NEO4J":
        inside_managed_block = True
        continue
    if line == "# END TIDEWISE UAT NEO4J":
        inside_managed_block = False
        continue
    if inside_managed_block:
        continue
    key = line.split("=", 1)[0].strip() if "=" in line else ""
    if key in managed_keys:
        continue
    kept.append(line)
text = "\n".join(kept).rstrip() + "\n\n" + fragment_path.read_text().strip() + "\n"
temporary = config_path.with_suffix(".conf.tidewise-new")
temporary.write_text(text)
temporary.chmod(0o640)
temporary.replace(config_path)
PY
chown root:neo4j "$neo4j_config/neo4j.conf"
install -m 0640 -o root -g neo4j /dev/null "$neo4j_config/apoc.conf"
printf '%s\n' \
  'apoc.export.file.enabled=true' \
  'apoc.import.file.enabled=true' >"$neo4j_config/apoc.conf"

runuser -u neo4j -- env NEO4J_CONF="$neo4j_config" NEO4J_HOME=/var/lib/neo4j \
  neo4j-admin dbms set-initial-password "$neo4j_admin_password" --require-password-change=false
systemctl start neo4j
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

printf '%s\n' "$backup_root" >"$deploy_root/current-backup"
chmod 0640 "$deploy_root/current-backup"
sync "$deploy_root/current-backup"
adoption_started=false
neo4j_stop_attempted=false
trap - ERR
echo "PASS adopted UAT Neo4j for Reason; backup=$backup_root"
