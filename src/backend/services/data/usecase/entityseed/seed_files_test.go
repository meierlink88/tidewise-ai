package seed

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
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

	if got, want := len(manifest.Entities), 52; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	keys := make(map[string]struct{}, len(manifest.Entities))
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeSector {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeSector)
		}
		if strings.Contains(entity.Key, "ths") || strings.Contains(entity.Key, "openspec") || !strings.HasPrefix(entity.Key, "sector:") {
			t.Fatalf("entity key %q is not source independent", entity.Key)
		}
		if entity.Name == "" || entity.CanonicalName == "" || len(entity.Aliases) == 0 {
			t.Fatalf("entity %q missing Chinese primary name or English alias", entity.Key)
		}
		classification := profileString(t, entity.Profile, "classification_code")
		if classification != "industry_sector" && classification != "theme_sector" {
			t.Fatalf("entity %q classification = %q", entity.Key, classification)
		}
		if profileString(t, entity.Profile, "review_status") != "approved" {
			t.Fatalf("entity %q is not approved", entity.Key)
		}
		if _, exists := keys[entity.Key]; exists {
			t.Fatalf("duplicate stable key %q", entity.Key)
		}
		keys[entity.Key] = struct{}{}
	}
	coverageRepresentatives := []string{
		"sector:industry_banking",
		"sector:industry_power_utilities",
		"sector:industry_nonferrous_new_materials",
		"sector:industry_construction_infrastructure",
		"sector:industry_semiconductors_electronics",
		"sector:industry_software_communications",
		"sector:industry_automobiles_components",
		"sector:industry_pharma_biotech",
		"sector:industry_consumer_retail",
		"sector:industry_transportation_logistics",
		"sector:industry_defense_aerospace",
		"sector:theme_soe_reform_technology",
	}
	for _, key := range coverageRepresentatives {
		if _, exists := keys[key]; !exists {
			t.Errorf("missing transmission-cluster representative %q", key)
		}
	}
}

func TestReviewedSectorSourceMappingsCloseSixtyCandidatesAndEightMerges(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(
		filepath.Join(root, "sectors.json"),
		filepath.Join(root, "sector_source_mappings.json"),
	)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	if got, want := len(manifest.SectorSourceMappings), 60; got != want {
		t.Fatalf("sector source mappings = %d, want %d", got, want)
	}
	taxonomyCounts := map[string]int{}
	merged := map[string]string{}
	for _, mapping := range manifest.SectorSourceMappings {
		taxonomyCounts[mapping.SourceTaxonomyType]++
		if mapping.SourceSystem != "openspec_review" || mapping.SourceSectorCode != "" || mapping.SourceMarketScope != "cn_a_share" {
			t.Fatalf("mapping identity is not an honest code-free Review identity: %+v", mapping)
		}
		if mapping.MappingStatus == "merged" {
			merged[mapping.SourceSectorName] = mapping.SectorEntityKey
		}
	}
	for _, taxonomy := range []string{"industry", "concept", "index_sector"} {
		if got, want := taxonomyCounts[taxonomy], 20; got != want {
			t.Errorf("%s mappings = %d, want %d", taxonomy, got, want)
		}
	}
	wantMerged := map[string]string{
		"中证卫星产业":   "sector:theme_commercial_space_satellite",
		"中证卫星导航产业": "sector:theme_satellite_communications_navigation",
		"国证机器人产业":  "sector:theme_robotics_embodied_ai",
		"国证风电光伏装备": "sector:theme_wind_solar_equipment",
		"创业板人工智能":  "sector:theme_artificial_intelligence",
		"上证信息安全":   "sector:theme_cyber_data_security",
		"中证基建":     "sector:industry_construction_infrastructure",
		"中证全指汽车":   "sector:industry_automobiles_components",
	}
	if len(merged) != len(wantMerged) {
		t.Fatalf("merged mappings = %d, want %d: %v", len(merged), len(wantMerged), merged)
	}
	for name, key := range wantMerged {
		if merged[name] != key {
			t.Errorf("merged mapping %q = %q, want %q", name, merged[name], key)
		}
	}
}

func TestReviewedMarketSectorRelationshipsAreObjectiveAndClosed(t *testing.T) {
	manifest, err := LoadFiles(entityFoundationSeedPaths()...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}
	counts := map[string]int{}
	for _, relationship := range manifest.Relationships {
		counts[relationship.RelationType]++
		if relationship.RelationType == "covers_sector" && (!strings.HasPrefix(relationship.From, "market:") || !strings.HasPrefix(relationship.To, "sector:")) {
			t.Fatalf("invalid covers_sector direction: %+v", relationship)
		}
		if relationship.RelationType == "tracked_by_benchmark" && (!strings.HasPrefix(relationship.From, "sector:") || !strings.HasPrefix(relationship.To, "benchmark:")) {
			t.Fatalf("invalid tracked_by_benchmark direction: %+v", relationship)
		}
	}
	if got, want := counts["covers_sector"], 52; got != want {
		t.Errorf("covers_sector relationships = %d, want %d", got, want)
	}
	if got := counts["tracked_by_benchmark"]; got != 0 {
		t.Errorf("tracked_by_benchmark relationships = %d, want 0 until benchmark entities are reviewed", got)
	}
}

func TestSectorFixturesRejectReasoningAndInvestmentAdvice(t *testing.T) {
	for _, content := range []string{
		`{"entities":[{"key":"sector:theme_test","entity_type":"sector","layer_code":"sector","name":"测试主题","canonical_name":"测试主题","aliases":["Test Theme"],"profile":{"sector_system":"canonical","sector_type":"theme","classification_code":"theme_sector","review_status":"approved","investment_advice":"买入"}}]}`,
		`{"entities":[{"key":"market:test","entity_type":"market","layer_code":"market","name":"测试市场","canonical_name":"测试市场","profile":{"market_type":"stock_market"}},{"key":"sector:theme_test","entity_type":"sector","layer_code":"sector","name":"测试主题","canonical_name":"测试主题","aliases":["Test Theme"],"profile":{"sector_system":"canonical","sector_type":"theme","classification_code":"theme_sector","review_status":"approved"}}],"relationships":[{"key":"relationship:test","from":"market:test","to":"sector:theme_test","relation_type":"covers_sector","source_name":"测试来源","source_url":"https://example.com/source","verified_at":"2026-07-12T00:00:00Z","evidence_note":"该主题受益并建议买入"}]}`,
	} {
		if _, err := Load([]byte(content)); err == nil {
			t.Fatal("Load() error = nil, want forbidden reasoning rejection")
		}
	}
}

func TestReviewedSectorSeedURLsUseCandidateReviewCommitPermalink(t *testing.T) {
	const permalink = "https://github.com/meierlink88/tidewise-ai/blob/03273effecb946ba21c953f6d12165d65b3dee88/openspec/changes/add-market-sector-foundation/candidate-review.md"
	root := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")

	sectors, err := LoadFile(filepath.Join(root, "sectors.json"))
	if err != nil {
		t.Fatalf("LoadFile(sectors) error = %v", err)
	}
	for _, entity := range sectors.Entities {
		if got := profileString(t, entity.Profile, "methodology_url"); got != permalink {
			t.Fatalf("sector %q methodology_url = %q, want commit permalink", entity.Key, got)
		}
	}

	combined, err := LoadFiles(filepath.Join(root, "sectors.json"), filepath.Join(root, "sector_source_mappings.json"))
	if err != nil {
		t.Fatalf("LoadFiles(source mappings) error = %v", err)
	}
	for _, mapping := range combined.SectorSourceMappings {
		if mapping.SourceURL != permalink {
			t.Fatalf("mapping %q source_url = %q, want commit permalink", mapping.SourceSectorName, mapping.SourceURL)
		}
	}

	all, err := LoadFiles(entityFoundationSeedPaths()...)
	if err != nil {
		t.Fatalf("LoadFiles(all seeds) error = %v", err)
	}
	for _, relationship := range all.Relationships {
		if relationship.RelationType == "covers_sector" && relationship.SourceURL != permalink {
			t.Fatalf("relationship %q source_url = %q, want commit permalink", relationship.Key, relationship.SourceURL)
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

	if got, want := len(manifest.Entities), 43; got != want {
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

	if got, want := len(manifest.Relationships), 389; got != want {
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
	return legacyFixturePaths(filepath.Join("..", "..", "..", "..", "data", "entity_foundation"))
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
