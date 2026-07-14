package seed

import (
	"context"
	"database/sql"
	"fmt"
)

type PhaseAPreflightReference struct {
	Kind       string `json:"kind"`
	ObjectName string `json:"object_name"`
	Definition string `json:"definition"`
}

type PhaseAProtectedEntityBaseline struct {
	RowCount int64  `json:"row_count"`
	Checksum string `json:"checksum"`
}

type PhaseAPreflightReport struct {
	DatabaseName              string                                   `json:"database_name"`
	ServerVersion             string                                   `json:"server_version"`
	GooseVersion              int64                                    `json:"goose_version"`
	Metrics                   map[string]int64                         `json:"metrics"`
	References                []PhaseAPreflightReference               `json:"references"`
	ProtectedEntityBaseline   map[string]PhaseAProtectedEntityBaseline `json:"protected_entity_baseline"`
	EntityKeyGlobalUniqueSafe bool                                     `json:"entity_key_global_unique_safe"`
	BackupVerified            bool                                     `json:"backup_verified"`
	BackupStatus              string                                   `json:"backup_status"`
}

const phaseAPreflightMetricsSQL = `WITH metrics(metric, value) AS (
    SELECT 'entity_type.' || entity_type, count(*)::bigint FROM entity_nodes GROUP BY entity_type
    UNION ALL SELECT 'entity_nodes.total', count(*)::bigint FROM entity_nodes
    UNION ALL SELECT 'status.merged', count(*)::bigint FROM entity_nodes WHERE status = 'merged'
    UNION ALL SELECT 'entity_key.blank', count(*)::bigint FROM entity_nodes WHERE entity_key IS NULL OR btrim(entity_key) = ''
    UNION ALL SELECT 'entity_key.duplicate_groups', count(*)::bigint FROM (SELECT entity_key FROM entity_nodes GROUP BY entity_key HAVING count(*) > 1) duplicate_keys
    UNION ALL SELECT 'profile.chain_node', count(*)::bigint FROM chain_node_profiles
    UNION ALL SELECT 'profile.theme', count(*)::bigint FROM theme_profiles
    UNION ALL SELECT 'external_identifier.total', count(*)::bigint FROM entity_external_identifiers
    UNION ALL SELECT 'entity_edge.total', count(*)::bigint FROM entity_edges
    UNION ALL SELECT 'event_entity_link.total', count(*)::bigint FROM event_entity_links
    UNION ALL SELECT 'chain_node.definition_blank', count(*)::bigint FROM chain_node_profiles WHERE definition IS NULL OR btrim(definition) = ''
    UNION ALL SELECT 'orphan.chain_node_profile', count(*)::bigint FROM chain_node_profiles p LEFT JOIN entity_nodes n ON n.id = p.entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'orphan.theme_profile', count(*)::bigint FROM theme_profiles p LEFT JOIN entity_nodes n ON n.id = p.entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'orphan.external_identifier', count(*)::bigint FROM entity_external_identifiers i LEFT JOIN entity_nodes n ON n.id = i.entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'orphan.event_entity_link', count(*)::bigint FROM event_entity_links l LEFT JOIN entity_nodes n ON n.id = l.entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'orphan.entity_edge_from', count(*)::bigint FROM entity_edges e LEFT JOIN entity_nodes n ON n.id = e.from_entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'orphan.entity_edge_to', count(*)::bigint FROM entity_edges e LEFT JOIN entity_nodes n ON n.id = e.to_entity_id WHERE n.id IS NULL
    UNION ALL SELECT 'schema.retired_relation_exists', count(*)::bigint FROM unnest(ARRAY['sector_profiles','sector_source_mappings','industry_chain_profiles','industry_chain_memberships','industry_chain_topology_edges','industry_chain_physical_constraints','entity_convergence_manifests','entity_convergences','entity_convergence_reference_moves','entity_convergence_alias_moves']) relation_name WHERE to_regclass('public.' || relation_name) IS NOT NULL
    UNION ALL SELECT 'catalog.legacy_function_reference', count(*)::bigint FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE p.prokind IN ('f','p') AND n.nspname NOT IN ('pg_catalog','information_schema') AND pg_get_functiondef(p.oid) ~* '(sector_profiles|sector_source_mappings|industry_chain|entity_convergence)'
    UNION ALL SELECT 'catalog.legacy_trigger_reference', count(*)::bigint FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid WHERE NOT t.tgisinternal AND pg_get_triggerdef(t.oid) ~* '(sector_profiles|sector_source_mappings|industry_chain|entity_convergence)'
    UNION ALL SELECT 'catalog.legacy_view_reference', count(*)::bigint FROM pg_views WHERE schemaname NOT IN ('pg_catalog','information_schema') AND definition ~* '(sector_profiles|sector_source_mappings|industry_chain|entity_convergence)'
    UNION ALL SELECT 'catalog.legacy_rule_reference', count(*)::bigint FROM pg_rules WHERE schemaname NOT IN ('pg_catalog','information_schema') AND definition ~* '(sector_profiles|sector_source_mappings|industry_chain|entity_convergence)'
)
SELECT metric, value FROM metrics ORDER BY metric`

const phaseAPreflightReferencesSQL = `WITH target_tables(name) AS (
    VALUES
        ('entity_nodes'),
        ('chain_node_profiles'),
        ('theme_profiles'),
        ('entity_external_identifiers')
), catalog_references(reference_kind, object_name, definition) AS (
    SELECT
        'foreign_key',
        source_namespace.nspname || '.' || constraint_name.conname,
        source_table.relname || ' -> ' || target_table.relname || ': ' || pg_get_constraintdef(constraint_name.oid)
    FROM pg_constraint constraint_name
    JOIN pg_class source_table ON source_table.oid = constraint_name.conrelid
    JOIN pg_namespace source_namespace ON source_namespace.oid = source_table.relnamespace
    JOIN pg_class target_table ON target_table.oid = constraint_name.confrelid
    WHERE constraint_name.contype = 'f'
      AND (source_table.relname IN (SELECT name FROM target_tables)
        OR target_table.relname IN (SELECT name FROM target_tables))
    UNION ALL
    SELECT
        'trigger',
        table_namespace.nspname || '.' || trigger_name.tgname,
        pg_get_triggerdef(trigger_name.oid)
    FROM pg_trigger trigger_name
    JOIN pg_class trigger_table ON trigger_table.oid = trigger_name.tgrelid
    JOIN pg_namespace table_namespace ON table_namespace.oid = trigger_table.relnamespace
    WHERE NOT trigger_name.tgisinternal
      AND trigger_table.relname IN (SELECT name FROM target_tables)
    UNION ALL
    SELECT
        CASE WHEN procedure_name.prokind = 'p' THEN 'procedure' ELSE 'function' END,
        procedure_namespace.nspname || '.' || procedure_name.proname,
        pg_get_functiondef(procedure_name.oid)
    FROM pg_proc procedure_name
    JOIN pg_namespace procedure_namespace ON procedure_namespace.oid = procedure_name.pronamespace
    WHERE procedure_name.prokind IN ('f', 'p')
      AND procedure_namespace.nspname NOT IN ('pg_catalog', 'information_schema')
      AND pg_get_functiondef(procedure_name.oid) ~* '(chain_node_profiles|theme_profiles|entity_external_identifiers)'
    UNION ALL
    SELECT
        'view',
        schemaname || '.' || viewname,
        definition
    FROM pg_views
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
      AND definition ~* '(chain_node_profiles|theme_profiles|entity_external_identifiers)'
    UNION ALL
    SELECT
        'rule',
        schemaname || '.' || rulename,
        definition
    FROM pg_rules
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
      AND definition ~* '(chain_node_profiles|theme_profiles|entity_external_identifiers)'
)
SELECT reference_kind, object_name, definition
FROM catalog_references
ORDER BY reference_kind, object_name`

const phaseAPreflightProtectedBaselineSQL = `SELECT
    entity_type,
    count(*)::bigint,
    md5(COALESCE(string_agg(
        concat_ws(E'\x1f', id::text, entity_key, layer_code, name, canonical_name, aliases::text, status),
        E'\x1e' ORDER BY id
    ), '')) AS checksum
FROM entity_nodes
WHERE entity_type NOT IN ('sector', 'industry_chain', 'chain_node')
GROUP BY entity_type
ORDER BY entity_type`

func (r PostgresRepository) RunPhaseAPreflight(ctx context.Context) (PhaseAPreflightReport, error) {
	if r.root == nil {
		return PhaseAPreflightReport{}, fmt.Errorf("phase A preflight requires a transactional PostgreSQL repository")
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("begin read-only phase A preflight: %w", err)
	}
	defer tx.Rollback()

	report := PhaseAPreflightReport{
		Metrics:                 map[string]int64{},
		ProtectedEntityBaseline: map[string]PhaseAProtectedEntityBaseline{},
	}
	if err := tx.QueryRowContext(ctx, "SELECT current_database(), version()").Scan(&report.DatabaseName, &report.ServerVersion); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query current database identity: %w", err)
	}
	if err := tx.QueryRowContext(ctx, "SELECT version_id FROM goose_db_version ORDER BY id DESC LIMIT 1").Scan(&report.GooseVersion); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query current goose version: %w", err)
	}
	rows, err := tx.QueryContext(ctx, phaseAPreflightMetricsSQL)
	if err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query phase A preflight metrics: %w", err)
	}
	for rows.Next() {
		var metric string
		var value int64
		if err := rows.Scan(&metric, &value); err != nil {
			rows.Close()
			return PhaseAPreflightReport{}, fmt.Errorf("scan phase A preflight metric: %w", err)
		}
		report.Metrics[metric] = value
	}
	if err := rows.Close(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("close phase A preflight metrics: %w", err)
	}
	if err := rows.Err(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("read phase A preflight metrics: %w", err)
	}

	referenceRows, err := tx.QueryContext(ctx, phaseAPreflightReferencesSQL)
	if err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query phase A preflight references: %w", err)
	}
	for referenceRows.Next() {
		var reference PhaseAPreflightReference
		if err := referenceRows.Scan(&reference.Kind, &reference.ObjectName, &reference.Definition); err != nil {
			referenceRows.Close()
			return PhaseAPreflightReport{}, fmt.Errorf("scan phase A preflight reference: %w", err)
		}
		report.References = append(report.References, reference)
	}
	if err := referenceRows.Close(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("close phase A preflight references: %w", err)
	}
	if err := referenceRows.Err(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("read phase A preflight references: %w", err)
	}

	baselineRows, err := tx.QueryContext(ctx, phaseAPreflightProtectedBaselineSQL)
	if err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query protected entity baseline: %w", err)
	}
	for baselineRows.Next() {
		var entityType string
		var baseline PhaseAProtectedEntityBaseline
		if err := baselineRows.Scan(&entityType, &baseline.RowCount, &baseline.Checksum); err != nil {
			baselineRows.Close()
			return PhaseAPreflightReport{}, fmt.Errorf("scan protected entity baseline: %w", err)
		}
		report.ProtectedEntityBaseline[entityType] = baseline
	}
	if err := baselineRows.Close(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("close protected entity baseline: %w", err)
	}
	if err := baselineRows.Err(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("read protected entity baseline: %w", err)
	}

	report.EntityKeyGlobalUniqueSafe = report.Metrics["entity_key.blank"] == 0 && report.Metrics["entity_key.duplicate_groups"] == 0
	var archiveMode string
	if err := tx.QueryRowContext(ctx, "SELECT current_setting('archive_mode')").Scan(&archiveMode); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("query backup boundary: %w", err)
	}
	report.BackupStatus = "archive_mode=" + archiveMode + "; external backup has not been verified in read-only preflight"
	if err := tx.Commit(); err != nil {
		return PhaseAPreflightReport{}, fmt.Errorf("commit read-only phase A preflight: %w", err)
	}
	return report, nil
}
