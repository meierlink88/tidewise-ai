package domain

import (
	"testing"
	"time"
)

func TestEntityNodeValidate(t *testing.T) {
	node := EntityNode{
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

func TestEntityNodeValidateRejectsUnsupportedEntityType(t *testing.T) {
	node := EntityNode{
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

func TestBenchmarkEntityTypeAndProfileValidate(t *testing.T) {
	node := EntityNode{
		ID:            "benchmark-1",
		EntityType:    EntityTypeBenchmark,
		LayerCode:     "benchmark",
		Name:          "美国10年期国债收益率",
		CanonicalName: "美国10年期国债收益率",
		Status:        StatusActive,
	}
	if err := node.Validate(); err != nil {
		t.Fatalf("benchmark EntityNode.Validate() error = %v", err)
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

func TestRawDocumentValidate(t *testing.T) {
	document := RawDocument{
		ID:              "raw-1",
		ContractVersion: 2,
		ArtifactID:      "artifact-1",
		SourceRef:       "source:example:feed",
		SourceType:      "news",
		SourceName:      "示例来源",
		Title:           "示例标题",
		ContentHash:     "hash-1",
		CollectedAt:     time.Now(),
		IngestStatus:    IngestStatusCollected,
	}

	if err := document.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	document.ContentHash = ""
	if err := document.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing content hash error")
	}
}

func TestEventValidate(t *testing.T) {
	event := Event{
		ID:          "event-1",
		Title:       "示例事件",
		FirstSeenAt: time.Now(),
		EventStatus: EventStatusCandidate,
		FactStatus:  FactStatusUnverified,
		DedupeKey:   "event:demo",
		FactPayload: FactPayload{},
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	event.FactStatus = "certain"
	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid fact status error")
	}
}

func TestEventFactPayloadContract(t *testing.T) {
	validPayloads := []FactPayload{
		{},
		{"policy_rate": map[string]any{"value": 3.5}},
	}
	for _, payload := range validPayloads {
		event := Event{
			ID:          "event-1",
			Title:       "示例事件",
			FirstSeenAt: time.Now(),
			EventStatus: EventStatusCandidate,
			FactStatus:  FactStatusUnverified,
			DedupeKey:   "event:demo",
			FactPayload: payload,
		}
		if err := event.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	}

	for name, payload := range map[string]any{
		"nil":                 nil,
		"typed-nil":           FactPayload(nil),
		"non-object":          []string{"not", "an", "object"},
		"non-encodable-value": map[string]any{"invalid": func() {}},
		"prediction":          map[string]any{"price_prediction": "上涨"},
		"score":               map[string]any{"event_score": 80},
		"investment-advice":   map[string]any{"investment_advice": "buy"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateFactPayload(payload); err == nil {
				t.Fatalf("ValidateFactPayload(%#v) error = nil, want rejection", payload)
			}
		})
	}
}

func TestEventSourceValidateEvidenceAttribution(t *testing.T) {
	legacy := EventSource{ID: "source-1", EventID: "event-1", RawDocumentID: "raw-1", EvidenceHash: "hash-1"}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("legacy Validate() error = %v", err)
	}

	supports := legacy
	supports.EvidenceRelation = EvidenceRelationSupports
	supports.SupportsFields = []string{"policy_rate"}
	if err := supports.Validate(); err != nil {
		t.Fatalf("supports Validate() error = %v", err)
	}

	for name, source := range map[string]EventSource{
		"supports-without-fields":    {EvidenceRelation: EvidenceRelationSupports},
		"contradicts-without-fields": {EvidenceRelation: EvidenceRelationContradicts},
		"unsupported-relation":       {EvidenceRelation: EvidenceRelation("irrelevant")},
	} {
		t.Run(name, func(t *testing.T) {
			if err := source.Validate(); err == nil {
				t.Fatalf("Validate() error = nil, want rejection")
			}
		})
	}
}

func TestEventTagMapValidateAttribution(t *testing.T) {
	legacy := EventTagMap{ID: "map-1", EventID: "event-1", TagID: "tag-1"}
	if err := legacy.Validate(); err != nil {
		t.Fatalf("legacy Validate() error = %v", err)
	}

	confidence := 0.85
	aiAssignment := legacy
	aiAssignment.AssignSource = TagAssignSourceAI
	aiAssignment.Confidence = &confidence
	aiAssignment.AssignmentReason = "模型抽取的政策主题"
	if err := aiAssignment.Validate(); err != nil {
		t.Fatalf("AI assignment Validate() error = %v", err)
	}

	missingReason := aiAssignment
	missingReason.AssignmentReason = ""
	if err := missingReason.Validate(); err == nil {
		t.Fatal("AI assignment without reason Validate() error = nil, want rejection")
	}

	overLimit := 1.0001
	aiAssignment.Confidence = &overLimit
	if err := aiAssignment.Validate(); err == nil {
		t.Fatal("out-of-range confidence Validate() error = nil, want rejection")
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
