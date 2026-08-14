package region

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStorePersistsCanonicalRegions(t *testing.T) {
	db := openRegionTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	description := "地理上相邻的亚太区域"

	apac, err := store.Create(ctx, Region{
		ID:          "REG_APAC",
		Code:        "APAC",
		Name:        "亚太地区",
		NameEn:      "Asia Pacific",
		RegionType:  RegionTypeGeographic,
		Description: &description,
	})
	if err != nil {
		t.Fatalf("Create(APAC) error = %v", err)
	}
	if apac.CreatedAt.IsZero() || time.Since(apac.CreatedAt) > time.Minute {
		t.Fatalf("Create(APAC) created_at = %s, want recent database time", apac.CreatedAt)
	}
	if apac.Description == nil || *apac.Description != description {
		t.Fatalf("Create(APAC) description = %v, want %q", apac.Description, description)
	}

	emea, err := store.Create(ctx, Region{
		ID:         "REG_EMEA",
		Code:       "EMEA",
		Name:       "欧洲中东与非洲",
		NameEn:     "Europe, Middle East and Africa",
		RegionType: RegionTypeGeographic,
	})
	if err != nil {
		t.Fatalf("Create(EMEA) error = %v", err)
	}
	if emea.Description != nil {
		t.Fatalf("Create(EMEA) description = %v, want nil", emea.Description)
	}

	byID, err := store.GetByID(ctx, apac.ID)
	if err != nil {
		t.Fatalf("GetByID(APAC) error = %v", err)
	}
	byCode, err := store.GetByCode(ctx, apac.Code)
	if err != nil {
		t.Fatalf("GetByCode(APAC) error = %v", err)
	}
	if !reflect.DeepEqual(byID, apac) || !reflect.DeepEqual(byCode, apac) {
		t.Fatalf("canonical reads disagree: create=%+v byID=%+v byCode=%+v", apac, byID, byCode)
	}

	regions, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("List() length = %d, want 2", len(regions))
	}
	if regions[0].Code != "APAC" || regions[1].Code != "EMEA" {
		t.Fatalf("List() codes = %q, %q, want APAC, EMEA", regions[0].Code, regions[1].Code)
	}
}

func TestStoreClassifiesRegionFailures(t *testing.T) {
	db := openRegionTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := Region{
		ID:         "REG_APAC",
		Code:       "APAC",
		Name:       "亚太地区",
		NameEn:     "Asia Pacific",
		RegionType: RegionTypeGeographic,
	}
	if _, err := store.Create(ctx, valid); err != nil {
		t.Fatalf("Create(valid) error = %v", err)
	}

	invalidRegions := []Region{
		{ID: "REG_apac", Code: "apac", Name: "亚太地区", NameEn: "Asia Pacific", RegionType: RegionTypeGeographic},
		{ID: "REG_ASIA", Code: "APAC", Name: "亚太地区", NameEn: "Asia Pacific", RegionType: RegionTypeGeographic},
		{ID: "REG_ASIA", Code: "ASIA", Name: " ", NameEn: "Asia", RegionType: RegionTypeContinent},
		{ID: "REG_ASIA", Code: "ASIA", Name: "亚洲", NameEn: " ", RegionType: RegionTypeContinent},
		{ID: "REG_INVALID", Code: "INVALID", Name: "无效", NameEn: "Invalid", RegionType: RegionType("INVALID")},
		{ID: "REG_" + strings.Repeat("A", 29), Code: strings.Repeat("A", 29), Name: "过长编码", NameEn: "Long code", RegionType: RegionTypeGeographic},
		{ID: "REG_LONG_NAME", Code: "LONG_NAME", Name: strings.Repeat("区", 51), NameEn: "Long name", RegionType: RegionTypeGeographic},
		{ID: "REG_LONG_NAME_EN", Code: "LONG_NAME_EN", Name: "过长英文名", NameEn: strings.Repeat("a", 101), RegionType: RegionTypeGeographic},
	}
	for _, candidate := range invalidRegions {
		if _, err := store.Create(ctx, candidate); !errors.Is(err, ErrInvalidRegion) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidRegion", candidate, err)
		}
	}

	if _, err := store.Create(ctx, valid); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate identity) error = %v, want ErrConflict", err)
	}

	if _, err := store.GetByID(ctx, "REG_MISSING"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetByCode(ctx, "MISSING"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByCode(missing) error = %v, want ErrNotFound", err)
	}
}

func TestStorePreservesContextCancellation(t *testing.T) {
	db := openRegionTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	valid := Region{
		ID: "REG_APAC", Code: "APAC", Name: "亚太地区", NameEn: "Asia Pacific",
		RegionType: RegionTypeGeographic,
	}
	if _, err := store.Create(ctx, valid); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := store.GetByID(ctx, valid.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetByID(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := store.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("List(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestRegionDatabaseRejectsInvalidFacts(t *testing.T) {
	db := openRegionTestDatabase(t)
	statements := map[string]string{
		"null required name": `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_NULL_NAME', 'NULL_NAME', NULL, 'Null Name', 'GEOGRAPHIC')`,
		"invalid enum":       `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_BAD_ENUM', 'BAD_ENUM', '无效枚举', 'Bad Enum', 'INVALID')`,
		"mismatched id":      `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_ASIA', 'APAC', '亚太', 'Asia Pacific', 'GEOGRAPHIC')`,
		"lowercase code":     `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_apac', 'apac', '亚太', 'Asia Pacific', 'GEOGRAPHIC')`,
		"blank name":         `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_BLANK', 'BLANK', ' ', 'Blank', 'GEOGRAPHIC')`,
		"overlong code":      `INSERT INTO regions (id, code, name, name_en, region_type) VALUES ('REG_ABCDEFGHIJKLMNOPQRSTU', 'ABCDEFGHIJKLMNOPQRSTU', '过长', 'Long', 'GEOGRAPHIC')`,
	}
	for name, statement := range statements {
		t.Run(name, func(t *testing.T) {
			if _, err := db.ExecContext(context.Background(), statement); err == nil {
				t.Fatalf("regions accepted %s", name)
			}
		})
	}
}

func TestRegionSchemaAndPersistenceStayAligned(t *testing.T) {
	db := openRegionTestDatabase(t)
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "doctype", "region.schema"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, property := range []string{
		"id(区域标识): Text",
		"code(区域编码): Text",
		"name(区域中文名称): Text",
		"nameEn(区域英文名称): Text",
		"regionType(区域类型): Text",
		"description(区域说明): Text",
		"createdAt(创建时间): Text",
	} {
		if !strings.Contains(string(schema), property) {
			t.Fatalf("Region Object Schema is missing %q", property)
		}
	}

	rows, err := db.QueryContext(context.Background(), `
SELECT column_name, is_nullable, udt_name, character_maximum_length, column_default
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'regions'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type column struct {
		name       string
		nullable   string
		dataType   string
		maxLength  sql.NullInt64
		defaultSQL sql.NullString
	}
	columns := make([]column, 0, 7)
	for rows.Next() {
		var item column
		if err := rows.Scan(&item.name, &item.nullable, &item.dataType, &item.maxLength, &item.defaultSQL); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	wantColumns := []column{
		{name: "id", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 32, Valid: true}},
		{name: "code", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 20, Valid: true}},
		{name: "name", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 50, Valid: true}},
		{name: "name_en", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 100, Valid: true}},
		{name: "region_type", nullable: "NO", dataType: "region_type"},
		{name: "description", nullable: "YES", dataType: "text"},
		{name: "created_at", nullable: "NO", dataType: "timestamptz"},
	}
	if len(columns) != len(wantColumns) {
		t.Fatalf("regions columns = %#v, want %#v", columns, wantColumns)
	}
	for index, want := range wantColumns {
		got := columns[index]
		if got.name != want.name || got.nullable != want.nullable || got.dataType != want.dataType || got.maxLength != want.maxLength {
			t.Fatalf("regions column %d = %#v, want %#v", index, got, want)
		}
	}
	if !columns[6].defaultSQL.Valid || !strings.Contains(columns[6].defaultSQL.String, "now()") {
		t.Fatalf("regions.created_at default = %#v, want database now()", columns[6].defaultSQL)
	}

	enumRows, err := db.QueryContext(context.Background(), `
SELECT enumlabel
FROM pg_enum
JOIN pg_type ON pg_type.oid = pg_enum.enumtypid
JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
WHERE pg_type.typname = 'region_type'
  AND pg_namespace.nspname = current_schema()
ORDER BY enumsortorder`)
	if err != nil {
		t.Fatal(err)
	}
	defer enumRows.Close()
	var enumValues []string
	for enumRows.Next() {
		var value string
		if err := enumRows.Scan(&value); err != nil {
			t.Fatal(err)
		}
		enumValues = append(enumValues, value)
	}
	wantEnumValues := []string{"CONTINENT", "GEOGRAPHIC", "MULTILATERAL", "INVESTMENT"}
	if !reflect.DeepEqual(enumValues, wantEnumValues) {
		t.Fatalf("region_type enum = %q, want %q", enumValues, wantEnumValues)
	}
	goEnumValues := []string{
		string(RegionTypeContinent),
		string(RegionTypeGeographic),
		string(RegionTypeMultilateral),
		string(RegionTypeInvestment),
	}
	if !reflect.DeepEqual(goEnumValues, wantEnumValues) {
		t.Fatalf("RegionType constants = %q, want %q", goEnumValues, wantEnumValues)
	}
	if !strings.Contains(string(schema), `Enum="`+strings.Join(wantEnumValues, ",")+`"`) {
		t.Fatal("Region Object Schema enum does not match PostgreSQL region_type")
	}
}

func openRegionTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_region", migrationDir, 0)
}
