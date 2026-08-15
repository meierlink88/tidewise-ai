package organization_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
	organizationdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/organization"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStorePublishesCatalogAndPersistsOrganizationFacts(t *testing.T) {
	db := openOrganizationDatabase(t, "tw_organization_data")
	ctx := context.Background()
	publication := organizationdata.CurrentCatalog()
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
	if len(catalog.Categories) != 4 || len(catalog.Functions) != 7 || len(catalog.DomainTags) != 20 || catalog.Categories[0].NameZh != "多边对话与合作机制" {
		t.Fatalf("published catalog = %#v", catalog)
	}
	current := organizationdata.CurrentCatalog()
	if err := organizationdata.PublishCatalog(ctx, db, current); err != nil {
		t.Fatal(err)
	}
	if err := organizationdata.PublishCatalog(ctx, db, current); err != nil {
		t.Fatalf("idempotent catalog publication: %v", err)
	}
	seedOrganizationReferences(t, db)

	input := organizationFact()
	created, err := store.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || created.Category.NameZh != "政府间国际组织" || created.Function.NameZh != "安全与防务" || len(created.DomainTags) != 0 {
		t.Fatalf("created Organization = %#v", created)
	}
	if _, err := store.Create(ctx, input); !errors.Is(err, organizationbiz.ErrConflict) {
		t.Fatalf("duplicate Create() error = %v, want conflict", err)
	}
	tagged, err := store.ReplaceDomainTags(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", []string{"REGIONAL_SECURITY_DIALOGUE"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged.DomainTags) != 1 || tagged.DomainTags[0].FunctionCode != "SECURITY" {
		t.Fatalf("tagged Organization = %#v", tagged)
	}
	if _, err := store.ReplaceDomainTags(ctx, "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", []string{"AI_TECHNOLOGY_AND_GOVERNANCE"}); err == nil {
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

func TestStoreEnforcesMembershipHistoryAndClassifiesReferences(t *testing.T) {
	db := openOrganizationDatabase(t, "tw_organization_members")
	ctx := context.Background()
	if err := organizationdata.PublishCatalog(ctx, db, organizationdata.CurrentCatalog()); err != nil {
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
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "FULL_MEMBER", EffectiveDate: &effective2020,
	})
	if err != nil {
		t.Fatal(err)
	}
	effective2021 := date(2021, 1, 1)
	if _, err := store.CreateMember(ctx, organizationbiz.Member{
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
		OrganizationID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", CountryID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", MembershipType: "OBSERVER", EffectiveDate: &effective2021,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatalf("membership IDs did not advance: first=%d second=%d", first.ID, second.ID)
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
		Category: organizationbiz.CatalogTerm{Code: "INTERGOVERNMENTAL"}, Function: organizationbiz.CatalogTerm{Code: "SECURITY"},
		DominantPartyID: &countryID, BindingPowerLevel: &binding, InfluenceRating: &influence,
		HeadquartersCountryID: &countryID,
	}
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
