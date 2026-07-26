#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
release_sha="${GITHUB_SHA:-$(git -C "$repository_root" rev-parse HEAD)}"
control_plane_sha="$release_sha"
bundle_root="$(mktemp -d "${TMPDIR:-/tmp}/tidewise-uat-bundle-stage.XXXXXX")"
extract_root="$(mktemp -d "${TMPDIR:-/tmp}/tidewise-uat-bundle-extract.XXXXXX")"
image_tag="tidewise-uat-deploy-bundle:smoke-${release_sha:0:12}"
container_id=""

cleanup() {
  if [ -n "$container_id" ]; then
    docker rm -f "$container_id" >/dev/null 2>&1 || true
  fi
  docker image rm "$image_tag" >/dev/null 2>&1 || true
  rm -rf "$bundle_root" "$extract_root"
}
trap cleanup EXIT

"$repository_root/infra/uat/stage-deploy-bundle.sh" \
  "$repository_root" \
  "$repository_root" \
  "$bundle_root" \
  "$release_sha" \
  "$control_plane_sha"
docker build \
  --tag "$image_tag" \
  --file "$repository_root/infra/uat/deploy-bundle.Dockerfile" \
  "$bundle_root"
container_id="$(docker create "$image_tag")"
docker cp "${container_id}:/bundle/." "$extract_root"
(
  cd "$extract_root"
  sha256sum --check SHA256SUMS
)
grep -Fx "RELEASE_SHA=$release_sha" "$extract_root/metadata.env" >/dev/null
grep -Fx "CONTROL_PLANE_SHA=$control_plane_sha" "$extract_root/metadata.env" >/dev/null
echo "PASS UAT deployment bundle container smoke"
