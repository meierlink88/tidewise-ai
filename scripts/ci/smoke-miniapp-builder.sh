#!/usr/bin/env bash
set -euo pipefail

output_dir="$(mktemp -d)"
compose_file="infra/local/miniapp-builder.compose.yaml"
project_name="tidewise-miniapp-builder-smoke"

cleanup() {
  docker compose --project-name "$project_name" -f "$compose_file" down --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$output_dir"
}
trap cleanup EXIT

for platform in weapp tt; do
  MINIAPP_FRONTEND_IMAGE=tidewise-miniapp-frontend:ci docker compose \
    --project-name "$project_name" \
    -f "$compose_file" \
    --profile "miniapp-${platform}" \
    run --rm --no-deps \
    --env TARO_APP_RESEARCH_SOURCE=mock \
    --volume "${output_dir}:/workspace/miniapp/frontend/dist" \
    "miniapp-${platform}" \
    npm --workspace @tidewise/miniapp run "build:${platform}"

  if ! find "${output_dir}/${platform}" -type f -print -quit | grep -q .; then
    echo "containerized Miniapp ${platform} build did not write dist/${platform}" >&2
    exit 1
  fi
done

echo "containerized Miniapp weapp and tt builds wrote bind-mounted platform outputs"
