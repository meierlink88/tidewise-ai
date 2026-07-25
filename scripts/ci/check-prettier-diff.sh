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
while IFS= read -r -d '' status; do
  if [[ "$status" == R* ]]; then
    IFS= read -r -d '' old_file
    IFS= read -r -d '' file
    if [[ "$status" == "R100" ]]; then
      continue
    fi
  else
    IFS= read -r -d '' file
  fi

  case "$file" in
    .github/*.yml | .github/*.yaml | package.json | package-lock.json | miniapp/frontend/*.js | miniapp/frontend/*.mjs | miniapp/frontend/*.cjs | miniapp/frontend/*.ts | miniapp/frontend/*.tsx | miniapp/frontend/*.json | miniapp/frontend/*.css | miniapp/frontend/*.scss | miniapp/frontend/*.html | admin-portal/frontend/*.js | admin-portal/frontend/*.mjs | admin-portal/frontend/*.cjs | admin-portal/frontend/*.ts | admin-portal/frontend/*.tsx | admin-portal/frontend/*.json | admin-portal/frontend/*.css | admin-portal/frontend/*.scss | admin-portal/frontend/*.html)
      files+=("$file")
      ;;
  esac
done < <(git diff --name-status --find-renames --diff-filter=ACMR -z "${diff_range[@]}")

if (( ${#files[@]} == 0 )); then
  echo "No changed Prettier-managed files"
  exit 0
fi

npx --no-install prettier --check "${files[@]}"
