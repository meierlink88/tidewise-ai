#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
compose_file="$script_dir/compose.yaml"
runtime_env="${RUNTIME_ENV:?RUNTIME_ENV is required}"
action="${1:-apply}"
mysql_password="${OPENSPG_MYSQL_ROOT_PASSWORD:?OPENSPG_MYSQL_ROOT_PASSWORD is required}"
backup_table='uat_openspg_project_config_backup'
compose=(docker compose --env-file "$runtime_env" -f "$compose_file")

run_mysql() {
  docker exec -i -e MYSQL_PWD="$mysql_password" tidewise-infra-uat-mysql-1 \
    mysql --batch --raw --skip-column-names -uroot openspg
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
  );

UPDATE kg_project_info
SET config = JSON_SET(config, '$.graph_store.database', LOWER(namespace))
WHERE JSON_VALID(config) = 1
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.uri')) IN (
    'neo4j://neo4j:7687',
    'neo4j://release-openspg-neo4j:7687'
  );
SQL

    project_rows="$(run_mysql <<'SQL'
SELECT id, namespace, JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.database'))
FROM kg_project_info
WHERE JSON_VALID(config) = 1
  AND JSON_UNQUOTE(JSON_EXTRACT(config, '$.graph_store.uri')) IN (
    'neo4j://neo4j:7687',
    'neo4j://release-openspg-neo4j:7687'
  )
ORDER BY id;
SQL
)"
    if [ -z "$project_rows" ]; then
      echo 'PASS no UAT OpenSPG project databases are required yet'
    else
      while IFS=$'\t' read -r project_id namespace database; do
        [[ "$database" =~ ^[a-z][a-z0-9.-]{0,62}$ ]] || {
          echo "Unsafe OpenSPG database name for project $project_id ($namespace)" >&2
          exit 1
        }
        expected_database="$(tr '[:upper:]' '[:lower:]' <<<"$namespace")"
        [ "$database" = "$expected_database" ] || {
          echo "Project $project_id database $database does not match namespace $namespace" >&2
          exit 1
        }
        "${compose[@]}" exec -T neo4j bash -c '
          user="${NEO4J_AUTH%%/*}"
          password="${NEO4J_AUTH#*/}"
          cypher-shell -d system -u "$user" -p "$password" \
            "CREATE DATABASE \`$1\` IF NOT EXISTS"
        ' _ "$database" >/dev/null
        echo "PASS prepared OpenSPG project $project_id ($namespace) database $database"
      done <<<"$project_rows"
    fi
    ;;
  restore)
    ensure_backup_table
    run_mysql <<SQL
UPDATE kg_project_info AS project
JOIN ${backup_table} AS backup ON backup.project_id = project.id
SET project.config = backup.config;
SQL
    echo 'PASS restored backed-up UAT OpenSPG project configurations'
    ;;
  *)
    echo "Usage: $0 [apply|restore]" >&2
    exit 2
    ;;
esac
