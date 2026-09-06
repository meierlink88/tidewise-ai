package geopoliticrivalry

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

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestLoadCurrentGeopoliticalCatalogPackage(t *testing.T) {
	publication, err := LoadCatalog(context.Background(), geopoliticalCatalogPath(t))
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if publication.SchemaVersion != 1 || publication.PublicationMode != CatalogPublicationModeReconcile ||
		len(publication.Domains) != expectedDomainCount || len(publication.Storylines) != expectedStorylineCount {
		t.Fatalf("catalog metadata = schema %d mode %q domains %d storylines %d",
			publication.SchemaVersion, publication.PublicationMode, len(publication.Domains), len(publication.Storylines))
	}
	seenDomains := make(map[string]struct{}, expectedDomainCount)
	for _, domain := range publication.Domains {
		if len(domain.Tactics) != expectedTacticsPerDomain {
			t.Fatalf("domain %q tactic count = %d, want %d", domain.Code, len(domain.Tactics), expectedTacticsPerDomain)
		}
		seenDomains[domain.Code] = struct{}{}
	}
	if len(seenDomains) != len(expectedDomainCodes) {
		t.Fatalf("unique domain count = %d, want %d", len(seenDomains), len(expectedDomainCodes))
	}
	for code := range expectedDomainCodes {
		if _, exists := seenDomains[code]; !exists {
			t.Fatalf("catalog is missing domain %q", code)
		}
	}
	for _, requiredStoryline := range []string{"俄乌战争", "台海军事安全态势", "美伊军事冲突"} {
		found := false
		for _, storyline := range publication.Storylines {
			if storyline.Name == requiredStoryline {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("catalog is missing required storyline %q", requiredStoryline)
		}
	}
}

func TestValidateGeopoliticalCatalogRejectsInvalidPackages(t *testing.T) {
	base, err := LoadCatalog(context.Background(), geopoliticalCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*CatalogPublication)
	}{
		{"unknown schema", func(value *CatalogPublication) { value.SchemaVersion = 2 }},
		{"unknown mode", func(value *CatalogPublication) { value.PublicationMode = "replace" }},
		{"missing domain", func(value *CatalogPublication) { value.Domains = value.Domains[1:] }},
		{"missing tactic", func(value *CatalogPublication) { value.Domains[0].Tactics = value.Domains[0].Tactics[1:] }},
		{"duplicate tactic", func(value *CatalogPublication) { value.Domains[0].Tactics[1] = value.Domains[0].Tactics[0] }},
		{"missing storyline", func(value *CatalogPublication) { value.Storylines = value.Storylines[1:] }},
		{"duplicate storyline", func(value *CatalogPublication) { value.Storylines[1].Name = value.Storylines[0].Name }},
		{"unknown domain reference", func(value *CatalogPublication) { value.Storylines[0].DomainCode = "UNKNOWN" }},
		{"empty core proposition", func(value *CatalogPublication) { value.Storylines[0].CoreProposition = " " }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := cloneCatalog(t, base)
			test.mutate(&publication)
			if err := validateCatalog(publication); !errors.Is(err, ErrInvalidGeopoliticCatalog) {
				t.Fatalf("validateCatalog() error = %v, want ErrInvalidGeopoliticCatalog", err)
			}
		})
	}
}

func TestLoadGeopoliticalCatalogRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	payload, err := os.ReadFile(geopoliticalCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := strings.Replace(string(payload), `"schema_version": 1`, `"unknown": true, "schema_version": 1`, 1)
	if _, err := LoadCatalog(context.Background(), writeGeopoliticalCatalog(t, []byte(withUnknown))); err == nil {
		t.Fatal("LoadCatalog() accepted an unknown field")
	}
	if _, err := LoadCatalog(context.Background(), writeGeopoliticalCatalog(t, append(payload, []byte("\n{}")...))); err == nil {
		t.Fatal("LoadCatalog() accepted trailing JSON")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadCatalog(cancelled, geopoliticalCatalogPath(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("LoadCatalog(cancelled) error = %v, want context.Canceled", err)
	}
}

func TestPublishGeopoliticalCatalogIsAtomicExactAndIdempotent(t *testing.T) {
	db := openGeopoliticalCatalogTestDatabase(t, "tw_geopolitical_catalog_publish")
	ctx := context.Background()
	publication, err := LoadCatalog(ctx, geopoliticalCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog() error = %v", err)
	}
	assertPublishedCatalog(t, db, publication)

	storylineID, err := coreid.Derive(coreid.GeopoliticRivalry, "geopolitic-rivalry", "俄乌战争")
	if err != nil {
		t.Fatal(err)
	}
	var firstUpdatedAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM geopolitic_rivalries WHERE id = $1`, storylineID).Scan(&firstUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); err != nil {
		t.Fatalf("PublishCatalog(repeat) error = %v", err)
	}
	var repeatedUpdatedAt time.Time
	if err := db.QueryRowContext(ctx, `SELECT updated_at FROM geopolitic_rivalries WHERE id = $1`, storylineID).Scan(&repeatedUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if !repeatedUpdatedAt.Equal(firstUpdatedAt) {
		t.Fatalf("idempotent publication changed updated_at: first %s repeat %s", firstUpdatedAt, repeatedUpdatedAt)
	}

	unexpectedDomainID, err := coreid.New(coreid.GeopoliticDomain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO geopolitic_domains (id, code, name, description, tactics)
VALUES ($1, 'UNEXPECTED', '未审核领域', '未审核领域', '[{"name":"手段","description":"描述"}]'::jsonb)`, unexpectedDomainID); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); !errors.Is(err, ErrGeopoliticCatalogConflict) {
		t.Fatalf("PublishCatalog(unexpected identity) error = %v, want ErrGeopoliticCatalogConflict", err)
	}
}

func TestPublishGeopoliticalCatalogRejectsNaturalKeyIdentityDrift(t *testing.T) {
	db := openGeopoliticalCatalogTestDatabase(t, "tw_geopolitical_catalog_identity_drift")
	ctx := context.Background()
	publication, err := LoadCatalog(ctx, geopoliticalCatalogPath(t))
	if err != nil {
		t.Fatal(err)
	}
	driftedID, err := coreid.New(coreid.GeopoliticDomain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO geopolitic_domains (id, code, name, description, tactics)
VALUES ($1, 'MILITARY', '军事/防务线', '漂移身份', '[{"name":"手段","description":"描述"}]'::jsonb)`, driftedID); err != nil {
		t.Fatal(err)
	}
	if err := PublishCatalog(ctx, db, publication); !errors.Is(err, ErrGeopoliticCatalogConflict) {
		t.Fatalf("PublishCatalog(identity drift) error = %v, want ErrGeopoliticCatalogConflict", err)
	}
	var domainCount, storylineCount int
	if err := db.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM geopolitic_domains),
    (SELECT count(*) FROM geopolitic_rivalries)`).Scan(&domainCount, &storylineCount); err != nil {
		t.Fatal(err)
	}
	if domainCount != 1 || storylineCount != 0 {
		t.Fatalf("failed publication left domains %d storylines %d, want 1 and 0", domainCount, storylineCount)
	}
}

func assertPublishedCatalog(t *testing.T, db *sql.DB, publication CatalogPublication) {
	t.Helper()
	var domainCount, tacticCount, storylineCount, orphanCount int
	if err := db.QueryRow(`SELECT count(*), COALESCE(sum(jsonb_array_length(tactics)), 0) FROM geopolitic_domains`).Scan(&domainCount, &tacticCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM geopolitic_rivalries`).Scan(&storylineCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM geopolitic_rivalries storyline
LEFT JOIN geopolitic_domains domain ON domain.id = storyline.geopolitic_domain_id
WHERE domain.id IS NULL`).Scan(&orphanCount); err != nil {
		t.Fatal(err)
	}
	if domainCount != expectedDomainCount || tacticCount != expectedDomainCount*expectedTacticsPerDomain ||
		storylineCount != expectedStorylineCount || orphanCount != 0 {
		t.Fatalf("published counts = domains %d tactics %d storylines %d orphans %d", domainCount, tacticCount, storylineCount, orphanCount)
	}
	if len(publication.Domains) != domainCount || len(publication.Storylines) != storylineCount {
		t.Fatalf("published counts do not match package")
	}

	wantDomains := make(map[string]DomainCatalogItem, len(publication.Domains))
	for _, domain := range publication.Domains {
		wantDomains[domain.Code] = domain
	}
	domainIDs := make(map[string]string, len(publication.Domains))
	rows, err := db.Query(`SELECT id, code, name, description, tactics FROM geopolitic_domains ORDER BY code`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, code, name, description string
		var tacticsJSON []byte
		if err := rows.Scan(&id, &code, &name, &description, &tacticsJSON); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		var tactics []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(tacticsJSON, &tactics); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		want := wantDomains[code]
		if name != want.Name || description != want.Description || len(tactics) != len(want.Tactics) {
			_ = rows.Close()
			t.Fatalf("published domain %q does not match package", code)
		}
		for index := range tactics {
			if tactics[index].Name != want.Tactics[index].Name || tactics[index].Description != want.Tactics[index].Description {
				_ = rows.Close()
				t.Fatalf("published domain %q tactic %d does not match package", code, index)
			}
		}
		domainIDs[code] = id
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	wantStorylines := make(map[string]StorylineCatalogItem, len(publication.Storylines))
	for _, storyline := range publication.Storylines {
		wantStorylines[storyline.Name] = storyline
	}
	rows, err = db.Query(`SELECT name, category, geopolitic_domain_id, core_proposition, core_actors, main_transmission
FROM geopolitic_rivalries ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var got StorylineCatalogItem
		var domainID string
		if err := rows.Scan(&got.Name, &got.Category, &domainID, &got.CoreProposition, &got.CoreActors, &got.MainTransmission); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		want := wantStorylines[got.Name]
		got.DomainCode = want.DomainCode
		if domainID != domainIDs[want.DomainCode] || !reflect.DeepEqual(got, want) {
			_ = rows.Close()
			t.Fatalf("published storyline %q = %#v domain %q, want %#v domain %q", got.Name, got, domainID, want, domainIDs[want.DomainCode])
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
}

func cloneCatalog(t *testing.T, input CatalogPublication) CatalogPublication {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var result CatalogPublication
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func geopoliticalCatalogPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "..", "initdata", "geopolitical-storylines-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func writeGeopoliticalCatalog(t *testing.T, payload []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "geopolitical-storylines.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func openGeopoliticalCatalogTestDatabase(t *testing.T, name string) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, name, migrationDir, 0)
}
