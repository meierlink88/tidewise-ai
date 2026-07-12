package seed

import (
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestValidateIndustryChainManifest(t *testing.T) {
	manifest := validIndustryChainManifest()
	if err := Validate(manifest); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateIndustryChainManifestRejectsInvalidCandidates(t *testing.T) {
	tests := map[string]func(*Manifest){
		"duplicate membership": func(m *Manifest) {
			m.IndustryChainMemberships = append(m.IndustryChainMemberships, m.IndustryChainMemberships[0])
		},
		"non member topology endpoint": func(m *Manifest) { m.IndustryChainTopologyEdges[0].ToChainNodeKey = "chain_node:missing" },
		"self topology": func(m *Manifest) {
			m.IndustryChainTopologyEdges[0].ToChainNodeKey = m.IndustryChainTopologyEdges[0].FromChainNodeKey
		},
		"AI candidate approved":   func(m *Manifest) { m.IndustryChainPhysicalConstraints[0].ReviewStatus = domain.ReviewStatusApproved },
		"non physical constraint": func(m *Manifest) { m.IndustryChainPhysicalConstraints[0].ConstraintType = "supplier_concentration" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			manifest := validIndustryChainManifest()
			mutate(&manifest)
			if err := Validate(manifest); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestIndustryChainRelationshipPolicy(t *testing.T) {
	entities := map[string]Entity{
		"industry_chain:test": {Key: "industry_chain:test", EntityType: domain.EntityTypeIndustryChain},
		"chain_node:test":     {Key: "chain_node:test", EntityType: domain.EntityTypeChainNode},
		"economy:cn":          {Key: "economy:cn", EntityType: domain.EntityTypeEconomy},
		"commodity:test":      {Key: "commodity:test", EntityType: domain.EntityTypeCommodity},
		"benchmark:test":      {Key: "benchmark:test", EntityType: domain.EntityTypeBenchmark},
		"sector:cn":           {Key: "sector:cn", EntityType: domain.EntityTypeSector, Profile: []byte(`{"primary_economy_entity_id":"economy:cn"}`)},
		"market:us":           {Key: "market:us", EntityType: domain.EntityTypeMarket, Profile: []byte(`{"market_type":"equity","economy_entity_id":"economy:us"}`)},
	}
	base := Relationship{Key: "relationship:test", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: time.Now()}
	valid := []Relationship{
		withRelationship(base, "industry_chain:test", "scoped_to_economy", "economy:cn"),
		withRelationship(base, "chain_node:test", "uses_commodity", "commodity:test"),
		withRelationship(base, "chain_node:test", "produces_commodity", "commodity:test"),
		withRelationship(base, "industry_chain:test", "observed_by_benchmark", "benchmark:test"),
		withRelationship(base, "chain_node:test", "mapped_to_sector", "sector:cn"),
	}
	for _, relationship := range valid {
		if err := validateRelationshipPolicy(relationship, entities); err != nil {
			t.Fatalf("validateRelationshipPolicy(%s) error = %v", relationship.RelationType, err)
		}
	}
	invalid := withRelationship(base, "market:us", "covers_sector", "sector:cn")
	if err := validateRelationshipPolicy(invalid, entities); err == nil || !strings.Contains(err.Error(), "overseas market") {
		t.Fatalf("validateRelationshipPolicy(covers_sector) error = %v", err)
	}
}

func validIndustryChainManifest() Manifest {
	verifiedAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	return Manifest{
		Entities: []Entity{
			{Key: "industry_chain:test", EntityType: domain.EntityTypeIndustryChain, LayerCode: "industry_chain", Name: "测试链", CanonicalName: "测试链", Profile: []byte(`{"chain_code":"test","definition":"test","scope_type":"global","version":1,"review_status":"approved","source_name":"review","source_url":"https://example.com","verified_at":"2026-07-12T00:00:00Z"}`)},
			{Key: "chain_node:a", EntityType: domain.EntityTypeChainNode, LayerCode: "industry_chain", Name: "节点A", CanonicalName: "节点A", Profile: []byte(`{"chain_position":"upstream"}`)},
			{Key: "chain_node:b", EntityType: domain.EntityTypeChainNode, LayerCode: "industry_chain", Name: "节点B", CanonicalName: "节点B", Profile: []byte(`{"chain_position":"downstream"}`)},
		},
		IndustryChainMemberships: []IndustryChainMembershipSeed{
			{ID: "membership:a", IndustryChainKey: "industry_chain:test", ChainNodeKey: "chain_node:a", StageCode: domain.IndustryChainStageUpstream, RoleCode: domain.IndustryChainRoleComponent, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive},
			{ID: "membership:b", IndustryChainKey: "industry_chain:test", ChainNodeKey: "chain_node:b", StageCode: domain.IndustryChainStageDownstream, RoleCode: domain.IndustryChainRoleProduct, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive},
		},
		IndustryChainTopologyEdges:       []IndustryChainTopologySeed{{ID: "edge:a-b", IndustryChainKey: "industry_chain:test", FromChainNodeKey: "chain_node:a", ToChainNodeKey: "chain_node:b", RelationType: domain.IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: domain.StatusActive}},
		IndustryChainPhysicalConstraints: []IndustryChainPhysicalConstraintSeed{{ID: "constraint:a", IndustryChainKey: "industry_chain:test", ChainNodeKey: "chain_node:a", ConstraintType: domain.PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, ReviewStatus: domain.ReviewStatusCandidate, Status: domain.StatusActive, GeneratedByAI: true}},
	}
}

func withRelationship(base Relationship, from, relationType, to string) Relationship {
	base.From, base.RelationType, base.To = from, relationType, to
	return base
}
