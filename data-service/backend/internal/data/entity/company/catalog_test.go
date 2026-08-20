package company

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

const (
	catalogAlphaID = "COMaaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	catalogBetaID  = "COMbbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func TestLoadCurrentCompanyCatalogPackage(t *testing.T) {
	publication, err := LoadCatalog(context.Background(), companyCatalogPath(t))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if publication.SchemaVersion != 1 || publication.PublicationMode != CatalogPublicationModeReplace ||
		publication.AsOf != "2026-08-20" || publication.ExpectedCompanyCount != 13264 ||
		len(publication.Companies) != 13264 {
		t.Fatalf("catalog metadata = schema %d mode %q as_of %q expected %d actual %d",
			publication.SchemaVersion, publication.PublicationMode, publication.AsOf,
			publication.ExpectedCompanyCount, len(publication.Companies))
	}
	if publication.SourceSnapshot.ExcludedSecurityRows != 7999 || publication.SourceSnapshot.AHCrosswalkPairs != 197 {
		t.Fatalf("source snapshot = %#v", publication.SourceSnapshot)
	}
	if got := publication.Companies[0]; got.Code != "cn_360_security_technology_inc" || got.ID != "COM523010a3-6e4d-51fb-bbcc-b81b18aa25d5" {
		t.Fatalf("first Company = %#v", got)
	}
	if got := publication.Companies[len(publication.Companies)-1]; got.Code != "us_zymeworks_inc" || got.ID != "COM7c0d7a94-6cca-56a5-9f90-60c02c85dbb7" {
		t.Fatalf("last Company = %#v", got)
	}
}

func TestValidateCompanyCatalogRejectsInvalidPackages(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CatalogPublication)
	}{
		{"unknown schema", func(value *CatalogPublication) { value.SchemaVersion = 2 }},
		{"unknown mode", func(value *CatalogPublication) { value.PublicationMode = "reconcile" }},
		{"invalid as of", func(value *CatalogPublication) { value.AsOf = "2026/08/20" }},
		{"count mismatch", func(value *CatalogPublication) { value.ExpectedCompanyCount++ }},
		{"duplicate id", func(value *CatalogPublication) { value.Companies[1].ID = value.Companies[0].ID }},
		{"duplicate code", func(value *CatalogPublication) { value.Companies[1].Code = value.Companies[0].Code }},
		{"invalid company", func(value *CatalogPublication) { value.Companies[0].Status = "unknown" }},
		{"unsorted aliases", func(value *CatalogPublication) { value.Companies[0].Aliases = []string{"Z", "A"} }},
		{"invalid checksum", func(value *CatalogPublication) { value.SourceSnapshot.Files[0].SHA256 = "bad" }},
		{"excluded count mismatch", func(value *CatalogPublication) { value.SourceSnapshot.ExcludedSecurityRows = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := testCompanyCatalog()
			test.mutate(&publication)
			if err := validateCatalog(publication); !errors.Is(err, ErrInvalidCompanyCatalog) {
				t.Fatalf("validateCatalog() error = %v, want ErrInvalidCompanyCatalog", err)
			}
		})
	}
}

func TestLoadCompanyCatalogRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	publication := testCompanyCatalog()
	payload, err := json.Marshal(publication)
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := strings.Replace(string(payload), `"schema_version":1`, `"unknown":true,"schema_version":1`, 1)
	if _, err := LoadCatalog(context.Background(), writeCompanyCatalog(t, []byte(withUnknown))); err == nil {
		t.Fatal("LoadCatalog() accepted an unknown field")
	}
	if _, err := LoadCatalog(context.Background(), writeCompanyCatalog(t, append(payload, []byte("\n{}")...))); err == nil {
		t.Fatal("LoadCatalog() accepted trailing JSON")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadCatalog(cancelled, companyCatalogPath(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadCatalog(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestPublishCompanyCatalogReplacesAtomicallyAndIsIdempotent(t *testing.T) {
	db := openCompanyCatalogTestDatabase(t, "tw_company_catalog_publish")
	ctx := context.Background()
	publication := testCompanyCatalog()
	if _, err := db.ExecContext(ctx, `INSERT INTO company (id, code, name, aliases, status) VALUES (
    'COMcccccccc-cccc-4ccc-8ccc-cccccccccccc', 'legacy', 'Legacy Company', '{}', 'active'
)`); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog() error = %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.List(ctx)
	if err != nil || len(first) != 2 {
		t.Fatalf("List() = %#v, %v", first, err)
	}
	var legacyCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM company WHERE code = 'legacy'`).Scan(&legacyCount); err != nil {
		t.Fatal(err)
	}
	if legacyCount != 0 {
		t.Fatalf("legacy Company count = %d, want 0", legacyCount)
	}
	var alphaUpdatedAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM company WHERE id = $1`, catalogAlphaID).Scan(&alphaUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog(repeat) error = %v", err)
	}
	var repeatedUpdatedAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM company WHERE id = $1`, catalogAlphaID).Scan(&repeatedUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if !repeatedUpdatedAt.Equal(alphaUpdatedAt) {
		t.Fatalf("idempotent publication changed updated_at: first %s repeat %s", alphaUpdatedAt, repeatedUpdatedAt)
	}

	if _, err := db.ExecContext(ctx, `UPDATE company SET name = 'Drifted Alpha' WHERE id = $1`, catalogAlphaID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog(replace) error = %v", err)
	}
	after, err := store.Get(ctx, companybiz.ID(catalogAlphaID))
	if err != nil {
		t.Fatal(err)
	}
	if after.Name != publication.Companies[0].Name || len(after.Industries) != 0 {
		t.Fatalf("replaced Company = %#v", after)
	}
	assertCompanyIdentityTriggerEnabled(t, db)
}

func TestPublishCurrentCompanyCatalogHandlesBulkWithoutExhaustingTransactionLocks(t *testing.T) {
	db := openCompanyCatalogTestDatabase(t, "tw_company_catalog_bulk")
	publication, err := LoadCatalog(context.Background(), companyCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(context.Background(), db, publication); err != nil {
		t.Fatalf("PublishCatalog(%d Companies) error = %v", len(publication.Companies), err)
	}
	var companies, links int
	if err := db.QueryRow(`SELECT count(*) FROM company`).Scan(&companies); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM company_industry_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if companies != publication.ExpectedCompanyCount || links != 0 {
		t.Fatalf("published counts = %d Companies, %d Industry links", companies, links)
	}
}

func TestPublishCompanyCatalogFailsClosedWhenIndustryLinksExist(t *testing.T) {
	db := openCompanyCatalogTestDatabase(t, "tw_company_catalog_links")
	ctx := context.Background()
	publication := testCompanyCatalog()
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO industry (
    id, classification_system, industry_code, hierarchy_path_codes,
    definition, review_status, name, aliases
) VALUES (
    $1, 'TIDEWISE', 'CATALOG_TEST', ARRAY['CATALOG_TEST'],
    'Catalog test Industry', 'approved', 'Catalog test', '{}'
)`, industryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO company_industry_links (id, company_id, industry_id) VALUES ($1, $2, $3)`, linkID, catalogAlphaID, industryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE company SET name = 'Drifted Alpha' WHERE id = $1`, catalogAlphaID); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); !errors.Is(err, ErrCompanyCatalogConflict) {
		t.Fatalf("PublishCatalog() error = %v, want ErrCompanyCatalogConflict", err)
	}
	var name string
	var links int
	if err := db.QueryRowContext(ctx, `SELECT name FROM company WHERE id = $1`, catalogAlphaID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM company_industry_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if name != "Drifted Alpha" || links != 1 {
		t.Fatalf("rollback facts = name %q links %d", name, links)
	}
	assertCompanyIdentityTriggerEnabled(t, db)
}

func TestPublishCompanyCatalogRequiresDatabase(t *testing.T) {
	if err := PublishCatalog(context.Background(), nil, testCompanyCatalog()); err == nil {
		t.Fatal("PublishCatalog(nil) error = nil")
	}
}

func testCompanyCatalog() CatalogPublication {
	alphaNameEn := "Alpha Company"
	alphaIPO := "2020-01-02"
	return CatalogPublication{
		SchemaVersion: 1, PublicationMode: CatalogPublicationModeReplace,
		AsOf: "2026-08-20", ExpectedCompanyCount: 2,
		SourceSnapshot: CatalogSourceSnapshot{
			Files:                []CatalogSourceFile{{Name: "companies.csv", SHA256: strings.Repeat("a", 64), Bytes: 10}},
			ExcludedReasonCounts: map[string]int{},
		},
		Companies: []CatalogItem{
			{ID: catalogAlphaID, Code: "alpha", Name: "Alpha", NameEn: &alphaNameEn, Aliases: []string{"Alpha Co"}, IPODate: &alphaIPO, Status: companybiz.StatusActive},
			{ID: catalogBetaID, Code: "beta", Name: "Beta", Aliases: []string{}, Status: companybiz.StatusActive},
		},
	}
}

func assertCompanyIdentityTriggerEnabled(t *testing.T, db *sql.DB) {
	t.Helper()
	var enabled bool
	if err := db.QueryRow(`
SELECT tgenabled = 'O'
FROM pg_trigger
WHERE tgrelid = 'company'::regclass
  AND tgname = 'trg_company_object_identity_unique'`).Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("Company identity trigger is disabled")
	}
}

func companyCatalogPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "companies-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func writeCompanyCatalog(t *testing.T, payload []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "companies.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func openCompanyCatalogTestDatabase(t *testing.T, name string) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, name, migrationDir, 0)
}

func TestCatalogCompanyDoesNotMutateInput(t *testing.T) {
	publication := testCompanyCatalog()
	want := append([]string{}, publication.Companies[0].Aliases...)
	if _, err := catalogCompany(publication.Companies[0]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(publication.Companies[0].Aliases, want) {
		t.Fatalf("catalogCompany() mutated aliases: got %#v want %#v", publication.Companies[0].Aliases, want)
	}
}
