package organization_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
	organizationdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/organization"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStorePublishesCatalogAndPersistsOrganizationFacts(t *testing.T) {
	db := openOrganizationDatabase(t, "tw_organization_data")
	ctx := context.Background()
	publication := currentCatalog(t)
	publication.Categories[0].NameZh = "多边对话与合作机制"
	publication.DomainTags = publication.DomainTags[:len(publication.DomainTags)-1]
	if err := organizationdata.PublishCatalog(ctx, db, publication); err != nil {
		t.Fatal(err)
	}
	store, err := organizationdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := store.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Categories) != 4 || len(catalog.Functions) != 7 || len(catalog.DomainTags) != 20 ||
		catalog.Categories[0].ID == "" || catalog.Functions[0].ID == "" || catalog.DomainTags[0].ID == "" || catalog.Categories[0].NameZh != "多边对话与合作机制" {
		t.Fatalf("published catalog = %#v", catalog)
	}
	current := currentCatalog(t)
	if err := organizationdata.PublishCatalog(ctx, db, current); err != nil {
		t.Fatal(err)
	}
	if err := organizationdata.PublishCatalog(ctx, db, current); err != nil {
		t.Fatalf("idempotent catalog publication: %v", err)
	}
	drifted := currentCatalog(t)
	for index := range drifted.Functions {
		if drifted.Functions[index].Code == "SECURITY" {
			drifted.Functions[index].ID = "OFN11111111-1111-4111-8111-111111111111"
		}
	}
	if err := organizationdata.PublishCatalog(ctx, db, drifted); !errors.Is(err, organizationbiz.ErrConflict) {
		t.Fatalf("drifted Function identity error = %v, want conflict", err)
	}
	seedOrganizationReferences(t, db)

	input := organizationFact()
	created, err := store.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || created.Category.NameZh != "政府间国际组织" || created.Function.ID != "OFN1cd93122-d11c-5059-87aa-ddb8b2f2d25b" || created.Function.NameZh != "安全与防务" || len(created.DomainTags) != 0 {
		t.Fatalf("created Organization = %#v", created)
	}
	if _, err := store.Create(ctx, input); !errors.Is(err, organizationbiz.ErrConflict) {
		t.Fatalf("duplicate Create() error = %v, want conflict", err)
	}
	tagged, err := store.ReplaceDomainTags(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", []organizationbiz.DomainTagLink{{ID: "ODL72d3c5ae-74ec-5d5e-9d3e-0d5ebbd189e8", DomainTagID: "ODT37166e5a-05da-5972-b5a8-ff2c85ddc76a", DomainTagCode: "REGIONAL_SECURITY_DIALOGUE"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged.DomainTags) != 1 || tagged.DomainTags[0].FunctionCode != "SECURITY" {
		t.Fatalf("tagged Organization = %#v", tagged)
	}
	var storedLinkID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM organization_domain_tag_links WHERE organization_id=$1`, tagged.ID).Scan(&storedLinkID); err != nil {
		t.Fatal(err)
	}
	if storedLinkID != "ODL72d3c5ae-74ec-5d5e-9d3e-0d5ebbd189e8" {
		t.Fatalf("Organization Domain Tag Link ID = %q", storedLinkID)
	}
	assertPostgresCode(t, db, "23514", `INSERT INTO organization_categories(id,code,name_zh) VALUES('BAD','BAD_CATEGORY_ID','Bad')`)
	assertPostgresCode(t, db, "23505", `INSERT INTO organization_categories(id,code,name_zh) VALUES('OCA11111111-1111-4111-8111-111111111111','INTERGOVERNMENTAL','Duplicate')`)
	assertPostgresCode(t, db, "23514", `INSERT INTO organization_functions(id,code,name_zh) VALUES('BAD','BAD_FUNCTION_ID','Bad')`)
	assertPostgresCode(t, db, "23505", `INSERT INTO organization_functions(id,code,name_zh) VALUES('OFN11111111-1111-4111-8111-111111111111','SECURITY','Duplicate')`)
	assertPostgresCode(t, db, "23514", `INSERT INTO organization_domain_tags(id,code,function_code,name_zh) VALUES('BAD','BAD_TAG_ID','SECURITY','Bad')`)
	assertPostgresCode(t, db, "23505", `INSERT INTO organization_domain_tags(id,code,function_code,name_zh) VALUES('ODT11111111-1111-4111-8111-111111111111','REGIONAL_SECURITY_DIALOGUE','SECURITY','Duplicate')`)
	assertPostgresCode(t, db, "23514", `INSERT INTO organization_domain_tag_links(id,organization_id,function_code,domain_tag_code) VALUES('BAD',$1,'SECURITY','REGIONAL_SECURITY_DIALOGUE')`, tagged.ID)
	assertPostgresCode(t, db, "23505", `INSERT INTO organization_domain_tag_links(id,organization_id,function_code,domain_tag_code) VALUES('ODL11111111-1111-4111-8111-111111111111',$1,'SECURITY','REGIONAL_SECURITY_DIALOGUE')`, tagged.ID)
	if _, err := store.ReplaceDomainTags(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", []organizationbiz.DomainTagLink{{ID: "ODL876bcfd7-84cd-53b2-a2ac-e8bf42977183", DomainTagID: "ODT910b9189-86fa-5fcb-93c2-568212eb9d29", DomainTagCode: "AI_TECHNOLOGY_AND_GOVERNANCE"}}); err == nil {
		t.Fatal("cross-Function Domain Tag error = nil")
	} else {
		var reference *organizationbiz.ReferenceError
		if !errors.As(err, &reference) {
			t.Fatalf("cross-Function Domain Tag error = %T %v", err, err)
		}
	}
	listed, err := store.List(ctx, organizationbiz.Filter{CategoryCode: "INTERGOVERNMENTAL", FunctionCode: "SECURITY", RegionID: "REG13802abf-d1ef-5dec-95ec-a47d35813827"})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" {
		t.Fatalf("filtered Organizations = %#v", listed)
	}
}

func TestOrganizationFunctionSchemaAndPersistenceStayAligned(t *testing.T) {
	db := openOrganizationDatabase(t, "tw_organization_function_schema")
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "doctype", "organization-function.schema"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{
		"id(核心职能标识): Text",
		`constraint: NotNull, Regular="^OFN[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"`,
		"code(核心职能代码): Text",
		"nameZh(中文名称): Text",
		"createdAt(创建时间): Text",
		"updatedAt(更新时间): Text",
	} {
		if !strings.Contains(string(schema), contract) {
			t.Fatalf("Organization Function Object Schema is missing %q", contract)
		}
	}

	type column struct {
		name       string
		nullable   string
		maxLength  sql.NullInt64
		defaultSQL sql.NullString
	}
	rows, err := db.QueryContext(context.Background(), `
SELECT column_name, is_nullable, character_maximum_length, column_default
FROM information_schema.columns
WHERE table_schema=current_schema() AND table_name='organization_functions'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := make(map[string]column)
	for rows.Next() {
		var item column
		if err := rows.Scan(&item.name, &item.nullable, &item.maxLength, &item.defaultSQL); err != nil {
			t.Fatal(err)
		}
		columns[item.name] = item
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for name, maxLength := range map[string]int64{"id": 39, "code": 30, "name_zh": 100} {
		item, exists := columns[name]
		if !exists || item.nullable != "NO" || !item.maxLength.Valid || item.maxLength.Int64 != maxLength || item.defaultSQL.Valid {
			t.Fatalf("organization_functions.%s = %#v", name, item)
		}
	}
	for _, name := range []string{"created_at", "updated_at"} {
		item, exists := columns[name]
		if !exists || item.nullable != "NO" || !item.defaultSQL.Valid || !strings.Contains(item.defaultSQL.String, "now()") {
			t.Fatalf("organization_functions.%s = %#v", name, item)
		}
	}

	var primaryKeyColumn string
	if err := db.QueryRowContext(context.Background(), `
SELECT kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON kcu.constraint_schema=tc.constraint_schema AND kcu.constraint_name=tc.constraint_name
WHERE tc.table_schema=current_schema() AND tc.table_name='organization_functions'
  AND tc.constraint_type='PRIMARY KEY'`).Scan(&primaryKeyColumn); err != nil {
		t.Fatal(err)
	}
	if primaryKeyColumn != "id" {
		t.Fatalf("Organization Function primary key = %q", primaryKeyColumn)
	}
	var codeUnique bool
	if err := db.QueryRowContext(context.Background(), `
SELECT EXISTS(
    SELECT 1
    FROM information_schema.table_constraints tc
    JOIN information_schema.key_column_usage kcu
      ON kcu.constraint_schema=tc.constraint_schema AND kcu.constraint_name=tc.constraint_name
    WHERE tc.table_schema=current_schema() AND tc.table_name='organization_functions'
      AND tc.constraint_type='UNIQUE' AND kcu.column_name='code'
)`).Scan(&codeUnique); err != nil {
		t.Fatal(err)
	}
	if !codeUnique {
		t.Fatal("Organization Function code is not uniquely constrained")
	}
}

func assertPostgresCode(t *testing.T, db *sql.DB, want, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != want {
		t.Fatalf("PostgreSQL error = %T %v, want SQLSTATE %s", err, err, want)
	}
}

func TestStoreEnforcesMembershipHistoryAndClassifiesReferences(t *testing.T) {
	db := openOrganizationDatabase(t, "tw_organization_members")
	ctx := context.Background()
	if err := organizationdata.PublishCatalog(ctx, db, currentCatalog(t)); err != nil {
		t.Fatal(err)
	}
	seedOrganizationReferences(t, db)
	store, err := organizationdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, organizationFact()); err != nil {
		t.Fatal(err)
	}
	effective2020 := date(2020, 1, 1)
	first, err := store.CreateMember(ctx, organizationbiz.Member{
		ID:             "OMB11111111-1111-4111-8111-111111111111",
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "FULL_MEMBER", EffectiveDate: &effective2020,
	})
	if err != nil {
		t.Fatal(err)
	}
	effective2021 := date(2021, 1, 1)
	if _, err := store.CreateMember(ctx, organizationbiz.Member{
		ID:             "OMB22222222-2222-4222-8222-222222222222",
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "OBSERVER", EffectiveDate: &effective2021,
	}); !errors.Is(err, organizationbiz.ErrConflict) {
		t.Fatalf("overlapping membership error = %v, want conflict", err)
	}
	expiry2020 := date(2020, 12, 31)
	first.ExpiryDate = &expiry2020
	if _, err := store.UpdateMember(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", first.ID, first); err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateMember(ctx, organizationbiz.Member{
		ID:             "OMB33333333-3333-4333-8333-333333333333",
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "OBSERVER", EffectiveDate: &effective2021,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatalf("membership IDs are equal: first=%s second=%s", first.ID, second.ID)
	}
	asOf2020 := date(2020, 6, 1)
	members, err := store.ListMembers(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", &asOf2020)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].MembershipType != "FULL_MEMBER" {
		t.Fatalf("2020 membership = %#v", members)
	}
	organizations, err := store.List(ctx, organizationbiz.Filter{CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", AsOfDate: &effective2021})
	if err != nil {
		t.Fatal(err)
	}
	if len(organizations) != 1 || organizations[0].ID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" {
		t.Fatalf("Country membership filter = %#v", organizations)
	}
	if _, err := store.CreateMember(ctx, organizationbiz.Member{
		ID:             "OMB44444444-4444-4444-8444-444444444444",
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COU9975a356-4385-5e0f-822c-22463a8a1ad2", MembershipType: "OBSERVER",
	}); err == nil {
		t.Fatal("missing Country reference error = nil")
	} else {
		var reference *organizationbiz.ReferenceError
		if !errors.As(err, &reference) {
			t.Fatalf("missing Country error = %T %v", err, err)
		}
	}
}

func openOrganizationDatabase(t *testing.T, prefix string) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, prefix, migrationDir, 0)
}

func organizationFact() organizationbiz.Organization {
	binding, influence := "HIGH", "S"
	regionID, countryID := "REG13802abf-d1ef-5dec-95ec-a47d35813827", "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b"
	return organizationbiz.Organization{
		ID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", Code: "UN", Name: "联合国", NameEn: "United Nations", RegionID: &regionID,
		Category: organizationbiz.Category{Code: "INTERGOVERNMENTAL"}, Function: organizationbiz.Function{Code: "SECURITY"},
		DominantPartyID: &countryID, BindingPowerLevel: &binding, InfluenceRating: &influence,
		HeadquartersCountryID: &countryID,
	}
}

func currentCatalog(t *testing.T) organizationbiz.Catalog {
	t.Helper()
	catalog, err := organizationbiz.AssignCatalogIdentities(organizationdata.CurrentCatalog())
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func seedOrganizationReferences(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO regions(id,code,name,name_en,region_type)
VALUES('REG13802abf-d1ef-5dec-95ec-a47d35813827','GLOBAL','全球','Global','GEOGRAPHIC');
INSERT INTO countries(id,code,name,name_en)
VALUES('COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b','CN','中国','China');`); err != nil {
		t.Fatal(err)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
