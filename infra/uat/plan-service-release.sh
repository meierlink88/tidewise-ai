#!/usr/bin/env bash

set -euo pipefail

base_sha="${1:-}"
target_sha="${2:-}"

print_plan() {
  local deploy_all="$1"
  local deploy_data="$2"
  local deploy_agentrun="$3"
  local deploy_miniapp="$4"
  local deploy_adminportal="$5"
  local deploy_admin="$6"
  local reason="$7"
  printf '%s\n' \
    "deploy_all=${deploy_all}" \
    "deploy_data=${deploy_data}" \
    "deploy_agentrun=${deploy_agentrun}" \
    "deploy_miniapp=${deploy_miniapp}" \
    "deploy_adminportal=${deploy_adminportal}" \
    "deploy_admin=${deploy_admin}" \
    "scope_reason=${reason}"
}

print_full_plan() {
  print_plan true true true true true true "$1"
  exit 0
}

if ! [[ "$base_sha" =~ ^[0-9a-fA-F]{40}$ ]] || ! git cat-file -e "${base_sha}^{commit}" 2>/dev/null; then
  print_full_plan unavailable_previous_release
fi
if ! [[ "$target_sha" =~ ^[0-9a-fA-F]{40}$ ]] || ! git cat-file -e "${target_sha}^{commit}" 2>/dev/null; then
  print_full_plan unavailable_target_release
fi
if [ "$base_sha" = "$target_sha" ]; then
  print_full_plan explicit_same_release_redeploy
fi
if ! git merge-base --is-ancestor "$base_sha" "$target_sha"; then
  print_full_plan divergent_release_history
fi

deploy_data=false
deploy_agentrun=false
deploy_miniapp=false
deploy_adminportal=false
deploy_admin=false
changed=false
outside_application_directories=false

while IFS= read -r -d '' path; do
  changed=true
  case "$path" in
    data-service/*) deploy_data=true ;;
    agent-run/*) deploy_agentrun=true ;;
    miniapp/backend/*) deploy_miniapp=true ;;
    admin-portal/backend/*) deploy_adminportal=true ;;
    admin-portal/frontend/*) deploy_admin=true ;;
    *) outside_application_directories=true ;;
  esac
done < <(git diff --name-only --no-renames -z "$base_sha" "$target_sha" --)

if [ "$changed" != true ]; then
  print_full_plan empty_release_diff
fi
if [ "$outside_application_directories" = true ]; then
  print_full_plan outside_application_directories
fi

print_plan \
  false \
  "$deploy_data" \
  "$deploy_agentrun" \
  "$deploy_miniapp" \
  "$deploy_adminportal" \
  "$deploy_admin" \
  application_directories_only
