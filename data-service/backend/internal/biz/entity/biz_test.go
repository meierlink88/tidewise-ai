package entity

import (
	"testing"
)

const testEntityID = "ENT550e8400-e29b-41d4-a716-446655440000"

func TestDomainObjectIdentityRequiresPrefixAndUUID(t *testing.T) {
	for value, want := range map[string]bool{
		"ORG550e8400-e29b-41d4-a716-446655440000":  true,
		"ORG_550e8400-e29b-41d4-a716-446655440000": false,
		"ORG550E8400-E29B-41D4-A716-446655440000":  false,
		"6f845f9f-10e2-44dd-b08a-e482e32d3558":     false,
	} {
		if got := IsOrganizationID(value); got != want {
			t.Errorf("IsOrganizationID(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestObjectIdentityRecognizesIndependentDomainPrefixes(t *testing.T) {
	suffix := "550e8400-e29b-41d4-a716-446655440000"
	for objectType, prefix := range map[string]string{
		ObjectTypeIndustry:      "IND",
		ObjectTypeConcept:       "CON",
		ObjectTypeChainNode:     "CND",
		ObjectTypeIndustryChain: "ICH",
	} {
		value := prefix + suffix
		if !IsObjectID(value) {
			t.Errorf("IsObjectID(%q) = false", value)
		}
		if !ObjectTypeMatchesID(objectType, value) {
			t.Errorf("ObjectTypeMatchesID(%q, %q) = false", objectType, value)
		}
		if ObjectTypeMatchesID(objectType, "ENT"+suffix) {
			t.Errorf("ObjectTypeMatchesID(%q, legacy ENT ID) = true", objectType)
		}
	}
}

func TestThemeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile interface{ Validate() error }
		wantErr bool
	}{
		{name: "theme", profile: ThemeProfile{EntityID: "theme", Definition: "自有投研视角", BoundaryNote: "不等同于产业链节点"}},
		{name: "theme missing boundary", profile: ThemeProfile{EntityID: "theme", Definition: "自有投研视角"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.profile.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestEntityAcceptsTheme(t *testing.T) {
	node := Entity{ID: testEntityID, EntityType: EntityTypeTheme, LayerCode: "theme", Name: "主题", CanonicalName: "主题", Status: StatusActive}
	if err := node.Validate(); err != nil {
		t.Fatalf("Entity.Validate() error = %v", err)
	}
}

func TestEntityExternalIdentifierValidation(t *testing.T) {
	valid := EntityExternalIdentifier{
		ID:                 "identifier-id",
		EntityID:           "entity-id",
		SourceSystem:       "eastmoney",
		SourceTaxonomyType: "concept_sector",
		ExternalCode:       "BK0619",
		ExternalName:       "3D打印",
		Status:             StatusActive,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []EntityExternalIdentifier{
		{EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: " ", SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: " ", ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: " ", ExternalName: valid.ExternalName, Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: " ", Status: valid.Status},
		{ID: valid.ID, EntityID: valid.EntityID, SourceSystem: valid.SourceSystem, SourceTaxonomyType: valid.SourceTaxonomyType, ExternalCode: valid.ExternalCode, ExternalName: valid.ExternalName, Status: StatusMerged},
	}
	for i, candidate := range tests {
		if err := candidate.Validate(); err == nil {
			t.Fatalf("case %d Validate() error = nil", i)
		}
	}
}

func TestGenericEntityRejectsIndependentObjectTypes(t *testing.T) {
	for _, entityType := range []EntityType{EntityTypeIndustry, EntityTypeConcept, EntityTypeChainNode, EntityTypeIndustryChain} {
		node := Entity{ID: testEntityID, EntityType: entityType, LayerCode: string(entityType), Name: "人工智能", CanonicalName: "人工智能", Status: StatusActive}
		if err := node.Validate(); err == nil {
			t.Fatalf("Entity.Validate(%q) error = nil", entityType)
		}
	}
}

func TestIndustryChainMasterDataTypesValidateNewSchemaVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{ Validate() error }
		wantErr bool
	}{
		{
			name: "membership",
			value: IndustryChainNodeMembership{
				IndustryChainID: "chain", ChainNodeID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStageUpstream, ReviewStatus: ReviewStatusApproved, Status: StatusActive,
			},
		},
		{
			name: "direct graph edge",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainID: "chain", FromChainNodeID: "a", ToChainNodeID: "b",
				RelationType: ChainNodeRelationInputTo, Mechanism: "A 的产出进入 B", SegmentKind: IndustryChainSegmentDirectCandidate,
				ReviewStatus: ReviewStatusCandidate, Status: StatusActive,
			},
		},
		{
			name: "legacy stage is rejected",
			value: IndustryChainNodeMembership{
				IndustryChainID: "chain", ChainNodeID: "node", Position: 1,
				ContextualStage: IndustryChainContextualStage("infrastructure"), ReviewStatus: ReviewStatusApproved, Status: StatusActive,
			},
			wantErr: true,
		},
		{
			name: "compressed edge requires omitted step",
			value: IndustryChainGraphEdge{
				ID: "edge", IndustryChainID: "chain", FromChainNodeID: "a", ToChainNodeID: "b",
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

func TestEntityValidate(t *testing.T) {
	node := Entity{
		ID:            testEntityID,
		EntityType:    EntityTypeCompany,
		LayerCode:     "company",
		Name:          "示例公司",
		CanonicalName: "示例公司",
		Status:        StatusActive,
	}

	if err := node.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	node.Status = "unknown"
	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid status error")
	}
}

func TestEntityValidateRejectsUnsupportedEntityType(t *testing.T) {
	node := Entity{
		ID:            "entity-1",
		EntityType:    "unknown",
		LayerCode:     "unknown",
		Name:          "未知实体",
		CanonicalName: "未知实体",
		Status:        StatusActive,
	}

	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported entity type error")
	}
}

func TestProductEntityAndProfileValidate(t *testing.T) {
	entity := Entity{ID: testEntityID, EntityType: EntityTypeProduct, LayerCode: "product", Name: "AI芯片", CanonicalName: "AI芯片", Status: StatusActive}
	if err := entity.Validate(); err != nil {
		t.Fatalf("product Entity.Validate() error = %v", err)
	}
	profile := ProductProfile{EntityID: entity.ID, ProductCategory: "semiconductor", Specification: "accelerator", ReviewStatus: ReviewStatusApproved}
	if err := profile.Validate(); err != nil {
		t.Fatalf("ProductProfile.Validate() error = %v", err)
	}
	profile.ReviewStatus = ReviewStatusRejected
	if err := profile.Validate(); err == nil {
		t.Fatal("ProductProfile.Validate() expected unsupported review status")
	}
}

func TestEntityValidateRejectsRetiredEntityTypes(t *testing.T) {
	for _, entityType := range []EntityType{"benchmark", "metric"} {
		node := Entity{ID: testEntityID, EntityType: entityType, LayerCode: string(entityType), Name: "retired", CanonicalName: "retired", Status: StatusActive}
		if err := node.Validate(); err == nil {
			t.Fatalf("Entity.Validate() type %q error = nil, want rejection", entityType)
		}
	}
}

func TestSectorProfileAndSourceMappingValidateSemanticBoundaries(t *testing.T) {
	profile := SectorProfile{EntityID: "sector-id", ClassificationCode: SectorClassificationTheme, ReviewStatus: SectorReviewApproved}
	if err := profile.Validate(); err != nil {
		t.Fatalf("SectorProfile.Validate() error = %v", err)
	}
	profile.ClassificationCode = SectorClassification("index_sector")
	if err := profile.Validate(); err == nil {
		t.Fatal("SectorProfile.Validate() expected index_sector rejection")
	}
	profile.ClassificationCode = SectorClassificationTheme
	profile.PrimaryCountryID = "legacy-country-id"
	if err := profile.Validate(); err == nil {
		t.Fatal("SectorProfile.Validate() expected legacy country identity rejection")
	}
	mapping := SectorSourceMapping{
		ID: "mapping-id", SectorEntityID: "sector-id", SourceSystem: "ths",
		SourceTaxonomyType: SectorSourceTaxonomyIndexSector,
		SourceSectorName:   "人工智能", SourceSectorNameNormalized: "人工智能",
		MappingStatus: SectorSourceMappingApproved,
	}
	if err := mapping.Validate(); err != nil {
		t.Fatalf("SectorSourceMapping.Validate() error = %v", err)
	}
}
