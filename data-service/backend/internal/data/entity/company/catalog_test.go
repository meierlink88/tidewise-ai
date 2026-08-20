package company

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

const catalogAlphaID = "COM8b19b1f0-1040-54be-b864-434ab109b398"

func TestLoadCurrentCompanyCatalogPackage(t *testing.T) {
	publication, err := LoadCatalog(context.Background(), companyCatalogPath(t))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if publication.SchemaVersion != 2 || publication.PublicationMode != CatalogPublicationModeReplace ||
		publication.AsOf != "2026-08-20" || publication.ExpectedCompanyCount != 13264 ||
		publication.CompanyCodeSetSHA256 != "68ef09e792cc4122626462184fc7d944a662b91b024f182040153cf34cc7f824" ||
		len(publication.Companies) != 13264 {
		t.Fatalf("catalog metadata = schema %d mode %q as_of %q expected %d actual %d",
			publication.SchemaVersion, publication.PublicationMode, publication.AsOf,
			publication.ExpectedCompanyCount, len(publication.Companies))
	}
	if publication.SourceSnapshot.ExcludedSecurityRows != 7999 || publication.SourceSnapshot.AHCrosswalkPairs != 197 {
		t.Fatalf("source snapshot = %#v", publication.SourceSnapshot)
	}
	if got := publication.Companies[0]; got.Code != "cn_360_security_technology_inc" || got.ID != "" {
		t.Fatalf("first Company = %#v", got)
	}
	if got := publication.Companies[len(publication.Companies)-1]; got.Code != "us_zymeworks_inc" || got.ID != "" {
		t.Fatalf("last Company = %#v", got)
	}
	countryIDs := canonicalCountryIDs(t)
	for _, company := range publication.Companies {
		if company.RegistrationCountryID == nil {
			t.Fatalf("Company %q has no registration_country_id", company.Code)
		}
		if _, exists := countryIDs[*company.RegistrationCountryID]; !exists {
			t.Fatalf("Company %q references unknown Country %q", company.Code, *company.RegistrationCountryID)
		}
	}
}

func TestLoadLegacyCompanyCatalogPackageRemainsCompatible(t *testing.T) {
	legacy, err := LoadCatalog(context.Background(), companyCatalogV1Path(t))
	if err != nil {
		t.Fatalf("LoadCatalog(v1) error = %v", err)
	}
	if legacy.SchemaVersion != 1 || legacy.CompanyCodeSetSHA256 != "" ||
		legacy.ExpectedCompanyCount != 13264 || len(legacy.Companies) != 13264 {
		t.Fatalf("legacy catalog metadata = schema %d digest %q expected %d actual %d",
			legacy.SchemaVersion, legacy.CompanyCodeSetSHA256,
			legacy.ExpectedCompanyCount, len(legacy.Companies))
	}
	current, err := LoadCatalog(context.Background(), companyCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	legacyIDs := make(map[string]string, len(legacy.Companies))
	for _, company := range legacy.Companies {
		legacyIDs[company.Code] = company.ID
	}
	for _, item := range current.Companies {
		company, err := catalogCompany(current.SchemaVersion, item)
		if err != nil {
			t.Fatal(err)
		}
		if string(company.ID) != legacyIDs[item.Code] {
			t.Fatalf("Company %q derived ID = %q, legacy ID = %q", item.Code, company.ID, legacyIDs[item.Code])
		}
	}
}

func TestCompanyCountryInferenceAuditMatchesCurrentCatalog(t *testing.T) {
	publication, err := LoadCatalog(context.Background(), companyCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	companies := make(map[string]string, len(publication.Companies))
	for _, company := range publication.Companies {
		companies[company.Code] = *company.RegistrationCountryID
	}

	file, err := os.Open(companyCountryAuditPath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(publication.Companies)+1 {
		t.Fatalf("audit row count = %d, want %d", len(rows)-1, len(publication.Companies))
	}
	if !reflect.DeepEqual(rows[0], []string{"company_code", "company_name", "country_code", "registration_country_id", "method", "confidence", "evidence"}) {
		t.Fatalf("audit header = %#v", rows[0])
	}
	confidenceCounts := map[string]int{}
	seen := make(map[string]struct{}, len(publication.Companies))
	for _, row := range rows[1:] {
		companyCode, registrationCountryID, confidence := row[0], row[3], row[5]
		if companies[companyCode] != registrationCountryID || row[2] == "" || row[4] == "" || row[6] == "" {
			t.Fatalf("invalid audit row for Company %q: %#v", companyCode, row)
		}
		if _, duplicate := seen[companyCode]; duplicate {
			t.Fatalf("duplicate audit Company %q", companyCode)
		}
		seen[companyCode] = struct{}{}
		confidenceCounts[confidence]++
	}
	if !reflect.DeepEqual(confidenceCounts, map[string]int{"high": 10593, "medium": 708, "low": 1963}) {
		t.Fatalf("audit confidence counts = %#v", confidenceCounts)
	}
}

func TestValidateCompanyCatalogRejectsInvalidPackages(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CatalogPublication)
	}{
		{"unknown schema", func(value *CatalogPublication) { value.SchemaVersion = 3 }},
		{"unknown mode", func(value *CatalogPublication) { value.PublicationMode = "reconcile" }},
		{"invalid as of", func(value *CatalogPublication) { value.AsOf = "2026/08/20" }},
		{"count mismatch", func(value *CatalogPublication) { value.ExpectedCompanyCount++ }},
		{"caller id in v2", func(value *CatalogPublication) { value.Companies[0].ID = catalogAlphaID }},
		{"duplicate code", func(value *CatalogPublication) { value.Companies[1].Code = value.Companies[0].Code }},
		{"invalid company", func(value *CatalogPublication) { value.Companies[0].Status = "unknown" }},
		{"unsorted aliases", func(value *CatalogPublication) { value.Companies[0].Aliases = []string{"Z", "A"} }},
		{"invalid checksum", func(value *CatalogPublication) { value.SourceSnapshot.Files[0].SHA256 = "bad" }},
		{"code set checksum mismatch", func(value *CatalogPublication) { value.CompanyCodeSetSHA256 = strings.Repeat("0", 64) }},
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
	withUnknown := strings.Replace(string(payload), `"schema_version":2`, `"unknown":true,"schema_version":2`, 1)
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
	seedCanonicalCountries(t, db)
	if err := PublishCatalog(context.Background(), db, publication); err != nil {
		t.Fatalf("PublishCatalog(%d Companies) error = %v", len(publication.Companies), err)
	}
	var companies, nullCountries, links int
	if err := db.QueryRow(`SELECT count(*) FROM company`).Scan(&companies); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM company WHERE registration_country_id IS NULL`).Scan(&nullCountries); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM company_industry_links`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if companies != publication.ExpectedCompanyCount || nullCountries != 0 || links != 0 {
		t.Fatalf("published counts = %d Companies, %d null Countries, %d Industry links", companies, nullCountries, links)
	}
	var firstUpdatedAtHash string
	if err := db.QueryRow(`SELECT md5(string_agg(id || ':' || updated_at::text, ',' ORDER BY id)) FROM company`).Scan(&firstUpdatedAtHash); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(context.Background(), db, publication); err != nil {
		t.Fatalf("PublishCatalog(repeat %d Companies) error = %v", len(publication.Companies), err)
	}
	var repeatedUpdatedAtHash string
	if err := db.QueryRow(`SELECT md5(string_agg(id || ':' || updated_at::text, ',' ORDER BY id)) FROM company`).Scan(&repeatedUpdatedAtHash); err != nil {
		t.Fatal(err)
	}
	if repeatedUpdatedAtHash != firstUpdatedAtHash {
		t.Fatalf("full catalog repeat changed updated_at hash: first %s repeat %s", firstUpdatedAtHash, repeatedUpdatedAtHash)
	}
}

func TestPublishCompanyCatalogRollsBackForUnknownCountry(t *testing.T) {
	db := openCompanyCatalogTestDatabase(t, "tw_company_catalog_country_reference")
	ctx := context.Background()
	publication := testCompanyCatalog()
	unknownCountryID := "COU11111111-1111-4111-8111-111111111111"
	publication.Companies[0].RegistrationCountryID = &unknownCountryID
	if _, err := db.ExecContext(ctx, `INSERT INTO company (id, code, name, aliases, status) VALUES (
    'COMcccccccc-cccc-4ccc-8ccc-cccccccccccc', 'legacy', 'Legacy Company', '{}', 'active'
)`); err != nil {
		t.Fatal(err)
	}

	err := PublishCatalog(ctx, db, publication)
	var referenceError *companybiz.ReferenceError
	if !errors.As(err, &referenceError) || referenceError.Field != "registration_country_id" {
		t.Fatalf("PublishCatalog() error = %v, want registration_country_id ReferenceError", err)
	}
	var companyCount, legacyCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE code = 'legacy') FROM company`).Scan(&companyCount, &legacyCount); err != nil {
		t.Fatal(err)
	}
	if companyCount != 1 || legacyCount != 1 {
		t.Fatalf("rollback counts = %d Companies, %d legacy", companyCount, legacyCount)
	}
	assertCompanyIdentityTriggerEnabled(t, db)
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
	publication := CatalogPublication{
		SchemaVersion: 2, PublicationMode: CatalogPublicationModeReplace,
		AsOf: "2026-08-20", ExpectedCompanyCount: 2,
		SourceSnapshot: CatalogSourceSnapshot{
			Files:                []CatalogSourceFile{{Name: "companies.csv", SHA256: strings.Repeat("a", 64), Bytes: 10}},
			ExcludedReasonCounts: map[string]int{},
		},
		Companies: []CatalogItem{
			{Code: "alpha", Name: "Alpha", NameEn: &alphaNameEn, Aliases: []string{"Alpha Co"}, IPODate: &alphaIPO, Status: companybiz.StatusActive},
			{Code: "beta", Name: "Beta", Aliases: []string{}, Status: companybiz.StatusActive},
		},
	}
	publication.CompanyCodeSetSHA256 = companyCodeSetSHA256(publication.Companies)
	return publication
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
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "companies-v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func companyCatalogV1Path(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "companies-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func companyCountryAuditPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "company-country-inferences-v1.csv"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func countryCatalogPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "countries-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

type canonicalCountry struct {
	Code                 string  `json:"code"`
	Name                 string  `json:"name"`
	NameEn               string  `json:"name_en"`
	StrategicPositioning *string `json:"strategic_positioning"`
	KeyResources         *string `json:"key_resources"`
}

func loadCanonicalCountries(t *testing.T) []canonicalCountry {
	t.Helper()
	payload, err := os.ReadFile(countryCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Countries []canonicalCountry `json:"countries"`
	}
	if err := json.Unmarshal(payload, &catalog); err != nil {
		t.Fatal(err)
	}
	return catalog.Countries
}

func canonicalCountryIDs(t *testing.T) map[string]struct{} {
	t.Helper()
	result := make(map[string]struct{})
	for _, country := range loadCanonicalCountries(t) {
		result[canonicalCountryID(t, country.Code)] = struct{}{}
	}
	return result
}

func seedCanonicalCountries(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, country := range loadCanonicalCountries(t) {
		if _, err := db.Exec(`
INSERT INTO countries (id, code, name, name_en, strategic_positioning, key_resources)
VALUES ($1, $2, $3, $4, $5, $6)`, canonicalCountryID(t, country.Code), country.Code, country.Name, country.NameEn, country.StrategicPositioning, country.KeyResources); err != nil {
			t.Fatal(err)
		}
	}
}

func canonicalCountryID(t *testing.T, code string) string {
	t.Helper()
	id, err := coreid.Derive(coreid.Country, "country", code)
	if err != nil {
		t.Fatal(err)
	}
	return id
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
	if _, err := catalogCompany(publication.SchemaVersion, publication.Companies[0]); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(publication.Companies[0].Aliases, want) {
		t.Fatalf("catalogCompany() mutated aliases: got %#v want %#v", publication.Companies[0].Aliases, want)
	}
}
