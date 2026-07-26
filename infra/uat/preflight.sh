#!/usr/bin/env bash

set -euo pipefail

deployment_root="${DEPLOY_ROOT:?DEPLOY_ROOT is required}"
expected_runner="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
swr_registry="${SWR_REGISTRY:?SWR_REGISTRY is required}"
public_base_url="${UAT_PUBLIC_BASE_URL:?UAT_PUBLIC_BASE_URL is required}"
agentrun_artifact_gid="10001"

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
import pathlib
import re
import socket
from urllib.parse import urlparse

def database_config(path):
    values = {}
    inside = False
    for line in pathlib.Path(path).read_text().splitlines():
        if line == "database:":
            inside = True
            continue
        if inside and line and not line.startswith(" "):
            break
        if inside:
            match = re.match(r"^  (host|port|name|user|ssl_mode):\s*(.+?)\s*$", line)
            if match:
                values[match.group(1)] = match.group(2)
    required = {"host", "port", "name", "user", "ssl_mode"}
    if set(values) != required:
        raise SystemExit(f"FAIL rds-config: {path} requires {sorted(required)}")
    if values["ssl_mode"] != "require":
        raise SystemExit(f"FAIL rds-config: {path} requires ssl_mode=require")
    return values

paths = (
    "analyse-data-service/backend/configs/config.uat.yaml",
    "agent-run/backend/configs/config.uat.yaml",
)
endpoints = [database_config(path) for path in paths]
data_endpoint, agentrun_endpoint = endpoints
if agentrun_endpoint["name"] != "tidewise_ai_server":
    raise SystemExit("FAIL rds-config: AgentRun database must be tidewise_ai_server")
if data_endpoint["name"] == agentrun_endpoint["name"]:
    raise SystemExit("FAIL rds-config: Data and AgentRun must use different database names")
if data_endpoint["user"] == agentrun_endpoint["user"]:
    raise SystemExit("FAIL rds-config: Data and AgentRun must use different database users")
for endpoint in endpoints:
    with socket.create_connection((endpoint["host"], int(endpoint["port"])), timeout=10):
        pass

public_endpoint = urlparse(os.environ["UAT_PUBLIC_BASE_URL"])
if public_endpoint.scheme != "http" or not public_endpoint.hostname:
    raise SystemExit("FAIL public-base-url: an http URL with hostname is required")
if public_endpoint.port or public_endpoint.path not in ("", "/") or public_endpoint.query or public_endpoint.fragment:
    raise SystemExit("FAIL public-base-url: port, path, query, and fragment are not allowed")
PY
pass data-and-agentrun-rds-private-tcp-and-config
pass public-base-url

artifact_dir="${AGENTRUN_ARTIFACT_DIR:-${deployment_root}/agentrun-artifacts}"
[ -d "$artifact_dir" ] || fail agentrun-artifact-directory "$artifact_dir is missing"
[ -w "$artifact_dir" ] || fail agentrun-artifact-directory "$artifact_dir is not writable"
[ "$(stat -c '%U' "$artifact_dir")" = tidewise-deploy ] || fail agentrun-artifact-directory "$artifact_dir owner must be tidewise-deploy"
[ "$(stat -c '%g' "$artifact_dir")" = "$agentrun_artifact_gid" ] || fail agentrun-artifact-directory "$artifact_dir group must use GID $agentrun_artifact_gid"
[ "$(stat -c '%a' "$artifact_dir")" = 2770 ] || fail agentrun-artifact-directory "$artifact_dir mode must be 2770"
pass agentrun-artifact-directory

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
