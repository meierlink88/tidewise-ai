#!/usr/bin/env bash
set -euo pipefail

GITLEAKS_VERSION="8.30.1"
release_base="https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}"

: "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"

sha_pattern='^[0-9a-f]{40}$'
if [[ ! "$BASE_SHA" =~ $sha_pattern ]] || [[ ! "$HEAD_SHA" =~ $sha_pattern ]]; then
  echo "BASE_SHA and HEAD_SHA must be full lowercase Git commit SHAs" >&2
  exit 2
fi

git cat-file -e "${BASE_SHA}^{commit}"
git cat-file -e "${HEAD_SHA}^{commit}"

case "$(uname -s):$(uname -m)" in
  Darwin:arm64)
    archive_name="gitleaks_${GITLEAKS_VERSION}_darwin_arm64.tar.gz"
    expected_sha256="b40ab0ae55c505963e365f271a8d3846efbc170aa17f2607f13df610a9aeb6a5"
    ;;
  Darwin:x86_64)
    archive_name="gitleaks_${GITLEAKS_VERSION}_darwin_x64.tar.gz"
    expected_sha256="dfe101a4db2255fc85120ac7f3d25e4342c3c20cf749f2c20a18081af1952709"
    ;;
  Linux:aarch64 | Linux:arm64)
    archive_name="gitleaks_${GITLEAKS_VERSION}_linux_arm64.tar.gz"
    expected_sha256="e4a487ee7ccd7d3a7f7ec08657610aa3606637dab924210b3aee62570fb4b080"
    ;;
  Linux:x86_64 | Linux:amd64)
    archive_name="gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz"
    expected_sha256="551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb"
    ;;
  *)
    echo "unsupported platform for Gitleaks: $(uname -s) $(uname -m)" >&2
    exit 2
    ;;
esac

temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT

archive_path="${temp_dir}/${archive_name}"
curl --fail --silent --show-error --location --retry 3 \
  --output "$archive_path" "${release_base}/${archive_name}"

if command -v sha256sum >/dev/null 2>&1; then
  actual_sha256="$(sha256sum "$archive_path" | awk '{print $1}')"
else
  actual_sha256="$(shasum -a 256 "$archive_path" | awk '{print $1}')"
fi

if [[ "$actual_sha256" != "$expected_sha256" ]]; then
  echo "Gitleaks archive checksum mismatch" >&2
  exit 2
fi

tar -xzf "$archive_path" -C "$temp_dir" gitleaks

report_path="${GITLEAKS_REPORT_PATH:-${RUNNER_TEMP:-/tmp}/gitleaks-results.sarif}"
mkdir -p "$(dirname "$report_path")"

"${temp_dir}/gitleaks" git \
  --redact=100 \
  --report-format=sarif \
  --report-path="$report_path" \
  --log-opts="--diff-merges=first-parent ${BASE_SHA}..${HEAD_SHA}" \
  .
