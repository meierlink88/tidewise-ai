package graphprojection

import (
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestMapEntityNodePreservesProjectionFields(t *testing.T) {
	now := time.Date(2026, 7, 10, 11, 0, 0, 0, time.UTC)

	node, err := MapEntityNode(repositories.GraphEntityNode{
		ID:            "entity-1",
		EntityKey:     "economy:cn",
		EntityType:    domain.EntityTypeEconomy,
		LayerCode:     "economy",
		Name:          "中国",
		CanonicalName: "中国",
		Aliases:       []string{"China", "PRC"},
		Status:        domain.StatusActive,
		UpdatedAt:     now,
	}, "tidewise")
	if err != nil {
		t.Fatalf("MapEntityNode() error = %v", err)
	}

	if node.EntityID != "entity-1" || node.EntityKey != "economy:cn" {
		t.Fatalf("node ids = %+v, want entity id and key", node)
	}
	if node.EntityType != "economy" || node.LayerCode != "economy" {
		t.Fatalf("node classification = %+v, want economy/economy", node)
	}
	if node.Name != "中国" || node.CanonicalName != "中国" || node.Status != "active" {
		t.Fatalf("node display fields = %+v", node)
	}
	if len(node.Aliases) != 2 || node.Aliases[0] != "China" || node.Aliases[1] != "PRC" {
		t.Fatalf("node aliases = %v, want PostgreSQL aliases", node.Aliases)
	}
	if node.Namespace != "tidewise" || !node.UpdatedAt.Equal(now) {
		t.Fatalf("node projection fields = %+v", node)
	}
}

func TestMapEntityNodeProjectsOnlyValidSectorClassification(t *testing.T) {
	for _, classification := range []domain.SectorClassification{domain.SectorClassificationIndustry, domain.SectorClassificationTheme} {
		node := repositories.GraphEntityNode{
			ID: "sector-1", EntityKey: "sector:test", EntityType: domain.EntityTypeSector,
			LayerCode: "sector", Name: "测试板块", CanonicalName: "测试板块",
			ClassificationCode: classification, Status: domain.StatusActive,
		}
		mapped, err := MapEntityNode(node, "tidewise")
		if err != nil {
			t.Fatalf("MapEntityNode(%q) error = %v", classification, err)
		}
		if mapped.ClassificationCode != classification {
			t.Fatalf("classification = %q, want %q", mapped.ClassificationCode, classification)
		}
	}

	nonSector, err := MapEntityNode(repositories.GraphEntityNode{
		ID: "market-1", EntityKey: "market:test", EntityType: domain.EntityTypeMarket,
		LayerCode: "market", Name: "测试市场", CanonicalName: "测试市场", Status: domain.StatusActive,
	}, "tidewise")
	if err != nil {
		t.Fatalf("MapEntityNode(non-sector) error = %v", err)
	}
	if nonSector.ClassificationCode != "" {
		t.Fatalf("non-sector classification = %q, want empty", nonSector.ClassificationCode)
	}

	for _, classification := range []domain.SectorClassification{"", "index_sector"} {
		_, err := MapEntityNode(repositories.GraphEntityNode{
			ID: "sector-invalid", EntityKey: "sector:invalid", EntityType: domain.EntityTypeSector,
			LayerCode: "sector", Name: "非法板块", CanonicalName: "非法板块",
			ClassificationCode: classification, Status: domain.StatusActive,
		}, "tidewise")
		if err == nil || !strings.Contains(err.Error(), "sector classification") {
			t.Fatalf("classification %q error = %v, want sector classification error", classification, err)
		}
	}
}

func TestMapEntityNodeRejectsMissingRequiredFields(t *testing.T) {
	_, err := MapEntityNode(repositories.GraphEntityNode{
		ID:         "entity-1",
		EntityType: domain.EntityTypeEconomy,
		Name:       "中国",
		Status:     domain.StatusActive,
	}, "tidewise")
	if err == nil {
		t.Fatal("MapEntityNode() error = nil, want missing field error")
	}
	if !strings.Contains(err.Error(), "entity key") {
		t.Fatalf("MapEntityNode() error = %q, want entity key context", err.Error())
	}
}

func TestMapRelationTypeUsesSafeKnownTypes(t *testing.T) {
	cases := map[string]string{
		"member_of":             "MEMBER_OF",
		"HAS_MARKET":            "HAS_MARKET",
		"tracks_index":          "TRACKS_INDEX",
		"issues":                "ISSUES",
		"participates_in":       "PARTICIPATES_IN",
		"affiliated_with":       "AFFILIATED_WITH",
		"applies_to":            "APPLIES_TO",
		"observes_benchmark":    "OBSERVES_BENCHMARK",
		"covers_sector":         "COVERS_SECTOR",
		"tracked_by_benchmark":  "TRACKED_BY_BENCHMARK",
		"measures":              "MEASURES",
		"references":            "REFERENCES",
		"member_of_chain":       "MEMBER_OF_CHAIN",
		"supplies_to":           "SUPPLIES_TO",
		"is_subcategory_of":     "IS_SUBCATEGORY_OF",
		"is_component_of":       "IS_COMPONENT_OF",
		"input_to":              "INPUT_TO",
		"depends_on":            "DEPENDS_ON",
		"substitutes_for":       "SUBSTITUTES_FOR",
		"scoped_to_economy":     "SCOPED_TO_ECONOMY",
		"uses_commodity":        "USES_COMMODITY",
		"produces_commodity":    "PRODUCES_COMMODITY",
		"observed_by_benchmark": "OBSERVED_BY_BENCHMARK",
		"mapped_to_sector":      "MAPPED_TO_SECTOR",
	}

	for input, want := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := MapRelationType(input)
			if err != nil {
				t.Fatalf("MapRelationType() error = %v", err)
			}
			if got != want {
				t.Fatalf("MapRelationType() = %q, want %q", got, want)
			}
		})
	}
}

func TestMapChainNodeRelationshipsPreservesDirectionAndSource(t *testing.T) {
	nodes := map[string]GraphNode{
		"from": {EntityID: "from"},
		"to":   {EntityID: "to"},
	}
	for _, relationType := range []string{"is_subcategory_of", "is_component_of", "input_to", "depends_on"} {
		t.Run(relationType, func(t *testing.T) {
			edge := repositories.GraphEntityEdge{
				ID: "edge-" + relationType, FromEntityID: "from", ToEntityID: "to",
				RelationType: relationType, Source: "postgres_chain_node_relations", Status: domain.StatusActive,
			}

			relationship, report := MapEntityRelationship(edge, nodes, "tidewise")
			if report.Status != RelationshipMapStatusProjected || relationship == nil {
				t.Fatalf("mapping report = %+v, relationship = %+v", report, relationship)
			}
			if relationship.FromEntityID != "from" || relationship.ToEntityID != "to" {
				t.Fatalf("relationship direction = %s -> %s", relationship.FromEntityID, relationship.ToEntityID)
			}
			if relationship.Source != "postgres_chain_node_relations" {
				t.Fatalf("relationship source = %q", relationship.Source)
			}
		})
	}
}

func TestMapMarketSectorRelationshipsPreservesOriginalType(t *testing.T) {
	nodes := map[string]GraphNode{
		"market":    {EntityID: "market"},
		"sector":    {EntityID: "sector"},
		"benchmark": {EntityID: "benchmark"},
	}
	cases := []struct {
		from         string
		to           string
		relationType string
		mappedType   string
	}{
		{from: "market", to: "sector", relationType: "covers_sector", mappedType: "COVERS_SECTOR"},
		{from: "sector", to: "benchmark", relationType: "tracked_by_benchmark", mappedType: "TRACKED_BY_BENCHMARK"},
	}
	for _, tc := range cases {
		t.Run(tc.relationType, func(t *testing.T) {
			relationship, report := MapEntityRelationship(repositories.GraphEntityEdge{
				ID: "edge-" + tc.relationType, FromEntityID: tc.from, ToEntityID: tc.to,
				RelationType: tc.relationType, Status: domain.StatusActive,
			}, nodes, "tidewise")
			if report.Status != RelationshipMapStatusProjected || relationship == nil {
				t.Fatalf("mapping report = %+v, relationship = %+v", report, relationship)
			}
			if relationship.RelationshipType != tc.mappedType || relationship.OriginalRelationType != tc.relationType {
				t.Fatalf("relationship = %+v", relationship)
			}
		})
	}
}

func TestMapRelationTypeFallsBackForUnknownOrUnsafeValues(t *testing.T) {
	for _, input := range []string{"unknown_type", "member_of) MATCH (n)"} {
		t.Run(input, func(t *testing.T) {
			got, err := MapRelationType(input)
			if err != nil {
				t.Fatalf("MapRelationType() error = %v", err)
			}
			if got != "RELATED_TO" {
				t.Fatalf("MapRelationType() = %q, want RELATED_TO", got)
			}
		})
	}
}

func TestMapRelationTypeRejectsEmptyValues(t *testing.T) {
	if _, err := MapRelationType(" "); err == nil {
		t.Fatal("MapRelationType() error = nil, want empty relation type error")
	}
}

func TestMapEntityRelationshipSkipsMissingEndpointsAndInactiveEdges(t *testing.T) {
	nodes := map[string]GraphNode{
		"entity-1": {EntityID: "entity-1"},
	}
	now := time.Date(2026, 7, 10, 11, 15, 0, 0, time.UTC)

	missing, report := MapEntityRelationship(repositories.GraphEntityEdge{
		ID:           "edge-missing",
		FromEntityID: "entity-1",
		ToEntityID:   "missing",
		RelationType: "member_of",
		Status:       domain.StatusActive,
		UpdatedAt:    now,
	}, nodes, "tidewise")
	if missing != nil {
		t.Fatalf("missing endpoint relationship = %+v, want nil", missing)
	}
	if report.Status != RelationshipMapStatusSkipped || !strings.Contains(report.Reason, "missing endpoint") {
		t.Fatalf("missing endpoint report = %+v", report)
	}

	inactive, report := MapEntityRelationship(repositories.GraphEntityEdge{
		ID:           "edge-inactive",
		FromEntityID: "entity-1",
		ToEntityID:   "entity-1",
		RelationType: "member_of",
		Status:       domain.StatusInactive,
		UpdatedAt:    now,
	}, nodes, "tidewise")
	if inactive != nil {
		t.Fatalf("inactive relationship = %+v, want nil", inactive)
	}
	if report.Status != RelationshipMapStatusSkipped || !strings.Contains(report.Reason, "inactive") {
		t.Fatalf("inactive report = %+v", report)
	}
}

func TestMapEntityRelationshipPreservesRelationshipProperties(t *testing.T) {
	nodes := map[string]GraphNode{
		"entity-1": {EntityID: "entity-1"},
		"entity-2": {EntityID: "entity-2"},
	}
	now := time.Date(2026, 7, 10, 11, 30, 0, 0, time.UTC)

	relationship, report := MapEntityRelationship(repositories.GraphEntityEdge{
		ID:           "edge-1",
		FromEntityID: "entity-1",
		ToEntityID:   "entity-2",
		RelationType: "member_of",
		EvidenceNote: "基础关系",
		Status:       domain.StatusActive,
		UpdatedAt:    now,
	}, nodes, "tidewise")
	if report.Status != RelationshipMapStatusProjected {
		t.Fatalf("relationship report = %+v, want projected", report)
	}
	if relationship == nil {
		t.Fatal("relationship = nil, want mapped relationship")
	}
	if relationship.EdgeID != "edge-1" || relationship.RelationshipType != "MEMBER_OF" || relationship.OriginalRelationType != "member_of" {
		t.Fatalf("relationship identity = %+v", relationship)
	}
	if relationship.Source != "postgres_entity_edges" || relationship.Confidence != 1 {
		t.Fatalf("relationship source/confidence = %+v", relationship)
	}
	if relationship.Status != "active" || relationship.Namespace != "tidewise" || !relationship.UpdatedAt.Equal(now) {
		t.Fatalf("relationship projection fields = %+v", relationship)
	}
}

func TestMapEntityRelationshipsSkipsDuplicateEdges(t *testing.T) {
	nodes := map[string]GraphNode{
		"entity-1": {EntityID: "entity-1"},
		"entity-2": {EntityID: "entity-2"},
	}
	edges := []repositories.GraphEntityEdge{
		{ID: "edge-1", FromEntityID: "entity-1", ToEntityID: "entity-2", RelationType: "member_of", Status: domain.StatusActive},
		{ID: "edge-1", FromEntityID: "entity-1", ToEntityID: "entity-2", RelationType: "member_of", Status: domain.StatusActive},
	}

	relationships, reports := MapEntityRelationships(edges, nodes, "tidewise")

	if len(relationships) != 1 {
		t.Fatalf("relationships length = %d, want 1", len(relationships))
	}
	if len(reports) != 2 {
		t.Fatalf("reports length = %d, want 2", len(reports))
	}
	if reports[1].Status != RelationshipMapStatusSkipped || !strings.Contains(reports[1].Reason, "duplicate") {
		t.Fatalf("duplicate report = %+v", reports[1])
	}
}
