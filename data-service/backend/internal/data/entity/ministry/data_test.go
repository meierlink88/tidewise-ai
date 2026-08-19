package ministry

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

var (
	ministryChinaID = "COU11111111-1111-4111-8111-111111111111"
	ministryOrgID   = "ORG22222222-2222-4222-8222-222222222222"
)

func TestStoreCreatesGetsAndListsMinistriesByOwner(t *testing.T) {
	db := openMinistryTestDatabase(t)
	seedMinistryOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tags := []string{"金融监管", "银行监管"}
	positioning := "维护国家金融稳定"
	jurisdiction := JurisdictionScopeFederal

	countryOwned, err := store.Create(ctx, CreateInput{
		Code: "PBOC-MIN", Name: "金融监管部门", NameEn: "Financial Regulatory Ministry",
		CountryID: &ministryChinaID, AgencyLevel: AgencyLevelCabinet,
		HasSanctionPower: true, HasRegulatoryPower: true, HasEnforcementPower: false,
		JurisdictionScope: &jurisdiction, DomainTags: tags, StrategicPositioning: &positioning,
	})
	if err != nil {
		t.Fatalf("Create(Country-owned) error = %v", err)
	}
	if !coreid.Is(countryOwned.ID, coreid.Ministry) || len(countryOwned.ID) != 39 {
		t.Fatalf("Create(Country-owned) ID = %q, want canonical MIN identity", countryOwned.ID)
	}
	if countryOwned.IsSupranational || countryOwned.OrganizationID != nil || countryOwned.CountryID == nil || *countryOwned.CountryID != ministryChinaID {
		t.Fatalf("Create(Country-owned) owner = %+v", countryOwned)
	}
	if !reflect.DeepEqual(countryOwned.DomainTags, tags) || countryOwned.JurisdictionScope == nil || *countryOwned.JurisdictionScope != jurisdiction {
		t.Fatalf("Create(Country-owned) optional fields = %+v", countryOwned)
	}
	if countryOwned.CreatedAt.IsZero() || !countryOwned.CreatedAt.Equal(countryOwned.UpdatedAt) || time.Since(countryOwned.CreatedAt) > time.Minute {
		t.Fatalf("Create(Country-owned) times = %s, %s", countryOwned.CreatedAt, countryOwned.UpdatedAt)
	}

	orgOwned, err := store.Create(ctx, CreateInput{
		Code: "BIS-REG", Name: "国际监管部门", NameEn: "International Regulatory Ministry",
		OrganizationID: &ministryOrgID, AgencyLevel: AgencyLevelIndependentRegulator,
		HasSanctionPower: false, HasRegulatoryPower: true, HasEnforcementPower: false,
	})
	if err != nil {
		t.Fatalf("Create(Organization-owned) error = %v", err)
	}
	if !orgOwned.IsSupranational || orgOwned.CountryID != nil || orgOwned.OrganizationID == nil || *orgOwned.OrganizationID != ministryOrgID {
		t.Fatalf("Create(Organization-owned) owner = %+v", orgOwned)
	}
	if orgOwned.DomainTags != nil || orgOwned.JurisdictionScope != nil || orgOwned.StrategicPositioning != nil || orgOwned.Description != nil {
		t.Fatalf("Create(Organization-owned) nullable fields = %+v", orgOwned)
	}

	got, err := store.Get(ctx, countryOwned.ID)
	if err != nil || !reflect.DeepEqual(got, countryOwned) {
		t.Fatalf("Get(Country-owned) = %+v, %v; want %+v", got, err, countryOwned)
	}
	all, err := store.List(ctx, Filter{})
	if err != nil || len(all) != 2 || all[0].Code != "BIS-REG" || all[1].Code != "PBOC-MIN" {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	byCountry, err := store.List(ctx, Filter{CountryID: &ministryChinaID})
	if err != nil || len(byCountry) != 1 || byCountry[0].ID != countryOwned.ID {
		t.Fatalf("List(Country) = %+v, %v", byCountry, err)
	}
	byOrganization, err := store.List(ctx, Filter{OrganizationID: &ministryOrgID})
	if err != nil || len(byOrganization) != 1 || byOrganization[0].ID != orgOwned.ID {
		t.Fatalf("List(Organization) = %+v, %v", byOrganization, err)
	}
}

func TestStoreEnforcesMinistryPersistenceContracts(t *testing.T) {
	db := openMinistryTestDatabase(t)
	seedMinistryOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Code: "ministry code 1", Name: "有效部门", NameEn: "Valid Ministry",
		CountryID: &ministryChinaID, AgencyLevel: AgencyLevelSubCabinet,
		HasSanctionPower: false, HasRegulatoryPower: false, HasEnforcementPower: true,
		DomainTags: []string{},
	}
	parent, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if parent.DomainTags == nil || len(parent.DomainTags) != 0 {
		t.Fatalf("Create(present empty domain tags) = %#v, want non-nil empty", parent.DomainTags)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate code) error = %v, want ErrConflict", err)
	}

	childInput := valid
	childInput.Code = "child"
	childInput.ParentMinistryID = &parent.ID
	child, err := store.Create(ctx, childInput)
	if err != nil || child.ParentMinistryID == nil || *child.ParentMinistryID != parent.ID {
		t.Fatalf("Create(child) = %+v, %v", child, err)
	}
	unknownParent := valid
	unknownParent.Code = "unknown-parent"
	unknownParent.ParentMinistryID = stringPointer("MIN33333333-3333-4333-8333-333333333333")
	if _, err := store.Create(ctx, unknownParent); !errors.Is(err, ErrParentNotFound) {
		t.Fatalf("Create(unknown parent) error = %v, want ErrParentNotFound", err)
	}
	unknownOwner := valid
	unknownOwner.Code = "unknown-owner"
	unknownOwner.CountryID = stringPointer("COU33333333-3333-4333-8333-333333333333")
	if _, err := store.Create(ctx, unknownOwner); !errors.Is(err, ErrOwnerNotFound) {
		t.Fatalf("Create(unknown owner) error = %v, want ErrOwnerNotFound", err)
	}

	invalid := []CreateInput{
		{Code: "missing-owner", Name: "无归属", NameEn: "Missing Owner", AgencyLevel: AgencyLevelCabinet},
		{Code: "two-owners", Name: "双归属", NameEn: "Two Owners", CountryID: &ministryChinaID, OrganizationID: &ministryOrgID, AgencyLevel: AgencyLevelCabinet},
		{Code: "unknown-agency", Name: "未知层级", NameEn: "Unknown Agency", CountryID: &ministryChinaID, AgencyLevel: AgencyLevel("cabinet_level")},
		{Code: "unknown-scope", Name: "未知范围", NameEn: "Unknown Scope", CountryID: &ministryChinaID, AgencyLevel: AgencyLevelCabinet, JurisdictionScope: jurisdictionPointer(JurisdictionScope("UNKNOWN"))},
		{Code: " ", Name: "空代码", NameEn: "Blank Code", CountryID: &ministryChinaID, AgencyLevel: AgencyLevelCabinet},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidMinistry) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidMinistry", input, err)
		}
	}
	if _, err := store.List(ctx, Filter{CountryID: &ministryChinaID, OrganizationID: &ministryOrgID}); !errors.Is(err, ErrInvalidMinistry) {
		t.Fatalf("List(two owner filters) error = %v, want ErrInvalidMinistry", err)
	}
	if _, err := store.Get(ctx, "COU33333333-3333-4333-8333-333333333333"); !errors.Is(err, ErrInvalidMinistry) {
		t.Fatalf("Get(Country ID) error = %v, want ErrInvalidMinistry", err)
	}
	if _, err := store.Get(ctx, "MIN44444444-4444-4444-8444-444444444444"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM countries WHERE id = $1`, ministryChinaID); err == nil {
		t.Fatal("deleting a referenced Country succeeded")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM ministries WHERE id = $1`, parent.ID); err == nil {
		t.Fatal("deleting a referenced parent Ministry succeeded")
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, parent.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := store.List(cancelled, Filter{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestStoreFailsClosedForUnknownPersistedMinistryEnums(t *testing.T) {
	db := openMinistryTestDatabase(t)
	seedMinistryOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	scope := JurisdictionScopeFederal
	created, err := store.Create(ctx, CreateInput{
		Code: "enum-drift", Name: "枚举漂移测试", NameEn: "Enum Drift Test",
		CountryID: &ministryChinaID, AgencyLevel: AgencyLevelCabinet,
		JurisdictionScope: &scope,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE ministries ALTER COLUMN agency_level TYPE TEXT USING agency_level::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE ministries SET agency_level = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted agency level) error = %v, want ErrPersistence", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE ministries SET agency_level = 'CABINET_LEVEL' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE ministries ALTER COLUMN jurisdiction_scope TYPE TEXT USING jurisdiction_scope::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE ministries SET jurisdiction_scope = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted jurisdiction scope) error = %v, want ErrPersistence", err)
	}
}

func TestMinistryDatabaseSchemaMatchesThePublicContract(t *testing.T) {
	db := openMinistryTestDatabase(t)
	seedMinistryOwners(t, db)
	ctx := context.Background()
	type column struct {
		Name      string
		Nullable  string
		Type      string
		MaxLength sql.NullInt64
	}
	rows, err := db.QueryContext(ctx, `
SELECT column_name, is_nullable, udt_name, character_maximum_length
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'ministries'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []column
	for rows.Next() {
		var item column
		if err := rows.Scan(&item.Name, &item.Nullable, &item.Type, &item.MaxLength); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, item)
	}
	want := []column{
		{Name: "id", Nullable: "NO", Type: "varchar", MaxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{Name: "code", Nullable: "NO", Type: "varchar", MaxLength: sql.NullInt64{Int64: 30, Valid: true}},
		{Name: "name", Nullable: "NO", Type: "varchar", MaxLength: sql.NullInt64{Int64: 100, Valid: true}},
		{Name: "name_en", Nullable: "NO", Type: "varchar", MaxLength: sql.NullInt64{Int64: 100, Valid: true}},
		{Name: "country_id", Nullable: "YES", Type: "varchar", MaxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{Name: "org_id", Nullable: "YES", Type: "varchar", MaxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{Name: "is_supranational", Nullable: "NO", Type: "bool"},
		{Name: "parent_ministry_id", Nullable: "YES", Type: "varchar", MaxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{Name: "agency_level", Nullable: "NO", Type: "ministry_agency_level"},
		{Name: "has_sanction_power", Nullable: "NO", Type: "bool"},
		{Name: "has_regulatory_power", Nullable: "NO", Type: "bool"},
		{Name: "has_enforcement_power", Nullable: "NO", Type: "bool"},
		{Name: "jurisdiction_scope", Nullable: "YES", Type: "ministry_jurisdiction_scope"},
		{Name: "domain_tags", Nullable: "YES", Type: "_text"},
		{Name: "strategic_positioning", Nullable: "YES", Type: "text"},
		{Name: "description", Nullable: "YES", Type: "text"},
		{Name: "created_at", Nullable: "NO", Type: "timestamptz"},
		{Name: "updated_at", Nullable: "NO", Type: "timestamptz"},
	}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("ministries columns = %#v, want %#v", columns, want)
	}
	for enumName, expected := range map[string][]string{
		"ministry_agency_level":       {"CABINET_LEVEL", "SUB_CABINET", "INDEPENDENT_REGULATOR"},
		"ministry_jurisdiction_scope": {"FEDERAL", "STATE", "SUPRANATIONAL"},
	} {
		if got := postgresEnumValues(t, db, enumName); !reflect.DeepEqual(got, expected) {
			t.Errorf("%s = %q, want %q", enumName, got, expected)
		}
	}
	var supranationalDefault, createdDefault, updatedDefault string
	if err := db.QueryRowContext(ctx, `
SELECT
    max(column_default) FILTER (WHERE column_name = 'is_supranational'),
    max(column_default) FILTER (WHERE column_name = 'created_at'),
    max(column_default) FILTER (WHERE column_name = 'updated_at')
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'ministries'`).Scan(
		&supranationalDefault, &createdDefault, &updatedDefault,
	); err != nil {
		t.Fatal(err)
	}
	if supranationalDefault != "false" || !strings.Contains(createdDefault, "now()") || !strings.Contains(updatedDefault, "now()") {
		t.Fatalf("ministries defaults = %q, %q, %q", supranationalDefault, createdDefault, updatedDefault)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO ministries (
    id, code, name, name_en, country_id, agency_level,
    has_sanction_power, has_regulatory_power, has_enforcement_power
) VALUES (
    'MIN55555555-5555-4555-8555-555555555555', 'valid', '有效部门', 'Valid Ministry',
    $1, 'CABINET_LEVEL', FALSE, TRUE, FALSE
)`, ministryChinaID); err != nil {
		t.Fatal(err)
	}
	invalidStatements := map[string]string{
		"invalid identity": `INSERT INTO ministries (id,code,name,name_en,country_id,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MIN_BAD','bad-id','错误','Invalid','` + ministryChinaID + `','CABINET_LEVEL',FALSE,FALSE,FALSE)`,
		"missing owner":    `INSERT INTO ministries (id,code,name,name_en,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MIN66666666-6666-4666-8666-666666666666','missing','错误','Missing','CABINET_LEVEL',FALSE,FALSE,FALSE)`,
		"two owners":       `INSERT INTO ministries (id,code,name,name_en,country_id,org_id,is_supranational,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MIN77777777-7777-4777-8777-777777777777','two','错误','Two','` + ministryChinaID + `','` + ministryOrgID + `',TRUE,'CABINET_LEVEL',FALSE,FALSE,FALSE)`,
		"owner flag drift": `INSERT INTO ministries (id,code,name,name_en,org_id,is_supranational,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MIN88888888-8888-4888-8888-888888888888','drift','错误','Drift','` + ministryOrgID + `',FALSE,'CABINET_LEVEL',FALSE,FALSE,FALSE)`,
		"unknown enum":     `INSERT INTO ministries (id,code,name,name_en,country_id,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MIN99999999-9999-4999-8999-999999999999','enum','错误','Enum','` + ministryChinaID + `','cabinet_level',FALSE,FALSE,FALSE)`,
		"null boolean":     `INSERT INTO ministries (id,code,name,name_en,country_id,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MINaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','boolean','错误','Boolean','` + ministryChinaID + `','CABINET_LEVEL',NULL,FALSE,FALSE)`,
		"duplicate code":   `INSERT INTO ministries (id,code,name,name_en,country_id,agency_level,has_sanction_power,has_regulatory_power,has_enforcement_power) VALUES ('MINbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','valid','重复','Duplicate','` + ministryChinaID + `','CABINET_LEVEL',FALSE,FALSE,FALSE)`,
	}
	for name, statement := range invalidStatements {
		t.Run(name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, statement); err == nil {
				t.Fatalf("ministries accepted %s", name)
			}
		})
	}
}

func postgresEnumValues(t *testing.T, db *sql.DB, enumName string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), `
SELECT enumlabel
FROM pg_enum
JOIN pg_type ON pg_type.oid = pg_enum.enumtypid
JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
WHERE pg_namespace.nspname = current_schema() AND pg_type.typname = $1
ORDER BY enumsortorder`, enumName)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	return values
}

func jurisdictionPointer(value JurisdictionScope) *JurisdictionScope { return &value }

func stringPointer(value string) *string { return &value }

func openMinistryTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_ministry", migrationDir, 0)
}

func seedMinistryOwners(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO countries (id, code, name, name_en)
VALUES ('COU11111111-1111-4111-8111-111111111111', 'CN', '中国', 'China');
INSERT INTO organization_categories (id, code, name_zh)
VALUES ('OCA11111111-1111-4111-8111-111111111111', 'INTERGOVERNMENTAL', '政府间组织');
INSERT INTO organization_functions (id, code, name_zh)
VALUES ('OFN11111111-1111-4111-8111-111111111111', 'FINANCE', '金融协调');
INSERT INTO organizations (id, code, name, name_en, category_code, function_code)
VALUES ('ORG22222222-2222-4222-8222-222222222222', 'BIS', '国际清算银行', 'Bank for International Settlements', 'INTERGOVERNMENTAL', 'FINANCE');`); err != nil {
		t.Fatal(err)
	}
}
