#!/usr/bin/env bash

set -Eeuo pipefail

deploy_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
compose_file="${COMPOSE_FILE:?COMPOSE_FILE is required}"
script_dir="$(cd "$(dirname "$0")" && pwd)"
release_sha="${RELEASE_SHA:?RELEASE_SHA is required}"
public_base="${RAW_EVIDENCE_PUBLIC_BASE_URL:?RAW_EVIDENCE_PUBLIC_BASE_URL is required}"
state_dir="$deploy_root/state"
current_runtime="$deploy_root/runtime.env"
current_compose="$state_dir/current.compose.yaml"
current_sha="$state_dir/current.sha"
previous_runtime="$state_dir/previous.runtime.env"
previous_compose="$state_dir/previous.compose.yaml"
previous_sha="$state_dir/previous.sha"
rollback_in_progress=false
unowned_containers=(
  tidewise-infra-uat-mysql-1
  tidewise-uat-data-1
  tidewise-uat-miniapp-1
  tidewise-uat-adminportal-1
  tidewise-uat-admin-1
  tidewise-uat-qdrant
  tidewise-uat-openspg-neo4j
  tidewise-agentos-uat-agentos-1
)

for name in MINIO_ROOT_USER MINIO_ROOT_PASSWORD MINIO_ACCESS_KEY MINIO_SECRET_KEY; do
  [ -n "${!name:-}" ] || { echo "FAIL secret: $name is required" >&2; exit 1; }
done

compose_for() {
  local env_file="$1"
  local file="$2"
  shift 2
  docker compose --env-file "$env_file" -f "$file" "$@"
}

verify_release() {
  local env_file="$1"
  local file="$2"
  compose_for "$env_file" "$file" exec -T -e MYSQL_PWD="$OPENSPG_MYSQL_ROOT_PASSWORD" mysql mysqladmin ping -h 127.0.0.1 --silent >/dev/null
  compose_for "$env_file" "$file" exec -T minio mc ready local >/dev/null
  curl --fail --silent --show-error http://127.0.0.1:9001/ >/dev/null
  compose_for "$env_file" "$file" exec -T mysql getent hosts mysql >/dev/null
  docker run --rm --network tidewise-uat \
    -e MINIO_ROOT_USER -e MINIO_ROOT_PASSWORD --entrypoint sh \
    spg-registry.cn-hangzhou.cr.aliyuncs.com/spg/openspg-minio@sha256:9493c8e8f77edb10d556255d49ba8b5761b0fe57889235dfd10619c0513da007 \
    -c 'MC_CONFIG_DIR=/tmp/mc mc alias set network http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null && MC_CONFIG_DIR=/tmp/mc mc ready network' >/dev/null
  echo "PASS mysql-minio-health-and-network"
}

unowned_fingerprint() {
  local container
  for container in "${unowned_containers[@]}"; do
    docker inspect --format '{{.Id}}|{{.State.StartedAt}}' "$container"
  done
}

verify_unowned_services() {
  local container
  for container in \
    tidewise-uat-data-1 tidewise-uat-miniapp-1 tidewise-uat-adminportal-1 \
    tidewise-uat-admin-1 tidewise-agentos-uat-agentos-1; do
    [ "$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' "$container")" = healthy ]
  done
  docker exec tidewise-uat-data-1 wget -qO- http://127.0.0.1:9011/readyz >/dev/null
  docker exec tidewise-uat-data-1 /usr/local/bin/dbmigrate >/dev/null
  docker exec tidewise-uat-data-1 wget -qO- http://qdrant:6333/healthz >/dev/null
  curl -fsS --connect-timeout 3 --max-time 5 http://127.0.0.1:9081/health >/dev/null
  curl -fsS --connect-timeout 3 --max-time 5 http://127.0.0.1:7474/ >/dev/null
  [ "$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}missing{{end}}' tidewise-uat-openspg-neo4j)" = healthy ]
  echo "PASS unowned-uat-services-and-rds-readiness"
}

rollback() {
  rollback_in_progress=true
  if [ -s "$current_runtime" ] && [ -s "$current_compose" ] && [ -s "$current_sha" ]; then
    compose_for "$current_runtime" "$current_compose" up -d --wait --wait-timeout 180 mysql minio
    verify_release "$current_runtime" "$current_compose"
    echo "PASS rollback-previous-uat-infrastructure" >&2
  else
    compose_for "$runtime_env" "$compose_file" stop --timeout 30 mysql minio || true
    echo "PASS rollback-first-candidate-stopped-volumes-preserved" >&2
  fi
}

on_error() {
  local code="$1"
  trap - ERR
  if [ "$rollback_in_progress" = false ]; then
    set +e
    rollback
    rollback_code="$?"
    set -e
    [ "$rollback_code" -eq 0 ] || echo "FAIL rollback: manual recovery required" >&2
  fi
  exit "$code"
}
trap 'on_error $?' ERR

exec 8>/opt/tidewise/uat/deploy.lock
flock -n 8 || { echo "FAIL shared-uat-lock: another UAT deployment is running" >&2; exit 1; }
exec 9>"$deploy_root/deploy.lock"
flock -n 9 || { echo "FAIL infrastructure-lock: another infrastructure deployment is running" >&2; exit 1; }

verify_unowned_services
unowned_before="$(unowned_fingerprint)"
compose_for "$runtime_env" "$compose_file" config --quiet
compose_for "$runtime_env" "$compose_file" pull mysql minio
compose_for "$runtime_env" "$compose_file" up -d --wait --wait-timeout 180 mysql minio
verify_release "$runtime_env" "$compose_file"

mc_exec=(compose_for "$runtime_env" "$compose_file" exec -T -e MC_CONFIG_DIR=/tmp/tidewise-mc minio mc)
"${mc_exec[@]}" alias set root http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null
"${mc_exec[@]}" mb --ignore-existing root/raw-evidence >/dev/null

policy_name=agentos-raw-evidence-v1
policy='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetBucketLocation","s3:ListBucket"],"Resource":["arn:aws:s3:::raw-evidence"]},{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject","s3:DeleteObject"],"Resource":["arn:aws:s3:::raw-evidence/*"]}]}'
printf '%s' "$policy" | compose_for "$runtime_env" "$compose_file" exec -T minio sh -c 'cat > /tmp/agentos-raw-evidence-policy.json'
"${mc_exec[@]}" admin policy create root "$policy_name" /tmp/agentos-raw-evidence-policy.json >/dev/null
policy_info="$("${mc_exec[@]}" admin policy info root "$policy_name")"
POLICY="$policy" POLICY_INFO="$policy_info" python3 "$script_dir/verify-policy.py" admin
if ! "${mc_exec[@]}" admin user info root "$MINIO_ACCESS_KEY" >/dev/null 2>&1; then
  "${mc_exec[@]}" admin user add root "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null
fi
"${mc_exec[@]}" admin policy attach root "$policy_name" --user "$MINIO_ACCESS_KEY" >/dev/null
anonymous_policy='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::raw-evidence/*"]}]}'
printf '%s' "$anonymous_policy" | compose_for "$runtime_env" "$compose_file" exec -T minio sh -c 'cat > /tmp/raw-evidence-anonymous-policy.json'
"${mc_exec[@]}" anonymous set-json /tmp/raw-evidence-anonymous-policy.json root/raw-evidence >/dev/null
anonymous_policy_info="$("${mc_exec[@]}" anonymous get-json root/raw-evidence)"
POLICY="$anonymous_policy" POLICY_INFO="$anonymous_policy_info" \
  python3 "$script_dir/verify-policy.py" anonymous
"${mc_exec[@]}" alias set agentos http://127.0.0.1:9000 "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null

canary_key="_preflight/${release_sha}.md"
anonymous_put_key="_preflight/${release_sha}-anonymous-put.md"
canary_body=$'# Raw Evidence UAT infrastructure\n\nBrowser-readable Markdown.\n'
headers="$(mktemp)"
body="$(mktemp)"
cleanup_canary() {
  "${mc_exec[@]}" rm --force "agentos/raw-evidence/$canary_key" >/dev/null 2>&1 || true
  "${mc_exec[@]}" rm --force "agentos/raw-evidence/$anonymous_put_key" >/dev/null 2>&1 || true
  rm -f "$headers" "$body"
}
trap cleanup_canary EXIT
printf '%s' "$canary_body" | compose_for "$runtime_env" "$compose_file" exec -T minio sh -c \
  "MC_CONFIG_DIR=/tmp/tidewise-mc mc pipe --attr 'Content-Type=text/markdown;Content-Disposition=inline' 'agentos/raw-evidence/$canary_key'" >/dev/null
"${mc_exec[@]}" cat "agentos/raw-evidence/$canary_key" > "$body"
printf '%s' "$canary_body" | cmp -s - "$body"

hostname="$(python3 -c 'from urllib.parse import urlparse; import sys; print(urlparse(sys.argv[1]).hostname)' "$public_base")"
curl -sS --fail --connect-timeout 5 --max-time 15 \
  --resolve "${hostname}:443:127.0.0.1" --dump-header "$headers" --output "$body" \
  "${public_base%/}/raw-evidence/$canary_key"
printf '%s' "$canary_body" | cmp -s - "$body"
grep -Eiq '^content-type:[[:space:]]*text/markdown([;[:space:]]|$)' "$headers"
grep -Eiq '^content-disposition:[[:space:]]*inline' "$headers"
grep -Eiq '^x-tidewise-upstream:[[:space:]]*minio-uat' "$headers"
anonymous_status="$(curl -sS --connect-timeout 5 --max-time 15 \
  --resolve "${hostname}:443:127.0.0.1" --request PUT \
  --header 'Content-Type: text/plain' --data-binary 'must-be-denied' \
  --output /dev/null --write-out '%{http_code}' \
  "${public_base%/}/raw-evidence/$anonymous_put_key")"
[ "$anonymous_status" = 403 ]
"${mc_exec[@]}" rm --force "agentos/raw-evidence/$canary_key" >/dev/null
if "${mc_exec[@]}" stat "agentos/raw-evidence/$canary_key" >/dev/null 2>&1; then
  echo "FAIL raw-evidence-authenticated-delete: canary still exists" >&2
  exit 1
fi
cleanup_canary
trap - EXIT
echo "PASS raw-evidence-authenticated-write-read-delete-public-read-anonymous-write-denied"

compose_for "$runtime_env" "$compose_file" restart minio
compose_for "$runtime_env" "$compose_file" up -d --wait --wait-timeout 180 mysql minio
verify_release "$runtime_env" "$compose_file"
"${mc_exec[@]}" alias set root http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null
"${mc_exec[@]}" stat root/raw-evidence >/dev/null
"${mc_exec[@]}" admin user info root "$MINIO_ACCESS_KEY" >/dev/null
compose_for "$runtime_env" "$compose_file" exec -T -e MYSQL_PWD="$OPENSPG_MYSQL_ROOT_PASSWORD" mysql \
  mysql -NBe "SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME='openspg'" \
  | grep -qx openspg
echo "PASS minio-restart-and-mysql-persistence"

verify_unowned_services
unowned_after="$(unowned_fingerprint)"
[ "$unowned_before" = "$unowned_after" ] || {
  echo "FAIL unowned-uat-services: one or more existing workloads restarted" >&2
  exit 1
}
echo "PASS unowned-uat-services-not-restarted"

if [ -s "$current_runtime" ] && [ -s "$current_compose" ] && [ -s "$current_sha" ]; then
  install -m 0600 "$current_runtime" "$previous_runtime"
  install -m 0640 "$current_compose" "$previous_compose"
  install -m 0640 "$current_sha" "$previous_sha"
fi
install -m 0600 "$runtime_env" "$current_runtime"
install -m 0640 "$compose_file" "$current_compose"
printf '%s\n' "$release_sha" > "$current_sha"
chmod 0640 "$current_sha"
sync "$current_runtime" "$current_compose" "$current_sha"
trap - ERR
echo "PASS deployed-uat-infrastructure $release_sha"
