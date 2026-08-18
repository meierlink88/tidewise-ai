#!/usr/bin/env bash

validate_neo4j_backup_target() {
  local target="$1"
  [[ "$target" =~ ^/opt/tidewise/neo4j-uat/backups/[0-9]{8}T[0-9]{6}Z$ ]]
}

restore_neo4j_files() {
  local data_dir="$1"
  local config_dir="$2"
  local plugins_dir="$3"
  local backup_dir="$4"
  local required

  for required in data etc-neo4j plugins; do
    if [ ! -d "$backup_dir/$required" ]; then
      echo "FAIL rollback: missing $backup_dir/$required" >&2
      return 1
    fi
  done
  for required in failed-data failed-etc-neo4j failed-plugins; do
    if [ -e "$backup_dir/$required" ]; then
      echo "FAIL rollback: recovery target already exists: $backup_dir/$required" >&2
      return 1
    fi
  done

  if [ -d "$data_dir" ]; then
    mv "$data_dir" "$backup_dir/failed-data" || return 1
  fi
  mv "$backup_dir/data" "$data_dir" || return 1
  if [ -d "$config_dir" ]; then
    mv "$config_dir" "$backup_dir/failed-etc-neo4j" || return 1
  fi
  cp -a "$backup_dir/etc-neo4j" "$config_dir" || return 1
  if [ -d "$plugins_dir" ]; then
    mv "$plugins_dir" "$backup_dir/failed-plugins" || return 1
  fi
  cp -a "$backup_dir/plugins" "$plugins_dir"
}

restore_neo4j_service() {
  local data_dir="$1"
  local config_dir="$2"
  local plugins_dir="$3"
  local backup_dir="$4"

  systemctl stop neo4j || return 1
  restore_neo4j_files "$data_dir" "$config_dir" "$plugins_dir" "$backup_dir" || return 1
  chown -R neo4j:neo4j "$data_dir" "$plugins_dir" || return 1
  systemctl start neo4j
}

recover_neo4j_after_failure() {
  local mutation_started="$1"
  local stop_attempted="$2"
  local data_dir="$3"
  local config_dir="$4"
  local plugins_dir="$5"
  local backup_dir="$6"

  if [ "$mutation_started" = true ]; then
    restore_neo4j_service "$data_dir" "$config_dir" "$plugins_dir" "$backup_dir"
  elif [ "$stop_attempted" = true ]; then
    systemctl start neo4j
  fi
}
