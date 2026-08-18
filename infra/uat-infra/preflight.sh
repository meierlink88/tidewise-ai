#!/usr/bin/env bash

set -euo pipefail

deploy_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
expected_runner="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
public_base="${RAW_EVIDENCE_PUBLIC_BASE_URL:?RAW_EVIDENCE_PUBLIC_BASE_URL is required}"

pass() { echo "PASS $1"; }
fail() { echo "FAIL $1: $2" >&2; exit 1; }

[ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || fail platform "expected Linux x86_64"
[ "$(id -un)" = tidewise-deploy ] || fail deploy-user "expected tidewise-deploy"
[ "${RUNNER_NAME:-}" = "$expected_runner" ] || fail runner-name "expected $expected_runner"
pass runtime-identity

for command in docker curl python3 flock ss systemctl; do
  command -v "$command" >/dev/null || fail dependency "$command is missing"
done
docker info >/dev/null || fail docker-engine "docker info failed"
docker compose version >/dev/null || fail docker-compose "Docker Compose v2 is unavailable"
docker network inspect tidewise-uat >/dev/null || fail docker-network "tidewise-uat is missing"
pass docker-runtime-and-network

for directory in "$deploy_root" "$deploy_root/state" /opt/tidewise/uat; do
  [ -d "$directory" ] || fail deployment-directory "$directory is missing"
done
[ -w "$deploy_root/state" ] || fail deployment-directory "$deploy_root/state is not writable"
[ -w /opt/tidewise/uat ] || fail shared-lock "/opt/tidewise/uat is not writable"
pass deployment-storage

available_kb="$(df -Pk "$deploy_root" | awk 'NR == 2 {print $4}')"
[ "$available_kb" -ge 5242880 ] || fail disk-space "at least 5 GiB is required"
pass disk-space

python3 - <<'PY'
import os
from urllib.parse import urlparse

url = urlparse(os.environ["RAW_EVIDENCE_PUBLIC_BASE_URL"])
if url.scheme != "https" or url.hostname != "tideai.tripwise.cn":
    raise SystemExit("FAIL public-base-url: expected https://tideai.tripwise.cn")
if url.port or url.path not in ("", "/") or url.query or url.fragment:
    raise SystemExit("FAIL public-base-url: port, path, query and fragment are not allowed")
PY
pass public-base-url

curl -fsS --connect-timeout 3 --max-time 5 http://127.0.0.1:7474/ >/dev/null \
  || fail neo4j "host-native Neo4j is unavailable"
pass host-native-neo4j

for port in 3306 9000 9001; do
  while read -r container_id; do
    [ -z "$container_id" ] && continue
    project="$(docker inspect --format '{{ index .Config.Labels "com.docker.compose.project" }}' "$container_id")"
    [ "$project" = tidewise-infra-uat ] || fail "port-$port" "published outside tidewise-infra-uat"
  done < <(docker ps --filter "publish=$port" --format '{{.ID}}')
  if [ -z "$(docker ps --filter "publish=$port" --format '{{.ID}}')" ] && [ -n "$(ss -H -ltn "sport = :$port")" ]; then
    fail "port-$port" "occupied by a non-Docker listener"
  fi
done
pass loopback-ports

hostname="$(python3 -c 'from urllib.parse import urlparse; import os; print(urlparse(os.environ["RAW_EVIDENCE_PUBLIC_BASE_URL"]).hostname)')"
headers="$(curl -sS --connect-timeout 3 --max-time 5 --resolve "${hostname}:443:127.0.0.1" -D - -o /dev/null "${public_base%/}/raw-evidence/_preflight/not-present" || true)"
grep -Eiq '^X-Tidewise-Upstream:[[:space:]]*minio-uat' <<<"$headers" \
  || fail nginx-route "raw-evidence Nginx route is not installed"
pass nginx-route
