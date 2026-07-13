package seed

import (
	"strings"
	"testing"
)

func TestBuildFirstBatchDryRunProducesStableIdentitiesAliasesAndCounts(t *testing.T) {
	draft := FirstBatchDraft{
		Nodes: []FirstBatchNodeDraft{
			{EntityKey: "chain_node:ice_snow_industry", CanonicalName: "冰雪产业", OriginalNames: []string{"冰雪经济", "冰雪产业", "冰雪经济"}, Definition: "围绕冰雪资源开发、装备与消费服务形成的产业活动集合", BoundaryNote: "包含冰雪装备和冰雪服务，不包含单次赛事热度标签", WideBoundary: true},
			{EntityKey: "chain_node:aircraft_engine", CanonicalName: "航空发动机", OriginalNames: []string{"航空发动机"}, Definition: "为航空器提供推进力的动力装置及其核心系统"},
		},
		Mappings: []FirstBatchMappingDraft{
			{CanonicalName: "冰雪产业", SourceSystem: "eastmoney", SourceTaxonomyType: "concept_sector", ExternalCode: "BK0001", ExternalName: "冰雪经济", TaxonomyResolved: true},
			{CanonicalName: "冰雪产业", SourceSystem: "ths", SourceTaxonomyType: "concept_sector", ExternalCode: "300001", ExternalName: "冰雪产业", TaxonomyResolved: true},
			{CanonicalName: "航空发动机", SourceSystem: "eastmoney", SourceTaxonomyType: "industry_sector", ExternalCode: "BK0002", ExternalName: "航空发动机", TaxonomyResolved: true},
		},
	}
	expectations := FirstBatchExpectations{Nodes: 2, OriginalNames: 3, Mappings: 3, EastmoneyMappings: 2, THSMappings: 1, DualSourceNodes: 1}
	report := BuildFirstBatchDryRun(draft, FirstBatchIdentitySnapshot{}, expectations)
	if !report.Ready || len(report.Blockers) != 0 || len(report.Conflicts) != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.NodeCount != 2 || report.OriginalNameCount != 3 || report.MappingCount != 3 || report.DualSourceNodeCount != 1 {
		t.Fatalf("counts = %+v", report)
	}
	if got := report.ProviderCounts["eastmoney"]; got != 2 {
		t.Fatalf("eastmoney count = %d", got)
	}
	if got := report.Nodes[0].Aliases; len(got) != 1 || got[0] != "冰雪经济" {
		t.Fatalf("aliases = %v", got)
	}
	if report.Nodes[0].EntityID != entitySeedUUID("chain_node:ice_snow_industry") {
		t.Fatalf("entity id = %q", report.Nodes[0].EntityID)
	}
	if len(report.Mappings) != 3 || report.Mappings[0].ID == "" || report.Mappings[0].EntityID == "" {
		t.Fatalf("mapping report = %+v", report.Mappings)
	}
}

func TestBuildFirstBatchDryRunBlocksUnresolvedTaxonomyAndDefinitionShortcuts(t *testing.T) {
	draft := FirstBatchDraft{
		Nodes:    []FirstBatchNodeDraft{{EntityKey: "chain_node:baijiu", CanonicalName: "白酒", OriginalNames: []string{"白酒"}, Definition: "白酒", WideBoundary: true}},
		Mappings: []FirstBatchMappingDraft{{CanonicalName: "白酒", SourceSystem: "eastmoney", ExternalCode: "BK0896", ExternalName: "白酒", TaxonomyResolved: false}},
	}
	report := BuildFirstBatchDryRun(draft, FirstBatchIdentitySnapshot{}, FirstBatchExpectations{Nodes: 1, OriginalNames: 1, Mappings: 1, EastmoneyMappings: 1})
	if report.Ready {
		t.Fatalf("report unexpectedly ready: %+v", report)
	}
	joined := strings.Join(report.Blockers, "\n")
	for _, expected := range []string{"definition", "boundary_note", "taxonomy"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("blockers %q missing %q", joined, expected)
		}
	}
}

func TestBuildFirstBatchDryRunBlocksIdentityConflicts(t *testing.T) {
	draft := FirstBatchDraft{Nodes: []FirstBatchNodeDraft{{EntityKey: "chain_node:aircraft_engine", CanonicalName: "航空发动机", OriginalNames: []string{"航空发动机"}, Definition: "为航空器提供推进力的动力装置及其核心系统"}}}
	existing := FirstBatchIdentity{EntityID: "legacy-id", EntityKey: "chain_node:aircraft_engine", CanonicalName: "旧航空发动机"}
	snapshot := FirstBatchIdentitySnapshot{ByEntityKey: map[string]FirstBatchIdentity{existing.EntityKey: existing}}
	report := BuildFirstBatchDryRun(draft, snapshot, FirstBatchExpectations{Nodes: 1, OriginalNames: 1})
	if report.Ready || len(report.Conflicts) == 0 || !strings.Contains(report.Conflicts[0], "entity_key") {
		t.Fatalf("report = %+v", report)
	}
}

func TestApprovedFirstBatchExpectations(t *testing.T) {
	got := ApprovedFirstBatchExpectations()
	if got.Nodes != 842 || got.OriginalNames != 950 || got.Mappings != 1156 || got.EastmoneyMappings != 811 || got.THSMappings != 345 || got.DualSourceNodes != 241 {
		t.Fatalf("expectations = %+v", got)
	}
}
