package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

const contractVersion = "uat-excluded-fact-audit.v1"

const emptyTableFingerprint = "d41d8cd98f00b204e9800998ecf8427e"

var auditedTables = []string{
	"event_entity_links",
	"event_publication_receipts",
	"event_semantic_candidate_snapshots",
	"event_semantic_context_leases",
	"event_semantic_resolution_bindings",
	"event_semantic_review_snapshots",
	"event_semantic_submissions",
	"event_sources",
	"event_tag_defs",
	"event_tag_maps",
	"events",
	"raw_documents",
	"research_reasoning_tree_events",
	"research_reasoning_tree_import_receipts",
	"research_reasoning_tree_node_signals",
	"research_reasoning_tree_nodes",
	"research_reasoning_trees",
	"research_theme_events",
	"research_theme_impacts",
	"research_theme_import_receipts",
	"research_themes",
}

type tableAudit struct {
	RowCount    int64  `json:"row_count"`
	Fingerprint string `json:"fingerprint"`
}

type auditReport struct {
	ContractVersion string                `json:"contract_version"`
	Tables          map[string]tableAudit `json:"tables"`
}

func main() {
	if err := execute(os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func execute(output io.Writer) error {
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return fmt.Errorf("load Data configuration: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := postgres.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open PostgreSQL for excluded fact audit: %w", err)
	}
	defer database.Close()
	return run(ctx, database, output)
}

func run(ctx context.Context, database *sql.DB, output io.Writer) error {
	if database == nil || output == nil {
		return fmt.Errorf("excluded fact audit dependencies are required")
	}
	report := auditReport{
		ContractVersion: contractVersion,
		Tables:          make(map[string]tableAudit, len(auditedTables)),
	}
	for _, table := range auditedTables {
		value, err := auditTable(ctx, database, table)
		if err != nil {
			return fmt.Errorf("audit excluded fact table %s: %w", table, err)
		}
		report.Tables[table] = value
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func auditTable(ctx context.Context, database *sql.DB, table string) (tableAudit, error) {
	var relation sql.NullString
	if err := database.QueryRowContext(ctx, "SELECT to_regclass($1)", table).Scan(&relation); err != nil {
		return tableAudit{}, err
	}
	if !relation.Valid {
		return tableAudit{Fingerprint: emptyTableFingerprint}, nil
	}
	var value tableAudit
	if err := database.QueryRowContext(ctx, auditQuery(table)).Scan(&value.RowCount, &value.Fingerprint); err != nil {
		return tableAudit{}, err
	}
	return value, nil
}

func auditQuery(table string) string {
	return fmt.Sprintf(`
		SELECT count(*),
		       COALESCE(md5(string_agg(row_fingerprint, '' ORDER BY row_fingerprint)), md5(''))
		FROM (
		    SELECT md5((%s)::text) AS row_fingerprint
		    FROM %s source_row
		) audited_rows
	`, auditRowExpression(table), table)
}

func auditRowExpression(table string) string {
	switch table {
	case "events":
		return "to_jsonb(source_row) - 'primary_source_id'"
	case "event_sources":
		return `(to_jsonb(source_row) - 'evidence_excerpt' - 'is_primary' - 'contract_version')
		        || jsonb_build_object(
		            'evidence_statement', COALESCE(
		                to_jsonb(source_row) -> 'evidence_statement',
		                to_jsonb(source_row) -> 'evidence_excerpt',
		                'null'::jsonb
		            ),
		            'contract_version', CASE
		                WHEN to_jsonb(source_row) ->> 'contract_version' = '2' THEN '3'::jsonb
		                ELSE to_jsonb(source_row) -> 'contract_version'
		            END
		        )`
	case "event_semantic_context_leases":
		return `to_jsonb(source_row)
		        || jsonb_build_object(
		            'context_manifest', to_jsonb(source_row) -> 'context_manifest'
		        )`
	default:
		return "to_jsonb(source_row)"
	}
}
