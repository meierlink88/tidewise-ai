package company

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

const (
	companyID  = companybiz.ID("COM11111111-1111-4111-8111-111111111111")
	countryID  = "COU22222222-2222-4222-8222-222222222222"
	industryID = companybiz.IndustryID("IND33333333-3333-4333-8333-333333333333")
	linkID     = "CIL44444444-4444-4444-8444-444444444444"
)

func TestStorePersistsCompanyAndAtomicallyReplacesIndustries(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_company_store", migrationDir, 0)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
INSERT INTO countries (id, code, name, name_en)
VALUES ($1, 'TC', '测试国', 'Test Country')`, countryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
    id, classification_system, industry_code, hierarchy_path_codes,
    definition, review_status, name, aliases
) VALUES (
    $1, 'TIDEWISE', 'SEMICONDUCTOR', ARRAY['SEMICONDUCTOR'],
    '半导体行业', 'approved', '半导体', '{}'
)`, industryID); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	nameEn := "Taiwan Semiconductor Manufacturing Company"
	legalName := "Taiwan Semiconductor Manufacturing Company Limited"
	operatingArea := "全球"
	headquartersCity := "新竹"
	legalForm := "PUBLIC"
	strategicPositioning := "先进制程晶圆代工领导者"
	description := "全球晶圆代工企业"
	foundingDate := time.Date(1987, time.February, 21, 0, 0, 0, 0, time.UTC)
	ipoDate := time.Date(1994, time.September, 5, 0, 0, 0, 0, time.UTC)
	ownership := companybiz.OwnershipDispersed
	created, err := store.Create(ctx, companybiz.Company{
		ID: companyID, Code: "TSM", Name: "台积电", NameEn: &nameEn, LegalName: &legalName,
		Aliases: []string{"TSMC"}, RegistrationCountryID: stringPointer(countryID),
		OperatingArea: &operatingArea, HeadquartersCity: &headquartersCity,
		FoundingDate: &foundingDate, IPODate: &ipoDate, LegalForm: &legalForm,
		OwnershipType: &ownership, StrategicPositioning: &strategicPositioning,
		Description: &description, Status: companybiz.StatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Code != "TSM" || created.Industries == nil || len(created.Industries) != 0 {
		t.Fatalf("Create() = %#v", created)
	}
	if _, err := db.ExecContext(ctx, `UPDATE company SET code = 'NEW' WHERE id = $1`, companyID); err == nil {
		t.Fatal("direct Company code update error = nil")
	}

	linked, err := store.ReplaceIndustries(ctx, companyID, []companybiz.IndustryLink{{ID: linkID, IndustryID: industryID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(linked.Industries) != 1 || linked.Industries[0].ID != industryID || linked.Industries[0].Name != "半导体" {
		t.Fatalf("ReplaceIndustries() industries = %#v", linked.Industries)
	}

	updatedDescription := "更新后的公司描述"
	updated, err := store.Update(ctx, companyID, companybiz.Update{
		Name: "台积电", NameEn: &nameEn, LegalName: &legalName, Aliases: []string{"TSMC", "台积电"},
		RegistrationCountryID: stringPointer(countryID), OperatingArea: &operatingArea,
		HeadquartersCity: &headquartersCity, FoundingDate: &foundingDate, IPODate: &ipoDate,
		LegalForm: &legalForm, OwnershipType: &ownership, StrategicPositioning: &strategicPositioning,
		Description: &updatedDescription, Status: companybiz.StatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Code != "TSM" || updated.Description == nil || *updated.Description != updatedDescription {
		t.Fatalf("Update() = %#v", updated)
	}

	listed, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != companyID || len(listed[0].Industries) != 1 {
		t.Fatalf("List() = %#v", listed)
	}

	cleared, err := store.ReplaceIndustries(ctx, companyID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(cleared.Industries) != 0 {
		t.Fatalf("cleared industries = %#v", cleared.Industries)
	}
}

func TestStoreRejectsUnknownCompanyReferences(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_company_reference", migrationDir, 0)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	unknownCountry := "COU55555555-5555-4555-8555-555555555555"
	_, err = store.Create(context.Background(), companybiz.Company{
		ID: companyID, Code: "UNKNOWN", Name: "Unknown", Aliases: []string{},
		RegistrationCountryID: &unknownCountry, Status: companybiz.StatusActive,
	})
	var referenceError *companybiz.ReferenceError
	if !errors.As(err, &referenceError) || referenceError.Field != "registration_country_id" {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestStoreFailsClosedWhenPersistedCompanyStateIsCorrupt(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_company_corrupt", migrationDir, 0)
	ctx := context.Background()
	for _, constraint := range []string{
		"chk_company_identity",
		"chk_company_aliases",
		"chk_company_date_order",
		"chk_company_status",
		"chk_company_timestamp_order",
		"company_profiles_registration_country_id_fkey",
		"chk_company_industry_link_identity",
		"company_industry_links_industry_id_fkey",
	} {
		table := "company"
		if constraint == "chk_company_industry_link_identity" || constraint == "company_industry_links_industry_id_fkey" {
			table = "company_industry_links"
		}
		if _, err := db.ExecContext(ctx, `ALTER TABLE `+table+` DROP CONSTRAINT `+constraint); err != nil {
			t.Fatalf("drop %s: %v", constraint, err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
    id, classification_system, industry_code, hierarchy_path_codes,
    definition, review_status, name, aliases
) VALUES (
    'IND99999999-9999-4999-8999-999999999999', 'TIDEWISE', 'VALID', ARRAY['VALID'],
    'Valid Industry', 'approved', 'Valid Industry', '{}'
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO company (
    id, code, name, aliases, registration_country_id,
    founding_date, ipo_date, status, created_at, updated_at
) VALUES
    ('COM_bad', 'BAD-ID', 'Bad ID', '{}', NULL, NULL, NULL, 'active', now(), now()),
    ('COMaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', 'BAD-ALIASES', 'Bad aliases', ARRAY['same', 'same'], NULL, NULL, NULL, 'active', now(), now()),
    ('COMbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', 'BAD-DATES', 'Bad dates', '{}', NULL, DATE '2020-01-02', DATE '2020-01-01', 'active', now(), now()),
    ('COMcccccccc-cccc-4ccc-8ccc-cccccccccccc', 'BAD-STATUS', 'Bad status', '{}', NULL, NULL, NULL, 'unknown', now(), now()),
    ('COMdddddddd-dddd-4ddd-8ddd-dddddddddddd', 'BAD-TIMES', 'Bad timestamps', '{}', NULL, NULL, NULL, 'active', now(), now() - interval '1 second'),
	    ('COMeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee', 'BAD-COUNTRY', 'Bad country', '{}', 'COUffffffff-ffff-4fff-8fff-ffffffffffff', NULL, NULL, 'active', now(), now()),
	    ('COMf1111111-1111-4111-8111-111111111111', 'BAD-LINK-ID', 'Bad link ID', '{}', NULL, NULL, NULL, 'active', now(), now()),
	    ('COMf2222222-2222-4222-8222-222222222222', 'BAD-INDUSTRY', 'Bad Industry reference', '{}', NULL, NULL, NULL, 'active', now(), now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO company_industry_links (id, company_id, industry_id) VALUES
    ('CIL_bad', 'COMf1111111-1111-4111-8111-111111111111', 'IND99999999-9999-4999-8999-999999999999'),
    ('CIL77777777-7777-4777-8777-777777777777', 'COMf2222222-2222-4222-8222-222222222222', 'IND88888888-8888-4888-8888-888888888888')`); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []companybiz.ID{
		"COM_bad",
		"COMaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		"COMbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		"COMcccccccc-cccc-4ccc-8ccc-cccccccccccc",
		"COMdddddddd-dddd-4ddd-8ddd-dddddddddddd",
		"COMeeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee",
		"COMf1111111-1111-4111-8111-111111111111",
		"COMf2222222-2222-4222-8222-222222222222",
	} {
		if _, err := store.Get(ctx, id); !errors.Is(err, companybiz.ErrPersistence) {
			t.Errorf("Get(%q) error = %v, want ErrPersistence", id, err)
		}
	}
}

func TestNewStoreRequiresDatabase(t *testing.T) {
	if _, err := NewStore(nil); err == nil {
		t.Fatal("NewStore(nil) error = nil")
	}
}

func TestStoreListsStableCompanyProjectionAndRejectsSnapshotDrift(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_company_projection", migrationDir, 0)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	firstID := companybiz.ID("COMaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	secondID := companybiz.ID("COMbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	for _, input := range []companybiz.Company{
		{ID: secondID, Code: "000002.SZ", Name: "Second", Aliases: []string{}, Status: companybiz.StatusActive},
		{ID: firstID, Code: "000001.SZ", Name: "First", Aliases: []string{}, Status: companybiz.StatusActive},
	} {
		if _, err := store.Create(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.ListProjection(ctx, companybiz.ProjectionListQuery{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(first.SnapshotID) || len(first.Items) != 1 || first.Items[0].ID != firstID || !first.HasMore || first.Items[0].IndustryLinks == nil {
		t.Fatalf("first projection page = %#v", first)
	}
	second, err := store.ListProjection(ctx, companybiz.ProjectionListQuery{
		PageSize: 1, SnapshotID: first.SnapshotID,
		After: &companybiz.ProjectionListKey{Code: first.Items[0].Code, ID: first.Items[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != secondID || second.HasMore || second.SnapshotID != first.SnapshotID {
		t.Fatalf("second projection page = %#v", second)
	}
	if _, err := db.ExecContext(ctx, `UPDATE company SET name = 'Changed', updated_at = now() WHERE id = $1`, secondID); err != nil {
		t.Fatal(err)
	}
	_, err = store.ListProjection(ctx, companybiz.ProjectionListQuery{
		PageSize: 1, SnapshotID: first.SnapshotID,
		After: &companybiz.ProjectionListKey{Code: first.Items[0].Code, ID: first.Items[0].ID},
	})
	if !errors.Is(err, companybiz.ErrProjectionSnapshotChanged) {
		t.Fatalf("projection drift error = %v", err)
	}
	current, err := store.ListProjection(ctx, companybiz.ProjectionListQuery{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
    id, classification_system, industry_code, hierarchy_path_codes,
    definition, review_status, name, aliases
) VALUES (
    'IND33333333-3333-4333-8333-333333333333', 'TIDEWISE', 'PROJECTION', ARRAY['PROJECTION'],
    'Projection Industry', 'approved', 'Projection Industry', '{}'
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO company_industry_links (id, company_id, industry_id) VALUES (
    'CIL44444444-4444-4444-8444-444444444444', $1, 'IND33333333-3333-4333-8333-333333333333'
)`, firstID); err != nil {
		t.Fatal(err)
	}
	_, err = store.ListProjection(ctx, companybiz.ProjectionListQuery{PageSize: 1, SnapshotID: current.SnapshotID})
	if !errors.Is(err, companybiz.ErrProjectionSnapshotChanged) {
		t.Fatalf("CompanyIndustryLink drift error = %v", err)
	}
}

func TestProjectionSnapshotIsIndependentOfTheDatabaseSessionTimezone(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_company_projection_timezone", migrationDir, 0)
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, companybiz.Company{
		ID: companyID, Code: "TIMEZONE", Name: "Timezone", Aliases: []string{}, Status: companybiz.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `SET TIME ZONE 'Asia/Shanghai'`); err != nil {
		t.Fatal(err)
	}
	shanghai, err := store.ListProjection(ctx, companybiz.ProjectionListQuery{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `SET TIME ZONE 'UTC'`); err != nil {
		t.Fatal(err)
	}
	utc, err := store.ListProjection(ctx, companybiz.ProjectionListQuery{PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if shanghai.SnapshotID != utc.SnapshotID {
		t.Fatalf("snapshot depends on session timezone: Asia/Shanghai=%s UTC=%s", shanghai.SnapshotID, utc.SnapshotID)
	}
}

func stringPointer(value string) *string { return &value }
