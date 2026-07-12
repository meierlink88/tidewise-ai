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

	if got, want := len(manifest.Entities), 47; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	requiredMarketTypes := map[string]string{
		"market:cn_bond":                   "bond_market",
		"market:us_treasury":               "bond_market",
		"market:euro_area_government_bond": "bond_market",
		"market:jgb":                       "bond_market",
		"market:uk_gilt":                   "bond_market",
		"market:ine":                       "commodity_futures_exchange",
		"market:dce":                       "commodity_futures_exchange",
		"market:czce":                      "commodity_futures_exchange",
		"market:lme":                       "commodity_futures_exchange",
		"market:ice_futures_europe":        "commodity_futures_exchange",
		"market:saudi_stock":               "stock_market",
		"market:indonesia_stock":           "stock_market",
		"market:vietnam_stock":             "stock_market",
		"market:global_equity":             "stock_market",
		"market:global_precious_metals":    "commodity_spot_market",
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
		if wantType, ok := requiredMarketTypes[entity.Key]; ok {
			if gotType := profileString(t, entity.Profile, "market_type"); gotType != wantType {
				t.Fatalf("entity %q market_type = %q, want %q", entity.Key, gotType, wantType)
			}
			delete(requiredMarketTypes, entity.Key)
		}
	}
	if len(requiredMarketTypes) > 0 {
		t.Fatalf("missing reviewed market entities: %v", requiredMarketTypes)
	}
}

func TestIndexSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "indices.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 43; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	requiredMarketAssignments := map[string]string{
		"index:dji":               "market:us_stock",
		"index:msci_world":        "market:global_equity",
		"index:msci_em":           "market:global_equity",
		"index:ccdc_bond":         "market:cn_bond",
		"index:tasi":              "market:saudi_stock",
		"index:jakarta_composite": "market:indonesia_stock",
		"index:vn_index":          "market:vietnam_stock",
	}
	deferredBenchmarkKeys := map[string]struct{}{
		"index:cn_10y_government_bond_yield":        {},
		"index:us_10y_treasury_yield":               {},
		"index:euro_area_10y_government_bond_yield": {},
		"index:jgb_10y_yield":                       {},
		"index:uk_10y_gilt_yield":                   {},
		"index:brent_continuous":                    {},
		"index:wti_continuous":                      {},
		"index:xau_spot":                            {},
		"index:btc_price":                           {},
		"index:eth_price":                           {},
	}
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeIndex {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeIndex)
		}
		if strings.Contains(entity.Name, "台湾") && !strings.Contains(entity.Name, "中国台湾") {
			t.Fatalf("entity %q name = %q, want 中国台湾 political naming rule", entity.Key, entity.Name)
		}
		if wantMarket, ok := requiredMarketAssignments[entity.Key]; ok {
			if gotMarket := profileString(t, entity.Profile, "market_entity_id"); gotMarket != wantMarket {
				t.Fatalf("entity %q market_entity_id = %q, want %q", entity.Key, gotMarket, wantMarket)
			}
			delete(requiredMarketAssignments, entity.Key)
		}
		if _, deferred := deferredBenchmarkKeys[entity.Key]; deferred {
			t.Fatalf("benchmark candidate %q must not remain in index seed", entity.Key)
		}
	}
	if len(requiredMarketAssignments) > 0 {
		t.Fatalf("missing reviewed index assignments: %v", requiredMarketAssignments)
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

	if got, want := len(manifest.Relationships), 306; got != want {
		t.Fatalf("relationships = %d, want %d reviewed relationships", got, want)
	}

	wantCounts := map[string]int{
		"alliance_org:opec_plus":  9,
		"alliance_org:opec":       5,
		"alliance_org:g7":         8,
		"alliance_org:g20":        20,
		"alliance_org:wto":        48,
		"alliance_org:imf":        46,
		"alliance_org:world_bank": 46,
		"alliance_org:oecd":       22,
		"alliance_org:eu":         9,
		"alliance_org:brics":      10,
	}
	gotCounts := make(map[string]int, len(wantCounts))
	relationTypeCounts := make(map[string]int)
	forbiddenHasMarketTargets := map[string]struct{}{
		"market:europe_stock":             {},
		"market:ice":                      {},
		"market:global_fx":                {},
		"market:global_commodity_futures": {},
		"market:global_crypto":            {},
		"market:global_equity":            {},
		"market:global_precious_metals":   {},
	}
	for _, relationship := range manifest.Relationships {
		if relationship.SourceName == "" || relationship.SourceURL == "" || relationship.VerifiedAt.IsZero() {
			t.Fatalf("relationship %q provenance is incomplete", relationship.Key)
		}
		relationTypeCounts[relationship.RelationType]++
		if relationship.RelationType == "member_of" {
			gotCounts[relationship.To]++
		}
		if relationship.RelationType == "has_market" {
			if _, forbidden := forbiddenHasMarketTargets[relationship.To]; forbidden {
				t.Fatalf("deferred has_market target %q was included", relationship.To)
			}
		}
	}
	if got, want := relationTypeCounts["member_of"], 223; got != want {
		t.Errorf("member_of count = %d, want %d", got, want)
	}
	if got, want := relationTypeCounts["has_market"], 40; got != want {
		t.Errorf("has_market count = %d, want %d", got, want)
	}
	if got, want := relationTypeCounts["tracks_index"], 43; got != want {
		t.Errorf("tracks_index count = %d, want %d", got, want)
	}
	for target, want := range wantCounts {
		if got := gotCounts[target]; got != want {
			t.Errorf("member_of target %q count = %d, want %d", target, got, want)
		}
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
