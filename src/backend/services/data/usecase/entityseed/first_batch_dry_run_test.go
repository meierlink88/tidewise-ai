package seed

import (
	"reflect"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
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
	expectations := FirstBatchExpectations{Nodes: 2, OriginalNames: 3, WideBoundaryNodes: 1, Mappings: 3, EastmoneyMappings: 2, THSMappings: 1, DualSourceNodes: 1}
	report := BuildFirstBatchDryRun(draft, FirstBatchIdentitySnapshot{}, expectations)
	if !report.Ready || len(report.Blockers) != 0 || len(report.Conflicts) != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.NodeCount != 2 || report.OriginalNameCount != 3 || report.WideBoundaryNodeCount != 1 || report.MappingCount != 3 || report.DualSourceNodeCount != 1 {
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
	report := BuildFirstBatchDryRun(draft, FirstBatchIdentitySnapshot{}, FirstBatchExpectations{Nodes: 1, OriginalNames: 1, WideBoundaryNodes: 1, Mappings: 1, EastmoneyMappings: 1})
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
	if got.Nodes != 842 || got.OriginalNames != 950 || got.WideBoundaryNodes != 79 || got.Mappings != 1156 || got.EastmoneyMappings != 811 || got.THSMappings != 345 || got.DualSourceNodes != 241 {
		t.Fatalf("expectations = %+v", got)
	}
}

func TestBuildFirstBatchDryRunReportsNodeDriftAndExactIdempotency(t *testing.T) {
	draft := FirstBatchDraft{Nodes: []FirstBatchNodeDraft{{
		EntityKey: "chain_node:aircraft_engine", CanonicalName: "航空发动机", OriginalNames: []string{"航空发动机", "航空引擎"},
		Definition: "为航空器提供推进力的动力装置及其核心系统", BoundaryNote: "包含整机及核心系统，不包含一般航空零部件",
	}}}
	expectations := FirstBatchExpectations{Nodes: 1, OriginalNames: 2}
	wanted := buildFirstBatchIdentity(draft.Nodes[0])

	for _, test := range []struct {
		name       string
		existing   FirstBatchIdentity
		wantAction string
		wantReady  bool
	}{
		{name: "exact unchanged", existing: wanted, wantAction: string(WriteUnchanged), wantReady: true},
		{name: "alias drift updated", existing: withNodeAliases(wanted, []string{"旧别名"}), wantAction: string(WriteUpdated), wantReady: true},
		{name: "definition drift updated", existing: withNodeDefinition(wanted, "旧定义"), wantAction: string(WriteUpdated), wantReady: true},
		{name: "boundary drift updated", existing: withNodeBoundary(wanted, "旧边界"), wantAction: string(WriteUpdated), wantReady: true},
		{name: "wrong type conflict", existing: withNodeType(wanted, domain.EntityTypeTheme), wantAction: "conflict", wantReady: false},
		{name: "inactive conflict", existing: withNodeStatus(wanted, domain.StatusInactive), wantAction: "conflict", wantReady: false},
		{name: "merged conflict", existing: withNodeStatus(wanted, domain.Status("merged")), wantAction: "conflict", wantReady: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := nodeSnapshot(test.existing)
			report := BuildFirstBatchDryRun(draft, snapshot, expectations)
			if got := report.Nodes[0].Action; got != test.wantAction {
				t.Fatalf("action = %q, want %q; report=%+v", got, test.wantAction, report)
			}
			if report.Ready != test.wantReady {
				t.Fatalf("ready = %v, want %v; report=%+v", report.Ready, test.wantReady, report)
			}
		})
	}
}

func TestBuildFirstBatchDryRunTreatsAliasOrderAsStableIdentity(t *testing.T) {
	draft := singleMappingDraft()
	draft.Nodes[0].OriginalNames = []string{" 增材制造 ", "3D打印", "航空发动机", "增材制造"}
	wanted := buildFirstBatchIdentity(draft.Nodes[0])

	reordered := singleMappingDraft()
	reordered.Nodes[0].OriginalNames = []string{"航空发动机", "增材制造", "3D打印"}
	rebuilt := buildFirstBatchIdentity(reordered.Nodes[0])
	if !reflect.DeepEqual(wanted.Aliases, rebuilt.Aliases) {
		t.Fatalf("aliases differ by input order: %v != %v", wanted.Aliases, rebuilt.Aliases)
	}

	report := BuildFirstBatchDryRun(reordered, nodeSnapshot(wanted), FirstBatchExpectations{
		Nodes: 1, OriginalNames: 3, Mappings: 1, EastmoneyMappings: 1,
	})
	if !report.Ready || report.Nodes[0].Action != string(WriteUnchanged) {
		t.Fatalf("report = %+v", report)
	}
}

func TestBuildFirstBatchDryRunBlocksCrossIndexIdentitySnapshotConflict(t *testing.T) {
	draft := singleMappingDraft()
	wanted := buildFirstBatchIdentity(draft.Nodes[0])
	snapshot := nodeSnapshot(wanted)
	snapshot.ByCanonicalName[wanted.CanonicalName] = withNodeAliases(wanted, []string{"冲突别名"})

	report := BuildFirstBatchDryRun(FirstBatchDraft{Nodes: draft.Nodes}, snapshot, FirstBatchExpectations{Nodes: 1, OriginalNames: 1})
	if report.Ready || report.Nodes[0].Action != "conflict" || !strings.Contains(strings.Join(report.Conflicts, "\n"), "snapshot indexes disagree") {
		t.Fatalf("report = %+v", report)
	}
}

func TestBuildFirstBatchDryRunBlocksIncompleteNodeSnapshotIndexes(t *testing.T) {
	draft := singleMappingDraft()
	wanted := buildFirstBatchIdentity(draft.Nodes[0])
	snapshot := FirstBatchIdentitySnapshot{ByEntityKey: map[string]FirstBatchIdentity{wanted.EntityKey: wanted}}

	report := BuildFirstBatchDryRun(FirstBatchDraft{Nodes: draft.Nodes}, snapshot, FirstBatchExpectations{Nodes: 1, OriginalNames: 1})
	if report.Ready || report.Nodes[0].Action != "conflict" || !strings.Contains(strings.Join(report.Conflicts, "\n"), "snapshot indexes are incomplete") {
		t.Fatalf("report = %+v", report)
	}
}

func TestBuildFirstBatchDryRunReportsMappingSnapshotActionsAndConflicts(t *testing.T) {
	draft := singleMappingDraft()
	wantedNode := buildFirstBatchIdentity(draft.Nodes[0])
	wantedMapping := firstBatchMappingSnapshot(draft.Mappings[0], wantedNode.EntityID)
	expectations := FirstBatchExpectations{Nodes: 1, OriginalNames: 1, Mappings: 1, EastmoneyMappings: 1}

	for _, test := range []struct {
		name       string
		snapshot   FirstBatchExternalIdentifierSnapshot
		wantAction string
		wantReady  bool
	}{
		{name: "created", wantAction: string(WriteCreated), wantReady: true},
		{name: "unchanged", snapshot: externalSnapshot(wantedMapping), wantAction: string(WriteUnchanged), wantReady: true},
		{name: "incomplete snapshot conflict", snapshot: FirstBatchExternalIdentifierSnapshot{ByIdentity: map[string]FirstBatchExternalIdentifier{externalIdentifierIdentity(wantedMapping.SourceSystem, wantedMapping.SourceTaxonomyType, wantedMapping.ExternalCode): wantedMapping}}, wantAction: "conflict", wantReady: false},
		{name: "name drift updated", snapshot: externalSnapshot(withMappingName(wantedMapping, "旧名称")), wantAction: string(WriteUpdated), wantReady: true},
		{name: "status drift updated", snapshot: externalSnapshot(withMappingStatus(wantedMapping, domain.StatusInactive)), wantAction: string(WriteUpdated), wantReady: true},
		{name: "tuple rebound conflict", snapshot: externalSnapshot(withMappingEntity(wantedMapping, entitySeedUUID("chain_node:other"))), wantAction: "conflict", wantReady: false},
		{name: "tuple deterministic id mismatch", snapshot: FirstBatchExternalIdentifierSnapshot{ByIdentity: map[string]FirstBatchExternalIdentifier{externalIdentifierIdentity(wantedMapping.SourceSystem, wantedMapping.SourceTaxonomyType, wantedMapping.ExternalCode): withMappingID(wantedMapping, entitySeedUUID("chain_node:other"))}}, wantAction: "conflict", wantReady: false},
		{name: "deterministic id collision", snapshot: FirstBatchExternalIdentifierSnapshot{ByID: map[string]FirstBatchExternalIdentifier{wantedMapping.ID: withMappingCode(wantedMapping, "OTHER")}}, wantAction: "conflict", wantReady: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := nodeSnapshot(wantedNode)
			snapshot.ExternalIdentifiers = test.snapshot
			report := BuildFirstBatchDryRun(draft, snapshot, expectations)
			if len(report.Mappings) != 1 || report.Mappings[0].Action != test.wantAction {
				t.Fatalf("mappings = %+v, want action %q", report.Mappings, test.wantAction)
			}
			if report.Ready != test.wantReady {
				t.Fatalf("ready = %v, want %v; report=%+v", report.Ready, test.wantReady, report)
			}
		})
	}
}

func TestBuildFirstBatchDryRunBlocksWideBoundaryCountLoss(t *testing.T) {
	draft := FirstBatchDraft{Nodes: []FirstBatchNodeDraft{{EntityKey: "chain_node:a", CanonicalName: "A", OriginalNames: []string{"A"}, Definition: "一种可判定的产业对象"}}}
	report := BuildFirstBatchDryRun(draft, FirstBatchIdentitySnapshot{}, FirstBatchExpectations{Nodes: 1, OriginalNames: 1, WideBoundaryNodes: 1})
	if report.Ready || report.WideBoundaryNodeCount != 0 || !strings.Contains(strings.Join(report.Blockers, "\n"), "wide_boundary_nodes") {
		t.Fatalf("report = %+v", report)
	}
}

func nodeSnapshot(identity FirstBatchIdentity) FirstBatchIdentitySnapshot {
	return FirstBatchIdentitySnapshot{
		ByEntityID:      map[string]FirstBatchIdentity{identity.EntityID: identity},
		ByEntityKey:     map[string]FirstBatchIdentity{identity.EntityKey: identity},
		ByCanonicalName: map[string]FirstBatchIdentity{identity.CanonicalName: identity},
	}
}

func withNodeAliases(identity FirstBatchIdentity, aliases []string) FirstBatchIdentity {
	identity.Aliases = aliases
	return identity
}
func withNodeDefinition(identity FirstBatchIdentity, definition string) FirstBatchIdentity {
	identity.Definition = definition
	return identity
}
func withNodeBoundary(identity FirstBatchIdentity, boundary string) FirstBatchIdentity {
	identity.BoundaryNote = boundary
	return identity
}
func withNodeType(identity FirstBatchIdentity, entityType domain.EntityType) FirstBatchIdentity {
	identity.EntityType = entityType
	return identity
}
func withNodeStatus(identity FirstBatchIdentity, status domain.Status) FirstBatchIdentity {
	identity.Status = status
	return identity
}

func singleMappingDraft() FirstBatchDraft {
	return FirstBatchDraft{
		Nodes:    []FirstBatchNodeDraft{{EntityKey: "chain_node:3d_printing", CanonicalName: "3D打印", OriginalNames: []string{"3D打印"}, Definition: "通过逐层增材方式制造实体部件的工艺与设备类别"}},
		Mappings: []FirstBatchMappingDraft{{CanonicalName: "3D打印", SourceSystem: "eastmoney", SourceTaxonomyType: "concept_sector", ExternalCode: "BK0619", ExternalName: "3D打印", TaxonomyResolved: true}},
	}
}

func firstBatchMappingSnapshot(mapping FirstBatchMappingDraft, entityID string) FirstBatchExternalIdentifier {
	identity := externalIdentifierIdentity(mapping.SourceSystem, mapping.SourceTaxonomyType, mapping.ExternalCode)
	return FirstBatchExternalIdentifier{ID: externalIdentifierSeedUUID(identity), EntityID: entityID, SourceSystem: mapping.SourceSystem, SourceTaxonomyType: mapping.SourceTaxonomyType, ExternalCode: mapping.ExternalCode, ExternalName: mapping.ExternalName, Status: domain.StatusActive}
}

func externalSnapshot(identifier FirstBatchExternalIdentifier) FirstBatchExternalIdentifierSnapshot {
	return FirstBatchExternalIdentifierSnapshot{ByIdentity: map[string]FirstBatchExternalIdentifier{externalIdentifierIdentity(identifier.SourceSystem, identifier.SourceTaxonomyType, identifier.ExternalCode): identifier}, ByID: map[string]FirstBatchExternalIdentifier{identifier.ID: identifier}}
}

func withMappingName(identifier FirstBatchExternalIdentifier, name string) FirstBatchExternalIdentifier {
	identifier.ExternalName = name
	return identifier
}
func withMappingStatus(identifier FirstBatchExternalIdentifier, status domain.Status) FirstBatchExternalIdentifier {
	identifier.Status = status
	return identifier
}
func withMappingEntity(identifier FirstBatchExternalIdentifier, entityID string) FirstBatchExternalIdentifier {
	identifier.EntityID = entityID
	return identifier
}
func withMappingID(identifier FirstBatchExternalIdentifier, id string) FirstBatchExternalIdentifier {
	identifier.ID = id
	return identifier
}
func withMappingCode(identifier FirstBatchExternalIdentifier, code string) FirstBatchExternalIdentifier {
	identifier.ExternalCode = code
	return identifier
}
