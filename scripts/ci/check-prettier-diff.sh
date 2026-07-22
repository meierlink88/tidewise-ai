#!/usr/bin/env bash
set -euo pipefail

: "${BASE_SHA:?BASE_SHA is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"

if [[ "$BASE_SHA" =~ ^0+$ ]]; then
  BASE_SHA="$(git rev-list --max-parents=0 "$HEAD_SHA")"
fi

diff_range=("$BASE_SHA")
if [[ "$HEAD_SHA" != "WORKTREE" ]]; then
  diff_range+=("$HEAD_SHA")
fi

files=()
while IFS= read -r -d '' file; do
  case "$file" in
    .github/*.yml | .github/*.yaml | package.json | package-lock.json | src/frontend/*.js | src/frontend/*.mjs | src/frontend/*.cjs | src/frontend/*.ts | src/frontend/*.tsx | src/frontend/*.json | src/frontend/*.css | src/frontend/*.scss | src/frontend/*.html)
      files+=("$file")
      ;;
  esac
done < <(git diff --name-only --diff-filter=ACMR -z "${diff_range[@]}")

if (( ${#files[@]} == 0 )); then
  echo "No changed Prettier-managed files"
  exit 0
fi

npx --no-install prettier --check "${files[@]}"
