package seed

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAllianceEconomyDependencyAuditBlocksCrossDomainEdges(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyCountsSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"scope", "relation_type", "from_type", "to_type", "row_count"}).
			AddRow("entity_nodes", "", "", "", 50).
			AddRow("entity_edges", "has_market", "economy", "market", 2),
	)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyFingerprintsSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}).AddRow("node|one"))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyForeignKeysSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"table_name", "column_name", "referenced_table", "delete_rule"}).
			AddRow("market_profiles", "economy_entity_id", "entity_nodes", "NO ACTION"),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "market_profiles" r JOIN entity_nodes n ON n.id=r."economy_entity_id" WHERE n.entity_type IN ('alliance_org','economy')`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(47))

	report, err := NewPostgresRepository(db).AuditAllianceEconomyRebuildDependencies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Blocked || len(report.CrossDomainEdges) != 1 || report.CrossDomainEdges[0].RelationType != "has_market" {
		t.Fatalf("report = %+v", report)
	}
	if report.Checksum == "" || len(report.ForeignKeys) != 1 || len(report.Fingerprints) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAllianceEconomyDependencyAuditRejectsWrongDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_prod"))
	if _, err := NewPostgresRepository(db).AuditAllianceEconomyRebuildDependencies(context.Background()); err == nil || !strings.Contains(err.Error(), "tidewise_local") {
		t.Fatalf("AuditAllianceEconomyRebuildDependencies() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAllianceEconomyLocalFailsClosedBeforeTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyCountsSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"scope", "relation_type", "from_type", "to_type", "row_count"}).
			AddRow("entity_edges", "has_market", "economy", "market", 1),
	)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyFingerprintsSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyForeignKeysSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"table_name", "column_name", "referenced_table", "delete_rule"}),
	)
	mock.ExpectRollback()

	_, err = NewPostgresRepository(db).CleanupAllianceEconomyLocal(context.Background(), "reviewed-checksum")
	if err == nil || !regexp.MustCompile(`cross-domain`).MatchString(err.Error()) {
		t.Fatalf("CleanupAllianceEconomyLocal() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAllianceEconomyLocalUsesReviewedSnapshotAndAtomicExactCleanup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"scope", "relation_type", "from_type", "to_type", "row_count"}).
		AddRow("entity_nodes", "", "alliance_org", "", 10).
		AddRow("entity_edges", "member_of", "economy", "alliance_org", 223)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyCountsSQL())).WillReturnRows(rows)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyFingerprintsSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyForeignKeysSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"table_name", "column_name", "referenced_table", "delete_rule"}),
	)
	report := dependencyReportForTest(t, []AllianceEconomyDependencyCount{{Scope: "entity_nodes", FromType: "alliance_org", RowCount: 10}, {Scope: "entity_edges", RelationType: "member_of", FromType: "economy", ToType: "alliance_org", RowCount: 223}}, nil)

	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"deleted_edges", "deleted_alliance_profiles", "deleted_economy_profiles", "deleted_entities", "remaining_entities", "remaining_profiles", "remaining_edges"}).
			AddRow(223, 10, 50, 60, 0, 0, 0),
	)
	mock.ExpectCommit()

	result, err := NewPostgresRepository(db).CleanupAllianceEconomyLocal(context.Background(), report.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemainingEntities != 0 || result.RemainingProfiles != 0 || result.RemainingEdges != 0 {
		t.Fatalf("result = %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildApprovedAllianceEconomyLocalIsAtomicExactAndIdempotent(t *testing.T) {
	manifest := approvedManifest(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for run := 0; run < 2; run++ {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
		mock.ExpectExec(regexp.QuoteMeta(`LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`)).WillReturnResult(sqlmock.NewResult(0, 0))
		preflight := []driver.Value{true, 0, 0, 0, 0, 0, 0, 0, 0, 0}
		if run == 1 {
			preflight = []driver.Value{true, 0, 0, 0, 0, 45, 45, 79, 79, 133}
		}
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildPreflightSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"schema_ready", "id_conflicts", "key_conflicts", "unexpected_target_nodes", "unexpected_incident_edges", "alliances", "alliance_profiles", "economies", "economy_profiles", "member_of"}).AddRow(preflight...))
		if run == 1 {
			mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 133, 0, 0, 0))
		}
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyEntityRebuildSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyProfileRebuildSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyMemberRebuildSQL())).
			WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 133))
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 133, 0, 0, 0))
		mock.ExpectCommit()
		result, err := NewPostgresRepository(db).RebuildApprovedAllianceEconomyLocal(context.Background(), manifest)
		if err != nil {
			t.Fatal(err)
		}
		if result.ManifestChecksum != approvedAllianceEconomyManifestSHA256 || result.Alliances != 45 || result.AllianceProfiles != 45 || result.Economies != 79 || result.EconomyProfiles != 79 || result.MemberOf != 133 || result.Orphans != 0 || result.DuplicateTuples != 0 || result.Mismatches != 0 {
			t.Fatalf("run %d result = %+v", run+1, result)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildApprovedAllianceEconomyLocalRollsBackOnAssertionFailure(t *testing.T) {
	manifest := approvedManifest(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectExec(regexp.QuoteMeta(`LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildPreflightSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"schema_ready", "id_conflicts", "key_conflicts", "unexpected_target_nodes", "unexpected_incident_edges", "alliances", "alliance_profiles", "economies", "economy_profiles", "member_of"}).AddRow(true, 0, 0, 0, 0, 0, 0, 0, 0, 0))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyEntityRebuildSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyProfileRebuildSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyMemberRebuildSQL())).
		WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 133))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 132, 0, 0, 0))
	mock.ExpectRollback()

	if _, err := NewPostgresRepository(db).RebuildApprovedAllianceEconomyLocal(context.Background(), manifest); err == nil {
		t.Fatal("RebuildApprovedAllianceEconomyLocal() error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildApprovedAllianceEconomyLocalRejectsIdentityCollision(t *testing.T) {
	manifest := approvedManifest(t)
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectExec(regexp.QuoteMeta(`LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildPreflightSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"schema_ready", "id_conflicts", "key_conflicts", "unexpected_target_nodes", "unexpected_incident_edges", "alliances", "alliance_profiles", "economies", "economy_profiles", "member_of"}).AddRow(true, 0, 1, 0, 0, 0, 0, 0, 0, 0))
	mock.ExpectRollback()
	if _, err := NewPostgresRepository(db).RebuildApprovedAllianceEconomyLocal(context.Background(), manifest); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("RebuildApprovedAllianceEconomyLocal() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAllianceProfileRepositoryUsesOnlyApprovedFields(t *testing.T) {
	statement, args, err := buildProfileUpsert("alliance_org:g7", "alliance_org", []byte(`{"abbreviation":"G7","leadership_summary":"轮值主席国","influence_scope_summary":"全球协调"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"abbreviation", "leadership_summary", "influence_scope_summary"} {
		if !regexp.MustCompile(`\b` + field + `\b`).MatchString(statement) {
			t.Fatalf("statement lacks %s: %s", field, statement)
		}
	}
	for _, forbidden := range []string{"org_code", "org_type", "primary_domain", "scope_region", "official_url", "categories", "name"} {
		if regexp.MustCompile(`\b` + forbidden + `\b`).MatchString(statement) {
			t.Fatalf("statement contains %s: %s", forbidden, statement)
		}
	}
	if len(args) != 4 {
		t.Fatalf("args = %v", args)
	}
}

func TestAllianceEconomyExactQueryChecksApprovedFieldValues(t *testing.T) {
	query := allianceEconomyExactQuerySQL()
	for _, fragment := range []string{
		`n.name IS DISTINCT FROM i."Name"`,
		`n.aliases IS DISTINCT FROM i."Aliases"`,
		`p.abbreviation IS DISTINCT FROM i."Abbreviation"`,
		`p.leadership_summary IS DISTINCT FROM i."LeadershipSummary"`,
		`p.influence_scope_summary IS DISTINCT FROM i."InfluenceScopeSummary"`,
		`p.country_code IS DISTINCT FROM i."CountryCode"`,
		`p.currency_code IS DISTINCT FROM i."CurrencyCode"`,
		`p.region IS DISTINCT FROM i."Region"`,
		`e.source_name IS DISTINCT FROM i."SourceName"`,
		`e.source_url IS DISTINCT FROM i."SourceURL"`,
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("exact query lacks %q", fragment)
		}
	}
}

func dependencyReportForTest(t *testing.T, counts []AllianceEconomyDependencyCount, keys []AllianceEconomyForeignKey) AllianceEconomyDependencyReport {
	t.Helper()
	report, err := buildAllianceEconomyDependencyReport(counts, keys, nil)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

var _ = sql.LevelSerializable
