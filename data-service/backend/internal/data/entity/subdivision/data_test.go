package subdivision

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

const (
	chinaID        = "COU11111111-1111-4111-8111-111111111111"
	unitedStatesID = "COU22222222-2222-4222-8222-222222222222"
	canadaID       = "COU33333333-3333-4333-8333-333333333333"
)

func TestStorePersistsAndListsSubdivisionsByCountry(t *testing.T) {
	db := openSubdivisionTestDatabase(t)
	seedSubdivisionCountries(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	positioning := "国际金融与航运中心"
	resources := "国际资本与专业服务"

	hongKong, err := store.Create(ctx, withSubdivisionID(t, Subdivision{
		Code:                 "HK",
		Name:                 "香港特别行政区",
		NameEn:               "Hong Kong",
		CountryID:            chinaID,
		SubdivisionType:      SubdivisionTypeSAR,
		StrategicPositioning: &positioning,
		KeyResources:         &resources,
	}))
	if err != nil {
		t.Fatalf("Create(HK) error = %v", err)
	}
	if !coreid.Is(hongKong.ID, coreid.Subdivision) || len(hongKong.ID) != 39 {
		t.Fatalf("Create(HK) ID = %q, want canonical SUB identity", hongKong.ID)
	}
	if hongKong.CreatedAt.IsZero() || hongKong.UpdatedAt.IsZero() || !hongKong.CreatedAt.Equal(hongKong.UpdatedAt) || time.Since(hongKong.CreatedAt) > time.Minute {
		t.Fatalf("Create(HK) times = %s, %s, want recent equal database times", hongKong.CreatedAt, hongKong.UpdatedAt)
	}

	californiaChina, err := store.Create(ctx, withSubdivisionID(t, Subdivision{
		Code: "CA", Name: "测试省", NameEn: "Test Province", CountryID: chinaID,
		SubdivisionType: SubdivisionTypeProvince,
	}))
	if err != nil {
		t.Fatalf("Create(CN-CA) error = %v", err)
	}
	if californiaChina.StrategicPositioning != nil || californiaChina.KeyResources != nil {
		t.Fatalf("Create(CN-CA) optional text = %v, %v, want nil", californiaChina.StrategicPositioning, californiaChina.KeyResources)
	}

	if _, err := store.Create(ctx, withSubdivisionID(t, Subdivision{
		Code: "CA", Name: "加利福尼亚州", NameEn: "California", CountryID: unitedStatesID,
		SubdivisionType: SubdivisionTypeState,
	})); err != nil {
		t.Fatalf("Create(US-CA) error = %v", err)
	}

	byID, err := store.Get(ctx, hongKong.ID)
	if err != nil {
		t.Fatalf("Get(HK) error = %v", err)
	}
	if !reflect.DeepEqual(byID, hongKong) {
		t.Fatalf("Get(HK) = %+v, want %+v", byID, hongKong)
	}

	china, err := store.ListByCountry(ctx, chinaID)
	if err != nil {
		t.Fatalf("ListByCountry(CN) error = %v", err)
	}
	if len(china) != 2 || china[0].Code != "CA" || china[1].Code != "HK" {
		t.Fatalf("ListByCountry(CN) = %+v, want CA then HK", china)
	}
	unitedStates, err := store.ListByCountry(ctx, unitedStatesID)
	if err != nil {
		t.Fatalf("ListByCountry(US) error = %v", err)
	}
	if len(unitedStates) != 1 || unitedStates[0].Code != "CA" {
		t.Fatalf("ListByCountry(US) = %+v, want only CA", unitedStates)
	}
	empty, err := store.ListByCountry(ctx, canadaID)
	if err != nil {
		t.Fatalf("ListByCountry(CA) error = %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("ListByCountry(CA) = %#v, want non-nil empty list", empty)
	}
}

func TestStoreEnforcesSubdivisionContractsAndClassifiesFailures(t *testing.T) {
	db := openSubdivisionTestDatabase(t)
	seedSubdivisionCountries(t, db)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := Subdivision{
		Code: "11", Name: "北京市", NameEn: "Beijing", CountryID: chinaID,
		SubdivisionType: SubdivisionTypeProvince,
	}
	created, err := store.Create(ctx, withSubdivisionID(t, valid))
	if err != nil {
		t.Fatal(err)
	}
	for index, subdivisionType := range []SubdivisionType{
		SubdivisionTypeState, SubdivisionTypeSAR, SubdivisionTypeTerritory,
	} {
		candidate := withSubdivisionID(t, valid)
		candidate.Code = []string{"ST", "HK", "T1"}[index]
		candidate.Name = []string{"测试州", "香港特别行政区", "测试领地"}[index]
		candidate.NameEn = []string{"Test State", "Hong Kong", "Test Territory"}[index]
		candidate.SubdivisionType = subdivisionType
		if _, err := store.Create(ctx, candidate); err != nil {
			t.Fatalf("Create(%s) error = %v", subdivisionType, err)
		}
	}

	if _, err := store.Create(ctx, withSubdivisionID(t, valid)); !errors.Is(err, ErrConflict) {
		t.Fatalf("Create(duplicate Country/code) error = %v, want ErrConflict", err)
	}
	unknownCountry := withSubdivisionID(t, valid)
	unknownCountry.Code = "UK"
	unknownCountry.CountryID = "COU44444444-4444-4444-8444-444444444444"
	if _, err := store.Create(ctx, unknownCountry); !errors.Is(err, ErrCountryNotFound) {
		t.Fatalf("Create(unknown Country) error = %v, want ErrCountryNotFound", err)
	}

	blank := " \t"
	invalid := []Subdivision{
		{Code: "", Name: "空代码", NameEn: "Empty Code", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "hk", Name: "小写", NameEn: "Lowercase", CountryID: chinaID, SubdivisionType: SubdivisionTypeSAR},
		{Code: "H-K", Name: "连接符", NameEn: "Hyphen", CountryID: chinaID, SubdivisionType: SubdivisionTypeSAR},
		{Code: "ABCDEFGHIJK", Name: "过长代码", NameEn: "Long Code", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "BN", Name: " ", NameEn: "Blank Name", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "BE", Name: "空英文名", NameEn: " ", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "LN", Name: strings.Repeat("区", 101), NameEn: "Long Name", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "LE", Name: "过长英文名", NameEn: strings.Repeat("a", 101), CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince},
		{Code: "SP", Name: "空战略定位", NameEn: "Blank Positioning", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince, StrategicPositioning: &blank},
		{Code: "KR", Name: "空关键资源", NameEn: "Blank Resources", CountryID: chinaID, SubdivisionType: SubdivisionTypeProvince, KeyResources: &blank},
		{Code: "PF", Name: "未支持类型", NameEn: "Unsupported Type", CountryID: chinaID, SubdivisionType: SubdivisionType("PREFECTURE")},
		{Code: "CI", Name: "错误国家身份", NameEn: "Invalid Country ID", CountryID: "SUB33333333-3333-4333-8333-333333333333", SubdivisionType: SubdivisionTypeProvince},
	}
	for _, candidate := range invalid {
		candidate = withSubdivisionID(t, candidate)
		if _, err := store.Create(ctx, candidate); !errors.Is(err, ErrInvalidSubdivision) {
			t.Errorf("Create(%+v) error = %v, want ErrInvalidSubdivision", candidate, err)
		}
	}

	missingID := "SUB44444444-4444-4444-8444-444444444444"
	if _, err := store.Get(ctx, missingID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := store.Get(ctx, "COU44444444-4444-4444-8444-444444444444"); !errors.Is(err, ErrInvalidSubdivision) {
		t.Fatalf("Get(Country ID) error = %v, want ErrInvalidSubdivision", err)
	}
	if _, err := store.ListByCountry(ctx, "SUB44444444-4444-4444-8444-444444444444"); !errors.Is(err, ErrInvalidSubdivision) {
		t.Fatalf("ListByCountry(Subdivision ID) error = %v, want ErrInvalidSubdivision", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM countries WHERE id = $1`, chinaID); err == nil {
		t.Fatal("deleting a Country with Subdivisions succeeded")
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.Get(cancelled, created.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get(cancelled) error = %v, want context.Canceled", err)
	}
	if _, err := store.ListByCountry(cancelled, chinaID); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListByCountry(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestSubdivisionDatabaseSchemaAndConstraints(t *testing.T) {
	db := openSubdivisionTestDatabase(t)
	seedSubdivisionCountries(t, db)
	ctx := context.Background()

	type column struct {
		name       string
		nullable   string
		dataType   string
		maxLength  sql.NullInt64
		defaultSQL sql.NullString
	}
	rows, err := db.QueryContext(ctx, `
SELECT column_name, is_nullable, udt_name, character_maximum_length, column_default
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'subdivisions'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := make([]column, 0, 10)
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
		{name: "id", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{name: "code", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 10, Valid: true}},
		{name: "name", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 100, Valid: true}},
		{name: "name_en", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 100, Valid: true}},
		{name: "country_id", nullable: "NO", dataType: "varchar", maxLength: sql.NullInt64{Int64: 39, Valid: true}},
		{name: "subdivision_type", nullable: "NO", dataType: "subdivision_type"},
		{name: "strategic_positioning", nullable: "YES", dataType: "text"},
		{name: "key_resources", nullable: "YES", dataType: "text"},
		{name: "created_at", nullable: "NO", dataType: "timestamptz"},
		{name: "updated_at", nullable: "NO", dataType: "timestamptz"},
	}
	if len(columns) != len(wantColumns) {
		t.Fatalf("subdivisions columns = %#v, want %#v", columns, wantColumns)
	}
	for index, want := range wantColumns {
		got := columns[index]
		if got.name != want.name || got.nullable != want.nullable || got.dataType != want.dataType || got.maxLength != want.maxLength {
			t.Fatalf("subdivisions column %d = %#v, want %#v", index, got, want)
		}
	}
	for _, index := range []int{8, 9} {
		if !columns[index].defaultSQL.Valid || !strings.Contains(columns[index].defaultSQL.String, "now()") {
			t.Fatalf("subdivisions.%s default = %#v, want database now()", columns[index].name, columns[index].defaultSQL)
		}
	}

	enumRows, err := db.QueryContext(ctx, `
SELECT enumlabel
FROM pg_enum
JOIN pg_type ON pg_type.oid = pg_enum.enumtypid
JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
WHERE pg_type.typname = 'subdivision_type'
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
	if want := []string{"PROVINCE", "STATE", "SAR", "TERRITORY"}; !reflect.DeepEqual(enumValues, want) {
		t.Fatalf("subdivision_type enum = %q, want %q", enumValues, want)
	}

	const validID = "SUB55555555-5555-4555-8555-555555555555"
	if _, err := db.ExecContext(ctx, `
INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type)
VALUES ($1, 'HK', '香港特别行政区', 'Hong Kong', $2, 'SAR')`, validID, chinaID); err != nil {
		t.Fatal(err)
	}
	statements := map[string]string{
		"invalid identity":       `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUB_HK', 'I1', '错误身份', 'Invalid ID', '` + chinaID + `', 'PROVINCE')`,
		"lowercase code":         `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUB66666666-6666-4666-8666-666666666666', 'hk', '小写', 'Lowercase', '` + chinaID + `', 'SAR')`,
		"punctuated code":        `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUB77777777-7777-4777-8777-777777777777', 'H-K', '连接符', 'Hyphen', '` + chinaID + `', 'SAR')`,
		"overlong code":          `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUB88888888-8888-4888-8888-888888888888', 'ABCDEFGHIJK', '过长', 'Overlong', '` + chinaID + `', 'PROVINCE')`,
		"blank name":             `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUB99999999-9999-4999-8999-999999999999', 'B1', ' ', 'Blank', '` + chinaID + `', 'PROVINCE')`,
		"blank optional":         `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type, key_resources) VALUES ('SUBaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', 'B2', '空资源', 'Blank Resources', '` + chinaID + `', 'PROVINCE', ' ')`,
		"unsupported type":       `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUBbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', 'PF', '县', 'Prefecture', '` + chinaID + `', 'PREFECTURE')`,
		"unknown Country":        `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUBcccccccc-cccc-4ccc-8ccc-cccccccccccc', 'UK', '未知国家', 'Unknown Country', 'COUeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee', 'PROVINCE')`,
		"duplicate Country code": `INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type) VALUES ('SUBdddddddd-dddd-4ddd-8ddd-dddddddddddd', 'HK', '重复香港', 'Duplicate Hong Kong', '` + chinaID + `', 'SAR')`,
	}
	for name, statement := range statements {
		t.Run(name, func(t *testing.T) {
			if _, err := db.ExecContext(ctx, statement); err == nil {
				t.Fatalf("subdivisions accepted %s", name)
			}
		})
	}
}

func TestStoreFailsClosedForInvalidPersistedSubdivision(t *testing.T) {
	db := openSubdivisionTestDatabase(t)
	seedSubdivisionCountries(t, db)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `ALTER TABLE subdivisions DROP CONSTRAINT chk_subdivisions_identity`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO subdivisions (id, code, name, name_en, country_id, subdivision_type)
VALUES ('SUB_HK', 'HK', '香港特别行政区', 'Hong Kong', $1, 'SAR')`, chinaID); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ListByCountry(ctx, chinaID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("ListByCountry(invalid persisted ID) error = %v, want ErrPersistence", err)
	}
}

func openSubdivisionTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_subdivision", migrationDir, 0)
}

func seedSubdivisionCountries(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO countries (id, code, name, name_en)
VALUES
    ($1, 'CN', '中国', 'China'),
    ($2, 'US', '美国', 'United States'),
    ($3, 'CA', '加拿大', 'Canada')`, chinaID, unitedStatesID, canadaID); err != nil {
		t.Fatal(err)
	}
}

func withSubdivisionID(t *testing.T, input Subdivision) Subdivision {
	t.Helper()
	id, err := coreid.New(coreid.Subdivision)
	if err != nil {
		t.Fatal(err)
	}
	input.ID = id
	return input
}
