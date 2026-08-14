package entity

import (
	"testing"
	"time"
)

func TestOrganizationIdentityRequiresStableORGCode(t *testing.T) {
	for value, want := range map[string]bool{
		"ORG_UN":                               true,
		"ORG_G7_2026":                          true,
		"ORG_1UN":                              false,
		"ORG_un":                               false,
		"6f845f9f-10e2-44dd-b08a-e482e32d3558": false,
	} {
		if got := IsOrganizationID(value); got != want {
			t.Errorf("IsOrganizationID(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestChainNodeAndThemeProfiles(t *testing.T) {
	tests := []struct {
		name    string
		profile interface{ Validate() error }
		wantErr bool
	}{
		{name: "chain node", profile: ChainNodeProfile{EntityID: "node", Definition: "可独立链接的产业概念"}},
		{name: "chain node optional boundary", profile: ChainNodeProfile{EntityID: "node", Definition: "可独立链接的产业概念", BoundaryNote: "不含行情标签"}},
		{name: "chain node blank definition", profile: ChainNodeProfile{EntityID: "node", Definition: " "}, wantErr: true},
		{name: "chain node blank optional boundary", profile: ChainNodeProfile{EntityID: "node", Definition: "节点", BoundaryNote: " "}, wantErr: true},
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
	node := Entity{ID: "theme", EntityType: EntityTypeTheme, LayerCode: "theme", Name: "主题", CanonicalName: "主题", Status: StatusActive}
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

func TestEntityAcceptsIndustryAndConceptAsDistinctTypes(t *testing.T) {
	for _, entityType := range []EntityType{EntityTypeIndustry, EntityTypeConcept} {
		node := Entity{ID: "id", EntityType: entityType, LayerCode: string(entityType), Name: "人工智能", CanonicalName: "人工智能", Status: StatusActive}
		if err := node.Validate(); err != nil {
			t.Fatalf("Entity.Validate(%q) error = %v", entityType, err)
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
				Geography: "global_with_china_research_focus", PrimaryCountryID: "COU_CHN",
				AsOfDate: validDate, ReviewStatus: ReviewStatusCandidate,
			},
		},
		{
			name: "definition rejects legacy country identity",
			value: IndustryChainDefinition{
				EntityID: "chain", Scope: "AI 算力主链", TargetOutput: "可用算力", EndUse: "AI 训练与推理",
				Geography: "china", PrimaryCountryID: "legacy-country-id", AsOfDate: validDate, ReviewStatus: ReviewStatusCandidate,
			},
			wantErr: true,
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

func TestEntityValidate(t *testing.T) {
	node := Entity{
		ID:            "entity-1",
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
	entity := Entity{ID: "product-id", EntityType: EntityTypeProduct, LayerCode: "product", Name: "AI芯片", CanonicalName: "AI芯片", Status: StatusActive}
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

func TestBenchmarkEntityTypeAndProfileValidate(t *testing.T) {
	node := Entity{
		ID:            "benchmark-1",
		EntityType:    EntityTypeBenchmark,
		LayerCode:     "benchmark",
		Name:          "美国10年期国债收益率",
		CanonicalName: "美国10年期国债收益率",
		Status:        StatusActive,
	}
	if err := node.Validate(); err != nil {
		t.Fatalf("benchmark Entity.Validate() error = %v", err)
	}

	profile := BenchmarkProfile{
		EntityID:           "benchmark-1",
		BenchmarkType:      BenchmarkTypeGovernmentBondYield,
		OfficialSeriesCode: "",
		Provider:           "us_treasury",
		Tenor:              "10Y",
		CurrencyCode:       "USD",
		Unit:               "percent",
		Frequency:          "daily",
		SourceURL:          "https://home.treasury.gov/",
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("BenchmarkProfile.Validate() error = %v", err)
	}

	profile.BenchmarkType = "index"
	if err := profile.Validate(); err == nil {
		t.Fatal("BenchmarkProfile.Validate() error = nil, want invalid benchmark type error")
	}
}

func TestBenchmarkObservationQualityStatusValidate(t *testing.T) {
	validStatuses := []BenchmarkObservationQualityStatus{
		BenchmarkObservationQualityRaw,
		BenchmarkObservationQualityValidated,
		BenchmarkObservationQualitySuspect,
		BenchmarkObservationQualityRejected,
	}
	for _, status := range validStatuses {
		observation := BenchmarkObservation{
			ID:                "observation-1",
			BenchmarkEntityID: "benchmark-1",
			ObservedAt:        time.Now(),
			Value:             "4.25",
			Unit:              "percent",
			SourceName:        "US Treasury",
			QualityStatus:     status,
		}
		if err := observation.Validate(); err != nil {
			t.Fatalf("BenchmarkObservation.Validate() status %q error = %v", status, err)
		}
	}

	observation := BenchmarkObservation{
		ID:                "observation-1",
		BenchmarkEntityID: "benchmark-1",
		ObservedAt:        time.Now(),
		Value:             "4.25",
		Unit:              "percent",
		SourceName:        "US Treasury",
		QualityStatus:     "estimated",
	}
	if err := observation.Validate(); err == nil {
		t.Fatal("BenchmarkObservation.Validate() error = nil, want invalid quality status error")
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
