package country_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

func TestCountryInitializationCatalogUsesCanonicalIdentityAndAlpha2Codes(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "initdata", "countries-v1.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Countries []struct {
			Code   string `json:"code"`
			Name   string `json:"name"`
			NameEn string `json:"name_en"`
		} `json:"countries"`
	}
	if err := json.Unmarshal(payload, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Countries) != 201 {
		t.Fatalf("Country count = %d, want 201", len(catalog.Countries))
	}
	required := map[string]bool{"CN": false, "US": false, "HK": false, "MO": false, "TW": false, "EH": false}
	seenCodes := make(map[string]struct{}, len(catalog.Countries))
	for _, country := range catalog.Countries {
		if len(country.Code) != 2 || country.Code[0] < 'A' || country.Code[0] > 'Z' ||
			country.Code[1] < 'A' || country.Code[1] > 'Z' {
			t.Fatalf("Country code %q is not ISO 3166-1 alpha-2 shaped", country.Code)
		}
		_, err := coreid.Derive(coreid.Country, "country", country.Code)
		if err != nil {
			t.Fatal(err)
		}
		if country.Name == "" || country.NameEn == "" {
			t.Fatalf("Country %s has a blank name", country.Code)
		}
		if _, exists := seenCodes[country.Code]; exists {
			t.Fatalf("duplicate Country code %s", country.Code)
		}
		seenCodes[country.Code] = struct{}{}
		if _, exists := required[country.Code]; exists {
			required[country.Code] = true
		}
	}
	for code, found := range required {
		if !found {
			t.Fatalf("required Country code %s is missing", code)
		}
	}
}
