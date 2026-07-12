package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestLoadManifestValidatesEntitiesAndRelationships(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{
				"key": "alliance_org:opec_plus",
				"entity_type": "alliance_org",
				"layer_code": "alliance",
				"name": "OPEC+",
				"canonical_name": "OPEC+",
				"aliases": ["OPEC Plus"],
				"profile": {
					"org_code": "OPEC_PLUS",
					"org_type": "energy_alliance",
					"primary_domain": "energy",
					"scope_region": "global",
					"official_url": "https://www.opec.org"
				}
			},
			{
				"key": "economy:cn",
				"entity_type": "economy",
				"layer_code": "economy",
				"name": "中国",
				"canonical_name": "中国",
				"profile": {
					"country_code": "CN",
					"currency_code": "CNY",
					"region": "asia"
				}
			}
		],
		"relationships": [
			{
				"key": "relationship:cn_member_of_opec_plus",
				"from": "economy:cn",
				"to": "alliance_org:opec_plus",
				"relation_type": "member_of",
				"source_name": "OPEC",
				"source_url": "https://www.opec.org/opec_web/en/about_us/25.htm",
				"verified_at": "2026-07-10T00:00:00Z"
			}
		]
	}`)

	manifest, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 2; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}
	if got, want := len(manifest.Relationships), 1; got != want {
		t.Fatalf("relationships = %d, want %d", got, want)
	}
}

func TestDefaultSeedPathsIncludeBenchmarkFiles(t *testing.T) {
	paths := DefaultSeedPaths("seed-root")
	required := map[string]struct{}{
		filepath.Join("seed-root", "benchmarks.json"):                          {},
		filepath.Join("seed-root", "relationships", "observes_benchmark.json"): {},
		filepath.Join("seed-root", "relationships", "measures.json"):           {},
		filepath.Join("seed-root", "relationships", "references.json"):         {},
	}
	for _, path := range paths {
		delete(required, path)
	}
	if len(required) > 0 {
		t.Fatalf("DefaultSeedPaths() missing benchmark paths: %v", required)
	}
}

func TestReviewedBenchmarkSeedFixtures(t *testing.T) {
	seedRoot := filepath.Join("..", "..", "..", "..", "data", "entity_foundation")
	manifest, err := LoadFiles(DefaultSeedPaths(seedRoot)...)
	if err != nil {
		t.Fatalf("LoadFiles() error = %v", err)
	}

	entities := map[string]Entity{}
	for _, entity := range manifest.Entities {
		entities[entity.Key] = entity
	}
	benchmarkKeys := []string{
		"benchmark:cn_10y_government_bond_yield",
		"benchmark:us_10y_treasury_par_yield",
		"benchmark:de_10y_federal_bond_yield",
		"benchmark:jp_10y_jgb_constant_maturity_yield",
		"benchmark:uk_10y_gilt_nominal_par_yield",
		"benchmark:ice_brent_crude_front_month_settlement",
		"benchmark:nymex_wti_crude_front_month_settlement",
		"benchmark:lbma_gold_price_pm",
		"benchmark:cme_cf_bitcoin_reference_rate",
		"benchmark:cme_cf_ether_dollar_reference_rate",
	}
	for _, key := range benchmarkKeys {
		entity, ok := entities[key]
		if !ok {
			t.Fatalf("missing reviewed benchmark %q", key)
		}
		if entity.EntityType != domain.EntityTypeBenchmark {
			t.Fatalf("entity %q type = %q, want benchmark", key, entity.EntityType)
		}
		if !containsScript(entity.Name, unicode.Han) || !containsScript(entity.CanonicalName, unicode.Han) {
			t.Fatalf("benchmark %q must have Chinese name and canonical_name: %q / %q", key, entity.Name, entity.CanonicalName)
		}
		if !aliasesContainScript(entity.Aliases, unicode.Latin) {
			t.Fatalf("benchmark %q aliases = %v, want at least one English alias", key, entity.Aliases)
		}
	}
	if got := countEntitiesByType(manifest.Entities, domain.EntityTypeBenchmark); got != 10 {
		t.Fatalf("benchmark entities = %d, want 10", got)
	}
	if _, ok := entities["metric:fear_index"]; ok {
		t.Fatal("metric:fear_index must be removed from reviewed seed")
	}
	assertMetricProfile(t, entities["metric:implied_volatility"], "market_volatility", "percent", "trading_day")
	assertMetricProfile(t, entities["metric:gold_price"], "market_price", "price", "trading_day")

	counts := map[string]int{}
	relations := map[string]Relationship{}
	for _, relationship := range manifest.Relationships {
		counts[relationship.RelationType]++
		relations[relationship.From+"|"+relationship.RelationType+"|"+relationship.To] = relationship
		if relationship.SourceName == "" || relationship.SourceURL == "" || relationship.VerifiedAt.IsZero() {
			t.Fatalf("relationship %q missing provenance: %+v", relationship.Key, relationship)
		}
	}
	if counts["observes_benchmark"] != 10 || counts["measures"] != 10 || counts["references"] != 5 {
		t.Fatalf("benchmark relationship counts = %#v, want 10/10/5", counts)
	}
	assertRelationshipExists(t, relations, "market:global_commodity_futures", "observes_benchmark", "benchmark:nymex_wti_crude_front_month_settlement")
	assertRelationshipExists(t, relations, "benchmark:lbma_gold_price_pm", "measures", "metric:gold_price")
	assertRelationshipExists(t, relations, "benchmark:cme_cf_bitcoin_reference_rate", "measures", "metric:exchange_rate")
	assertRelationshipExists(t, relations, "benchmark:cme_cf_ether_dollar_reference_rate", "measures", "metric:exchange_rate")
	for key := range relations {
		if key == "market:cme|observes_benchmark|benchmark:nymex_wti_crude_front_month_settlement" ||
			(strings.Contains(key, "benchmark:lbma_gold_price_pm") && strings.Contains(key, "metric:latest_price")) ||
			(strings.Contains(key, "benchmark:cme_cf_") && strings.Contains(key, "metric:latest_price")) {
			t.Fatalf("reviewed seed contains rejected relationship tuple %q", key)
		}
	}
}

func TestValidateBenchmarkRequiresBilingualSearchNames(t *testing.T) {
	base := Entity{
		Key:           "benchmark:test",
		EntityType:    domain.EntityTypeBenchmark,
		LayerCode:     "benchmark",
		Name:          "测试基准",
		CanonicalName: "测试基准",
		Aliases:       []string{"Test Benchmark"},
		Profile:       json.RawMessage(`{"benchmark_type":"reference_rate","provider":"test","currency_code":"USD","unit":"points","frequency":"daily","source_url":"https://example.com"}`),
	}

	if err := Validate(Manifest{Entities: []Entity{base}}); err != nil {
		t.Fatalf("Validate(Chinese primary with English alias) error = %v", err)
	}

	englishPrimary := base
	englishPrimary.Name = "Test Benchmark"
	englishPrimary.CanonicalName = "Test Benchmark"
	englishPrimary.Aliases = []string{"测试基准"}
	if err := Validate(Manifest{Entities: []Entity{englishPrimary}}); err != nil {
		t.Fatalf("Validate(English primary with Chinese alias) error = %v", err)
	}

	for name, entity := range map[string]Entity{
		"Chinese primary without English alias": func() Entity {
			candidate := base
			candidate.Aliases = []string{"测试指标"}
			return candidate
		}(),
		"English primary without Chinese alias": func() Entity {
			candidate := englishPrimary
			candidate.Aliases = []string{"Reference Rate"}
			return candidate
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := Validate(Manifest{Entities: []Entity{entity}}); err == nil {
				t.Fatal("Validate() error = nil, want bilingual benchmark validation error")
			}
		})
	}
}

func containsScript(value string, script *unicode.RangeTable) bool {
	return strings.IndexFunc(value, func(r rune) bool { return unicode.Is(script, r) }) >= 0
}

func aliasesContainScript(aliases []string, script *unicode.RangeTable) bool {
	for _, alias := range aliases {
		if containsScript(alias, script) {
			return true
		}
	}
	return false
}

func countEntitiesByType(entities []Entity, entityType domain.EntityType) int {
	count := 0
	for _, entity := range entities {
		if entity.EntityType == entityType {
			count++
		}
	}
	return count
}

func assertMetricProfile(t *testing.T, entity Entity, metricType string, unit string, frequency string) {
	t.Helper()
	if entity.Key == "" {
		t.Fatal("metric entity is missing")
	}
	if got := profileString(t, entity.Profile, "metric_type"); got != metricType {
		t.Fatalf("metric %q type = %q, want %q", entity.Key, got, metricType)
	}
	if got := profileString(t, entity.Profile, "unit"); got != unit {
		t.Fatalf("metric %q unit = %q, want %q", entity.Key, got, unit)
	}
	if got := profileString(t, entity.Profile, "frequency"); got != frequency {
		t.Fatalf("metric %q frequency = %q, want %q", entity.Key, got, frequency)
	}
}

func assertRelationshipExists(t *testing.T, relationships map[string]Relationship, from string, relationType string, to string) {
	t.Helper()
	key := from + "|" + relationType + "|" + to
	if _, ok := relationships[key]; !ok {
		t.Fatalf("missing reviewed relationship %q", key)
	}
}

func TestValidateRelationshipPolicyAcceptsSupportedTypeDirections(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	entities := map[string]Entity{
		"economy:cn":          {Key: "economy:cn", EntityType: domain.EntityTypeEconomy},
		"alliance_org:g20":    {Key: "alliance_org:g20", EntityType: domain.EntityTypeAllianceOrg},
		"market:sse":          {Key: "market:sse", EntityType: domain.EntityTypeMarket},
		"market:us_treasury":  {Key: "market:us_treasury", EntityType: domain.EntityTypeMarket},
		"index:sse_composite": {Key: "index:sse_composite", EntityType: domain.EntityTypeIndex},
		"benchmark:us_10y":    {Key: "benchmark:us_10y", EntityType: domain.EntityTypeBenchmark},
		"company:test":        {Key: "company:test", EntityType: domain.EntityTypeCompany},
		"security:test":       {Key: "security:test", EntityType: domain.EntityTypeSecurity},
		"chain_node:test":     {Key: "chain_node:test", EntityType: domain.EntityTypeChainNode},
		"person:test":         {Key: "person:test", EntityType: domain.EntityTypePerson},
		"policy_body:test":    {Key: "policy_body:test", EntityType: domain.EntityTypePolicyBody},
		"metric:test":         {Key: "metric:test", EntityType: domain.EntityTypeMetric},
		"instrument:test":     {Key: "instrument:test", EntityType: domain.EntityTypeInstrument},
		"commodity:test":      {Key: "commodity:test", EntityType: domain.EntityTypeCommodity},
	}
	cases := []Relationship{
		{Key: "r1", From: "economy:cn", To: "alliance_org:g20", RelationType: "member_of"},
		{Key: "r2", From: "economy:cn", To: "market:sse", RelationType: "has_market"},
		{Key: "r3", From: "market:sse", To: "index:sse_composite", RelationType: "tracks_index"},
		{Key: "r4", From: "company:test", To: "security:test", RelationType: "issues"},
		{Key: "r5", From: "company:test", To: "chain_node:test", RelationType: "participates_in"},
		{Key: "r6", From: "person:test", To: "policy_body:test", RelationType: "affiliated_with"},
		{Key: "r7", From: "metric:test", To: "instrument:test", RelationType: "applies_to"},
		{Key: "r8", From: "metric:test", To: "commodity:test", RelationType: "applies_to"},
		{Key: "r9", From: "metric:test", To: "chain_node:test", RelationType: "applies_to"},
		{Key: "r10", From: "market:us_treasury", To: "benchmark:us_10y", RelationType: "observes_benchmark"},
		{Key: "r11", From: "benchmark:us_10y", To: "metric:test", RelationType: "measures"},
		{Key: "r12", From: "benchmark:us_10y", To: "commodity:test", RelationType: "references"},
		{Key: "r13", From: "benchmark:us_10y", To: "instrument:test", RelationType: "references"},
	}
	for _, relationship := range cases {
		relationship.SourceName = "官方来源"
		relationship.SourceURL = "https://example.com/source"
		relationship.VerifiedAt = verifiedAt
		if err := validateRelationshipPolicy(relationship, entities); err != nil {
			t.Fatalf("validateRelationshipPolicy(%s) error = %v", relationship.RelationType, err)
		}
	}
}

func TestLoadManifestAcceptsBenchmarkProfileWithNullableOfficialSeriesCode(t *testing.T) {
	manifest, err := Load([]byte(`{
	  "entities": [
	    {
	      "key": "benchmark:us_10y_treasury_yield",
	      "entity_type": "benchmark",
	      "layer_code": "benchmark",
	      "name": "美国10年期国债收益率",
	      "canonical_name": "美国10年期国债收益率",
	      "aliases": ["US 10Y Treasury Yield"],
	      "profile": {
	        "benchmark_type": "government_bond_yield",
	        "official_series_code": null,
	        "provider": "us_treasury",
	        "tenor": "10Y",
	        "currency_code": "USD",
	        "unit": "percent",
	        "frequency": "daily",
	        "source_url": "https://home.treasury.gov/"
	      }
	    }
	  ]
	}`))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := len(manifest.Entities), 1; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}
	if manifest.Entities[0].EntityType != domain.EntityTypeBenchmark {
		t.Fatalf("entity type = %q, want benchmark", manifest.Entities[0].EntityType)
	}
	if got := profileString(t, manifest.Entities[0].Profile, "official_series_code"); got != "" {
		t.Fatalf("official_series_code = %q, want empty for null", got)
	}
}

func TestValidateRelationshipPolicyRejectsInvalidFacts(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	entities := map[string]Entity{
		"economy:cn":       {Key: "economy:cn", EntityType: domain.EntityTypeEconomy},
		"alliance_org:g20": {Key: "alliance_org:g20", EntityType: domain.EntityTypeAllianceOrg},
	}
	valid := Relationship{
		Key: "relationship:cn_member_of_g20", From: "economy:cn", To: "alliance_org:g20",
		RelationType: "member_of", SourceName: "G20", SourceURL: "https://g20.org/members/", VerifiedAt: verifiedAt,
	}
	cases := map[string]Relationship{
		"wrong direction":  {Key: valid.Key, From: valid.To, To: valid.From, RelationType: valid.RelationType, SourceName: valid.SourceName, SourceURL: valid.SourceURL, VerifiedAt: valid.VerifiedAt},
		"missing source":   {Key: valid.Key, From: valid.From, To: valid.To, RelationType: valid.RelationType, SourceURL: valid.SourceURL, VerifiedAt: valid.VerifiedAt},
		"invalid url":      {Key: valid.Key, From: valid.From, To: valid.To, RelationType: valid.RelationType, SourceName: valid.SourceName, SourceURL: "://bad", VerifiedAt: valid.VerifiedAt},
		"missing verified": {Key: valid.Key, From: valid.From, To: valid.To, RelationType: valid.RelationType, SourceName: valid.SourceName, SourceURL: valid.SourceURL},
		"self relation":    {Key: valid.Key, From: valid.From, To: valid.From, RelationType: valid.RelationType, SourceName: valid.SourceName, SourceURL: valid.SourceURL, VerifiedAt: valid.VerifiedAt},
		"unknown type":     {Key: valid.Key, From: valid.From, To: valid.To, RelationType: "benefits_from", SourceName: valid.SourceName, SourceURL: valid.SourceURL, VerifiedAt: valid.VerifiedAt},
		"reasoning note":   {Key: valid.Key, From: valid.From, To: valid.To, RelationType: valid.RelationType, EvidenceNote: "利好相关公司", SourceName: valid.SourceName, SourceURL: valid.SourceURL, VerifiedAt: valid.VerifiedAt},
	}
	for name, relationship := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateRelationshipPolicy(relationship, entities); err == nil {
				t.Fatal("validateRelationshipPolicy() error = nil")
			}
		})
	}
}

func TestValidateRejectsDuplicateRelationshipTuple(t *testing.T) {
	manifest := relationshipTestManifest()
	duplicate := manifest.Relationships[0]
	duplicate.Key = "relationship:cn_member_of_g20_duplicate"
	manifest.Relationships = append(manifest.Relationships, duplicate)

	if err := Validate(manifest); err == nil || !strings.Contains(err.Error(), "duplicate relationship tuple") {
		t.Fatalf("Validate() error = %v, want duplicate relationship tuple", err)
	}
}

func relationshipTestManifest() Manifest {
	verifiedAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	return Manifest{
		Entities: []Entity{
			{Key: "economy:cn", EntityType: domain.EntityTypeEconomy, LayerCode: "economy", Name: "中国", CanonicalName: "中国", Profile: []byte(`{"country_code":"CN","currency_code":"CNY"}`)},
			{Key: "alliance_org:g20", EntityType: domain.EntityTypeAllianceOrg, LayerCode: "alliance", Name: "二十国集团", CanonicalName: "二十国集团", Profile: []byte(`{"org_code":"G20","org_type":"economic_forum"}`)},
		},
		Relationships: []Relationship{{
			Key: "relationship:cn_member_of_g20", From: "economy:cn", To: "alliance_org:g20", RelationType: "member_of",
			SourceName: "G20", SourceURL: "https://g20.org/members/", VerifiedAt: verifiedAt,
		}},
	}
}

func TestLoadManifestRejectsDuplicateEntityKeys(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{"key": "economy:cn", "entity_type": "economy", "layer_code": "economy", "name": "中国", "canonical_name": "中国", "profile": {"country_code": "CN", "currency_code": "CNY"}},
			{"key": "economy:cn", "entity_type": "economy", "layer_code": "economy", "name": "中国", "canonical_name": "中国", "profile": {"country_code": "CN", "currency_code": "CNY"}}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "duplicate entity key") {
		t.Fatalf("LoadFile() error = %v, want duplicate entity key", err)
	}
}

func TestLoadManifestRejectsDanglingRelationshipReferences(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{"key": "economy:cn", "entity_type": "economy", "layer_code": "economy", "name": "中国", "canonical_name": "中国", "profile": {"country_code": "CN", "currency_code": "CNY"}}
		],
		"relationships": [
			{"key": "relationship:missing", "from": "economy:cn", "to": "alliance_org:wto", "relation_type": "member_of"}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "unknown relationship target") {
		t.Fatalf("LoadFile() error = %v, want unknown relationship target", err)
	}
}

func TestLoadManifestRejectsDuplicateRelationshipKeys(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{"key": "economy:cn", "entity_type": "economy", "layer_code": "economy", "name": "中国", "canonical_name": "中国", "profile": {"country_code": "CN", "currency_code": "CNY"}},
			{"key": "alliance_org:g20", "entity_type": "alliance_org", "layer_code": "alliance", "name": "二十国集团", "canonical_name": "二十国集团", "profile": {"org_code": "G20", "org_type": "economic_forum"}}
		],
		"relationships": [
			{"key": "relationship:cn_member_of_g20", "from": "economy:cn", "to": "alliance_org:g20", "relation_type": "member_of", "source_name": "G20", "source_url": "https://g20.org/members/", "verified_at": "2026-07-10T00:00:00Z"},
			{"key": "relationship:cn_member_of_g20", "from": "economy:cn", "to": "alliance_org:g20", "relation_type": "member_of", "source_name": "G20", "source_url": "https://g20.org/members/", "verified_at": "2026-07-10T00:00:00Z"}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "duplicate relationship key") {
		t.Fatalf("LoadFile() error = %v, want duplicate relationship key", err)
	}
}

func TestLoadManifestRejectsDanglingProfileReferences(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{"key": "economy:cn", "entity_type": "economy", "layer_code": "economy", "name": "中国", "canonical_name": "中国", "profile": {"country_code": "CN", "currency_code": "CNY"}}
		],
		"profiles": [
			{"entity_key": "alliance_org:g20", "entity_type": "alliance_org", "data": {"org_code": "G20", "org_type": "economic_forum"}}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "unknown profile entity key") {
		t.Fatalf("LoadFile() error = %v, want unknown profile entity key", err)
	}
}

func TestLoadManifestRejectsForbiddenReasoningFields(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{
				"key": "metric:event_score",
				"entity_type": "metric",
				"layer_code": "metric",
				"name": "事件评分",
				"canonical_name": "事件评分",
				"profile": {"metric_type": "event_score", "unit": "score", "frequency": "event"},
				"event_score": 80
			}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "forbidden reasoning field") {
		t.Fatalf("LoadFile() error = %v, want forbidden reasoning field", err)
	}
}

func TestLoadManifestRejectsMissingRequiredProfileField(t *testing.T) {
	path := writeManifest(t, `{
		"entities": [
			{
				"key": "alliance_org:opec_plus",
				"entity_type": "alliance_org",
				"layer_code": "alliance",
				"name": "OPEC+",
				"canonical_name": "OPEC+",
				"profile": {"org_type": "energy_alliance"}
			}
		]
	}`)

	if _, err := LoadFile(path); err == nil || !strings.Contains(err.Error(), "org_code is required") {
		t.Fatalf("LoadFile() error = %v, want missing org code", err)
	}
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}
