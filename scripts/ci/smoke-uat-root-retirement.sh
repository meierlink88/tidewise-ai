#!/usr/bin/env bash

set -Eeuo pipefail

data_image="${1:?usage: smoke-uat-root-retirement.sh DATA_IMAGE}"
smoke_root="$(mktemp -d "${TMPDIR:-/tmp}/tidewise-root-retirement.XXXXXXXX")"
fixture_root="${smoke_root}/host"
helper_binary="${smoke_root}/uat-root-retirement"
fake_systemctl="${smoke_root}/systemctl"

cleanup() {
  find -P "$smoke_root" -depth -delete
}
trap cleanup EXIT

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -o "$helper_binary" \
  ./infra/uat/cmd/uat-root-retirement
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -o "$fake_systemctl" \
  ./scripts/ci/testdata/uat-fake-systemctl

mkdir -p \
  "${fixture_root}/usr/bin" \
  "${fixture_root}/usr/lib/systemd/system" \
  "${fixture_root}/etc/systemd/system" \
  "${fixture_root}/opt/tidewise/agentos-uat" \
  "${fixture_root}/opt/tidewise/uat/agentrun-artifacts" \
  "${fixture_root}/opt/tidewise/uat/logs/agentrun" \
  "${fixture_root}/opt/tidewise/uat/state" \
  "${fixture_root}/opt/tidewise/reason-uat" \
  "${fixture_root}/opt/tidewise/neo4j-uat"
cp "$fake_systemctl" "${fixture_root}/usr/bin/systemctl"
printf 'unit\n' > "${fixture_root}/etc/systemd/system/actions.runner.meierlink88-tidewise-agent-os.tidewise-agentos-uat-ecs.service"
printf 'unit\n' > "${fixture_root}/etc/systemd/system/actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service"
printf 'unit\n' > "${fixture_root}/usr/lib/systemd/system/neo4j.service"
printf 'enabled\n' > "${fixture_root}/etc/systemd/system/.uat-fake-agentos-enabled"
printf 'enabled\n' > "${fixture_root}/etc/systemd/system/.uat-fake-reason-enabled"
printf 'enabled\n' > "${fixture_root}/etc/systemd/system/.uat-fake-neo4j-enabled"
printf 'retired\n' > "${fixture_root}/opt/tidewise/agentos-uat/fixture"
printf 'retired\n' > "${fixture_root}/opt/tidewise/uat/agentrun-artifacts/fixture"
printf 'retired\n' > "${fixture_root}/opt/tidewise/uat/logs/agentrun/fixture"
printf 'retired\n' > "${fixture_root}/opt/tidewise/reason-uat/fixture"
printf 'retired\n' > "${fixture_root}/opt/tidewise/neo4j-uat/fixture"
printf 'retained\n' > "${fixture_root}/opt/tidewise/uat/state/current.sha"

run_helper() {
  local action="$1"
  docker run --rm \
    --user 0:0 \
    --privileged \
    --pid host \
    --network none \
    --read-only \
    --mount "type=bind,source=${fixture_root},target=/host,readonly" \
    --mount "type=bind,source=${fixture_root}/etc/systemd/system,target=/host/etc/systemd/system" \
    --mount "type=bind,source=${fixture_root}/opt/tidewise,target=/host/opt/tidewise" \
    --mount "type=bind,source=${helper_binary},target=/uat-root-retirement,readonly" \
    --entrypoint /uat-root-retirement \
    "$data_image" \
    "$action"
}

run_helper preflight
run_helper apply

for retired_path in \
  /opt/tidewise/agentos-uat \
  /opt/tidewise/uat/agentrun-artifacts \
  /opt/tidewise/uat/logs/agentrun \
  /opt/tidewise/reason-uat \
  /opt/tidewise/neo4j-uat \
  /etc/systemd/system/actions.runner.meierlink88-tidewise-agent-os.tidewise-agentos-uat-ecs.service \
  /etc/systemd/system/actions.runner.meierlink88-tidewise-reason.tidewise-reason-uat-ecs.service; do
  test ! -e "${fixture_root}${retired_path}"
done
test -f "${fixture_root}/usr/lib/systemd/system/neo4j.service"
test ! -e "${fixture_root}/etc/systemd/system/.uat-fake-agentos-enabled"
test ! -e "${fixture_root}/etc/systemd/system/.uat-fake-reason-enabled"
test ! -e "${fixture_root}/etc/systemd/system/.uat-fake-neo4j-enabled"
grep -qx retained "${fixture_root}/opt/tidewise/uat/state/current.sha"

echo "PASS UAT root-retirement container smoke"
