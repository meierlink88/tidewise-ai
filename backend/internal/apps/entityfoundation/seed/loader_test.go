package seed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestValidateRelationshipPolicyAcceptsSupportedTypeDirections(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	entities := map[string]Entity{
		"economy:cn":          {Key: "economy:cn", EntityType: domain.EntityTypeEconomy},
		"alliance_org:g20":    {Key: "alliance_org:g20", EntityType: domain.EntityTypeAllianceOrg},
		"market:sse":          {Key: "market:sse", EntityType: domain.EntityTypeMarket},
		"index:sse_composite": {Key: "index:sse_composite", EntityType: domain.EntityTypeIndex},
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
