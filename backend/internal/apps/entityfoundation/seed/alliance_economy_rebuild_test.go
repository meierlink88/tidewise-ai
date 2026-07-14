package seed

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCleanupAllianceEconomyLocalFailsClosedOnUnapprovedAllianceIncidentEdge(t *testing.T) {
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
			AddRow("entity_edges", "participates_in", "alliance_org", "economy", 1),
	)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyFingerprintsSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyForeignKeysSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"table_name", "column_name", "referenced_table", "delete_rule"}),
	)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupProtectionSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectRollback()

	_, err = NewPostgresRepository(db).CleanupAllianceEconomyLocal(context.Background(), "reviewed-checksum")
	if err == nil || !regexp.MustCompile(`cross-domain`).MatchString(err.Error()) {
		t.Fatalf("CleanupAllianceEconomyLocal() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupAllianceEconomyLocalPreservesEconomiesAndProtectedFacts(t *testing.T) {
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
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupProtectionSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	report := dependencyReportForTest(t, []AllianceEconomyDependencyCount{{Scope: "entity_nodes", FromType: "alliance_org", RowCount: 10}, {Scope: "entity_edges", RelationType: "member_of", FromType: "economy", ToType: "alliance_org", RowCount: 223}}, nil)

	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"deleted_member_of", "deleted_alliance_profiles", "deleted_alliances"}).AddRow(223, 10, 10),
	)
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupRemainingSQL())).WillReturnRows(
		sqlmock.NewRows([]string{"remaining_alliances", "remaining_alliance_profiles", "remaining_member_of", "remaining_economies", "remaining_economy_profiles"}).AddRow(0, 0, 0, 50, 50),
	)
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("tidewise_local"))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyCountsSQL())).WillReturnRows(sqlmock.NewRows([]string{"scope", "relation_type", "from_type", "to_type", "row_count"}).AddRow("entity_nodes", "", "economy", "", 50))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyDependencyFingerprintsSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyForeignKeysSQL())).WillReturnRows(sqlmock.NewRows([]string{"table_name", "column_name", "referenced_table", "delete_rule"}))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyCleanupProtectionSQL())).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
	mock.ExpectCommit()

	result, err := NewPostgresRepository(db).CleanupAllianceEconomyLocal(context.Background(), report.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemainingAlliances != 0 || result.RemainingAllianceProfiles != 0 || result.RemainingMemberOf != 0 || result.RemainingEconomies != 50 || result.RemainingEconomyProfiles != 50 {
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
		preflight := []driver.Value{true, 0, 0, 0, 0, 0, 0, 35, 35, 15, 15, 0}
		if run == 1 {
			preflight = []driver.Value{true, 0, 0, 0, 0, 45, 45, 79, 79, 15, 15, 133}
		}
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildPreflightSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"schema_ready", "id_conflicts", "key_conflicts", "unexpected_alliance_nodes", "unexpected_alliance_edges", "alliances", "alliance_profiles", "economies", "economy_profiles", "non_target_economies", "non_target_economy_profiles", "member_of"}).AddRow(preflight...))
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildProtectionSQL())).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
		if run == 1 {
			mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "non_target_economies", "non_target_economy_profiles", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 133, 15, 15, 0, 0, 0))
		}
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyEntityRebuildSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyProfileRebuildSQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
		mock.ExpectExec(regexp.QuoteMeta(allianceEconomyMemberRebuildSQL())).
			WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 133))
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "non_target_economies", "non_target_economy_profiles", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 133, 15, 15, 0, 0, 0))
		mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildProtectionSQL())).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}))
		mock.ExpectCommit()
		result, err := NewPostgresRepository(db).RebuildApprovedAllianceEconomyLocal(context.Background(), manifest)
		if err != nil {
			t.Fatal(err)
		}
		if result.ManifestChecksum != approvedAllianceEconomyManifestSHA256 || result.Alliances != 45 || result.AllianceProfiles != 45 || result.Economies != 79 || result.EconomyProfiles != 79 || result.MemberOf != 133 || result.NonTargetEconomies != 15 || result.NonTargetEconomyProfiles != 15 || result.Orphans != 0 || result.DuplicateTuples != 0 || result.Mismatches != 0 {
			t.Fatalf("run %d result = %+v", run+1, result)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRebuildApprovedAllianceEconomyLocalStopsOnProtectedFactDrift(t *testing.T) {
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
		WillReturnRows(sqlmock.NewRows([]string{"schema_ready", "id_conflicts", "key_conflicts", "unexpected_alliance_nodes", "unexpected_alliance_edges", "alliances", "alliance_profiles", "economies", "economy_profiles", "non_target_economies", "non_target_economy_profiles", "member_of"}).AddRow(true, 0, 0, 0, 0, 0, 0, 35, 35, 15, 15, 0))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildProtectionSQL())).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}).AddRow("protected-before"))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyEntityRebuildSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyProfileRebuildSQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 124))
	mock.ExpectExec(regexp.QuoteMeta(allianceEconomyMemberRebuildSQL())).
		WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 133))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyExactQuerySQL())).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"alliances", "alliance_profiles", "economies", "economy_profiles", "member_of", "non_target_economies", "non_target_economy_profiles", "orphans", "duplicate_tuples", "mismatches"}).AddRow(45, 45, 79, 79, 133, 15, 15, 0, 0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(allianceEconomyRebuildProtectionSQL())).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"fingerprint"}).AddRow("protected-after"))
	mock.ExpectRollback()

	if _, err := NewPostgresRepository(db).RebuildApprovedAllianceEconomyLocal(context.Background(), manifest); err == nil || !regexp.MustCompile(`protected cross-domain`).MatchString(err.Error()) {
		t.Fatalf("RebuildApprovedAllianceEconomyLocal() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func dependencyReportForTest(t *testing.T, counts []AllianceEconomyDependencyCount, keys []AllianceEconomyForeignKey) AllianceEconomyDependencyReport {
	t.Helper()
	report, err := buildAllianceEconomyDependencyReport(counts, keys, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

var _ = sql.LevelSerializable
