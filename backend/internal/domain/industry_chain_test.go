package domain

import (
	"strings"
	"testing"
	"time"
)

func TestIndustryChainDomainValidation(t *testing.T) {
	verifiedAt := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		err  error
	}{
		{name: "profile", err: (IndustryChainProfile{EntityID: "chain", ChainCode: "ai", Definition: "AI", ScopeType: IndustryChainScopeGlobal, Version: 1, ReviewStatus: ReviewStatusApproved, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt}).Validate()},
		{name: "membership", err: (IndustryChainMembership{ID: "membership", IndustryChainEntityID: "chain", ChainNodeEntityID: "node", StageCode: IndustryChainStageMidstream, RoleCode: IndustryChainRoleComponent, StageOrder: 10, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: StatusActive}).Validate()},
		{name: "topology", err: (IndustryChainTopologyEdge{ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "node-a", ToChainNodeEntityID: "node-b", RelationType: IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: StatusActive}).Validate()},
		{name: "constraint", err: (IndustryChainPhysicalConstraint{ID: "constraint", IndustryChainEntityID: "chain", ChainNodeEntityID: "node", ConstraintType: PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, ReviewStatus: ReviewStatusCandidate, Status: StatusActive}).Validate()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err != nil {
				t.Fatalf("Validate() error = %v", tc.err)
			}
		})
	}
}

func TestIndustryChainDomainRejectsInvalidValues(t *testing.T) {
	valid := IndustryChainPhysicalConstraint{ID: "constraint", IndustryChainEntityID: "chain", ChainNodeEntityID: "node", ConstraintType: PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: time.Now(), ReviewStatus: ReviewStatusCandidate, Status: StatusActive}
	tests := map[string]IndustryChainPhysicalConstraint{
		"unknown constraint": func() IndustryChainPhysicalConstraint {
			v := valid
			v.ConstraintType = "supplier_concentration"
			return v
		}(),
		"two subjects": func() IndustryChainPhysicalConstraint { v := valid; v.TopologyEdgeID = "edge"; return v }(),
		"reasoning field": func() IndustryChainPhysicalConstraint {
			v := valid
			v.ReasoningFields = map[string]string{"severity": "high"}
			return v
		}(),
	}
	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			if err := value.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestValidateIndustryChainBatchAssociationsAndApprovalGate(t *testing.T) {
	verifiedAt := time.Now()
	memberships := []IndustryChainMembership{
		{ID: "ma", IndustryChainEntityID: "chain", ChainNodeEntityID: "a", StageCode: IndustryChainStageUpstream, RoleCode: IndustryChainRoleComponent, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: StatusActive},
		{ID: "mb", IndustryChainEntityID: "chain", ChainNodeEntityID: "b", StageCode: IndustryChainStageDownstream, RoleCode: IndustryChainRoleProduct, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: StatusActive},
	}
	edges := []IndustryChainTopologyEdge{{ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b", RelationType: IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, Status: StatusActive}}
	constraint := IndustryChainPhysicalConstraint{ID: "constraint", IndustryChainEntityID: "chain", TopologyEdgeID: "edge", ConstraintType: PhysicalConstraintPowerCapacity, Mechanism: "power", SourceName: "review", SourceURL: "https://example.com", VerifiedAt: verifiedAt, ReviewStatus: ReviewStatusApproved, Status: StatusActive, GeneratedByAI: true}
	if err := ValidateIndustryChainBatch(memberships, edges, []IndustryChainPhysicalConstraint{constraint}, IndustryChainApprovalGate{}); err == nil || !strings.Contains(err.Error(), "human approval") {
		t.Fatalf("unapproved AI constraint error = %v", err)
	}
	gate := IndustryChainApprovalGate{HumanApprovedConstraintIDs: map[string]struct{}{"constraint": {}}}
	if err := ValidateIndustryChainBatch(memberships, edges, []IndustryChainPhysicalConstraint{constraint}, gate); err != nil {
		t.Fatalf("human-approved AI constraint error = %v", err)
	}
	invalidEdge := edges[0]
	invalidEdge.ToChainNodeEntityID = "missing"
	if err := ValidateIndustryChainBatch(memberships, []IndustryChainTopologyEdge{invalidEdge}, nil, IndustryChainApprovalGate{}); err == nil || !strings.Contains(err.Error(), "membership") {
		t.Fatalf("non-member topology error = %v", err)
	}
	invalidConstraint := constraint
	invalidConstraint.GeneratedByAI = false
	invalidConstraint.IndustryChainEntityID = "other-chain"
	if err := ValidateIndustryChainBatch(memberships, edges, []IndustryChainPhysicalConstraint{invalidConstraint}, IndustryChainApprovalGate{}); err == nil || !strings.Contains(err.Error(), "same chain") {
		t.Fatalf("cross-chain constraint error = %v", err)
	}
}

func TestValidateIndustryChainTopologyRejectsCanonicalDuplicates(t *testing.T) {
	edges := []IndustryChainTopologyEdge{
		{ID: "supply", IndustryChainEntityID: "chain", FromChainNodeEntityID: "supplier", ToChainNodeEntityID: "receiver", RelationType: IndustryChainRelationSuppliesTo, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: time.Now(), Status: StatusActive},
		{ID: "duplicate", IndustryChainEntityID: "chain", FromChainNodeEntityID: "receiver", ToChainNodeEntityID: "supplier", RelationType: IndustryChainRelationDependsOn, SourceName: "review", SourceURL: "https://example.com", VerifiedAt: time.Now(), Status: StatusActive},
	}
	for _, candidate := range [][]IndustryChainTopologyEdge{edges, []IndustryChainTopologyEdge{edges[1], edges[0]}} {
		if err := ValidateIndustryChainTopology(candidate); err == nil || !strings.Contains(err.Error(), "canonical") {
			t.Fatalf("ValidateIndustryChainTopology() error = %v, want canonical duplicate", err)
		}
	}
}

func TestEntityNodeAcceptsIndustryChainWithoutChangingChainNode(t *testing.T) {
	for _, entityType := range []EntityType{EntityTypeIndustryChain, EntityTypeChainNode} {
		node := EntityNode{ID: "id", EntityType: entityType, LayerCode: "industry_chain", Name: "name", CanonicalName: "name", Status: StatusActive}
		if err := node.Validate(); err != nil {
			t.Fatalf("EntityNode.Validate(%q) error = %v", entityType, err)
		}
	}
}
