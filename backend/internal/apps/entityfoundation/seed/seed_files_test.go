package seed

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestAllianceOrgSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "alliance_orgs.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 10; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	required := map[string]struct{}{
		"alliance_org:opec_plus":  {},
		"alliance_org:opec":       {},
		"alliance_org:g7":         {},
		"alliance_org:g20":        {},
		"alliance_org:wto":        {},
		"alliance_org:imf":        {},
		"alliance_org:world_bank": {},
		"alliance_org:oecd":       {},
		"alliance_org:eu":         {},
		"alliance_org:brics":      {},
	}
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeAllianceOrg {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeAllianceOrg)
		}
		delete(required, entity.Key)
	}
	if len(required) > 0 {
		t.Fatalf("missing alliance org entities: %v", required)
	}
}

func TestEconomySeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "economies.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 50; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	names := map[string]string{}
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeEconomy {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeEconomy)
		}
		names[entity.Key] = entity.Name
		if strings.Contains(entity.Name, "香港") && entity.Name != "中国香港" {
			t.Fatalf("entity %q name = %q, want 中国香港 political naming rule", entity.Key, entity.Name)
		}
		if strings.Contains(entity.Name, "台湾") && entity.Name != "中国台湾" {
			t.Fatalf("entity %q name = %q, want 中国台湾 political naming rule", entity.Key, entity.Name)
		}
	}
	if names["economy:hk"] != "中国香港" {
		t.Fatalf("economy:hk name = %q, want 中国香港", names["economy:hk"])
	}
	if names["economy:tw"] != "中国台湾" {
		t.Fatalf("economy:tw name = %q, want 中国台湾", names["economy:tw"])
	}
}

func TestPolicyBodySeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "policy_bodies.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 30; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypePolicyBody {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypePolicyBody)
		}
		if hasASCIILetter(entity.Name) {
			t.Fatalf("entity %q name = %q, want Chinese display name", entity.Key, entity.Name)
		}
	}
}

func TestMarketSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "markets.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 32; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeMarket {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeMarket)
		}
		if strings.Contains(entity.Name, "香港") && !strings.Contains(entity.Name, "中国香港") {
			t.Fatalf("entity %q name = %q, want 中国香港 political naming rule", entity.Key, entity.Name)
		}
		if strings.Contains(entity.Name, "台湾") && !strings.Contains(entity.Name, "中国台湾") {
			t.Fatalf("entity %q name = %q, want 中国台湾 political naming rule", entity.Key, entity.Name)
		}
	}
}

func TestIndexSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "indices.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 45; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeIndex {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeIndex)
		}
		if strings.Contains(entity.Name, "台湾") && !strings.Contains(entity.Name, "中国台湾") {
			t.Fatalf("entity %q name = %q, want 中国台湾 political naming rule", entity.Key, entity.Name)
		}
	}
}

func TestSectorSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "sectors.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 60; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeSector {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeSector)
		}
	}
}

func TestChainNodeSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "chain_nodes.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, wantMinimum := len(manifest.Entities), 33; got < wantMinimum {
		t.Fatalf("entities = %d, want at least %d", got, wantMinimum)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeChainNode {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeChainNode)
		}
	}
}

func TestMetricSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "metrics.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 42; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeMetric {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeMetric)
		}
	}
}

func TestCommoditySeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "commodities.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 45; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeCommodity {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeCommodity)
		}
	}
}

func TestCompanySecuritySeedFiles(t *testing.T) {
	companies, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "companies.json"))
	if err != nil {
		t.Fatalf("LoadFile(companies) error = %v", err)
	}
	securities, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "securities.json"))
	if err != nil {
		t.Fatalf("LoadFile(securities) error = %v", err)
	}

	if got, wantMinimum := len(companies.Entities), 70; got < wantMinimum {
		t.Fatalf("companies = %d, want at least %d", got, wantMinimum)
	}
	if got, want := len(securities.Entities), len(companies.Entities); got != want {
		t.Fatalf("securities = %d, want one primary security for each of %d companies", got, want)
	}

	companyKeys := make(map[string]struct{}, len(companies.Entities))
	for _, entity := range companies.Entities {
		if entity.EntityType != domain.EntityTypeCompany {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeCompany)
		}
		companyKeys[entity.Key] = struct{}{}
	}

	coveredCompanies := make(map[string]struct{}, len(securities.Entities))
	for _, entity := range securities.Entities {
		if entity.EntityType != domain.EntityTypeSecurity {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeSecurity)
		}
		issuer := profileString(t, entity.Profile, "issuer_company_entity_id")
		if _, ok := companyKeys[issuer]; !ok {
			t.Fatalf("security %q issuer %q does not exist in companies seed", entity.Key, issuer)
		}
		coveredCompanies[issuer] = struct{}{}
	}
	if got, want := len(coveredCompanies), len(companyKeys); got != want {
		t.Fatalf("covered companies = %d, want %d", got, want)
	}
}

func TestInstrumentSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "instruments.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 4; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeInstrument {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeInstrument)
		}
	}
}

func TestPersonSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "persons.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 30; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypePerson {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypePerson)
		}
		if hasASCIILetter(entity.Name) && entity.Name != "Meta" && entity.Name != "ABB" {
			t.Fatalf("entity %q name = %q, want Chinese display name", entity.Key, entity.Name)
		}
	}
}

func TestRelationshipSeedFile(t *testing.T) {
	manifest, err := LoadFiles(entityFoundationSeedPaths()...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}

	if got := len(manifest.Relationships); got != 0 {
		t.Fatalf("relationships = %d, want empty reviewed baseline", got)
	}
}

func entityFoundationSeedPaths() []string {
	return DefaultSeedPaths(filepath.Join("..", "..", "..", "..", "data", "entity_foundation"))
}

func profileString(t *testing.T, raw json.RawMessage, key string) string {
	t.Helper()

	var profile map[string]string
	if err := json.Unmarshal(raw, &profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return profile[key]
}

func hasASCIILetter(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}
