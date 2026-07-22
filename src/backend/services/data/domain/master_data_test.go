package domain

import (
	"testing"
	"time"
)

func TestTypedMasterDataProfilesValidateFrozenVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		profile interface{ Validate() error }
		wantErr bool
	}{
		{
			name: "industry",
			profile: IndustryProfile{
				EntityID: "industry", ClassificationSystem: "sw", ClassificationVersion: "workbook_snapshot_v1",
				IndustryCode: "801010", ClassificationLevel: 2, ParentIndustryEntityID: "parent",
				HierarchyPathCodes: []string{"801000", "801010"}, Definition: "二级行业", BoundaryNote: "行业边界",
				ReviewStatus: ReviewStatusApproved,
			},
		},
		{
			name: "industry path mismatch",
			profile: IndustryProfile{
				EntityID: "industry", ClassificationSystem: "sw", ClassificationVersion: "v1",
				IndustryCode: "801010", ClassificationLevel: 2, ParentIndustryEntityID: "parent",
				HierarchyPathCodes: []string{"801010"}, Definition: "行业", BoundaryNote: "边界",
				ReviewStatus: ReviewStatusApproved,
			},
			wantErr: true,
		},
		{
			name:    "concept",
			profile: ConceptProfile{EntityID: "concept", ConceptType: ConceptTypeTechnology, Definition: "跨行业技术聚合", BoundaryNote: "不是行业", ReviewStatus: ReviewStatusCandidate},
		},
		{
			name:    "concept rejects historical reviewed status",
			profile: ConceptProfile{EntityID: "concept", ConceptType: ConceptTypeTechnology, Definition: "跨行业技术聚合", BoundaryNote: "不是行业", ReviewStatus: ReviewStatusReviewed},
			wantErr: true,
		},
		{
			name:    "concept type",
			profile: ConceptProfile{EntityID: "concept", ConceptType: "sector", Definition: "错误聚合", BoundaryNote: "边界", ReviewStatus: ReviewStatusApproved},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.profile.Validate()
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestEntityNodeAcceptsIndustryAndConceptAsDistinctTypes(t *testing.T) {
	for _, entityType := range []EntityType{EntityTypeIndustry, EntityTypeConcept} {
		node := EntityNode{ID: "id", EntityType: entityType, LayerCode: string(entityType), Name: "人工智能", CanonicalName: "人工智能", Status: StatusActive}
		if err := node.Validate(); err != nil {
			t.Fatalf("EntityNode.Validate(%q) error = %v", entityType, err)
		}
	}
}

func TestChainNodeProfileKeepsLegacyReviewStatusUnclassified(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		status  ReviewStatus
		wantErr bool
	}{
		{name: "legacy null-equivalent", status: ""},
		{name: "candidate", status: ReviewStatusCandidate},
		{name: "approved", status: ReviewStatusApproved},
		{name: "reviewed is not master data status", status: ReviewStatusReviewed, wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := (ChainNodeProfile{EntityID: "node", Definition: "稳定经济节点", ReviewStatus: testCase.status}).Validate()
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestIndustryChainMasterDataTypesValidateNewSchemaVocabulary(t *testing.T) {
	validDate := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		value   interface{ Validate() error }
		wantErr bool
	}{
		{
			name: "definition",
			value: IndustryChainDefinition{
				EntityID: "chain", Scope: "AI 算力主链", TargetOutput: "可用算力", EndUse: "AI 训练与推理",
				Geography: "global_with_china_research_focus", AsOfDate: validDate, ReviewStatus: ReviewStatusCandidate,
			},
		},
		{
			name: "membership",
			value: IndustryChainNodeMembership{
				IndustryChainEntityID: "chain", ChainNodeEntityID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStageUpstream, ReviewStatus: ReviewStatusApproved, Status: StatusActive,
			},
		},
		{
			name: "direct graph edge",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b",
				RelationType: ChainNodeRelationInputTo, Mechanism: "A 的产出进入 B", SegmentKind: IndustryChainSegmentDirectCandidate,
				ReviewStatus: ReviewStatusCandidate, Status: StatusActive,
			},
		},
		{
			name: "legacy stage is rejected",
			value: IndustryChainNodeMembership{
				IndustryChainEntityID: "chain", ChainNodeEntityID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStage("infrastructure"), ReviewStatus: ReviewStatusApproved, Status: StatusActive,
			},
			wantErr: true,
		},
		{
			name: "compressed edge requires omitted step",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainEntityID: "chain", FromChainNodeEntityID: "a", ToChainNodeEntityID: "b",
				RelationType: ChainNodeRelationDependsOn, Mechanism: "跨环节依赖", SegmentKind: IndustryChainSegmentCompressedCandidate,
				ReviewStatus: ReviewStatusCandidate, Status: StatusActive,
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.value.Validate()
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}
