package seed

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPhaseAPreflightSQLIsReadOnlyAndCoversMigrationRisks(t *testing.T) {
	sql := strings.ToLower(phaseAPreflightMetricsSQL + phaseAPreflightReferencesSQL + phaseAPreflightProtectedBaselineSQL)
	if strings.Contains(sql, "), references(") {
		t.Fatal("preflight SQL must not use PostgreSQL reserved keyword references as a CTE name")
	}
	for _, required := range []string{
		"entity_nodes", "sector_profiles", "chain_node_profiles", "industry_chain_profiles",
		"industry_chain_memberships", "industry_chain_topology_edges", "industry_chain_physical_constraints",
		"event_entity_links", "entity_edges", "entity_convergence_manifests", "entity_convergences",
		"entity_convergence_reference_moves", "entity_convergence_alias_moves",
		"pg_constraint", "pg_trigger", "pg_proc", "pg_views", "pg_rules",
		"entity_key.blank", "entity_key.duplicate_groups", "orphan", "status.merged",
		"md5", "string_agg", "not in ('sector', 'industry_chain', 'chain_node')",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("preflight SQL missing %q", required)
		}
	}
	for _, forbidden := range []string{"insert ", "update ", "delete ", "alter ", "drop ", "truncate ", "create "} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("preflight SQL contains write keyword %q", forbidden)
		}
	}
}

func TestRunPhaseAPreflightReportsGlobalKeyGateAndBackupBoundary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("WITH retired_entities").WillReturnRows(sqlmock.NewRows([]string{"metric", "value"}).
		AddRow("entity_key.blank", 0).
		AddRow("entity_key.duplicate_groups", 1).
		AddRow("entity_type.sector", 52))
	mock.ExpectQuery("WITH target_tables").WillReturnRows(sqlmock.NewRows([]string{"reference_kind", "object_name", "definition"}).
		AddRow("foreign_key", "event_entity_links_entity_id_fkey", "event_entity_links.entity_id -> entity_nodes.id").
		AddRow("function", "public.lookup_sector", "function body references sector_profiles"))
	mock.ExpectQuery("SELECT entity_type").WillReturnRows(sqlmock.NewRows([]string{"entity_type", "row_count", "checksum"}).
		AddRow("market", 4, "abc"))
	mock.ExpectQuery("SELECT current_setting").WillReturnRows(sqlmock.NewRows([]string{"archive_mode"}).AddRow("off"))
	mock.ExpectCommit()
	report, err := NewPostgresRepository(db).RunPhaseAPreflight(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.EntityKeyGlobalUniqueSafe {
		t.Fatal("entity key global unique gate should be blocked")
	}
	if report.BackupVerified || !strings.Contains(report.BackupStatus, "archive_mode=off") {
		t.Fatalf("backup boundary = %+v", report)
	}
	if len(report.References) != 2 || report.References[1].ObjectName != "public.lookup_sector" {
		t.Fatalf("references = %+v", report.References)
	}
	if baseline := report.ProtectedEntityBaseline["market"]; baseline.RowCount != 4 || baseline.Checksum != "abc" {
		t.Fatalf("protected baseline = %+v", report.ProtectedEntityBaseline)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
