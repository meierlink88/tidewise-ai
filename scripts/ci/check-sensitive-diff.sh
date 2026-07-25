#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
base_sha="${1:-${BASE_SHA:-}}"
head_sha="${2:-${HEAD_SHA:-HEAD}}"

if [[ ! "$base_sha" =~ ^[0-9a-f]{40}$ ]] ||
  [[ "$base_sha" == "0000000000000000000000000000000000000000" ]]; then
  base_sha="$(git -C "$repo_root" rev-list --max-parents=0 "$head_sha" | tail -n 1)"
fi

changed_paths="$(mktemp "${TMPDIR:-/tmp}/tidewise-changed-paths.XXXXXX")"
added_diff="$(mktemp "${TMPDIR:-/tmp}/tidewise-added-diff.XXXXXX")"
cleanup() {
  rm -f -- "$changed_paths" "$added_diff"
}
trap cleanup EXIT

git -C "$repo_root" diff --check "$base_sha" "$head_sha"
git -C "$repo_root" diff --name-only "$base_sha" "$head_sha" >"$changed_paths"

if grep -Eq \
  '^agent-run/backend/(data|\.reference)(/|$)|(^|/)\.env$|(^|/)(\.DS_Store|midscene_run)(/|$)' \
  "$changed_paths"; then
  echo "Diff contains a forbidden AgentRun runtime, credential, or reference path" >&2
  exit 1
fi

git -C "$repo_root" diff --unified=0 "$base_sha" "$head_sha" >"$added_diff"
if grep -Eq \
  '^\+.*(sk-[A-Za-z0-9_-]{16,}|tvly-[A-Za-z0-9_-]{16,}|gh[pousr]_[A-Za-z0-9]{20,}|BEGIN (RSA|OPENSSH|EC) PRIVATE KEY)' \
  "$added_diff"; then
  echo "Diff contains a value matching a credential pattern" >&2
  exit 1
fi

echo "Sensitive diff checks passed"
