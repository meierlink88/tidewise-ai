#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/docker-compose.infra.yaml"
env_file="$script_dir/.env.local"
action="${1:-apply}"
backup_table='local_neo4j_526_project_config_backup'
compose=(docker compose --env-file "$env_file" -f "$compose_file")

run_mysql() {
  "${compose[@]}" exec -T mysql bash -c \
    'mysql --batch --raw --skip-column-names -uroot -p"$MYSQL_ROOT_PASSWORD" openspg'
}

ensure_backup_table() {
  run_mysql <<SQL
CREATE TABLE IF NOT EXISTS ${backup_table} (
  project_id BIGINT NOT NULL PRIMARY KEY,
  config LONGTEXT NOT NULL,
  backed_up_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;
SQL
}

case "$action" in
  apply)
    ensure_backup_table
    run_mysql <<SQL
INSERT IGNORE INTO ${backup_table} (project_id, config)
SELECT id, config
FROM kg_project_info
WHERE JSON_VALID(config) = 1
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.uri')) IN (
    'neo4j://neo4j:7687',
    'neo4j://release-openspg-neo4j:7687'
  )
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.database')) <> 'neo4j';

UPDATE kg_project_info
SET config = JSON_SET(config, '$.graph_store.database', 'neo4j')
WHERE JSON_VALID(config) = 1
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.uri')) IN (
    'neo4j://neo4j:7687',
    'neo4j://release-openspg-neo4j:7687'
  )
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.database')) <> 'neo4j';

SELECT CONCAT('backup_projects=', COUNT(*)) FROM ${backup_table};
SQL
    ;;
  restore)
    ensure_backup_table
    run_mysql <<SQL
UPDATE kg_project_info AS project
JOIN ${backup_table} AS backup ON backup.project_id = project.id
SET project.config = JSON_SET(
  project.config,
  '$.graph_store.database',
  JSON_UNQUOTE(JSON_EXTRACT(backup.config, '$.graph_store.database'))
);

SELECT CONCAT('restored_projects=', COUNT(*)) FROM ${backup_table};
SQL
    ;;
  *)
    echo "Usage: $0 [apply|restore]" >&2
    exit 2
    ;;
esac
