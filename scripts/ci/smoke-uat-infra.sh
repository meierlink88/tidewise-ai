#!/usr/bin/env bash

set -Eeuo pipefail

compose_file=infra/uat-infra/docker-compose.yaml
example_env=infra/uat-infra/.env.example
deploy_script=infra/uat-infra/deploy.sh
run_suffix="${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-$$}"
[[ "$run_suffix" =~ ^[A-Za-z0-9_-]+$ ]] || { echo "invalid smoke run suffix" >&2; exit 1; }

grep -q 'compose_for "$runtime_env" "$compose_file" restart minio' "$deploy_script"
if grep -Eiq 'openspg-mysql|mysql-data|compose_for .* mysql' "$deploy_script" "$compose_file"; then
  echo "retired UAT MySQL must not remain in the MinIO deployment" >&2
  exit 1
fi

network="tidewise-uat-infra-smoke-${run_suffix}"
minio_container="tidewise-uat-minio-smoke-${run_suffix}"
minio_volume="tidewise-uat-minio-smoke-${run_suffix}"
minio_root_user="smoke$(python3 -c 'import secrets; print(secrets.token_hex(6))')"
minio_root_password="$(python3 -c 'import secrets; print(secrets.token_hex(24))')"
minio_access_key="smoke$(python3 -c 'import secrets; print(secrets.token_hex(6))')"
minio_secret_key="$(python3 -c 'import secrets; print(secrets.token_hex(24))')"

compose_json="$(docker compose --env-file "$example_env" -f "$compose_file" config --format json)"
COMPOSE_JSON="$compose_json" python3 - <<'PY'
import json
import os

config = json.loads(os.environ["COMPOSE_JSON"])
assert set(config["services"]) == {"minio"}
assert config["services"]["minio"]["ports"] == [
    {"mode": "ingress", "target": 9000, "published": "9000", "protocol": "tcp", "host_ip": "127.0.0.1"},
    {"mode": "ingress", "target": 9001, "published": "9001", "protocol": "tcp", "host_ip": "0.0.0.0"},
]
assert set(config["volumes"]) == {"minio-data"}
PY
minio_image="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["services"]["minio"]["image"])' <<<"$compose_json")"
[[ "$minio_image" =~ @sha256:[0-9a-f]{64}$ ]]

cleanup() {
  docker rm -f "$minio_container" >/dev/null 2>&1 || true
  docker volume rm "$minio_volume" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT
cleanup

docker network create "$network" >/dev/null
docker volume create "$minio_volume" >/dev/null
docker run -d --rm --name "$minio_container" --network "$network" --network-alias minio \
  --mount "type=volume,source=${minio_volume},target=/data" \
  -e MINIO_ACCESS_KEY="$minio_root_user" \
  -e MINIO_SECRET_KEY="$minio_root_password" \
  -e MINIO_ROOT_USER="$minio_root_user" \
  -e MINIO_ROOT_PASSWORD="$minio_root_password" \
  "$minio_image" server --console-address :9001 /data >/dev/null

for attempt in $(seq 1 30); do
  if docker exec "$minio_container" mc ready local >/dev/null 2>&1; then
    break
  fi
  [ "$attempt" -lt 30 ] || { docker logs "$minio_container"; exit 1; }
  sleep 1
done

docker exec -e ROOT_USER="$minio_root_user" -e ROOT_PASSWORD="$minio_root_password" \
  "$minio_container" sh -c \
  'MC_CONFIG_DIR=/tmp/test-mc mc alias set root http://minio:9000 "$ROOT_USER" "$ROOT_PASSWORD" >/dev/null'
docker exec "$minio_container" sh -c 'MC_CONFIG_DIR=/tmp/test-mc mc mb root/raw-evidence >/dev/null'

agent_policy='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetBucketLocation","s3:ListBucket"],"Resource":["arn:aws:s3:::raw-evidence"]},{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject","s3:DeleteObject"],"Resource":["arn:aws:s3:::raw-evidence/*"]}]}'
docker exec -e POLICY="$agent_policy" "$minio_container" sh -c \
  'printf "%s" "$POLICY" > /tmp/agent-policy.json; MC_CONFIG_DIR=/tmp/test-mc mc admin policy create root agentos-raw-evidence-v1 /tmp/agent-policy.json >/dev/null'
agent_policy_info="$(docker exec "$minio_container" sh -c 'MC_CONFIG_DIR=/tmp/test-mc mc admin policy info root agentos-raw-evidence-v1')"
POLICY="$agent_policy" POLICY_INFO="$agent_policy_info" python3 infra/uat-infra/verify-policy.py admin
docker exec -e ACCESS_KEY="$minio_access_key" -e SECRET_KEY="$minio_secret_key" \
  "$minio_container" sh -c \
  'MC_CONFIG_DIR=/tmp/test-mc mc admin user add root "$ACCESS_KEY" "$SECRET_KEY" >/dev/null; MC_CONFIG_DIR=/tmp/test-mc mc admin policy attach root agentos-raw-evidence-v1 --user "$ACCESS_KEY" >/dev/null; MC_CONFIG_DIR=/tmp/test-mc mc alias set agentos http://minio:9000 "$ACCESS_KEY" "$SECRET_KEY" >/dev/null'

anonymous_policy='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::raw-evidence/*"]}]}'
docker exec -e POLICY="$anonymous_policy" "$minio_container" sh -c \
  'printf "%s" "$POLICY" > /tmp/anonymous-policy.json; MC_CONFIG_DIR=/tmp/test-mc mc anonymous set-json /tmp/anonymous-policy.json root/raw-evidence >/dev/null'
anonymous_policy_info="$(docker exec "$minio_container" sh -c 'MC_CONFIG_DIR=/tmp/test-mc mc anonymous get-json root/raw-evidence')"
POLICY="$anonymous_policy" POLICY_INFO="$anonymous_policy_info" python3 infra/uat-infra/verify-policy.py anonymous

printf 'contract-test\n' | docker exec -i "$minio_container" sh -c \
  'MC_CONFIG_DIR=/tmp/test-mc mc pipe agentos/raw-evidence/test.md >/dev/null'
docker run --rm --network "$network" --entrypoint curl "$minio_image" \
  -fsS http://minio:9000/raw-evidence/test.md | grep -qx contract-test
anonymous_status="$(docker run --rm --network "$network" --entrypoint curl "$minio_image" \
  -sS -X PUT --data-binary must-be-denied -o /dev/null -w '%{http_code}' \
  http://minio:9000/raw-evidence/anonymous-put.md)"
[ "$anonymous_status" = 403 ]

docker restart "$minio_container" >/dev/null
for attempt in $(seq 1 30); do
  docker exec "$minio_container" mc ready local >/dev/null 2>&1 && break
  [ "$attempt" -lt 30 ] || exit 1
  sleep 1
done
docker exec -e ROOT_USER="$minio_root_user" -e ROOT_PASSWORD="$minio_root_password" \
  "$minio_container" sh -c \
  'MC_CONFIG_DIR=/tmp/test-mc mc alias set root http://minio:9000 "$ROOT_USER" "$ROOT_PASSWORD" >/dev/null; MC_CONFIG_DIR=/tmp/test-mc mc stat root/raw-evidence >/dev/null'

echo "PASS UAT MinIO raw-evidence behavior smoke"
