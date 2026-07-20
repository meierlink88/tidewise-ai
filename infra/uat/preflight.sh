#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
expected_runner="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
swr_registry="${SWR_REGISTRY:?SWR_REGISTRY is required}"
public_base_url="${UAT_PUBLIC_BASE_URL:?UAT_PUBLIC_BASE_URL is required}"

pass() {
  echo "PASS $1"
}

fail() {
  echo "FAIL $1: $2" >&2
  exit 1
}

[ "$(uname -s)" = Linux ] || fail os "expected Linux"
[ "$(uname -m)" = x86_64 ] || fail architecture "expected x86_64"
pass os-architecture

[ "$(id -un)" = tidewise-deploy ] || fail deploy-user "expected tidewise-deploy"
[ "${RUNNER_NAME:-}" = "$expected_runner" ] || fail runner-name "expected $expected_runner"
[ "${RUNNER_OS:-Linux}" = Linux ] || fail runner-os "expected Linux"
[ "${RUNNER_ARCH:-X64}" = X64 ] || fail runner-arch "expected X64"
pass runner-identity
pass runner-label-route

for command in docker git curl python3 flock ss systemctl; do
  command -v "$command" >/dev/null || fail dependency "$command is missing"
done
docker info >/dev/null || fail docker-engine "docker info failed"
docker compose version >/dev/null || fail docker-compose "Docker Compose v2 is unavailable"
systemctl is-enabled docker.service >/dev/null || fail docker-autostart "docker.service is not enabled"
runner_unit="$(systemctl list-unit-files 'actions.runner.*.service' --state=enabled --no-legend | awk 'NR == 1 {print $1}')"
[ -n "$runner_unit" ] || fail runner-autostart "no enabled actions.runner service found"
systemctl is-active "$runner_unit" >/dev/null || fail runner-service "${runner_unit} is not active"
pass docker-compose-runner-services

[ -d "$deployment_root" ] || fail deploy-directory "$deployment_root is missing"
[ -d "$deployment_root/state" ] || fail state-directory "$deployment_root/state is missing"
[ -w "$deployment_root/state" ] || fail state-directory "$deployment_root/state is not writable"
[ "$(stat -c '%U' "$deployment_root")" = tidewise-deploy ] || fail deploy-directory "$deployment_root owner must be tidewise-deploy"
pass directory-permissions

available_kb="$(df -Pk "$deployment_root" | awk 'NR == 2 {print $4}')"
[ "$available_kb" -ge 10485760 ] || fail disk-space "at least 10 GiB is required"
pass disk-space

curl --fail --silent --show-error --connect-timeout 5 --max-time 15 https://github.com/ >/dev/null || fail github-https "github.com is unavailable"
curl --fail --silent --show-error --connect-timeout 5 --max-time 15 https://api.github.com/ >/dev/null || fail github-api "api.github.com is unavailable"
git ls-remote https://github.com/actions/checkout.git HEAD >/dev/null || fail github-git "checkout repository is unavailable"
pass github-connectivity

swr_status="$(curl --silent --show-error --connect-timeout 5 --max-time 15 --output /dev/null --write-out '%{http_code}' "https://${swr_registry}/v2/")"
case "$swr_status" in
  200|401) pass swr-registry-endpoint ;;
  *) fail swr-registry-endpoint "unexpected HTTP status $swr_status" ;;
esac

python3 - <<'PY'
import os
import socket
from urllib.parse import parse_qsl, urlparse

endpoint = urlparse(os.environ["UAT_DATABASE_URL"])
if not endpoint.hostname or not endpoint.path.strip("/") or not endpoint.username:
    raise SystemExit("FAIL rds-url: hostname, database, and username are required")
query = dict(parse_qsl(endpoint.query))
if query.get("sslmode") != "require":
    raise SystemExit("FAIL rds-url: sslmode=require is required")
with socket.create_connection((endpoint.hostname, endpoint.port or 5432), timeout=10):
    pass

public_endpoint = urlparse(os.environ["UAT_PUBLIC_BASE_URL"])
if public_endpoint.scheme != "http" or not public_endpoint.hostname:
    raise SystemExit("FAIL public-base-url: an http URL with hostname is required")
if public_endpoint.port or public_endpoint.path not in ("", "/") or public_endpoint.query or public_endpoint.fragment:
    raise SystemExit("FAIL public-base-url: port, path, query, and fragment are not allowed")
PY
pass rds-private-tcp-and-url
pass public-base-url

for port in 9012 9013 9014; do
  container_ids="$(docker ps --filter "publish=$port" --format '{{.ID}}')"
  while read -r container_id; do
    [ -z "$container_id" ] && continue
    project="$(docker inspect --format '{{ index .Config.Labels "com.docker.compose.project" }}' "$container_id")"
    [ "$project" = tidewise-uat ] || fail port-$port "published by a container outside tidewise-uat"
  done <<< "$container_ids"
  if [ -z "$container_ids" ] && [ -n "$(ss -H -ltn "sport = :$port")" ]; then
    fail port-$port "occupied by a non-Docker listener"
  fi
  pass port-$port
done
