package institution

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
	institutionChinaID = "COU11111111-1111-4111-8111-111111111111"
	institutionOrgID   = "ORG22222222-2222-4222-8222-222222222222"
)

func TestStoreCreatesGetsAndListsInstitutionsByOwner(t *testing.T) {
	db := openInstitutionTestDatabase(t)
	seedInstitutionOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	importance := SystemicImportanceDSIB
	positioning := "国家货币政策与金融稳定核心机构"
	description := "依法履行中央银行职责"

	countryOwned, err := store.Create(ctx, CreateInput{
		Code: "CN-CB", Name: "中国中央银行", NameEn: "Central Bank of China",
		CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCentralBank,
		ClearingCurrency: stringPointer("CNY"), SwiftBIC: stringPointer("PBOCCNBJXXX"),
		LEICode: stringPointer("12345678901234567890"), SystemicImportance: &importance,
		StrategicPositioning: &positioning, Description: &description,
	})
	if err != nil {
		t.Fatalf("Create(Country-owned) error = %v", err)
	}
	if !coreid.Is(countryOwned.ID, coreid.Institution) || len(countryOwned.ID) != 39 {
		t.Fatalf("Create(Country-owned) ID = %q, want canonical INS identity", countryOwned.ID)
	}
	if countryOwned.IsSupranational || countryOwned.OrganizationID != nil || countryOwned.CountryID == nil || *countryOwned.CountryID != institutionChinaID {
		t.Fatalf("Create(Country-owned) owner = %+v", countryOwned)
	}
	if countryOwned.SystemicImportance == nil || *countryOwned.SystemicImportance != importance || countryOwned.ClearingCurrency == nil || *countryOwned.ClearingCurrency != "CNY" {
		t.Fatalf("Create(Country-owned) optional fields = %+v", countryOwned)
	}
	if countryOwned.CreatedAt.IsZero() || !countryOwned.CreatedAt.Equal(countryOwned.UpdatedAt) || time.Since(countryOwned.CreatedAt) > time.Minute {
		t.Fatalf("Create(Country-owned) times = %s, %s", countryOwned.CreatedAt, countryOwned.UpdatedAt)
	}

	orgOwned, err := store.Create(ctx, CreateInput{
		Code: "BIS-CP", Name: "国际清算机构", NameEn: "International Clearing Institution",
		OrganizationID: &institutionOrgID, InstitutionType: InstitutionTypeClearingHouse,
	})
	if err != nil {
		t.Fatalf("Create(Organization-owned) error = %v", err)
	}
	if !orgOwned.IsSupranational || orgOwned.CountryID != nil || orgOwned.OrganizationID == nil || *orgOwned.OrganizationID != institutionOrgID {
		t.Fatalf("Create(Organization-owned) owner = %+v", orgOwned)
	}
	if orgOwned.SystemicImportance != nil || orgOwned.ClearingCurrency != nil || orgOwned.SwiftBIC != nil || orgOwned.LEICode != nil || orgOwned.StrategicPositioning != nil || orgOwned.Description != nil {
		t.Fatalf("Create(Organization-owned) nullable fields = %+v", orgOwned)
	}

	got, err := store.Get(ctx, countryOwned.ID)
	if err != nil || !reflect.DeepEqual(got, countryOwned) {
		t.Fatalf("Get(Country-owned) = %+v, %v; want %+v", got, err, countryOwned)
	}
	all, err := store.List(ctx, Filter{})
	if err != nil || len(all) != 2 || all[0].Code != "BIS-CP" || all[1].Code != "CN-CB" {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	byCountry, err := store.List(ctx, Filter{CountryID: &institutionChinaID})
	if err != nil || len(byCountry) != 1 || byCountry[0].ID != countryOwned.ID {
		t.Fatalf("List(Country) = %+v, %v", byCountry, err)
	}
	byOrganization, err := store.List(ctx, Filter{OrganizationID: &institutionOrgID})
	if err != nil || len(byOrganization) != 1 || byOrganization[0].ID != orgOwned.ID {
		t.Fatalf("List(Organization) = %+v, %v", byOrganization, err)
	}
}

func TestStoreEnforcesInstitutionPersistenceContracts(t *testing.T) {
	db := openInstitutionTestDatabase(t)
	seedInstitutionOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := CreateInput{
		Code: "institution code 1", Name: "有效机构", NameEn: "Valid Institution",
		CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCommercialBank,
	}
	created, err := store.Create(ctx, valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate code) error = %v, want ErrConflict", err)
	}

	for index, institutionType := range []InstitutionType{
		InstitutionTypePaymentSystem,
		InstitutionTypeDevelopmentBank,
		InstitutionTypeInternationalFinancialInstitution,
	} {
		input := valid
		input.Code = []string{"payment-system", "development-bank", "international-fi"}[index]
		input.InstitutionType = institutionType
		if _, err := store.Create(ctx, input); err != nil {
			t.Errorf("Create(%s) error = %v", institutionType, err)
		}
	}
	for index, importance := range []SystemicImportance{
		SystemicImportanceGSIB,
		SystemicImportanceDSIB,
		SystemicImportanceNonSIB,
	} {
		input := valid
		input.Code = []string{"g-sib", "d-sib", "non-sib"}[index]
		input.SystemicImportance = &importance
		if _, err := store.Create(ctx, input); err != nil {
			t.Errorf("Create(%s) error = %v", importance, err)
		}
	}

	unknownOwner := valid
	unknownOwner.Code = "unknown-owner"
	unknownOwner.CountryID = stringPointer("COU33333333-3333-4333-8333-333333333333")
	if _, err := store.Create(ctx, unknownOwner); !errors.Is(err, ErrOwnerNotFound) {
		t.Fatalf("Create(unknown owner) error = %v, want ErrOwnerNotFound", err)
	}
	invalid := []CreateInput{
		{Code: "missing-owner", Name: "无归属", NameEn: "Missing Owner", InstitutionType: InstitutionTypeCentralBank},
		{Code: "two-owners", Name: "双归属", NameEn: "Two Owners", CountryID: &institutionChinaID, OrganizationID: &institutionOrgID, InstitutionType: InstitutionTypeCentralBank},
		{Code: "unknown-type", Name: "未知类型", NameEn: "Unknown Type", CountryID: &institutionChinaID, InstitutionType: InstitutionType("central_bank")},
		{Code: "unknown-importance", Name: "未知重要性", NameEn: "Unknown Importance", CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCentralBank, SystemicImportance: systemicImportancePointer(SystemicImportance("UNKNOWN"))},
		{Code: "currency-too-long", Name: "过长货币", NameEn: "Long Currency", CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCentralBank, ClearingCurrency: stringPointer("USDX")},
		{Code: " ", Name: "空代码", NameEn: "Blank Code", CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCentralBank},
	}
	for _, input := range invalid {
		if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidInstitution) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidInstitution", input, err)
		}
	}
	if _, err := store.List(ctx, Filter{CountryID: &institutionChinaID, OrganizationID: &institutionOrgID}); !errors.Is(err, ErrInvalidInstitution) {
		t.Fatalf("List(two owner filters) error = %v, want ErrInvalidInstitution", err)
	}
	if _, err := store.Get(ctx, "MIN33333333-3333-4333-8333-333333333333"); !errors.Is(err, ErrInvalidInstitution) {
		t.Fatalf("Get(Ministry ID) error = %v, want ErrInvalidInstitution", err)
	}
	if _, err := store.Get(ctx, "INS44444444-4444-4444-8444-444444444444"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM countries WHERE id = $1`, institutionChinaID); err == nil {
		t.Fatal("deleting a referenced Country succeeded")
	}
	referenceRows, err := db.QueryContext(ctx, `
SELECT ccu.table_name
FROM information_schema.table_constraints tc
JOIN information_schema.constraint_column_usage ccu
  ON ccu.constraint_schema = tc.constraint_schema
 AND ccu.constraint_name = tc.constraint_name
WHERE tc.table_schema = current_schema()
  AND tc.table_name = 'institutions'
  AND tc.constraint_type = 'FOREIGN KEY'
ORDER BY ccu.table_name`)
	if err != nil {
		t.Fatal(err)
	}
	defer referenceRows.Close()
	var referenceTables []string
	for referenceRows.Next() {
		var table string
		if err := referenceRows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		referenceTables = append(referenceTables, table)
	}
	if want := []string{"countries", "organizations"}; !reflect.DeepEqual(referenceTables, want) {
		t.Fatalf("institutions foreign-key targets = %q, want %q", referenceTables, want)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, created.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := store.List(cancelled, Filter{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestStoreFailsClosedForUnknownPersistedInstitutionEnums(t *testing.T) {
	db := openInstitutionTestDatabase(t)
	seedInstitutionOwners(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	importance := SystemicImportanceDSIB
	created, err := store.Create(ctx, CreateInput{
		Code: "enum-drift", Name: "枚举漂移测试", NameEn: "Enum Drift Test",
		CountryID: &institutionChinaID, InstitutionType: InstitutionTypeCentralBank,
		SystemicImportance: &importance,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE institutions ALTER COLUMN institution_type TYPE TEXT USING institution_type::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE institutions SET institution_type = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted institution type) error = %v, want ErrPersistence", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE institutions SET institution_type = 'CENTRAL_BANK' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE institutions ALTER COLUMN systemic_importance TYPE TEXT USING systemic_importance::text`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE institutions SET systemic_importance = 'UNKNOWN' WHERE id = $1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted systemic importance) error = %v, want ErrPersistence", err)
	}
}

func TestInstitutionDatabaseSchemaMatchesThePublicContract(t *testing.T) {
	db := openInstitutionTestDatabase(t)
	seedInstitutionOwners(t, db)
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
WHERE table_schema = current_schema() AND table_name = 'institutions'
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
		{Name: "institution_type", Nullable: "NO", Type: "institution_type"},
		{Name: "clearing_currency", Nullable: "YES", Type: "bpchar", MaxLength: sql.NullInt64{Int64: 3, Valid: true}},
		{Name: "swift_bic", Nullable: "YES", Type: "bpchar", MaxLength: sql.NullInt64{Int64: 11, Valid: true}},
		{Name: "lei_code", Nullable: "YES", Type: "bpchar", MaxLength: sql.NullInt64{Int64: 20, Valid: true}},
		{Name: "systemic_importance", Nullable: "YES", Type: "institution_systemic_importance"},
		{Name: "strategic_positioning", Nullable: "YES", Type: "text"},
		{Name: "description", Nullable: "YES", Type: "text"},
		{Name: "created_at", Nullable: "NO", Type: "timestamptz"},
		{Name: "updated_at", Nullable: "NO", Type: "timestamptz"},
	}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("institutions columns = %#v, want %#v", columns, want)
	}
	for enumName, expected := range map[string][]string{
		"institution_type": {
			"CENTRAL_BANK", "COMMERCIAL_BANK", "CLEARING_HOUSE", "PAYMENT_SYSTEM",
			"DEVELOPMENT_BANK", "INTERNATIONAL_FINANCIAL_INSTITUTION",
		},
		"institution_systemic_importance": {"G_SIB", "D_SIB", "NON_SIB"},
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
WHERE table_schema = current_schema() AND table_name = 'institutions'`).Scan(
		&supranationalDefault, &createdDefault, &updatedDefault,
	); err != nil {
		t.Fatal(err)
	}
	if supranationalDefault != "false" || !strings.Contains(createdDefault, "now()") || !strings.Contains(updatedDefault, "now()") {
		t.Fatalf("institutions defaults = %q, %q, %q", supranationalDefault, createdDefault, updatedDefault)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO institutions (id, code, name, name_en, country_id, institution_type)
VALUES (
    'INS55555555-5555-4555-8555-555555555555', 'valid', '有效机构',
    'Valid Institution', $1, 'CENTRAL_BANK'
)`, institutionChinaID); err != nil {
		t.Fatal(err)
	}
	invalidStatements := map[string]string{
		"invalid identity": `INSERT INTO institutions (id,code,name,name_en,country_id,institution_type) VALUES ('INS_BAD','bad-id','错误','Invalid','` + institutionChinaID + `','CENTRAL_BANK')`,
		"missing owner":    `INSERT INTO institutions (id,code,name,name_en,institution_type) VALUES ('INS66666666-6666-4666-8666-666666666666','missing','错误','Missing','CENTRAL_BANK')`,
		"two owners":       `INSERT INTO institutions (id,code,name,name_en,country_id,org_id,is_supranational,institution_type) VALUES ('INS77777777-7777-4777-8777-777777777777','two','错误','Two','` + institutionChinaID + `','` + institutionOrgID + `',TRUE,'CENTRAL_BANK')`,
		"owner flag drift": `INSERT INTO institutions (id,code,name,name_en,org_id,is_supranational,institution_type) VALUES ('INS88888888-8888-4888-8888-888888888888','drift','错误','Drift','` + institutionOrgID + `',FALSE,'CENTRAL_BANK')`,
		"unknown enum":     `INSERT INTO institutions (id,code,name,name_en,country_id,institution_type) VALUES ('INS99999999-9999-4999-8999-999999999999','enum','错误','Enum','` + institutionChinaID + `','central_bank')`,
		"null type":        `INSERT INTO institutions (id,code,name,name_en,country_id,institution_type) VALUES ('INSaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','type','错误','Type','` + institutionChinaID + `',NULL)`,
		"duplicate code":   `INSERT INTO institutions (id,code,name,name_en,country_id,institution_type) VALUES ('INSbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb','valid','重复','Duplicate','` + institutionChinaID + `','CENTRAL_BANK')`,
	}
	for name, statement := range invalidStatements {
		t.Run(name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, statement); err == nil {
				t.Fatalf("institutions accepted %s", name)
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

func systemicImportancePointer(value SystemicImportance) *SystemicImportance { return &value }

func openInstitutionTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_institution", migrationDir, 0)
}

func seedInstitutionOwners(t *testing.T, db *sql.DB) {
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

func stringPointer(value string) *string { return &value }
