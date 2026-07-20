#!/usr/bin/env bash

set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "bootstrap-ecs.sh must be run manually as root" >&2
  exit 1
fi
if [ "$(uname -s)" != Linux ] || [ "$(uname -m)" != x86_64 ]; then
  echo "UAT bootstrap requires Linux x86_64" >&2
  exit 1
fi
if ! grep -q '^VERSION_ID="24.04"' /etc/os-release; then
  echo "UAT bootstrap requires Ubuntu 24.04" >&2
  exit 1
fi

deploy_user="${TIDEWISE_DEPLOY_USER:-tidewise-deploy}"
deploy_root="${TIDEWISE_DEPLOY_ROOT:-/opt/tidewise/uat}"
runner_root="${TIDEWISE_RUNNER_ROOT:-/opt/tidewise/actions-runner}"
runner_name="${UAT_RUNNER_NAME:?UAT_RUNNER_NAME is required}"
repository_url="${GITHUB_REPOSITORY_URL:?GITHUB_REPOSITORY_URL is required}"
registration_token="${GITHUB_RUNNER_REGISTRATION_TOKEN:?GITHUB_RUNNER_REGISTRATION_TOKEN is required}"
runner_archive="${ACTIONS_RUNNER_ARCHIVE:?ACTIONS_RUNNER_ARCHIVE is required}"
runner_archive_sha256="${ACTIONS_RUNNER_ARCHIVE_SHA256:?ACTIONS_RUNNER_ARCHIVE_SHA256 is required}"

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates curl git python3 util-linux iproute2 docker.io docker-compose-v2
systemctl enable --now docker.service

if ! id "$deploy_user" >/dev/null 2>&1; then
  useradd --create-home --shell /bin/bash "$deploy_user"
fi
usermod -aG docker "$deploy_user"

install -d -m 0750 -o "$deploy_user" -g "$deploy_user" "$deploy_root" "$deploy_root/state"

printf '%s  %s\n' "$runner_archive_sha256" "$runner_archive" | sha256sum --check --status

if [ -n "${OLD_RUNNER_ROOT:-}" ] && [ -x "${OLD_RUNNER_ROOT}/svc.sh" ]; then
  (
    cd "$OLD_RUNNER_ROOT"
    ./svc.sh stop || true
    ./svc.sh uninstall || true
  )
elif [ -n "${OLD_RUNNER_SERVICE:-}" ]; then
  systemctl disable --now "$OLD_RUNNER_SERVICE"
fi

if [ ! -x "$runner_root/config.sh" ]; then
  install -d -m 0750 -o "$deploy_user" -g "$deploy_user" "$runner_root"
  tar -xzf "$runner_archive" -C "$runner_root"
  chown -R "$deploy_user:$deploy_user" "$runner_root"
fi

if [ ! -f "$runner_root/.runner" ]; then
  (
    cd "$runner_root"
    runuser -u "$deploy_user" -- ./config.sh \
      --url "$repository_url" \
      --token "$registration_token" \
      --name "$runner_name" \
      --labels tidewise-uat-ecs \
      --unattended \
      --replace
  )
fi

(
  cd "$runner_root"
  if [ ! -f .service ]; then
    ./svc.sh install "$deploy_user"
  fi
  ./svc.sh start
)

runner_unit="$(systemctl list-unit-files 'actions.runner.*.service' --state=enabled --no-legend | awk 'NR == 1 {print $1}')"
systemctl is-active "$runner_unit" >/dev/null
systemctl is-enabled docker.service >/dev/null

echo "UAT ECS bootstrap complete. Re-login is required before ${deploy_user} receives docker group membership."
echo "Rotate any previously exposed root password and verify cloud security-group/RDS rules separately."
