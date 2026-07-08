package entityseed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
				"relation_type": "member_of"
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
			{"key": "relationship:cn_member_of_g20", "from": "economy:cn", "to": "alliance_org:g20", "relation_type": "member_of"},
			{"key": "relationship:cn_member_of_g20", "from": "economy:cn", "to": "alliance_org:g20", "relation_type": "member_of"}
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
